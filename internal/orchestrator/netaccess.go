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

		// Only proceed once every one of the team's networks is deployed AND every
		// guest has settled -- peering + ACLs need all networks to exist, and a
		// per-NIC ACL applied before its guest is created would silently skip that
		// guest. A guest still being created (pending/deploying, or deployed but
		// no reference recorded yet) is a WAIT, not a reason to run a partial
		// policy; a guest that terminally failed to deploy has no NIC to configure,
		// so it neither blocks nor joins the ACL set. The runner's own readiness
		// defer is the safety net for a guest that flips unready after this queues.
		var netObjs []db.DeployedObject
		var guestRefs []string
		allDeployed, allSettled := true, true
		for _, o := range objs {
			switch o.Kind {
			case "network":
				netObjs = append(netObjs, o)
				if o.Status != "deployed" && o.Status != "finished" {
					allDeployed = false
				}
			case "host", "container":
				switch o.Status {
				case "pending", "deploying":
					allSettled = false
				case "deploy_failed":
					// terminal: no instance to ACL -- excluded, doesn't block
				default: // running/building/finished/deployed/build_failed/invalid
					if ref := db.StrOrEmpty(o.ExternalRef); ref != "" {
						guestRefs = append(guestRefs, deployIdentity(ref, o.DeployedAt))
					} else {
						allSettled = false // deployed but still settling (no ref yet)
					}
				}
			}
		}
		if len(netObjs) == 0 || !allDeployed || !allSettled {
			continue
		}

		fp, err := networkAccessFingerprint(ctx, q, build.ContentRevisionID, netObjs, guestRefs)
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
// network's name, CIDR, and sorted visible_from -- plus the deploy identity of
// every network and ready guest. The reconciler re-runs when the policy changes
// AND whenever a network or guest is redeployed, so a recreated network gets its
// peers back and a replaced instance's NIC gets its ACL re-applied; an unchanged
// policy over unchanged instances never re-runs (the builder call is idempotent).
//
// The identity is the instance reference plus when it was deployed, not the
// reference alone: builders name instances deterministically, so a rebuild
// recreates the same reference. With refs alone a rebuilt network looked
// unchanged and its peering was never restored (cptc12-se team 2, 2026-11-04).
func networkAccessFingerprint(ctx context.Context, q *db.Queries, revID pgtype.UUID, netObjs []db.DeployedObject, guestRefs []string) (string, error) {
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
		lines = append(lines, fmt.Sprintf("%s|%s|%s|%s", net.Name, net.Cidr, strings.Join(visibleFrom, ","),
			deployIdentity(db.StrOrEmpty(o.ExternalRef), o.DeployedAt)))
	}
	sort.Strings(lines)
	refs := append([]string(nil), guestRefs...)
	sort.Strings(refs)
	payload := strings.Join(lines, "\n") + "\x00guests\x00" + strings.Join(refs, ",")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:]), nil
}

// deployIdentity names one deployment of an instance: its reference and when it
// was deployed. A redeploy under the same reference yields a new identity.
func deployIdentity(ref string, deployedAt pgtype.Timestamptz) string {
	if !deployedAt.Valid {
		return ref
	}
	return fmt.Sprintf("%s@%d", ref, deployedAt.Time.UnixNano())
}
