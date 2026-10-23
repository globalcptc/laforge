package orchestrator

import (
	"context"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestReconcileObjectHoldsStepOnlyChange: with a configured build holding
// hosts on step-only changes, an object whose fingerprint differs but whose
// infrastructure fingerprint still matches is marked steps-changed and left
// alone -- no destroy task, one event, and nothing more on later passes.
// Without the hold, or when the infrastructure changed too, it's recreated
// as it always was.
func TestReconcileObjectHoldsStepOnlyChange(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "hold-step-change")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	mk := func(as string) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as), Fingerprint: "old"})
		if err != nil {
			t.Fatal(err)
		}
		// Up, with the fingerprints a deploy records.
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = 'finished', fingerprint = 'old', infra_fingerprint = 'infra' WHERE id = $1", o.ID); err != nil {
			t.Fatal(err)
		}
		o, _ = q.GetDeployedObject(ctx, o.ID)
		return o
	}
	destroys := func(o db.DeployedObject) int {
		var n int
		pool.QueryRow(ctx, "SELECT count(*) FROM task WHERE deployed_object_id = $1 AND kind = 'destroy_host'", o.ID).Scan(&n)
		return n
	}
	events := func(kind string) int {
		var n int
		pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE build_id = $1 AND kind = $2", build.ID, kind).Scan(&n)
		return n
	}

	// Steps-only change, hold on: marked, not destroyed; a second pass is quiet.
	held := mk("web-held")
	for i := 0; i < 2; i++ {
		if _, err := reconcileObject(ctx, q, build.ID, held, "new", "infra", true, 0, true); err != nil {
			t.Fatalf("pass %d: %v", i, err)
		}
	}
	if n := destroys(held); n != 0 {
		t.Fatalf("held object got %d destroy task(s), want none", n)
	}
	got, _ := q.GetDeployedObject(ctx, held.ID)
	if !got.StepsChangedAt.Valid || got.StepsChangedFingerprint != "new" || got.Fingerprint != "old" {
		t.Fatalf("held object = steps_changed_at %v, steps_changed_fingerprint %q, fingerprint %q; want marked, 'new', 'old' kept", got.StepsChangedAt.Valid, got.StepsChangedFingerprint, got.Fingerprint)
	}
	if n := events("steps.changed"); n != 1 {
		t.Fatalf("steps.changed events = %d, want exactly 1 across two passes", n)
	}

	// Restarting it from a step adopts the new fingerprint and clears the mark.
	if _, err := q.SetDeployedObjectRestarting(ctx, held.ID); err != nil {
		t.Fatalf("SetDeployedObjectRestarting on a marked finished host: %v", err)
	}
	got, _ = q.GetDeployedObject(ctx, held.ID)
	if got.Status != "building" || got.Fingerprint != "new" || got.StepsChangedAt.Valid || got.StepsChangedFingerprint != "" {
		t.Fatalf("after restart = %s fingerprint %q marked %v; want building, 'new', unmarked", got.Status, got.Fingerprint, got.StepsChangedAt.Valid)
	}

	// Steps-only change, hold off: recreated.
	plain := mk("web-plain")
	if _, err := reconcileObject(ctx, q, build.ID, plain, "new", "infra", false, 0, true); err != nil {
		t.Fatal(err)
	}
	if n := destroys(plain); n != 1 {
		t.Fatalf("unheld object got %d destroy task(s), want 1", n)
	}

	// Infrastructure changed too, hold on: recreated regardless.
	infra := mk("web-infra")
	if _, err := reconcileObject(ctx, q, build.ID, infra, "new", "other-infra", true, 0, true); err != nil {
		t.Fatal(err)
	}
	if n := destroys(infra); n != 1 {
		t.Fatalf("object with changed infrastructure got %d destroy task(s), want 1", n)
	}

	// Deployed before infra_fingerprint existed (empty): recreated, as before.
	legacy := mk("web-legacy")
	pool.Exec(ctx, "UPDATE deployed_object SET infra_fingerprint = '' WHERE id = $1", legacy.ID)
	legacy, _ = q.GetDeployedObject(ctx, legacy.ID)
	if _, err := reconcileObject(ctx, q, build.ID, legacy, "new", "infra", true, 0, true); err != nil {
		t.Fatal(err)
	}
	if n := destroys(legacy); n != 1 {
		t.Fatalf("legacy object got %d destroy task(s), want 1", n)
	}

	// A finished host that isn't marked can't be restarted.
	if _, err := q.SetDeployedObjectRestarting(ctx, plain.ID); err == nil {
		t.Fatal("SetDeployedObjectRestarting succeeded on an unmarked finished host")
	}
}
