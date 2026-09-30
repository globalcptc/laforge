package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// TestExecuteDeployTransitionsThroughDeploying is the real fix for a gap
// found by direct audit: MarkDeployedObjectDeploying
// existed since this table's own CHECK constraint named "deploying" as a
// real state, and the UI already renders it, but nothing ever called it
// -- a real deploy that takes real wall-clock time (a VM's first image
// clone, per item 1's own live finding) read as "pending" for its entire
// duration. Proven with a real, genuinely in-flight builder call (the
// fake builder's own WorkDelay, the same "real hoster API call taking
// real wall-clock time" mechanism internal/chaos already uses), polled
// from a second goroutine while the first is still inside the call --
// not just asserting the final state, which would pass even if the
// intermediate transition never happened at all.
//
// Every other pending task for this build is drained first, at
// WorkDelay=0, so only the host's own task is left to lease by the time
// WorkDelay is turned on -- otherwise which task a generic "lease
// whatever's next" call picks up first is unordered, and this test would
// flake waiting behind some other, unrelated object's own slow call.
func TestExecuteDeployTransitionsThroughDeploying(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "status-transition-deploy")

	if err := orchestrator.Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild: %v", err)
	}
	objs, err := q.ListDeployedObjectsByTeam(ctx, teams[0].ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByTeam: %v", err)
	}
	var host db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			host = o
			break
		}
	}
	if !host.ID.Valid {
		t.Fatal("no host deployed_object found in team 1")
	}

	fakeBuilder := fake.New(pool)
	r := &Runner{
		Pool: pool, Builder: fakeBuilder, RepoRoot: "../../examples/lm-test",
		ID: "status-transition-test", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}

	// Drain every other task at full speed, holding the host's own task
	// aside, unexecuted, the moment it's leased. The chosen host may be one
	// held by depends_on ordering (its deploy task appears only after its
	// dependencies deploy), so when we run out of tasks, reconcile again to
	// create newly-unblocked tasks before giving up.
	var hostTask *db.Task
	reconciles := 0
	for hostTask == nil {
		task, err := r.LeaseOne(ctx)
		if err != nil {
			t.Fatalf("LeaseOne: %v", err)
		}
		if task == nil {
			if reconciles < 5 {
				reconciles++
				if err := orchestrator.Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
					t.Fatalf("Reconcile (drain): %v", err)
				}
				continue
			}
			t.Fatal("ran out of leasable tasks before finding the host's own deploy task")
		}
		if task.DeployedObjectID == host.ID {
			hostTask = task
			break
		}
		if err := r.ExecuteAndRecord(ctx, *task); err != nil {
			t.Fatalf("draining task %s: %v", task.ID, err)
		}
	}

	fakeBuilder.WorkDelay = 400 * time.Millisecond
	done := make(chan error, 1)
	go func() {
		done <- r.ExecuteAndRecord(ctx, *hostTask)
	}()

	// The deploy call is genuinely in flight for WorkDelay -- poll for
	// "deploying" well inside that window, bounded so a slow CI machine
	// still passes rather than flaking on a fixed sleep.
	deadline := time.Now().Add(300 * time.Millisecond)
	sawDeploying := false
	for time.Now().Before(deadline) {
		current, err := q.GetDeployedObject(ctx, host.ID)
		if err != nil {
			t.Fatalf("GetDeployedObject: %v", err)
		}
		if current.Status == "deploying" {
			sawDeploying = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !sawDeploying {
		t.Fatal("never observed status=deploying while the builder call was genuinely in flight")
	}

	if err := <-done; err != nil {
		t.Fatalf("ExecuteAndRecord: %v", err)
	}
	final, err := q.GetDeployedObject(ctx, host.ID)
	if err != nil {
		t.Fatalf("GetDeployedObject (final): %v", err)
	}
	if final.Status != "running" {
		t.Fatalf("final status = %q, want running", final.Status)
	}
}
