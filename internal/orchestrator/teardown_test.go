package orchestrator

import (
	"context"
	"testing"
	"time"

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

	// Deploy everything for real. Convergence (not a single Reconcile) because
	// depends_on ordering brings dependents up only after their dependencies.
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
