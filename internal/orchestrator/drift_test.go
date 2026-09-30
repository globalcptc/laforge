package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// TestDetectDriftAgainstARealDeployWithNoDrift is the drift-detection
// leftover, wired to a real caller for the first time:
// drives a genuine deploy through a real Runner (fake builder), then
// proves DetectDrift's real Inspect-vs-deployed_object comparison
// reports every live object as Tracked and nothing Orphaned/Missing when
// the hoster and the database genuinely agree.
func TestDetectDriftAgainstARealDeployWithNoDrift(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "drift-clean")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	r := &runner.Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "drift-test-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
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

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	wantTracked := 0
	for _, o := range objs {
		// Inspect only ever covers network/host/container (see
		// DetectDrift's own doc comment) -- dns objects are real and
		// deployed but outside what any builder's Inspect lists.
		if o.Status == "running" && (o.Kind == "network" || o.Kind == "host" || o.Kind == "container") {
			wantTracked++
		}
	}
	if wantTracked == 0 {
		t.Fatal("expected at least one really-deployed object")
	}

	report, err := DetectDrift(ctx, pool, build.ID)
	if err != nil {
		t.Fatalf("DetectDrift: %v", err)
	}
	if report.Tracked != wantTracked {
		t.Fatalf("Tracked = %d, want %d", report.Tracked, wantTracked)
	}
	if len(report.Orphaned) != 0 {
		t.Fatalf("Orphaned = %+v, want none (hoster and database genuinely agree)", report.Orphaned)
	}
	if len(report.Missing) != 0 {
		t.Fatalf("Missing = %+v, want none", report.Missing)
	}
}

// TestDetectDriftFindsOrphanedAndMissingResources directly manipulates
// fake_hoster_resource (the fake builder's own real backing table, not
// in-process memory -- see its own doc comment) to simulate the two real
// drift cases: a resource the hoster has that LaForge never recorded
// (orphaned), and a resource LaForge believes is deployed that the
// hoster no longer has (missing) -- proving DetectDrift's diff logic
// against both, not just the trivial no-drift case above.
func TestDetectDriftFindsOrphanedAndMissingResources(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "drift-dirty")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	r := &runner.Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "drift-dirty-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
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

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var lostRef string
	for _, o := range objs {
		if o.Status == "running" && o.ExternalRef != nil {
			lostRef = *o.ExternalRef
			break
		}
	}
	if lostRef == "" {
		t.Fatal("expected at least one deployed object with a real external_ref")
	}

	// Simulate the hoster losing this one resource outside LaForge's
	// control (someone deleted it by hand, a real outage, ...).
	if _, err := pool.Exec(ctx, "UPDATE fake_hoster_resource SET destroyed = true WHERE external_ref = $1", lostRef); err != nil {
		t.Fatalf("simulating a lost resource: %v", err)
	}
	// Simulate a real orphan: something the hoster has, in this exact
	// build's own name prefix (runner.deterministicExternalName's real
	// shape -- "build-<id>-team-N-kind-name"), that LaForge never
	// recorded in deployed_object at all.
	orphanRef := "build-" + build.ID.String() + "-team-1-host-drift-test-orphan"
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM fake_hoster_resource WHERE external_ref = $1", orphanRef)
	})
	if _, err := pool.Exec(ctx, "INSERT INTO fake_hoster_resource (external_ref, kind) VALUES ($1, 'host')", orphanRef); err != nil {
		t.Fatalf("simulating an orphaned resource: %v", err)
	}

	report, err := DetectDrift(ctx, pool, build.ID)
	if err != nil {
		t.Fatalf("DetectDrift: %v", err)
	}
	if len(report.Missing) != 1 || *report.Missing[0].ExternalRef != lostRef {
		t.Fatalf("Missing = %+v, want exactly the lost resource %q", report.Missing, lostRef)
	}
	var foundOrphan bool
	for _, o := range report.Orphaned {
		if o.ExternalRef == orphanRef {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Fatalf("Orphaned = %+v, want %q among them", report.Orphaned, orphanRef)
	}
}

func TestDetectDriftRequiresAConfiguredBuild(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "drift-no-configured-build")

	if _, err := DetectDrift(ctx, pool, build.ID); err == nil {
		t.Fatal("expected an error for a build with no configured_build to resolve a builder from")
	}
}
