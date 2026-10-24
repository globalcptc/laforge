package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// TestTeardownDestroysEveryRealObjectAndMarksTheBuildTornDown is
// teardown made real end to end: the spec calls for "teardown (infrastructure)... available from
// the UI and the API," and build.status's own CHECK constraint already
// named 'torn_down' as a real terminal state, but nothing anywhere ever
// reached it -- Reconcile's own destroy machinery only ever destroys an
// object that fell out of desired content, never a whole build on a
// deliberate operator action.
//
// Drives a real deploy first (through a real Runner, not a shortcut), so
// this proves teardown against genuinely deployed infrastructure, not an
// empty or still-pending build.
func TestTeardownDestroysEveryRealObjectAndMarksTheBuildTornDown(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "teardown-e2e")

	// Deploy everything for real (drive the runner to convergence).
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	// A runner to drain the destroy_* tasks Teardown creates later (teardown
	// has no ordering, so a single drain suffices there).
	r := &runner.Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "teardown-test-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}
	drain := func() {
		for {
			worked, err := r.LeaseAndExecuteOne(ctx)
			if err != nil {
				t.Fatalf("LeaseAndExecuteOne: %v", err)
			}
			if !worked {
				return
			}
		}
	}

	objsBefore, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objsBefore) == 0 {
		t.Fatal("expected real deployed objects before teardown, got none")
	}
	for _, o := range objsBefore {
		if o.Status != "running" {
			t.Fatalf("object %s (%s) status = %q before teardown, want running", o.ObjectName, o.Kind, o.Status)
		}
	}

	// A real caller (internal/api's own handleTeardownBuild) sets
	// 'tearing_down' before ever calling Teardown -- Teardown itself
	// never touches build.status except to advance it to 'torn_down',
	// exactly once every object is actually destroyed.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "tearing_down"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}

	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	afterFirstPass, err := q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if afterFirstPass.Status != "tearing_down" {
		t.Fatalf("build.Status after first Teardown pass = %q, want still tearing_down (nothing destroyed yet)", afterFirstPass.Status)
	}

	drain() // run the real destroy_* tasks Teardown just created

	// Idempotent: calling Teardown again is what actually notices
	// everything is destroyed now and flips the build -- the same real
	// polling shape cmd/laforge-orchestrator's own ticker uses.
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown (second pass): %v", err)
	}

	final, err := q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild (final): %v", err)
	}
	if final.Status != "torn_down" {
		t.Fatalf("build.Status = %q, want torn_down", final.Status)
	}
	objsAfter, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (final): %v", err)
	}
	if len(objsAfter) != len(objsBefore) {
		t.Fatalf("object count changed across teardown: %d before, %d after -- rows should stay (destroyed, not deleted)", len(objsBefore), len(objsAfter))
	}
	for _, o := range objsAfter {
		if o.Status != "destroyed" {
			t.Fatalf("object %s (%s) status = %q after teardown, want destroyed", o.ObjectName, o.Kind, o.Status)
		}
	}
}

// TestTeardownOfABuildWithNoObjectsTearsDownImmediately covers the
// degenerate but real case: a build that was never actually reconciled
// (no deployed_object rows at all) has nothing to destroy, so a single
// Teardown call should reach 'torn_down' straight away, not need a
// second pass.
func TestTeardownOfABuildWithNoObjectsTearsDownImmediately(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "teardown-empty")

	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	final, err := q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if final.Status != "torn_down" {
		t.Fatalf("build.Status = %q, want torn_down", final.Status)
	}
}

