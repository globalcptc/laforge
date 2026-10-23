package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestRetryTeardownAndForceTornDown covers the two operator paths out of a
// teardown that stopped on a permanent builder error: retry (levelManage, like
// teardown itself -- it acts on live infrastructure) puts destroy_failed
// objects back to destroying, clears the stuck banner and requests fresh
// destroys; force (instance-admin-only, like force-delete -- it orphans real
// instances if misused) declares the build torn down without a hoster call.
// Both refuse a build that is not tearing_down.
func TestRetryTeardownAndForceTornDown(t *testing.T) {
	env := setupAPITest(t)
	emptyInstanceAdmins(t, env)
	ctx := context.Background()
	if err := env.server.SeedInstanceAdmins(ctx, []string{"exchanged-im-root"}); err != nil {
		t.Fatal(err)
	}
	build := realBuildFixture(t, env)
	base := env.httpURL + "/builds/" + build.ID.String() + "/teardown"
	owner := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	root := signedInClient(t, env, "im-root")

	post := func(client *http.Client, url string) (int, db.Build) {
		t.Helper()
		resp, err := client.Post(url, "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var b db.Build
		json.NewDecoder(resp.Body).Decode(&b)
		return resp.StatusCode, b
	}

	// Neither path applies to a build that isn't tearing down.
	if got, _ := post(owner, base+"/retry"); got != http.StatusConflict {
		t.Fatalf("retry on a %s build: status = %d, want 409", build.Status, got)
	}
	if got, _ := post(root, base+"/force"); got != http.StatusConflict {
		t.Fatalf("force on a %s build: status = %d, want 409", build.Status, got)
	}

	if got, b := post(owner, base); got != http.StatusAccepted || b.Status != "tearing_down" {
		t.Fatalf("teardown: status = %d / %q, want 202 / tearing_down", got, b.Status)
	}

	// Stand in for the orchestrator having stopped on a permanent error. The
	// fixture's own deploy tasks are still open (nothing ran them); close them
	// first, since one open task per object is all the task table allows and a
	// destroy cannot be requested behind a pending deploy.
	if _, err := env.server.Pool.Exec(ctx, "UPDATE task SET status='done' WHERE build_id=$1 AND status IN ('pending','leased')", build.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.server.Pool.Exec(ctx, "UPDATE deployed_object SET status='destroy_failed', last_error='permanent configuration error: no hosts' WHERE team_id IN (SELECT id FROM team WHERE build_id=$1)", build.ID); err != nil {
		t.Fatal(err)
	}
	if err := env.q.SetBuildReconcileError(ctx, db.SetBuildReconcileErrorParams{ID: build.ID, ReconcileError: db.StrPtr("Teardown is stuck")}); err != nil {
		t.Fatal(err)
	}

	got, b := post(owner, base+"/retry")
	if got != http.StatusAccepted {
		t.Fatalf("retry: status = %d, want 202", got)
	}
	if b.Status != "tearing_down" || b.ReconcileError != nil {
		t.Fatalf("after retry: status=%q reconcile_error=%v, want tearing_down with the banner cleared", b.Status, b.ReconcileError)
	}
	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) == 0 {
		t.Fatal("fixture should have deployed objects")
	}
	for _, o := range objs {
		if o.Status != "destroying" || o.LastError != nil {
			t.Fatalf("object %s after retry: status=%q last_error=%v, want destroying with no error", o.ObjectName, o.Status, o.LastError)
		}
	}
	tasks, err := env.q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	open := 0
	for _, tk := range tasks {
		if tk.Status == "pending" && strings.HasPrefix(tk.Kind, "destroy_") {
			open++
		}
	}
	if open != len(objs) {
		t.Fatalf("open destroy tasks after retry = %d, want %d (one per object)", open, len(objs))
	}

	// Force is admin-only: the repository admin who may tear down may not force.
	if got, _ := post(owner, base+"/force"); got != http.StatusForbidden {
		t.Fatalf("force as repo admin: status = %d, want 403", got)
	}
	got, b = post(root, base+"/force")
	if got != http.StatusOK || b.Status != "torn_down" {
		t.Fatalf("force as instance admin: status = %d / %q, want 200 / torn_down", got, b.Status)
	}
	after, _ := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	for _, o := range after {
		if o.Status != "destroyed" {
			t.Fatalf("object %s after force: status=%q, want destroyed", o.ObjectName, o.Status)
		}
	}
	tasks, _ = env.q.ListTasksByBuild(ctx, build.ID)
	for _, tk := range tasks {
		if tk.Status == "pending" || tk.Status == "leased" {
			t.Fatalf("task %s still %s after force", tk.Kind, tk.Status)
		}
	}
}
