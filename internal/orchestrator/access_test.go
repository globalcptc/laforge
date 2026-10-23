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
		// Reduce moves the end to 16:30 (an open override). Before the new end the
		// team is open; at/after it the team stays CLOSED for the rest of the
		// window it trimmed -- the schedule does NOT re-open 16:30-17:00.
		{
			"reduced end: open before the new end",
			db.Team{AccessOverrideState: "open", AccessOverrideUntil: ts(at(t, "2026-10-01T16:30:00Z"))},
			"2026-10-01T16:00:00Z", "open", true,
		},
		{
			"reduced end: closed for the rest of the trimmed window",
			db.Team{AccessOverrideState: "open", AccessOverrideUntil: ts(at(t, "2026-10-01T16:30:00Z"))},
			"2026-10-01T16:45:00Z", "closed", true,
		},
	}
	for _, c := range cases {
		gotState, gotEnforce := desiredAccess(w, c.team, at(t, c.now))
		if gotState != c.wantState || gotEnforce != c.wantEnforce {
			t.Errorf("%s: desiredAccess = (%q, %v), want (%q, %v)", c.name, gotState, gotEnforce, c.wantState, c.wantEnforce)
		}
	}
}

func TestDesiredAccessReducedEndReopensNextWindow(t *testing.T) {
	// Two daily 9-5 windows. A reduced end on day 1 (open override until 16:30)
	// keeps day 1 closed after 16:30 but must NOT suppress day 2 -- a genuinely
	// new window still opens normally.
	windows := []schedule.AccessWindow{
		{Open: at(t, "2026-10-01T09:00:00Z"), Close: at(t, "2026-10-01T17:00:00Z")},
		{Open: at(t, "2026-10-02T09:00:00Z"), Close: at(t, "2026-10-02T17:00:00Z")},
	}
	team := db.Team{AccessOverrideState: "open", AccessOverrideUntil: ts(at(t, "2026-10-01T16:30:00Z"))}
	cases := []struct {
		name, now, want string
	}{
		{"day-1 after trimmed end", "2026-10-01T16:45:00Z", "closed"},
		{"overnight", "2026-10-02T03:00:00Z", "closed"},
		{"day-2 new window opens", "2026-10-02T12:00:00Z", "open"},
	}
	for _, c := range cases {
		if got, _ := desiredAccess(windows, team, at(t, c.now)); got != c.want {
			t.Errorf("%s: desiredAccess = %q, want %q", c.name, got, c.want)
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

// deployReadyHost creates a host object that's up at the hoster (running, with
// an external_ref), the state teamHosterReady requires before access is driven.
func deployReadyHost(t *testing.T, q *db.Queries, teamID pgtype.UUID, name string) db.DeployedObject {
	t.Helper()
	ctx := context.Background()
	o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: teamID, Kind: "host", ObjectName: name, AsName: db.StrPtr(name), Fingerprint: "f",
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject %s: %v", name, err)
	}
	if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
		ID: o.ID, ExternalRef: db.StrPtr("ref-" + name), Fingerprint: "f",
	}); err != nil {
		t.Fatalf("MarkDeployedObjectRunning %s: %v", name, err)
	}
	return o
}

func countTasksOfKind(t *testing.T, pool *pgxpool.Pool, buildID pgtype.UUID, kind string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM task WHERE build_id = $1 AND kind = $2", buildID, kind).Scan(&n); err != nil {
		t.Fatalf("counting %s tasks: %v", kind, err)
	}
	return n
}

// markTeamClosed simulates a completed close that the runner applied: a done
// close_access task (so HasCompletedTeamAccessTask is true) plus the 'closed'
// label executeAccess would have written. This is a team the reconciler should
// trust as genuinely closed.
func markTeamClosed(t *testing.T, pool *pgxpool.Pool, q *db.Queries, build db.Build, team db.Team) {
	t.Helper()
	ctx := context.Background()
	payload := fmt.Sprintf(`{"team":"%d"}`, team.TeamNumber)
	task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: build.ID, Kind: "close_access", Payload: []byte(payload)})
	if err != nil {
		t.Fatalf("seed close task: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE task SET status = 'done' WHERE id = $1", task.ID); err != nil {
		t.Fatalf("mark seed close done: %v", err)
	}
	if _, err := q.SetTeamAccessState(ctx, db.SetTeamAccessStateParams{ID: team.ID, AccessState: "closed"}); err != nil {
		t.Fatalf("seed closed label: %v", err)
	}
}