// TestTeardownStopsAfterPermanentDestroyError is the wedged-teardown fix:
// before it, Teardown re-requested a destroy for every not-yet-destroyed
// object on every pass regardless of why the last one failed, so a builder
// refusing a destroy for a configuration reason (incuspool's "team 2 would
// reuse a bridge host's subnets", a project name too long for the hoster)
// produced a fresh failed destroy task per object every two seconds, forever
// -- ~210,000 rows for two builds on prod. Now the runner records the
// classification (task.permanent) and Teardown stops: the object becomes
// destroy_failed, an event says why, the build's reconcile_error says what to
// do, and no further task is created until an operator retries explicitly.
func TestTeardownStopsAfterPermanentDestroyError(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "teardown-permanent")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	fb := fake.New(pool)
	fb.FailDestroy = fmt.Errorf("%w: team 2 would reuse a bridge host's subnets: configure at least 2 distinct hosts for 2 teams", builder.ErrPermanentConfig)
	r := &runner.Runner{
		Pool: pool, Builder: fb, RepoRoot: "../../examples/lm-test",
		ID: "teardown-permanent-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}
	// LeaseAndExecuteOne returns (true, execErr) for a task it worked and
	// failed; while the fake refuses destroys, that error must be exactly the
	// permanent one, and once it stops refusing, there must be none.
	drain := func() {
		t.Helper()
		for {
			worked, err := r.LeaseAndExecuteOne(ctx)
			if !worked {
				if err != nil {
					t.Fatalf("LeaseAndExecuteOne: %v", err)
				}
				return
			}
			if fb.FailDestroy == nil && err != nil {
				t.Fatalf("LeaseAndExecuteOne after the builder was fixed: %v", err)
			}
			if fb.FailDestroy != nil && !errors.Is(err, builder.ErrPermanentConfig) {
				t.Fatalf("LeaseAndExecuteOne error = %v, want the permanent builder error", err)
			}
		}
	}
	destroyTasks := func() int {
		tasks, err := q.ListTasksByBuild(ctx, build.ID)
		if err != nil {
			t.Fatalf("ListTasksByBuild: %v", err)
		}
		n := 0
		for _, tk := range tasks {
			if strings.HasPrefix(tk.Kind, "destroy_") {
				n++
			}
		}
		return n
	}
	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) == 0 {
		t.Fatal("expected deployed objects")
	}

	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "tearing_down"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown (pass 1): %v", err)
	}
	if got := destroyTasks(); got != len(objs) {
		t.Fatalf("destroy tasks after pass 1 = %d, want %d (one per object)", got, len(objs))
	}
	drain() // every destroy fails on its first attempt: permanent, not retried by the runner

	tasks, _ := q.ListTasksByBuild(ctx, build.ID)
	for _, tk := range tasks {
		if strings.HasPrefix(tk.Kind, "destroy_") && !(tk.Status == "failed" && tk.Permanent) {
			t.Fatalf("task %s status=%q permanent=%v, want failed+permanent", tk.Kind, tk.Status, tk.Permanent)
		}
	}

	// Pass 2 notices the permanent failures and stops instead of re-queuing.
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown (pass 2): %v", err)
	}
	if got := destroyTasks(); got != len(objs) {
		t.Fatalf("destroy tasks after pass 2 = %d, want still %d (no re-queue after a permanent error)", got, len(objs))
	}
	after, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range after {
		if o.Status != "destroy_failed" {
			t.Fatalf("object %s (%s) status = %q, want destroy_failed", o.ObjectName, o.Kind, o.Status)
		}
		if !strings.Contains(db.StrOrEmpty(o.LastError), "bridge host's subnets") {
			t.Fatalf("object %s last_error = %q, want the builder's error", o.ObjectName, db.StrOrEmpty(o.LastError))
		}
	}
	b, err := q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != "tearing_down" {
		t.Fatalf("build.Status = %q, want still tearing_down (stuck, not torn down)", b.Status)
	}
	if msg := db.StrOrEmpty(b.ReconcileError); !strings.Contains(msg, "Teardown is stuck") || !strings.Contains(msg, "bridge host's subnets") {
		t.Fatalf("build.reconcile_error = %q, want the stuck-teardown summary with the builder error", msg)
	}
	events, err := q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	stuckEvents := 0
	for _, ev := range events {
		if ev.Kind == "object.destroy_failed" {
			stuckEvents++
		}
	}
	if stuckEvents != len(objs) {
		t.Fatalf("object.destroy_failed events = %d, want %d (one per object)", stuckEvents, len(objs))
	}

	// Further passes are inert: the loop that filled prod's task table is gone.
	for pass := 3; pass <= 5; pass++ {
		if err := Teardown(ctx, pool, build.ID); err != nil {
			t.Fatalf("Teardown (pass %d): %v", pass, err)
		}
	}
	if got := destroyTasks(); got != len(objs) {
		t.Fatalf("destroy tasks after inert passes = %d, want %d", got, len(objs))
	}

	// The operator fixes the builder and retries explicitly: every stuck
	// object gets a fresh destroy at once, and the teardown then completes.
	fb.FailDestroy = nil
	created, err := RetryTeardown(ctx, pool, build.ID)
	if err != nil {
		t.Fatalf("RetryTeardown: %v", err)
	}
	if created != len(objs) {
		t.Fatalf("RetryTeardown created %d tasks, want %d", created, len(objs))
	}
	b, _ = q.GetBuild(ctx, build.ID)
	if b.ReconcileError != nil {
		t.Fatalf("reconcile_error after retry = %q, want cleared", *b.ReconcileError)
	}
	retried, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range retried {
		if o.Status != "destroying" {
			t.Fatalf("object %s status after retry = %q, want destroying", o.ObjectName, o.Status)
		}
	}
	drain()
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown (after retry): %v", err)
	}
	final, _ := q.GetBuild(ctx, build.ID)
	if final.Status != "torn_down" {
		t.Fatalf("build.Status after successful retry = %q, want torn_down", final.Status)
	}
	done, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range done {
		if o.Status != "destroyed" {
			t.Fatalf("object %s status = %q, want destroyed", o.ObjectName, o.Status)
		}
	}
}

