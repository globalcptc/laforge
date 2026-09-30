package orchestrator

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

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

// newTestBuild creates a real repository/content_revision/build row for
// examples/lm-test -- no git or webhook involved, just enough of
// the schema for a build row to exist, since Reconcile requires
// content_revision_id to point somewhere real.
func newTestBuild(t *testing.T, pool *pgxpool.Pool, name string) db.Build {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{
		GithubOwner: "laforge-test", GithubRepo: name,
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID)
	})
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "orchestrator-test-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	// Every test using this helper is actually testing deploy-task
	// creation (their own names and assertions say so), which only
	// happens for a build in "deploying" -- see Reconcile's own
	// deployTasks gate, added after this repo's chaos suite proved a
	// `planned` build was never actually inert before it. A build
	// that's genuinely testing the "planned" side uses its own build
	// row instead (TestReconcilePlannedBuildCreatesNoRealWork).
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"
	return build
}

// TestReconcileOnRealExampleRepo runs Reconcile against the real,
// hand-authored examples/lm-test repo (5 teams; per team: 4 networks,
// database/webserver/scoreboard/domain-controller/workstation/devdb/
// wireguard = 7 single copies + kali x2 = 9 host-or-container copies, 13
// deployed_object rows total per team) and checks the exact numbers, not
// just "it didn't error."
func TestReconcileOnRealExampleRepo(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "reconcile-real-repo")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != 5*13 {
		t.Fatalf("ListDeployedObjectsByBuild = %d rows, want %d (5 teams x 13 objects each)", len(objs), 5*13)
	}

	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild: %v", err)
	}
	if len(teams) != 5 {
		t.Fatalf("ListTeamsByBuild = %d, want 5", len(teams))
	}

	// The two-copies case: kali01 and kali02, from the same `kali` host
	// definition, in the same team, must be two distinct rows.
	var team1ID = teams[0].ID
	for _, tm := range teams {
		if tm.TeamNumber == 1 {
			team1ID = tm.ID
		}
	}
	byTeam1, err := q.ListDeployedObjectsByTeam(ctx, team1ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByTeam: %v", err)
	}
	kaliCopies := 0
	seenAsNames := map[string]bool{}
	for _, o := range byTeam1 {
		if o.ObjectName == "kali" {
			kaliCopies++
			as := db.StrOrEmpty(o.AsName)
			if seenAsNames[as] {
				t.Fatalf("duplicate as_name %q among kali copies", as)
			}
			seenAsNames[as] = true
		}
	}
	if kaliCopies != 2 {
		t.Fatalf("found %d deployed_object rows for the kali definition in team 1, want 2 (kali01, kali02)", kaliCopies)
	}
	if !seenAsNames["kali01"] || !seenAsNames["kali02"] {
		t.Fatalf("expected as_names kali01 and kali02, got %v", seenAsNames)
	}

	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	// 11 of a team's 13 objects get a deploy task on the first pass; the two
	// with unmet depends_on (webserver -> database, workstation ->
	// domain-controller) are held by the ordering gate until their
	// dependencies come up on a later pass.
	if len(tasks) != 5*11 {
		t.Fatalf("ListTasksByBuild = %d, want %d -- one deploy task per ready object (depends_on holds 2/team)", len(tasks), 5*11)
	}
	for _, tk := range tasks {
		if tk.Status != "pending" {
			t.Fatalf("task %s status = %q, want pending (nothing has leased it yet)", tk.ID, tk.Status)
		}
	}

	// --- idempotency: reconciling again must not create a single new task ---
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (second pass): %v", err)
	}
	tasksAfter, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild (after second reconcile): %v", err)
	}
	if len(tasksAfter) != len(tasks) {
		t.Fatalf("second Reconcile pass created %d extra task(s) -- CreateTaskIfNoneOpen's guard did not hold", len(tasksAfter)-len(tasks))
	}
}

// TestReconcileRebuildsOnFingerprintChange proves "rebuild means
// recreate": an object recorded as deployed with a stale fingerprint gets
// a destroy task, not another deploy task, on the next reconcile pass.
func TestReconcileRebuildsOnFingerprintChange(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "reconcile-fingerprint-change")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var target db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" && db.StrOrEmpty(o.AsName) == "web01" {
			target = o
		}
	}
	if target.ID.String() == "" {
		t.Fatal("could not find the web01 deployed_object")
	}

	// Complete its open deploy task by hand (no runner in this test) and
	// mark it deployed with a deliberately WRONG fingerprint, simulating
	// "the content changed since this was last deployed." Updated
	// directly by id rather than through LeaseTask -- LeaseTask always
	// grabs whichever task is earliest in the shared queue, not a
	// specific one, which is exactly right for a real runner and wrong
	// for pinpointing web01's task here.
	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	for _, tk := range tasks {
		if tk.DeployedObjectID == target.ID {
			if _, err := pool.Exec(ctx, "UPDATE task SET status = 'done' WHERE id = $1", tk.ID); err != nil {
				t.Fatalf("completing web01's task directly: %v", err)
			}
		}
	}
	if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
		ID: target.ID, ExternalRef: db.StrPtr("fake-ref-web01"), Fingerprint: "deliberately-stale-fingerprint",
	}); err != nil {
		t.Fatalf("MarkDeployedObjectRunning: %v", err)
	}

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (after marking stale): %v", err)
	}

	tasksAfter, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild (after): %v", err)
	}
	var destroyTask *db.Task
	for i := range tasksAfter {
		if tasksAfter[i].DeployedObjectID == target.ID && tasksAfter[i].Status == "pending" {
			tk := tasksAfter[i]
			destroyTask = &tk
		}
	}
	if destroyTask == nil {
		t.Fatal("expected a new pending task for web01 after its fingerprint went stale")
	}
	if destroyTask.Kind != "destroy_host" {
		t.Fatalf("task kind = %q, want destroy_host (rebuild means recreate: destroy before redeploy)", destroyTask.Kind)
	}
}