// TestReconcileAccessEnforcesWindows proves the enforcer creates the real
// open/close tasks the runner executes, that the access_state label is kept
// HONEST (a deployed team reads 'open' because it really is), that a team
// already in the desired state enqueues nothing, and the dedup (a slow builder
// call doesn't get a duplicate every pass).
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
	// Team 2 was genuinely closed by an applied close -- the reconciler should
	// trust that label and open it when the window is open.
	markTeamClosed(t, pool, q, build, team2)

	// Both teams must be hoster-ready (instances deployed) before the reconciler
	// drives access OR trusts their label -- nothing to open/close otherwise.
	deployReadyHost(t, q, team1.ID, "web-1")
	deployReadyHost(t, q, team2.ID, "web-2")

	// Inside the window (want=open): team 1 is deployed, never explicitly set, so
	// its label is truthed to 'open' (it really is) and -- already matching the
	// desire -- needs no task. Team 2 was genuinely closed, so it gets opened.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (in window): %v", err)
	}
	t1, _ := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: build.ID, TeamNumber: 1})
	if t1.AccessState != "open" {
		t.Fatalf("team 1 access_state = %q after deploy, want open (label must show the real state)", t1.AccessState)
	}
	if got := countTasksOfKind(t, pool, build.ID, "open_access"); got != 1 {
		t.Fatalf("open_access tasks after first pass = %d, want 1 (team 2 reopened)", got)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks after first pass = %d, want 1 (only the seeded one)", got)
	}

	// Second pass, still inside the window: team 2's open task is still pending,
	// so no duplicate is created.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (dedup pass): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "open_access"); got != 1 {
		t.Fatalf("open_access tasks after dedup pass = %d, want 1 (no duplicate)", got)
	}

	// A scheduled close is held while the build is still deploying/building (its
	// agents still need egress for downloads/steps) -- so even well after the
	// window, a building build closes nothing.
	if err := ReconcileAccess(ctx, pool, build.ID, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("ReconcileAccess (after window, still building): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks while building = %d, want 1 (only the seeded one -- closes held until built)", got)
	}

	// Once the build is fully built, the schedule enforces: team 1 (open, never
	// explicitly closed) must be closed now.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "finished"}); err != nil {
		t.Fatalf("SetBuildStatus finished: %v", err)
	}
	if err := ReconcileAccess(ctx, pool, build.ID, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("ReconcileAccess (after window, built): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 2 {
		t.Fatalf("close_access tasks after window = %d, want 2 (seeded + team 1)", got)
	}
}

// TestReconcileAccessClosesDeployedButNeverAppliedTeam is the Fix-1 regression:
// a team's access_state defaults to 'closed', but its freshly deployed instances
// are actually OPEN and nothing has ever applied a close. The old reconciler saw
// want=closed == state=closed and did nothing, so the team ran the whole
// pre-competition window externally reachable while the UI reported it closed.
// The reconciler must now issue a real close (reality != applied), then treat the
// team as converged only once that close task has actually completed.
func TestReconcileAccessClosesDeployedButNeverAppliedTeam(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "access-never-applied")

	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	if team.AccessState != "closed" {
		t.Fatalf("fresh team access_state = %q, want closed (the default)", team.AccessState)
	}
	// A live manual override makes the desire 'closed' with no schedule needed.
	if _, err := q.SetTeamAccessOverride(ctx, db.SetTeamAccessOverrideParams{
		ID: team.ID, AccessOverrideState: "closed",
		AccessOverrideUntil: ts(time.Now().Add(24 * time.Hour)),
	}); err != nil {
		t.Fatalf("SetTeamAccessOverride: %v", err)
	}

	now := time.Now().UTC()

	// Before any instance is deployed the reconciler must NOT fire a close: a
	// no-op close with nothing to act on would falsely mark the team applied,
	// and the real instances would then come up open and stay open.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (pre-deploy): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 0 {
		t.Fatalf("close_access tasks before deploy = %d, want 0 (nothing deployed yet)", got)
	}

	// Deploy an instance -> the team is hoster-ready and its reality is OPEN.
	deployReadyHost(t, q, team.ID, "web-1")

	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (deployed): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks after deploy = %d, want 1 -- a closed-labeled but never-closed team must actually be closed", got)
	}

	// With the close task still pending, no duplicate piles on.
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (dedup): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks after dedup pass = %d, want 1 (no duplicate)", got)
	}

	// Complete the close task AND record the 'closed' label, exactly as the
	// runner's executeAccess does -> now it's genuinely applied and the
	// reconciler leaves the converged team alone.
	if _, err := pool.Exec(ctx, "UPDATE task SET status = 'done' WHERE build_id = $1 AND kind = 'close_access'", build.ID); err != nil {
		t.Fatalf("completing close task: %v", err)
	}
	if _, err := q.SetTeamAccessState(ctx, db.SetTeamAccessStateParams{ID: team.ID, AccessState: "closed"}); err != nil {
		t.Fatalf("recording closed label: %v", err)
	}
	if err := ReconcileAccess(ctx, pool, build.ID, now); err != nil {
		t.Fatalf("ReconcileAccess (converged): %v", err)
	}
	if got := countTasksOfKind(t, pool, build.ID, "close_access"); got != 1 {
		t.Fatalf("close_access tasks after convergence = %d, want 1 (no new task once truly applied)", got)
	}
}