// TestTeardownBacksOffTransientDestroyFailures keeps the self-heal for a
// destroy that failed for a reason that may clear on its own (the documented
// "network currently in use while its instances detach" case) but spaces the
// re-requests out: a failed destroy task is re-queued only once a backoff that
// doubles with every failed task has elapsed since it failed, not on the very
// next two-second pass.
func TestTeardownBacksOffTransientDestroyFailures(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "teardown-transient")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	objs, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	n := len(objs)
	if n == 0 {
		t.Fatal("expected deployed objects")
	}
	destroyTasks := func(status string) int {
		tasks, err := q.ListTasksByBuild(ctx, build.ID)
		if err != nil {
			t.Fatal(err)
		}
		c := 0
		for _, tk := range tasks {
			if strings.HasPrefix(tk.Kind, "destroy_") && (status == "" || tk.Status == status) {
				c++
			}
		}
		return c
	}
	// Stand in for the runner: fail every open destroy transiently (not
	// permanent), as a runner that exhausted MaxAttempts on an ordinary error
	// does, and then age the failure so the backoff can be observed without
	// sleeping.
	failOpen := func() {
		if _, err := pool.Exec(ctx, "UPDATE task SET status='failed', last_error='network currently in use', permanent=false, updated_at=now() WHERE build_id=$1 AND status IN ('pending','leased')", build.ID); err != nil {
			t.Fatal(err)
		}
	}
	age := func(d time.Duration) {
		if _, err := pool.Exec(ctx, "UPDATE task SET updated_at = now() - $2::interval WHERE build_id=$1 AND status='failed'", build.ID, db.Interval(d)); err != nil {
			t.Fatal(err)
		}
	}

	q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "tearing_down"})
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if got := destroyTasks(""); got != n {
		t.Fatalf("destroy tasks = %d, want %d", got, n)
	}
	failOpen()

	// One failed task per object: the next pass is inside the 30s backoff.
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if got := destroyTasks(""); got != n {
		t.Fatalf("destroy tasks right after a transient failure = %d, want still %d (backoff)", got, n)
	}
	// Once the backoff has elapsed, exactly one fresh task per object.
	age(teardownRequeueBase + time.Second)
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if got := destroyTasks("pending"); got != n {
		t.Fatalf("pending destroy tasks after backoff = %d, want %d", got, n)
	}
	events, _ := q.ListEventsByBuild(ctx, build.ID)
	retries := 0
	for _, ev := range events {
		if ev.Kind == "teardown.retry" {
			retries++
		}
	}
	if retries != n {
		t.Fatalf("teardown.retry events = %d, want %d", retries, n)
	}

	// Two failed tasks per object: the backoff has doubled, so the first
	// interval is no longer enough.
	failOpen()
	age(teardownRequeueBase + time.Second)
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if got := destroyTasks("pending"); got != 0 {
		t.Fatalf("pending destroy tasks inside the doubled backoff = %d, want 0", got)
	}
	age(2*teardownRequeueBase + time.Second)
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if got := destroyTasks("pending"); got != n {
		t.Fatalf("pending destroy tasks after the doubled backoff = %d, want %d", got, n)
	}
	b, _ := q.GetBuild(ctx, build.ID)
	if b.Status != "tearing_down" || b.ReconcileError != nil {
		t.Fatalf("build status=%q reconcile_error=%v: a transient failure must not mark the teardown stuck", b.Status, b.ReconcileError)
	}
	objsNow, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range objsNow {
		if o.Status == "destroy_failed" {
			t.Fatalf("object %s became destroy_failed on a transient error", o.ObjectName)
		}
	}
}

