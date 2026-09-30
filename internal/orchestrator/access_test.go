package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/schedule"
)

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

func ts(tm time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: tm, Valid: true} }

// One 9-to-5 window on 2026-10-01.
func oneWindow(t *testing.T) []schedule.AccessWindow {
	return []schedule.AccessWindow{{Open: at(t, "2026-10-01T09:00:00Z"), Close: at(t, "2026-10-01T17:00:00Z")}}
}

func TestDesiredAccess(t *testing.T) {
	w := oneWindow(t)
	cases := []struct {
		name        string
		team        db.Team
		now         string
		wantState   string
		wantEnforce bool
	}{
		{"before window closed", db.Team{}, "2026-10-01T08:00:00Z", "closed", true},
		{"inside window open", db.Team{}, "2026-10-01T12:00:00Z", "open", true},
		{"after window closed", db.Team{}, "2026-10-01T20:00:00Z", "closed", true},
		{
			"active open override beats schedule (outside window)",
			db.Team{AccessOverrideState: "open", AccessOverrideUntil: ts(at(t, "2026-10-01T22:00:00Z"))},
			"2026-10-01T20:00:00Z", "open", true,
		},
		{
			"active closed override beats schedule (inside window)",
			db.Team{AccessOverrideState: "closed", AccessOverrideUntil: ts(at(t, "2026-10-01T13:00:00Z"))},
			"2026-10-01T12:00:00Z", "closed", true,
		},
		{
			"lapsed override falls back to schedule",
			db.Team{AccessOverrideState: "open", AccessOverrideUntil: ts(at(t, "2026-10-01T18:00:00Z"))},
			"2026-10-01T20:00:00Z", "closed", true,
		},
		{
			"override with no expiry holds until changed",
			db.Team{AccessOverrideState: "open"},
			"2026-10-01T20:00:00Z", "open", true,
		},
	}
	for _, c := range cases {
		gotState, gotEnforce := desiredAccess(w, c.team, at(t, c.now))
		if gotState != c.wantState || gotEnforce != c.wantEnforce {
			t.Errorf("%s: desiredAccess = (%q, %v), want (%q, %v)", c.name, gotState, gotEnforce, c.wantState, c.wantEnforce)
		}
	}
}

func TestDesiredAccessNoWindows(t *testing.T) {
	// No schedule and no override: nothing to enforce, leave the team alone.
	if _, enforce := desiredAccess(nil, db.Team{AccessState: "open"}, at(t, "2026-10-01T12:00:00Z")); enforce {
		t.Error("no windows + no override should not enforce")
	}
	// No schedule but an active override still enforces the override.
	got, enforce := desiredAccess(nil, db.Team{AccessOverrideState: "closed", AccessOverrideUntil: ts(at(t, "2026-10-01T13:00:00Z"))}, at(t, "2026-10-01T12:00:00Z"))
	if !enforce || got != "closed" {
		t.Errorf("override with no windows = (%q, %v), want (closed, true)", got, enforce)
	}
}

func countTasksOfKind(t *testing.T, pool *pgxpool.Pool, buildID pgtype.UUID, kind string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM task WHERE build_id = $1 AND kind = $2", buildID, kind).Scan(&n); err != nil {
		t.Fatalf("counting %s tasks: %v", kind, err)
	}
	return n
}

// TestReconcileAccessEnforcesWindows proves the enforcer actually creates the
// real open/close tasks the runner executes -- the fix for the audit's
// top finding, that authored access: windows were never enforced. It also proves
// the dedup (a slow builder call doesn't get a duplicate every pass) and that a
// team already in the desired state enqueues nothing.
func TestReconcileAccessEnforcesWindows(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "reconcile-access")

	// A window open right now (realNow +/- 1h), persisted on the environment
	// row the reconciler reads (no content checkout needed).
	now := time.Now().UTC()
	access := fmt.Sprintf(`[{"open":%q,"close":%q}]`,
		now.Add(-time.Hour).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339))
	if _, err := q.CreateEnvironment(ctx, db.CreateEnvironmentParams{
		ContentRevisionID: build.ContentRevisionID, Path: "lm-test.yaml", Name: "lm-test", Teams: 2,
		Dns:    []byte("{}"),
		Access: []byte(access), Vars: []byte("{}"), Tags: []byte("{}"), Findings: []byte("[]"),
	}); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM environment WHERE content_revision_id = $1", build.ContentRevisionID)
	})

	team1, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam 1: %v", err)
	}
	if team1.AccessState != "closed" {
		t.Fatalf("fresh team access_state = %q, want closed", team1.AccessState)
	}
	team2, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 2})
	if err != nil {
		t.Fatalf("EnsureTeam 2: %v", err)
	}
	// Pretend team 2 already got opened (e.g. an earlier pass), so we can later
	// prove it gets closed once outside the window.
	if _, err := q.SetTeamAccessState(ctx, db.SetTeamAccessStateParams{ID: team2.ID, AccessState: "open"}); err != nil {
		t.Fatalf("SetTeamAccessState team2 open: %v", err)
	}

	// Inside the window: team 1 (closed) must be opened; team 2 (already open)
	// must be left alone.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (in window): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "open_access"); got != 1 {
		t.Fatalf("open_access tasks after first pass = %d, want 1 (team 1)", got)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 0 {
		t.Fatalf("close_access tasks after first pass = %d, want 0 (team 2 already open)", got)
	}

	// Second pass, still inside the window: the open task for team 1 is still
	// pending, so no duplicate is created.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (dedup pass): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "open_access"); got != 1 {
		t.Fatalf("open_access tasks after dedup pass = %d, want 1 (no duplicate)", got)
	}

	// Now well after the window: team 2 (still open) must be closed. Team 1 is
	// closed already, so it gets no close task.
	if err := ReconcileAccess(ctx, pool, build.ID, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("ReconcileAccess (after window): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks after window = %d, want 1 (team 2)", got)
	}
}
