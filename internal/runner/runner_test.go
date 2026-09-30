package runner

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// drainToConvergence interleaves Reconcile and drainQueue until every object is
// running, returning the total tasks drained. depends_on ordering means a
// dependent's deploy task appears only after its dependency deploys, so a
// single Reconcile+drain can't deploy the whole example in one shot.
func drainToConvergence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, q *db.Queries, r *Runner, repo string, buildID pgtype.UUID) int {
	t.Helper()
	total := 0
	for pass := 0; pass < 10; pass++ {
		if err := orchestrator.Reconcile(ctx, pool, repo, buildID); err != nil {
			t.Fatalf("Reconcile (pass %d): %v", pass, err)
		}
		total += drainQueue(t, ctx, r, 200)
		objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
		if err != nil {
			t.Fatalf("ListDeployedObjectsByBuild: %v", err)
		}
		done := len(objs) > 0
		for _, o := range objs {
			if o.Status != "running" {
				done = false
				break
			}
		}
		if done {
			return total
		}
	}
	t.Fatalf("deploy did not converge within 10 passes")
	return total
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	conn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if conn == "" {
		if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err != nil {
			t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
		}
		conn = "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
	}
	_, pool, err := db.Open(context.Background(), conn)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newTestBuild(t *testing.T, pool *pgxpool.Pool, name string) db.Build {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-test", GithubRepo: name})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "runner-test-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "lm-test"})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	// This whole suite drives Reconcile + the real runner to prove
	// convergence, which only creates tasks for a build in "deploying"
	// -- see internal/orchestrator.Reconcile's deployTasks gate.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"
	// fake_hoster_resource has no foreign key back to build (deliberately
	// -- see migrations/00003: it outlives any single build, modeling a
	// real hoster's own persistence), so nothing about deleting the
	// repository above cleans up whatever this test deployed into it.
	// external_ref is always "build-<this build's id>-...", so this
	// scopes cleanup to exactly this test's own rows.
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM fake_hoster_resource WHERE external_ref LIKE $1", "build-"+build.ID.String()+"-%")
	})
	return build
}

// TestFullBuildConverges is the end-to-end proof that orchestrator +
// runner + fake builder actually complete a real multi-team build: every
// one of the 65 deployed objects (5 teams x 13 each, per
// internal/orchestrator's own test) reaches 'deployed', every one has a
// distinct external_ref backed by a real fake_hoster_resource row (no
// duplicates), and a second Reconcile pass afterward creates zero new
// tasks -- the build is genuinely finished, not just "the queue emptied
// once."
func TestFullBuildConverges(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "runner-full-build")

	r := &Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "test-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		// Scoped to this test's own build -- see LeaseTaskForBuild's doc
		// comment: Go parallelizes test packages by default, and this
		// package, internal/orchestrator, and internal/chaos all share
		// one real Postgres database and all reconcile builds under the
		// same "lm-test" environment concurrently. Without this, a
		// healthy runner here could lease (and complete) another
		// package's in-flight test task.
		BuildID: &build.ID,
	}

	// Interleave reconcile + drain to convergence -- depends_on ordering brings
	// dependents up only after their dependencies.
	drained := drainToConvergence(t, ctx, pool, q, r, "../../examples/lm-test", build.ID)
	if drained != 65 {
		t.Fatalf("drained %d tasks total, want 65 (5 teams x 13 objects)", drained)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != 65 {
		t.Fatalf("len(objs) = %d, want 65", len(objs))
	}
	seenRefs := map[string]bool{}
	for _, o := range objs {
		if o.Status != "running" {
			t.Errorf("object %s (%s) status = %q, want running", o.ObjectName, db.StrOrEmpty(o.AsName), o.Status)
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if ref == "" {
			t.Errorf("object %s (%s) has no external_ref after deploy", o.ObjectName, db.StrOrEmpty(o.AsName))
			continue
		}
		if seenRefs[ref] {
			t.Errorf("duplicate external_ref %q across deployed objects", ref)
		}
		seenRefs[ref] = true

		count, err := q.CountFakeHosterResourcesByRef(ctx, ref)
		if err != nil {
			t.Fatalf("CountFakeHosterResourcesByRef(%s): %v", ref, err)
		}
		if count != 1 {
			t.Errorf("fake_hoster_resource rows for %q = %d, want 1", ref, count)
		}
	}
	if len(seenRefs) != 65 {
		t.Fatalf("saw %d distinct external_refs, want 65", len(seenRefs))
	}

	// A real fix, found by direct audit: a host/container's own authored `steps:`
	// used to never run automatically at deploy time at all (see
	// materializeSteps' own doc comment) -- web01's real step list
	// (script:base, download, extract, script:vuln-sqli, from
	// examples/lm-test/hosts/webserver.yaml) must now have queued real
	// agent_task rows the instant its deploy task completed above, with
	// no separate action needed.
	var web01 db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" && o.AsName != nil && *o.AsName == "web01" {
			tm, err := q.GetTeam(ctx, o.TeamID)
			if err != nil {
				t.Fatal(err)
			}
			if tm.TeamNumber == 1 {
				web01 = o
				break
			}
		}
	}
	if !web01.ID.Valid {
		t.Fatal("team 1's web01 object not found")
	}
	agentTasks, err := q.ListAgentTasksByHost(ctx, web01.ID)
	if err != nil {
		t.Fatalf("ListAgentTasksByHost: %v", err)
	}
	// script: expands to two commands each (write_file the rendered
	// script, then execute it) -- see internal/gateway/steps.go's own
	// "script" case -- so base + download + extract + vuln-sqli is 6
	// real commands, not 4.
	if len(agentTasks) != 6 {
		t.Fatalf("web01 agent_task rows = %d, want 6 (script:base -> write_file+execute, download, extract, script:vuln-sqli -> write_file+execute)", len(agentTasks))
	}
	wantCommands := []string{"write_file", "execute", "download", "extract", "write_file", "execute"}
	for i, at := range agentTasks {
		if at.Command != wantCommands[i] {
			t.Errorf("web01 agent_task[%d].Command = %q, want %q", i, at.Command, wantCommands[i])
		}
		if int(at.StepIndex) != i {
			t.Errorf("web01 agent_task[%d].StepIndex = %d, want %d", i, at.StepIndex, i)
		}
	}

	// A second Reconcile pass over an already-fully-deployed build must
	// create nothing new -- fingerprints all match.
	if err := orchestrator.Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (after full deploy): %v", err)
	}
	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	for _, tk := range tasks {
		if tk.Status == "pending" || tk.Status == "leased" {
			t.Errorf("unexpected open task after full convergence: %s %s", tk.Kind, tk.Status)
		}
	}
}

