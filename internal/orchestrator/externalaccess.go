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
	"github.com/globalcptc/laforge/internal/loader"
)

// ReconcileExternalAccess enqueues a builder-level ConfigureExternalAccess for
// any team whose content declares `public:` ports and whose hosts are deployed,
// re-running only when that public-port set changes (a fingerprint of it rides
// in the task payload). It mirrors ReconcileNetworkAccess exactly -- the "when"
// half of external-access enforcement -- and the runner's
// executeConfigureExternalAccess is the "what" (gather the public hosts, call the
// builder, record the endpoints). Needs the content checkout because `public:`
// lives in the topology, not a DB table.
func ReconcileExternalAccess(ctx context.Context, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID) error {
	q := db.New(pool)
	c, err := loader.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	fp := externalAccessFingerprint(c)
	if fp == "" {
		return nil // no host/container declares `public:` ports
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
		// Only once at least one of the team's hosts is actually deployed at the
		// hoster (has an external ref) -- external access attaches to live instances.
		deployed := false
		for _, o := range objs {
			if (o.Kind == "host" || o.Kind == "container") && db.StrOrEmpty(o.ExternalRef) != "" &&
				(o.Status == "running" || o.Status == "building" || o.Status == "finished" || o.Status == "deployed") {
				deployed = true
				break
			}
		}
		if !deployed {
			continue
		}
		teamStr := strconv.FormatInt(int64(team.TeamNumber), 10)
		present, err := q.HasConfigureExternalAccessTask(ctx, db.HasConfigureExternalAccessTaskParams{
			BuildID: buildID, Team: teamStr, Fingerprint: fp,
		})
		if err != nil {
			return fmt.Errorf("team %d: checking for external-access task: %w", team.TeamNumber, err)
		}
		if present {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"team": teamStr, "fingerprint": fp})
		task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: buildID, Kind: "configure_external_access", Payload: payload})
		if err != nil {
			return fmt.Errorf("team %d: creating configure_external_access task: %w", team.TeamNumber, err)
		}
		q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: buildID, TaskID: task.ID, Kind: "external_access.reconciled",
			Message: fmt.Sprintf("realizing public: ports for team %d", team.TeamNumber),
			Payload: []byte("{}"),
		})
	}
	return nil
}

// externalAccessFingerprint hashes every host/container's `public:` (name,
// protocol, port), so the reconciler re-runs exactly when the public-port set
// changes and not otherwise. Empty when nothing is public. `public:` lives on
// the host definition now, so this is content-level (team-agnostic -- every team
// deploys the same hosts), which is right: the dedup is still per team task.
func externalAccessFingerprint(c *loader.Content) string {
	var lines []string
	add := func(name string, public *loader.Ports) {
		if public == nil {
			return
		}
		for _, p := range public.TCP {
			lines = append(lines, name+"|tcp|"+p)
		}
		for _, p := range public.UDP {
			lines = append(lines, name+"|udp|"+p)
		}
	}
	for _, h := range c.Hosts {
		add(h.Name, h.Public)
	}
	for _, ct := range c.Containers {
		add(ct.Name, ct.Public)
	}
	if len(lines) == 0 {
		return ""
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}
