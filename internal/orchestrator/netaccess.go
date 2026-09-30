package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// ReconcileNetworkAccess enqueues a builder-level ConfigureNetworkAccess for any
// team whose networks are all deployed and whose desired visible_from policy
// isn't already applied. It is the "when" half of visible_from enforcement;
// the runner's executeConfigureNetworkAccess is
// the "what" (it builds the per-network specs and calls the builder), the same
// orchestrator-decides / runner-acts split deploy already uses.
//
// The trigger is a fingerprint of the team's networks (names + CIDRs +
// visible_from). It fires once the networks are up, and again only if that
// fingerprint changes (a content edit, or a redeploy that recreated a network) --
// the builder call is idempotent, so re-asserting an unchanged policy is skipped.
func ReconcileNetworkAccess(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) error {
	q := db.New(pool)
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading build: %w", err)
	}
	teams, err := q.ListTeamsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing teams: %w", err)
	}

	for _, team := range teams {
		objs, err := q.ListDeployedObjectsByTeam(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("team %d: listing objects: %w", team.TeamNumber, err)
		}

		// Only proceed once every one of the team's networks is deployed --
		// peering + ACLs need all of them to exist.
		var netObjs []db.DeployedObject
		allDeployed := true
		for _, o := range objs {
			if o.Kind != "network" {
				continue
			}
			netObjs = append(netObjs, o)
			if o.Status != "deployed" && o.Status != "finished" {
				allDeployed = false
			}
		}
		if len(netObjs) == 0 || !allDeployed {
			continue
		}

		fp, err := networkAccessFingerprint(ctx, q, build.ContentRevisionID, netObjs)
		if err != nil {
			return fmt.Errorf("team %d: fingerprinting network access: %w", team.TeamNumber, err)
		}
		teamStr := strconv.FormatInt(int64(team.TeamNumber), 10)
		present, err := q.HasConfigureNetworkAccessTask(ctx, db.HasConfigureNetworkAccessTaskParams{
			BuildID: buildID, Team: teamStr, Fingerprint: fp,
		})
		if err != nil {
			return fmt.Errorf("team %d: checking for network-access task: %w", team.TeamNumber, err)
		}
		if present {
			continue
		}

		payload, _ := json.Marshal(map[string]string{"team": teamStr, "fingerprint": fp})
		task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: buildID, Kind: "configure_network_access", Payload: payload})
		if err != nil {
			return fmt.Errorf("team %d: creating configure_network_access task: %w", team.TeamNumber, err)
		}
		q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: buildID, TaskID: task.ID, Kind: "network_access.reconciled",
			Message: fmt.Sprintf("applying visible_from policy for team %d", team.TeamNumber),
			Payload: []byte("{}"),
		})
	}
	return nil
}

// networkAccessFingerprint hashes the team's desired inter-network policy -- each
// network's name, CIDR, and sorted visible_from -- so the reconciler re-runs
// exactly when that policy changes and not otherwise.
func networkAccessFingerprint(ctx context.Context, q *db.Queries, revID pgtype.UUID, netObjs []db.DeployedObject) (string, error) {
	lines := make([]string, 0, len(netObjs))
	for _, o := range netObjs {
		net, err := q.GetNetworkByRevisionAndName(ctx, db.GetNetworkByRevisionAndNameParams{
			ContentRevisionID: revID, Name: o.ObjectName,
		})
		if err != nil {
			return "", fmt.Errorf("network %s: %w", o.ObjectName, err)
		}
		var visibleFrom []string
		json.Unmarshal(net.VisibleFrom, &visibleFrom)
		sort.Strings(visibleFrom)
		lines = append(lines, fmt.Sprintf("%s|%s|%s", net.Name, net.Cidr, strings.Join(visibleFrom, ",")))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), nil
}
