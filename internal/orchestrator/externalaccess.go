package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// ReconcileExternalAccess enqueues a builder-level ConfigureExternalAccess for
// any team whose content declares `public:` ports and whose hosts are deployed,
// re-running when public ports or ready instance identities change (a fingerprint rides
// in the task payload). It mirrors ReconcileNetworkAccess exactly -- the "when"
// half of external-access enforcement -- and the runner's
// executeConfigureExternalAccess is the "what" (gather the public hosts, call the
// builder, record the endpoints). Needs the content checkout because `public:`
// lives in the topology, not a DB table.
func ReconcileExternalAccess(ctx context.Context, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID) error {
	q := db.New(pool)
	c, err := loader.LoadCached(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	fp := externalAccessFingerprint(c)
	// Separate-NIC MicroCloud access also reconciles removals and newly
	// deployed copies. Other builders retain their existing behavior.
	nicMode, err := microcloudPublicNICMode(ctx, q, buildID)
	if err != nil {
		return err
	}
	if fp == "" && !nicMode {
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
		teamFP, ready := externalAccessTeamFingerprint(c, objs, fp, nicMode)
		if !ready {
			continue
		}
		teamStr := strconv.FormatInt(int64(team.TeamNumber), 10)
		present, err := q.HasConfigureExternalAccessTask(ctx, db.HasConfigureExternalAccessTaskParams{
			BuildID: buildID, Team: teamStr, Fingerprint: teamFP,
		})
		if err != nil {
			return fmt.Errorf("team %d: checking for external-access task: %w", team.TeamNumber, err)
		}
		if present {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"team": teamStr, "fingerprint": teamFP})
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

func microcloudPublicNICMode(ctx context.Context, q *db.Queries, buildID pgtype.UUID) (bool, error) {
	b, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return false, err
	}
	if !b.ConfiguredBuildID.Valid {
		return false, nil
	}
	cb, err := q.GetConfiguredBuild(ctx, b.ConfiguredBuildID)
	if err != nil {
		return false, err
	}
	cfg, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // a removed/unconfigured builder cannot enable NIC mode
	}
	if err != nil {
		return false, err
	}
	var public struct {
		Type string `json:"type"`
	}
	if cfg.Kind == "microcloud" && len(cfg.MicrocloudPublicAccess) > 0 {
		if err := json.Unmarshal(cfg.MicrocloudPublicAccess, &public); err != nil {
			return false, err
		}
	}
	return public.Type == "nic", nil
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

// Include the identities of ready targets, not guest step progress. A private
// host becoming ready cannot consume the public task, and later public VMs
// produce a new fingerprint even when the content itself has not changed.
func externalAccessTeamFingerprint(c *loader.Content, objs []db.DeployedObject, contentFP string, nicMode bool) (string, bool) {
	lines := []string{"external-access-ready-v2", contentFP}
	for _, o := range objs {
		if (o.Kind != "host" && o.Kind != "container") || db.StrOrEmpty(o.ExternalRef) == "" || !builder.ExternalAccessDeployReady(o.Status) {
			continue
		}
		public := nicMode
		if h := findHost(c, o.ObjectName); h != nil && h.Public != nil {
			public = public || len(h.Public.TCP)+len(h.Public.UDP) > 0
		}
		if ct := findContainer(c, o.ObjectName); ct != nil && ct.Public != nil {
			public = public || len(ct.Public.TCP)+len(ct.Public.UDP) > 0
		}
		if public {
			lines = append(lines, o.ID.String()+"|"+db.StrOrEmpty(o.ExternalRef))
		}
	}
	if len(lines) == 2 {
		return "", false
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:]), true
}
