package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestPushSupersedesOlderPlannedBuild: with auto-deploy off, every CI-passing
// push used to leave one more 'planned' build behind forever (prod reached 153,
// each holding a reconcile lane). Now a second push leaves exactly one planned
// build -- the newest commit's -- and the older one is terminal 'discarded'
// with an event naming what superseded it. Removing the configured build then
// discards its last planned build too, instead of orphaning it as a detached
// planned row.
func TestPushSupersedesOlderPlannedBuild(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	cbBody, _ := json.Marshal(createConfiguredBuildRequest{
		Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", bytes.NewReader(cbBody))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create configured build: status = %d", resp.StatusCode)
	}
	// Auto-deploy off so pushes leave planned builds rather than deploying.
	if _, err := env.q.SetConfiguredBuildAutoDeploy(ctx, db.SetConfiguredBuildAutoDeployParams{ID: cb.ID, AutoDeployEnabled: false}); err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}

	push := func(after string) {
		resp := postWebhook(t, env.httpURL, "push", map[string]interface{}{
			"ref": "refs/heads/main", "after": after,
			"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("webhook push: status = %d", resp.StatusCode)
		}
	}
	push("ignored-in-favor-of-real-fetch")
	after1, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(after1) != 1 || after1[0].Status != "planned" {
		t.Fatalf("after push 1: want one planned build, got %+v", after1)
	}
	first := after1[0]

	sha2 := pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "SUPERSEDE_TEST.txt", "second commit\n")
	push(sha2)
	rev2, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{RepositoryID: env.repo.ID, CommitSha: sha2})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA (rev2): %v", err)
	}

	after2, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(after2) != 2 {
		t.Fatalf("after push 2: want two build rows (history is kept), got %d", len(after2))
	}
	var planned, discarded []db.ListBuildsByRepositoryRow
	for _, b := range after2 {
		switch b.Status {
		case "planned":
			planned = append(planned, b)
		case "discarded":
			discarded = append(discarded, b)
		default:
			t.Fatalf("unexpected build status %q after push 2", b.Status)
		}
	}
	if len(planned) != 1 || len(discarded) != 1 {
		t.Fatalf("after push 2: want exactly one planned and one discarded build, got planned=%d discarded=%d", len(planned), len(discarded))
	}
	if planned[0].ContentRevisionID != rev2.ID {
		t.Fatalf("the surviving planned build is not the newest commit's (rev %v, want %v)", planned[0].ContentRevisionID, rev2.ID)
	}
	if discarded[0].ID != first.ID {
		t.Fatalf("discarded build = %v, want the first push's build %v", discarded[0].ID, first.ID)
	}
	events, err := env.q.ListEventsByBuild(ctx, first.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Kind == "build.discarded" {
			found = true
			var payload struct {
				SupersededBy string `json:"superseded_by"`
			}
			if err := json.Unmarshal(e.Payload, &payload); err != nil || payload.SupersededBy != planned[0].ID.String() {
				t.Fatalf("build.discarded payload = %s, want superseded_by %s", e.Payload, planned[0].ID)
			}
		}
	}
	if !found {
		t.Fatalf("no build.discarded event on the superseded build; events = %+v", events)
	}

	// Removing the configured build discards its remaining planned build.
	del, _ := http.NewRequest(http.MethodDelete, env.httpURL+"/configured-builds/"+cb.ID.String(), nil)
	del.Header.Set("Authorization", "Bearer good-token")
	resp, err = http.DefaultClient.Do(del)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete configured build: status = %d", resp.StatusCode)
	}
	last, err := env.q.GetBuild(ctx, planned[0].ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if last.Status != "discarded" {
		t.Fatalf("planned build after its configured build was removed: status = %q, want discarded", last.Status)
	}
	if last.ConfiguredBuildID.Valid {
		t.Fatalf("configured_build_id should be NULL after the configured build was removed")
	}
	events, err = env.q.ListEventsByBuild(ctx, last.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	found = false
	for _, e := range events {
		if e.Kind == "build.discarded" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no build.discarded event on the orphaned planned build; events = %+v", events)
	}
}
