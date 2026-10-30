package orchestrator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// TestRebuildDoesNotDragInSiblingInstances guards the per-person-instance bug:
// the `people:` field deploys several copies of one host that share an
// object_name (distinguished only by as_name). Rebuilding one copy must not pull
// in a sibling copy. content.Dependents seeds its walk with the root names, and
// the old resolver both collapsed the duplicate object_name to one arbitrary row
// and failed to exclude the roots -- so rebuilding vdi-win-01 dragged in a random
// sibling (vdi-win-06) as a phantom "dependent".
func TestRebuildDoesNotDragInSiblingInstances(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-siblings")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	// Two copies of one host: same object_name, different as_name (one per person).
	a, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "vdi-windows", AsName: db.StrPtr("vdi-win-01"), Fingerprint: "f"})
	if err != nil {
		t.Fatalf("EnsureDeployedObject vdi-win-01: %v", err)
	}
	if _, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "vdi-windows", AsName: db.StrPtr("vdi-win-02"), Fingerprint: "f"}); err != nil {
		t.Fatalf("EnsureDeployedObject vdi-win-02: %v", err)
	}

	// Nothing actually depends on vdi-windows.
	content := &loader.Content{Hosts: []loader.Host{{Name: "vdi-windows"}}}
	affected, err := resolveRebuildSet(ctx, q, build.ID, content, AdHocTarget{IDs: []string{a.ID.String()}}, true)
	if err != nil {
		t.Fatalf("resolveRebuildSet: %v", err)
	}
	if len(affected) != 1 || affected[0].ID != a.ID {
		names := make([]string, len(affected))
		for i, o := range affected {
			names[i] = db.StrOrEmpty(o.AsName)
		}
		t.Fatalf("rebuilding vdi-win-01 affected %d object(s) %v, want only vdi-win-01 -- a sibling that merely shares the object_name must not be dragged in", len(affected), names)
	}
}

