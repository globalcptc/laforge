package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// TestListObjectEventsFiltersToOneObject drives the real task queue with
// the real runner against the fake builder (internal/builder/fake --
// exactly the same proof pattern) so every deploy task actually
// completes and logs a real event tied to its own deployed_object_id,
// then confirms GET .../objects/{id}/events returns only that object's
// events, not the whole build's.
func TestListObjectEventsFiltersToOneObject(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	b := fake.New(env.server.Pool)
	r := &runner.Runner{
		Pool: env.server.Pool, Builder: b, RepoRoot: "../../examples/m7-two-team", ID: "test-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: 5 * time.Second, BuildID: &build.ID,
	}
	for i := 0; i < 20; i++ {
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !did {
			break
		}
	}

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var target, other db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			if target.ID.String() == "" {
				target = o
			} else if other.ID.String() == "" {
				other = o
			}
		}
	}
	if target.ID.String() == "" || other.ID.String() == "" {
		t.Fatal("fixture didn't produce two distinct hosts to compare")
	}

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + target.ID.String() + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var events []db.Event
	json.NewDecoder(resp.Body).Decode(&events)
	if len(events) == 0 {
		t.Fatal("expected at least one real event for the target object (its deploy_host task completed)")
	}

	// Every returned event's task must actually belong to target, not
	// other -- checked by cross-referencing the build's full event list
	// against which task each one came from.
	allEvents, err := env.q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	allTasks, err := env.q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	otherTaskIDs := map[string]bool{}
	for _, tk := range allTasks {
		if tk.DeployedObjectID == other.ID {
			otherTaskIDs[tk.ID.String()] = true
		}
	}
	for _, ev := range events {
		if otherTaskIDs[ev.TaskID.String()] {
			t.Fatalf("event %s belongs to a different object's task, leaked into target's filtered list", ev.ID)
		}
	}
	if len(allEvents) <= len(events) {
		t.Fatalf("filtered list (%d) should be strictly smaller than the build's full journal (%d) with two real objects involved", len(events), len(allEvents))
	}
}

// TestListObjectEventsRejectsIDFromAnotherBuild is the per-object-logs
// counterpart to internal/api/tasks.go's cross-build hardening: an
// object id real in build A must not
// leak its events through build B's endpoint.
func TestListObjectEventsRejectsIDFromAnotherBuild(t *testing.T) {
	env := setupAPITest(t)
	buildA := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	revB, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: env.repo.ID, CommitSha: "object-events-build-b",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision (B): %v", err)
	}
	buildB, err := env.q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: revB.ID, EnvironmentName: "m7-two-team",
	})
	if err != nil {
		t.Fatalf("CreateBuild (B): %v", err)
	}

	objsA, err := env.q.ListDeployedObjectsByBuild(ctx, buildA.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objsA) == 0 {
		t.Fatal("build A has no objects")
	}

	resp, err := client.Get(env.httpURL + "/builds/" + buildB.ID.String() + "/objects/" + objsA[0].ID.String() + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (build A's object id rejected against build B)", resp.StatusCode)
	}
}

// TestListObjectHeartbeatsFiltersAndOrdersReal proves the per-object
// heartbeat history endpoint (the "live troubleshooting" half of
// migrations/00007's append-only log): real rows for the target object
// only, newest first, carrying the real remote address each one
// recorded -- the signal that makes this useful for "rules checking"
// (a changed address mid-competition), not just a liveness count.
func TestListObjectHeartbeatsFiltersAndOrdersReal(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var target, other db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			if target.ID.String() == "" {
				target = o
			} else {
				other = o
			}
		}
	}
	if target.ID.String() == "" || other.ID.String() == "" {
		t.Fatal("fixture didn't produce two distinct hosts")
	}

	for _, addr := range []string{"10.0.0.5:4433", "10.0.0.9:4433"} { // simulates a real remote_addr change across two heartbeats
		if _, err := env.server.Pool.Exec(ctx,
			`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr) VALUES ($1,$2,$3)`,
			target.ID, "fp", addr,
		); err != nil {
			t.Fatalf("inserting heartbeat: %v", err)
		}
	}
	if _, err := env.server.Pool.Exec(ctx,
		`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr) VALUES ($1,$2,$3)`,
		other.ID, "fp-other", "10.0.0.99:4433",
	); err != nil {
		t.Fatalf("inserting other object's heartbeat: %v", err)
	}

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + target.ID.String() + "/heartbeats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var heartbeats []db.AgentHeartbeat
	json.NewDecoder(resp.Body).Decode(&heartbeats)
	if len(heartbeats) != 2 {
		t.Fatalf("got %d heartbeat(s), want exactly 2 (target's own, not other's)", len(heartbeats))
	}
	// Newest first.
	if heartbeats[0].RemoteAddr == nil || *heartbeats[0].RemoteAddr != "10.0.0.9:4433" {
		t.Fatalf("heartbeats[0].RemoteAddr = %v, want the most recently inserted address", heartbeats[0].RemoteAddr)
	}
	for _, h := range heartbeats {
		if h.DeployedObjectID != target.ID {
			t.Fatalf("heartbeat for a different object (%s) leaked into target's list", h.DeployedObjectID)
		}
	}
}

// TestListObjectHeartbeatsRejectsIDFromAnotherBuild mirrors
// TestListObjectEventsRejectsIDFromAnotherBuild for the heartbeat
// history endpoint.
func TestListObjectHeartbeatsRejectsIDFromAnotherBuild(t *testing.T) {
	env := setupAPITest(t)
	buildA := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	revB, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: env.repo.ID, CommitSha: "object-heartbeats-build-b",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision (B): %v", err)
	}
	buildB, err := env.q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: revB.ID, EnvironmentName: "m7-two-team",
	})
	if err != nil {
		t.Fatalf("CreateBuild (B): %v", err)
	}

	objsA, err := env.q.ListDeployedObjectsByBuild(ctx, buildA.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objsA) == 0 {
		t.Fatal("build A has no objects")
	}

	resp, err := client.Get(env.httpURL + "/builds/" + buildB.ID.String() + "/objects/" + objsA[0].ID.String() + "/heartbeats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (build A's object id rejected against build B)", resp.StatusCode)
	}
}
