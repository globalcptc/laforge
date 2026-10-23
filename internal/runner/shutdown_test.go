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
