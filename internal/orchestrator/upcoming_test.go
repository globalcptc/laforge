package orchestrator

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestDiffUpcomingReportsNewForNeverReconciledBuild is the baseline case:
// a build with zero deployed_object rows (Reconcile never ran) has every
// one of its 65 objects (5 teams x 13, same real examples/lm-test
// topology Reconcile's own tests use) reported as "new" -- there's
// nothing deployed yet to compare against.
func TestDiffUpcomingReportsNewForNeverReconciledBuild(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "diff-upcoming-new")

	changes, err := DiffUpcoming(ctx, pool, "../../examples/lm-test", build.ID)
	if err != nil {
		t.Fatalf("DiffUpcoming: %v", err)
	}
	if len(changes) != 65 {
		t.Fatalf("len(changes) = %d, want 65 (nothing deployed yet, everything is new)", len(changes))
	}
	for _, c := range changes {
		if c.Change != "new" {
			t.Fatalf("change %+v has Change = %q, want new", c, c.Change)
		}
	}
}

// TestDiffUpcomingDetectsChangedAndRemoved covers the two real diff
// paths that matter once a build is actually deployed: a real
// fingerprint mismatch on an object marked "deployed" (would rebuild),
// and an object still recorded but no longer in the desired topology at
// all (would be destroyed, not replaced) -- while confirming a freshly
// reconciled but not-yet-deployed build (every object still "pending")
// correctly reports zero changes, since Reconcile itself would just
// retry the same desired state for those, not alter anything.
func TestDiffUpcomingDetectsChangedAndRemoved(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "diff-upcoming-changed-removed")

	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	noneYet, err := DiffUpcoming(ctx, pool, "../../examples/lm-test", build.ID)
	if err != nil {
		t.Fatalf("DiffUpcoming (nothing deployed yet): %v", err)
	}
	if len(noneYet) != 0 {
		t.Fatalf("DiffUpcoming on a freshly reconciled (all-pending) build = %+v, want no changes", noneYet)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01 db.DeployedObject
	var team1ID pgtype.UUID
	for _, o := range objs {
		if o.Kind == "host" && db.StrOrEmpty(o.AsName) == "web01" {
			tm, err := q.GetTeam(ctx, o.TeamID)
			if err != nil {
				t.Fatal(err)
			}
			if tm.TeamNumber == 1 {
				web01 = o
				team1ID = o.TeamID
			}
		}
	}
	if !web01.ID.Valid {
		t.Fatal("team 1's web01 deployed_object not found after reconcile")
	}
	if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
		ID: web01.ID, ExternalRef: db.StrPtr("fake-ref-web01"), Fingerprint: "deliberately-stale-fingerprint",
	}); err != nil {
		t.Fatalf("MarkDeployedObjectRunning: %v", err)
	}

	// A deployed_object with no counterpart anywhere in examples/
	// lm-test's real content -- standing in for "this host was removed
	// from the environment file," the other real DiffUpcoming path.
	if _, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team1ID, Kind: "host", ObjectName: "long-gone", AsName: db.StrPtr("ghost01"), NetworkName: db.StrPtr("prod"),
	}); err != nil {
		t.Fatalf("EnsureDeployedObject (ghost): %v", err)
	}

	changes, err := DiffUpcoming(ctx, pool, "../../examples/lm-test", build.ID)
	if err != nil {
		t.Fatalf("DiffUpcoming: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len(changes) = %d, want 2 (web01 changed, ghost01 removed); got %+v", len(changes), changes)
	}
	var sawChanged, sawRemoved bool
	for _, c := range changes {
		switch {
		case c.Team == 1 && c.AsName == "web01":
			if c.Change != "changed" || c.Kind != "host" || c.ObjectName != "webserver" {
				t.Fatalf("web01's change = %+v, want {team:1 kind:host object:webserver as:web01 change:changed}", c)
			}
			sawChanged = true
		case c.Team == 1 && c.AsName == "ghost01":
			if c.Change != "removed" || c.Kind != "host" || c.ObjectName != "long-gone" {
				t.Fatalf("ghost01's change = %+v, want {team:1 kind:host object:long-gone as:ghost01 change:removed}", c)
			}
			sawRemoved = true
		default:
			t.Fatalf("unexpected change: %+v", c)
		}
	}
	if !sawChanged || !sawRemoved {
		t.Fatalf("changes = %+v, want both a changed and a removed entry", changes)
	}
}
