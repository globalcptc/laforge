package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// TestShutdownHandsTasksBack: a runner stopped mid-task (a redeploy of
// LaForge) hands the task back -- pending, unleased, the lease not counted as
// an attempt -- instead of leaving it leased until the lease expires and then
// counting it. Redeploys used to burn every in-flight deploy's retry budget.
func TestShutdownHandsTasksBack(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "shutdown-handback")
	if err := orchestrator.Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatal(err)
	}
	b := fake.New(pool)
	b.WorkDelay = time.Minute // the builder call is still running when the runner stops
	r := &Runner{Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test", ID: "shutdown-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: time.Second, BuildID: &build.ID}
	task, err := r.LeaseOne(ctx)
	if err != nil || task == nil {
		t.Fatalf("lease: %v %v", task, err)
	}
	if task.Attempts != 1 {
		t.Fatalf("leased with attempts %d", task.Attempts)
	}
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- r.ExecuteAndRecord(runCtx, *task) }()
	time.Sleep(200 * time.Millisecond)
	stop() // SIGTERM
	if err := <-done; err != nil {
		t.Fatalf("ExecuteAndRecord on shutdown: %v", err)
	}
	got, err := db.New(pool).GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "pending" || got.Attempts != 0 || got.LeaseOwner != nil {
		t.Fatalf("after shutdown: status %s attempts %d owner %v; want pending, 0, none", got.Status, got.Attempts, got.LeaseOwner)
	}
}

// TestTaskFromAKilledRunnerIsPickedUp is the upgrade case for the runner
// itself: the runner being replaced (the old binary, which has no shutdown
// handling) is killed with a task leased. Nothing hands it back; its lease
// expires, and the new runner leases it again as its next attempt -- one
// attempt spent, not the task lost.
func TestTaskFromAKilledRunnerIsPickedUp(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "killed-runner")
	if err := orchestrator.Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatal(err)
	}
	old := &Runner{Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test", ID: "old-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: time.Second, BuildID: &build.ID}
	task, err := old.LeaseOne(ctx)
	if err != nil || task == nil {
		t.Fatalf("lease: %v %v", task, err)
	}
	// Killed: no heartbeat, no hand-back. Its lease runs out.
	if _, err := pool.Exec(ctx, "UPDATE task SET lease_expires_at = now() - interval '1 second' WHERE id = $1", task.ID); err != nil {
		t.Fatal(err)
	}
	fresh := &Runner{Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test", ID: "new-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: time.Second, BuildID: &build.ID}
	// The new runner leases tasks in its own order; drain until this one is done.
	for i := 0; i < 500; i++ {
		got, err := db.New(pool).GetTask(ctx, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == "done" {
			if got.Attempts != 2 {
				t.Fatalf("picked-up task finished with attempts %d, want 2 (one lost to the killed runner)", got.Attempts)
			}
			return
		}
		if did, err := fresh.LeaseAndExecuteOne(ctx); err != nil {
			t.Fatal(err)
		} else if !did {
			break
		}
	}
	t.Fatal("the killed runner's task was never picked up")
}