// TestExecuteAccessRecordsRealTeamState is a real fix found by direct
// audit: executeAccess
// used to call the real OpenAccess/CloseAccess builder call and stop --
// team.access_state (what BuildAccess.tsx/BuildOverview.tsx read to show
// an operator whether a close actually took effect) never got written,
// even though SetTeamAccessState's own doc comment already described
// this exact wiring. Proves both directions, plus the real
// access.closed/access.opened event this now also records.
func TestExecuteAccessRecordsRealTeamState(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "runner-access-state")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	if team.AccessState != "closed" {
		t.Fatalf("a fresh team's access_state = %q, want closed (schema default -- access is schedule-driven now)", team.AccessState)
	}

	r := &Runner{Pool: pool, Builder: fake.New(pool), ID: "test-runner-access", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second, BuildID: &build.ID}

	closePayload, _ := json.Marshal(map[string]string{"team": "1"})
	if _, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: build.ID, Kind: "close_access", Payload: closePayload}); err != nil {
		t.Fatalf("CreateTeamTask(close): %v", err)
	}
	if did, err := r.LeaseAndExecuteOne(ctx); err != nil || !did {
		t.Fatalf("close_access task: did=%v err=%v", did, err)
	}
	afterClose, err := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("GetTeamByNumber (after close): %v", err)
	}
	if afterClose.AccessState != "closed" {
		t.Fatalf("access_state after a real close_access = %q, want closed", afterClose.AccessState)
	}

	openPayload, _ := json.Marshal(map[string]string{"team": "1"})
	if _, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: build.ID, Kind: "open_access", Payload: openPayload}); err != nil {
		t.Fatalf("CreateTeamTask(open): %v", err)
	}
	if did, err := r.LeaseAndExecuteOne(ctx); err != nil || !did {
		t.Fatalf("open_access task: did=%v err=%v", did, err)
	}
	afterOpen, err := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("GetTeamByNumber (after open): %v", err)
	}
	if afterOpen.AccessState != "open" {
		t.Fatalf("access_state after a real open_access = %q, want open", afterOpen.AccessState)
	}

	events, err := q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	var sawClosed, sawOpened bool
	for _, e := range events {
		if e.Kind == "access.closed" {
			sawClosed = true
		}
		if e.Kind == "access.opened" {
			sawOpened = true
		}
	}
	if !sawClosed || !sawOpened {
		t.Fatalf("events = %+v, want both a real access.closed and access.opened event", events)
	}
}

// drainQueue calls LeaseAndExecuteOne until the queue reports empty (two
// consecutive empty leases, since a destroy-then-redeploy pair can
// legitimately leave the queue briefly empty between Reconcile passes --
// this test only reconciles once, so one empty read is really enough, but
// two is a cheap safety margin) or maxIterations is hit, and returns how
// many tasks were actually executed.
func drainQueue(t *testing.T, ctx context.Context, r *Runner, maxIterations int) int {
	t.Helper()
	executed := 0
	for i := 0; i < maxIterations; i++ {
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !did {
			return executed
		}
		executed++
	}
	t.Fatalf("drainQueue: did not empty after %d iterations (executed %d) -- possible infinite retry loop", maxIterations, executed)
	return executed
}