func TestTeardownRequeueBackoffDoublesAndCaps(t *testing.T) {
	cases := map[int]time.Duration{
		0: 30 * time.Second, 1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute,
		5: 8 * time.Minute, 6: 10 * time.Minute, 50: 10 * time.Minute,
	}
	for failed, want := range cases {
		if got := teardownRequeueBackoff(failed); got != want {
			t.Errorf("teardownRequeueBackoff(%d) = %s, want %s", failed, got, want)
		}
	}
}

// TestForceTornDownDeclaresBuildTornDownWithoutHoster is the operator escape
// hatch: a wedged teardown is declared done -- open tasks closed, every object
// recorded destroyed, build torn_down, scheduled tasks canceled -- and nothing
// is asked of the hoster, so the fake hoster's rows are still there afterwards.
func TestForceTornDownDeclaresBuildTornDownWithoutHoster(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "teardown-force")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	objs, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	n := len(objs)
	if n == 0 {
		t.Fatal("expected deployed objects")
	}
	hosterRows := func() int {
		var c int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM fake_hoster_resource WHERE external_ref IN (SELECT external_ref FROM deployed_object JOIN team ON team.id = deployed_object.team_id WHERE team.build_id = $1)", build.ID).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	before := hosterRows()
	if before == 0 {
		t.Fatal("expected fake hoster rows for the deployed objects")
	}

	q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "tearing_down"})
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	forced, err := ForceTornDown(ctx, pool, build.ID, "forced by test")
	if err != nil {
		t.Fatalf("ForceTornDown: %v", err)
	}
	if forced != int64(n) {
		t.Fatalf("ForceTornDown forced %d objects, want %d", forced, n)
	}
	b, _ := q.GetBuild(ctx, build.ID)
	if b.Status != "torn_down" {
		t.Fatalf("build.Status = %q, want torn_down", b.Status)
	}
	after, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range after {
		if o.Status != "destroyed" {
			t.Fatalf("object %s status = %q, want destroyed", o.ObjectName, o.Status)
		}
	}
	tasks, _ := q.ListTasksByBuild(ctx, build.ID)
	for _, tk := range tasks {
		if tk.Status == "pending" || tk.Status == "leased" {
			t.Fatalf("task %s still %s after force", tk.Kind, tk.Status)
		}
		if strings.HasPrefix(tk.Kind, "destroy_") && db.StrOrEmpty(tk.LastError) != "forced by test" {
			t.Fatalf("destroy task last_error = %q, want the force reason", db.StrOrEmpty(tk.LastError))
		}
	}
	if got := hosterRows(); got != before {
		t.Fatalf("fake hoster rows after force = %d, want %d (force must not touch the hoster)", got, before)
	}
	// Idempotent against the orchestrator's own loop: a torn_down build is no
	// longer tearing_down, but a stray Teardown call must stay harmless.
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatalf("Teardown after force: %v", err)
	}
}
