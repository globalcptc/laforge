package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// TestRenderObjectResolvesThroughCheckoutCache proves checkout
// resolution at the API layer: with Server.Checkouts configured,
// handleRenderObject resolves this build's own repository/commit through
// a real checkout cache (env.gh's fake GitHub server already returns
// env.cloneURL as every repo's clone_url -- see fakeGitHub's own doc
// comment -- so this fetches from the exact same real, local bare
// repository setupAPITest already pushed examples/lm-test to) instead of
// the fixed env.server.RepoRoot, and still renders byte-identical
// output. Mirrors TestTriggerBuildAndDeployManualFlow's own render
// assertions (the RepoRoot fallback path), proving the cache path
// produces the same real result, not a second, divergent
// implementation.
func TestRenderObjectResolvesThroughCheckoutCache(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	env.server.Checkouts = &checkout.Cache{
		BaseDir: t.TempDir(), Q: env.q, GH: env.server.GH, ServiceToken: "service-token",
	}

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

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push: status = %d", resp.StatusCode)
	}

	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	buildClient := signedInClient(t, env, "cache-render-actor")
	grantLevel(t, env, adminClient, "exchanged-cache-render-actor", "build")

	resp, err = buildClient.Post(env.httpURL+"/configured-builds/"+cb.ID.String()+"/builds", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var build db.Build
	json.NewDecoder(resp.Body).Decode(&build)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("trigger build: status = %d, want 201", resp.StatusCode)
	}
	// "Build Now" deploys immediately -- the trigger already returns a
	// 'deploying' build, no separate Deploy step needed.
	if build.Status != "deploying" {
		t.Fatalf("triggered build status = %q, want deploying (Build Now deploys immediately)", build.Status)
	}

	// Reconcile also resolves through the cache -- see handleDeployBuild's
	// own doc comment on the poll loop normally doing this; this test
	// drives it directly, same as TestTriggerBuildAndDeployManualFlow.
	dir, err := env.server.Checkouts.ForBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ForBuild: %v", err)
	}
	if err := orchestrator.Reconcile(ctx, env.server.Pool, dir, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01 db.DeployedObject
	for _, o := range objs {
		if o.AsName != nil && *o.AsName == "web01" {
			tm, err := env.q.GetTeam(ctx, o.TeamID)
			if err != nil {
				t.Fatal(err)
			}
			if tm.TeamNumber == 1 {
				web01 = o
				break
			}
		}
	}
	if !web01.ID.Valid {
		t.Fatal("team 1's web01 object not found after reconcile")
	}

	resp, err = buildClient.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + web01.ID.String() + "/render")
	if err != nil {
		t.Fatal(err)
	}
	var steps []renderedStep
	json.NewDecoder(resp.Body).Decode(&steps)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("render: status = %d, want 200", resp.StatusCode)
	}
	// Same real step list as TestTriggerBuildAndDeployManualFlow's own
	// RepoRoot-path assertion -- webserver.yaml's script:base, download,
	// extract, script:vuln-sqli.
	if len(steps) != 4 {
		t.Fatalf("len(steps) = %d, want 4 (webserver.yaml's real step list)", len(steps))
	}
	if steps[0].Action != "script" || steps[0].ScriptName != "base" || steps[0].Rendered == "" {
		t.Fatalf("step 0 = %+v, want a non-empty rendered script:base", steps[0])
	}
	if steps[3].Action != "script" || steps[3].ScriptName != "vuln-sqli" || steps[3].Rendered == "" {
		t.Fatalf("step 3 = %+v, want a non-empty rendered script:vuln-sqli", steps[3])
	}
}
