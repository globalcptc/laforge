package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// TestPollInstanceStates drives a real deploy (fake builder), then proves
// the poll writes each host/container's live power state from the hoster --
// "running" while it's there, "missing" once the hoster loses it -- so
// instance liveness is tracked from the builder, independent of the agent.
func TestPollInstanceStates(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "power-poll")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	r := &runner.Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "power-poll-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}
	for {
		worked, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !worked {
			break
		}
	}

	// First poll: every deployed instance the fake hoster has reads "running".
	if err := PollInstanceStates(ctx, pool, build.ID); err != nil {
		t.Fatalf("PollInstanceStates: %v", err)
	}
	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var oneRef string
	var oneID interface{ String() string }
	for _, o := range objs {
		if (o.Kind == "host" || o.Kind == "container") && o.Status == "running" && o.ExternalRef != nil {
			if o.PowerState != "running" {
				t.Fatalf("%s power_state = %q, want running", o.ObjectName, o.PowerState)
			}
			oneRef, oneID = *o.ExternalRef, o.ID
		}
		// Networks never get a power state.
		if o.Kind == "network" && o.PowerState != "" {
			t.Fatalf("network power_state = %q, want empty", o.PowerState)
		}
	}
	if oneRef == "" {
		t.Fatal("expected at least one deployed host/container with a real external_ref")
	}

	// The hoster loses that instance (crash / manual stop-and-remove).
	if _, err := pool.Exec(ctx, "UPDATE fake_hoster_resource SET destroyed = true WHERE external_ref = $1", oneRef); err != nil {
		t.Fatalf("simulating a lost instance: %v", err)
	}
	if err := PollInstanceStates(ctx, pool, build.ID); err != nil {
		t.Fatalf("second PollInstanceStates: %v", err)
	}
	objs, _ = q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range objs {
		if o.ID.String() == oneID.String() {
			if o.PowerState != "missing" {
				t.Fatalf("after the hoster lost it, power_state = %q, want missing", o.PowerState)
			}
			// The deploy status must be untouched -- power state is a
			// separate axis, not a lifecycle change.
			if o.Status != "running" {
				t.Fatalf("status = %q, want it left at running (power state is separate)", o.Status)
			}
		}
	}
}