// firstHost returns a deployed host object for the build, failing if none.
func firstHost(t *testing.T, q *db.Queries, buildID db.Build) db.DeployedObject {
	t.Helper()
	objs, err := q.ListDeployedObjectsByBuild(context.Background(), buildID.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	for _, o := range objs {
		if o.Kind == "host" && o.ExternalRef != nil && *o.ExternalRef != "" {
			return o
		}
	}
	t.Fatal("no deployed host with an external_ref found")
	return db.DeployedObject{}
}

// TestRebuildCarriesExternalRefInPayload is the issue-#1 regression: the destroy
// task a rebuild enqueues must carry the real hoster reference, so the runner
// deletes the actual instance. The object itself now keeps its reference too,
// shown as waiting to be destroyed until the runner gets to it.
func TestRebuildCarriesExternalRefInPayload(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-payload")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	host := firstHost(t, q, build)
	originalRef := *host.ExternalRef

	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{host.ID.String()}}, false, false); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	// Waiting to be destroyed -- not pending, which would look like a host
	// waiting for its first deploy -- with its hoster reference intact.
	reread, err := q.GetDeployedObject(ctx, host.ID)
	if err != nil {
		t.Fatalf("GetDeployedObject: %v", err)
	}
	if reread.Status != "destroy_queued" {
		t.Errorf("object status = %q, want destroy_queued", reread.Status)
	}
	if db.StrOrEmpty(reread.ExternalRef) != originalRef {
		t.Errorf("external_ref = %q, want it kept (%q) until the destroy runs", db.StrOrEmpty(reread.ExternalRef), originalRef)
	}

	// The destroy task must still know the real ref, via its payload.
	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	var found bool
	for _, tk := range tasks {
		if tk.DeployedObjectID == host.ID && tk.Kind == "destroy_host" {
			var p struct {
				Remove      bool   `json:"remove"`
				ExternalRef string `json:"external_ref"`
			}
			if err := json.Unmarshal(tk.Payload, &p); err != nil {
				t.Fatalf("decoding destroy payload: %v", err)
			}
			if p.ExternalRef != originalRef {
				t.Errorf("destroy payload external_ref = %q, want %q", p.ExternalRef, originalRef)
			}
			if p.Remove {
				t.Error("a rebuild destroy must not set remove")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no destroy_host task was enqueued for the rebuilt host")
	}

	// And the build is back in the deploy phase so reconcile re-queues the deploy.
	rb, err := q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if rb.Status != "deploying" {
		t.Errorf("build status = %q, want deploying", rb.Status)
	}
}

// TestFinalizeBuildStatusRecoversPendingRebuild is the issue-#2 regression: a
// build that already advanced to "building" but then has infra pending again (a
// rebuild reset an object) must be pulled back to "deploying" -- reconcile only
// queues deploy tasks there, so otherwise the replacement strands forever.
func TestFinalizeBuildStatusRecoversPendingRebuild(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-recover")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	t.Run("building + a pending object recovers to deploying", func(t *testing.T) {
		host := firstHost(t, q, build)
		if _, err := q.ResetDeployedObjectForRedeploy(ctx, host.ID); err != nil { // -> pending
			t.Fatalf("reset: %v", err)
		}
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "building"}); err != nil {
			t.Fatalf("SetBuildStatus: %v", err)
		}
		terminal, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if terminal {
			t.Error("a build with pending infra must not be terminal")
		}
		rb, _ := q.GetBuild(ctx, build.ID)
		if rb.Status != "deploying" {
			t.Errorf("build status = %q, want deploying (recovered)", rb.Status)
		}
		// Clean the object back up so the next subtest starts from convergence.
		deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	})

	t.Run("deploying + a destroying object is not prematurely terminal", func(t *testing.T) {
		host := firstHost(t, q, build)
		if _, err := q.MarkDeployedObjectDestroying(ctx, host.ID); err != nil {
			t.Fatalf("MarkDeployedObjectDestroying: %v", err)
		}
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
			t.Fatalf("SetBuildStatus: %v", err)
		}
		terminal, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if terminal {
			t.Error("a destroying object must count as infra-pending, not terminal")
		}
		rb, _ := q.GetBuild(ctx, build.ID)
		if rb.Status != "deploying" {
			t.Errorf("build status = %q, want deploying", rb.Status)
		}
	})
}

// TestRebuildConverges is the end-to-end issue-#3 regression: a scoped rebuild
// redeploys the object and the build reaches a terminal (non-stranded) state,
// rather than sitting in building/pending forever.
func TestRebuildConverges(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-converge")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	host := firstHost(t, q, build)
	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{host.ID.String()}}, false, false); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	reread, err := q.GetDeployedObject(ctx, host.ID)
	if err != nil {
		t.Fatalf("GetDeployedObject: %v", err)
	}
	switch reread.Status {
	case "running", "building", "finished":
	default:
		t.Errorf("rebuilt object status = %q, want it redeployed (running/building/finished), not stranded", reread.Status)
	}
	if reread.ExternalRef == nil || *reread.ExternalRef == "" {
		t.Error("rebuilt object has no external_ref -- it was not redeployed")
	}
	if _, err := AdvanceBuildStatus(ctx, pool, build.ID); err != nil {
		t.Fatalf("AdvanceBuildStatus: %v", err)
	}
	rb, _ := q.GetBuild(ctx, build.ID)
	if rb.Status == "building" {
		// building is only acceptable if an object genuinely still has an agent
		// working; with the fake builder everything finishes, so building here
		// means the stranding bug.
		objs, _ := q.ListDeployedObjectsByBuild(ctx, build.ID)
		for _, o := range objs {
			if o.Status == "pending" {
				t.Fatalf("build stranded in building with a pending object %s", o.ID.String())
			}
		}
	}
}

