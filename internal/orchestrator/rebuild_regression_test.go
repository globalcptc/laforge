package orchestrator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

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
// deletes the actual instance even though Rebuild immediately clears the row's
// external_ref to hold the build in the deploy phase.
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

	// The row's ref is cleared and it's pending again (holds infraPending > 0).
	reread, err := q.GetDeployedObject(ctx, host.ID)
	if err != nil {
		t.Fatalf("GetDeployedObject: %v", err)
	}
	if reread.Status != "pending" {
		t.Errorf("object status = %q, want pending", reread.Status)
	}
	if reread.ExternalRef != nil && *reread.ExternalRef != "" {
		t.Errorf("external_ref = %q, want cleared", *reread.ExternalRef)
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