// TestReconcilePlannedBuildCreatesNoRealWork is the real proof of
// "Build and deploy are separate verbs... Build touches no hoster... A
// build in planned, fully inspectable" --
// previously not true: nothing in Reconcile ever checked build.Status,
// so a `planned` build reconciled by cmd/laforge-orchestrator's own poll
// loop (which lists both "planned" and "deploying") would have silently
// created real deploy tasks. A build left in the schema's own default
// status (`planned`, no explicit SetBuildStatus -- unlike every other
// test in this file, which uses newTestBuild's now-forced "deploying"
// flip) must reconcile into a fully resolved, fully inspectable state
// -- real teams, real deployed_object rows, real fingerprints -- with
// zero tasks created. Flipping the SAME build to "deploying" and
// reconciling again must then create exactly the tasks the plain
// deploying-build test already proves, confirming the transition works
// on one real build row, not just in isolation.
func TestReconcilePlannedBuildCreatesNoRealWork(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{
		GithubOwner: "laforge-test", GithubRepo: "reconcile-planned-build",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "planned-build-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	if build.Status != "planned" {
		t.Fatalf("a freshly created build's status = %q, want the schema default planned", build.Status)
	}

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (planned): %v", err)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != 5*13 { // same real topology TestReconcileOnRealExampleRepo already proves
		t.Fatalf("ListDeployedObjectsByBuild = %d, want %d -- a planned build must still be fully resolved", len(objs), 5*13)
	}
	// deployed_object.fingerprint itself stays at its EnsureDeployedObject
	// default ('') until an actual deploy completes (MarkDeployedObjectRunning
	// is the only thing that ever writes it) -- a real, existing property
	// of this schema, not something this test's fix changed. The
	// fingerprint IS computed fresh every Reconcile pass (Fingerprint(),
	// called above regardless of deployTasks), just not persisted for
	// something that was never actually deployed -- consistent with how
	// a rendered-output endpoint needs to work too: recompute from
	// content on demand, don't rely on a column only a real deploy fills
	// in.
	for _, o := range objs {
		if o.Status != "pending" {
			t.Fatalf("deployed_object %s (%s) status = %q, want pending -- a planned build creates no tasks, so nothing should have moved off pending", o.ID, o.Kind, o.Status)
		}
	}

	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("ListTasksByBuild = %d, want 0 -- a planned build must touch no hoster", len(tasks))
	}

	// Now deploy it -- the same build row, the real transition an
	// operator's "Deploy" action performs.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (deploying): %v", err)
	}
	tasksAfterDeploy, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild (after deploy): %v", err)
	}
	// 11/team on the first deploying pass -- the two objects with unmet
	// depends_on are held by the ordering gate until dependencies come up.
	if len(tasksAfterDeploy) != 5*11 {
		t.Fatalf("ListTasksByBuild (after deploy) = %d, want %d -- deploying creates real tasks (depends_on holds 2/team on the first pass)", len(tasksAfterDeploy), 5*11)
	}
}

// TestDependsOnHoldsDeployUntilDependencyUp proves the ordering gate: a host
// with an unmet depends_on gets no deploy task, and one appears only after its
// dependency is up. webserver depends_on database in examples/lm-test.
func TestDependsOnHoldsDeployUntilDependencyUp(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "depends-on-order")

	countWebserverTasks := func() int {
		objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
		if err != nil {
			t.Fatalf("ListDeployedObjectsByBuild: %v", err)
		}
		webIDs := map[string]bool{}
		for _, o := range objs {
			if o.ObjectName == "webserver" {
				webIDs[o.ID.String()] = true
			}
		}
		tasks, err := q.ListTasksByBuild(ctx, build.ID)
		if err != nil {
			t.Fatalf("ListTasksByBuild: %v", err)
		}
		n := 0
		for _, tk := range tasks {
			if webIDs[tk.DeployedObjectID.String()] {
				n++
			}
		}
		return n
	}

	// First pass: database isn't up, so webserver is held -- no task.
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if got := countWebserverTasks(); got != 0 {
		t.Fatalf("webserver deploy tasks on first pass = %d, want 0 (held by depends_on)", got)
	}

	// Bring every team's database copy up, then reconcile again.
	if _, err := pool.Exec(ctx,
		"UPDATE deployed_object SET status='running' WHERE object_name='database' AND team_id IN (SELECT id FROM team WHERE build_id=$1)",
		build.ID); err != nil {
		t.Fatalf("marking database up: %v", err)
	}
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile (second pass): %v", err)
	}
	if got := countWebserverTasks(); got != 5 {
		t.Fatalf("webserver deploy tasks after database up = %d, want 5 (one per team)", got)
	}
}