// TestRebuildDestroysThenGoesPending: a rebuilt host waits to be destroyed,
// then is destroyed, and only then is pending for its fresh deploy. While it
// waits, reconcile passes do nothing for it: no deploy is queued, and nothing
// is logged about it (its status says what it's waiting for). The build can't
// be judged finished in the meantime.
func TestRebuildDestroysThenGoesPending(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-destroy-first")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	host := firstHost(t, q, build)
	originalRef := *host.ExternalRef

	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{host.ID.String()}}, false, false); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	var eventsBefore int
	pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE deployed_object_id = $1", host.ID).Scan(&eventsBefore)
	for pass := 0; pass < 3; pass++ {
		if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID, nil); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	}
	got, _ := q.GetDeployedObject(ctx, host.ID)
	if got.Status != "destroy_queued" {
		t.Fatalf("after reconcile passes, status = %q, want still destroy_queued (its destroy hasn't run)", got.Status)
	}
	var deploys, events int
	pool.QueryRow(ctx, "SELECT count(*) FROM task WHERE deployed_object_id = $1 AND kind = 'deploy_host' AND status IN ('pending','leased')", host.ID).Scan(&deploys)
	pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE deployed_object_id = $1", host.ID).Scan(&events)
	if deploys != 0 {
		t.Fatalf("%d deploy(s) queued while the old instance still exists", deploys)
	}
	if events != eventsBefore {
		t.Fatalf("reconcile passes logged %d event(s) about a host that's only waiting for its destroy", events-eventsBefore)
	}
	b, _ := q.GetBuild(ctx, build.ID)
	if b.Status != "deploying" {
		t.Fatalf("build status %q while a host waits to be destroyed, want deploying", b.Status)
	}

	// The runner destroys it (by its recorded reference), then it's
	// redeployed fresh.
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	var destroyed int
	pool.QueryRow(ctx, "SELECT count(*) FROM task WHERE deployed_object_id = $1 AND kind = 'destroy_host' AND status = 'done' AND payload->>'external_ref' = $2", host.ID, originalRef).Scan(&destroyed)
	if destroyed != 1 {
		t.Fatalf("destroy of %s done %d time(s), want 1", originalRef, destroyed)
	}
	got, _ = q.GetDeployedObject(ctx, host.ID)
	if got.Status == "destroy_queued" || got.Status == "destroying" || got.Status == "pending" {
		t.Fatalf("after the destroy and a redeploy, status = %q", got.Status)
	}
}

// TestRebuildNetworkCascadesToGuests: a failed network can be a rebuild target
// (unlike ad-hoc), and rebuilding it pulls in exactly the guests on that network
// -- recreating the subnet means they must redeploy -- without dragging in a
// guest on a different network. content is nil (no depends_on needed): the
// direct guests come from deployed_object.network_name.
func TestRebuildNetworkCascadesToGuests(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-network")
	q := db.New(pool)
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}

	net, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "network", ObjectName: "corp", Fingerprint: "f"})
	if err != nil {
		t.Fatalf("EnsureDeployedObject corp network: %v", err)
	}
	onCorp, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "dc", AsName: db.StrPtr("dc-01"), NetworkName: db.StrPtr("corp"), Fingerprint: "f"})
	if err != nil {
		t.Fatalf("EnsureDeployedObject dc on corp: %v", err)
	}
	if _, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr("web-01"), NetworkName: db.StrPtr("prod"), Fingerprint: "f"}); err != nil {
		t.Fatalf("EnsureDeployedObject web on prod: %v", err)
	}

	affected, err := resolveRebuildSet(ctx, q, build.ID, nil, AdHocTarget{IDs: []string{net.ID.String()}}, false)
	if err != nil {
		t.Fatalf("resolveRebuildSet: %v", err)
	}
	got := map[string]bool{}
	for _, o := range affected {
		got[o.ID.String()] = true
	}
	if !got[net.ID.String()] {
		t.Error("the targeted network itself must be in the rebuild set")
	}
	if !got[onCorp.ID.String()] {
		t.Error("a guest on the rebuilt network must be pulled in (it redeploys on the fresh subnet)")
	}
	if len(affected) != 2 {
		t.Fatalf("expected exactly the network + its one guest (2), got %d: %v", len(affected), got)
	}
}
