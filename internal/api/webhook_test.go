package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestPushAutoBuildsAFollowingConfiguredBuild is "auto-build is on by
// default, always... produces a resolved, rendered build. It costs
// nothing and touches no hoster, so there is no reason not to"
// proven end to end: register a configured build
// (follow_enabled defaults true -- handleCreateConfiguredBuild's own doc
// comment), push a real commit through the real webhook path, and
// confirm a real `planned` build already exists -- resolved against the
// tracked commit and the configured build's own environment file -- with
// no explicit POST to /configured-builds/{id}/builds anywhere in this
// test. TestTriggerBuildAndDeployManualFlow covers the manual button;
// this covers the webhook doing the same thing on its own.
func TestPushAutoBuildsAFollowingConfiguredBuild(t *testing.T) {
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
	if !cb.AutoDeployEnabled {
		t.Fatalf("a freshly created configured build should default to auto_deploy_enabled=true")
	}

	before, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository (before push): %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("expected zero builds before any push, got %d", len(before))
	}

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push: status = %d", resp.StatusCode)
	}

	rev, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}

	after, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository (after push): %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("expected exactly one auto-built build after the push, got %d", len(after))
	}
	build := after[0]
	// Auto-deploy is on by default and nothing is live yet, so the first push
	// both builds AND deploys: the fresh build goes straight to 'deploying'
	// rather than sitting in 'planned' for a manual click ("with auto-deploy on,
	// deploy means deploy" -- webhook.go's reconcile).
	if build.Status != "deploying" {
		t.Fatalf("auto-built build status = %q, want deploying (auto-deploy on, nothing live -> deploy the first build)", build.Status)
	}
	if build.ConfiguredBuildID != cb.ID {
		t.Fatalf("auto-built build configured_build_id = %v, want %v", build.ConfiguredBuildID, cb.ID)
	}
	if build.ContentRevisionID != rev.ID {
		t.Fatalf("auto-built build content_revision_id = %v, want the just-pushed revision %v", build.ContentRevisionID, rev.ID)
	}
	if build.EnvironmentName != "lm-test" {
		t.Fatalf("auto-built build environment_name = %q, want lm-test", build.EnvironmentName)
	}

	// Auto-deploy fired exactly once (for this first build); re-reading it shows
	// the same 'deploying', no further drift.
	stillDeploying, err := env.q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if stillDeploying.Status != "deploying" {
		t.Fatalf("build status drifted to %q, want a stable deploying", stillDeploying.Status)
	}
}

// TestPushDoesNotAutoBuildWhenLocked: a locked configured build
// (competition_started) is frozen -- no auto-build, no auto-deploy. Auto-
// build is otherwise always on; the lock is the only thing that stops a
// push from advancing/building it.
func TestPushDoesNotAutoBuildWhenLocked(t *testing.T) {
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

	lockReq, _ := http.NewRequest(http.MethodPost, env.httpURL+"/configured-builds/"+cb.ID.String()+"/lock", bytes.NewReader(mustJSON(t, map[string]bool{"started": true})))
	lockReq.Header.Set("Authorization", "Bearer good-token")
	lockResp, err := http.DefaultClient.Do(lockReq)
	if err != nil {
		t.Fatal(err)
	}
	lockResp.Body.Close()
	if lockResp.StatusCode != http.StatusOK {
		t.Fatalf("locking: status = %d", lockResp.StatusCode)
	}

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push: status = %d", resp.StatusCode)
	}

	after, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected zero builds while locked, got %d", len(after))
	}

	updatedCB, err := env.q.GetConfiguredBuild(ctx, cb.ID)
	if err != nil {
		t.Fatalf("GetConfiguredBuild: %v", err)
	}
	// Create auto-pulls the branch HEAD, so the tracked revision is set at create
	// time (before the lock). The LOCKED push must not ADVANCE it -- with a single
	// commit that means it stays exactly what create set, and (asserted above) no
	// build is created.
	if updatedCB.CurrentContentRevisionID != cb.CurrentContentRevisionID {
		t.Fatalf("locked push changed the tracked revision from %v to %v", cb.CurrentContentRevisionID, updatedCB.CurrentContentRevisionID)
	}
}

// TestPushAppliesAnAlreadyDeployedConfiguredBuildAutomatically is the
// real auto-deploy path: "if it is
// already deployed, the commit is applied to it",
// fully automatic when a repository's own auto_deploy_enabled is on (the
// real default -- migrations/00011). Deploys a build manually once, then
// pushes a second, different commit through the real webhook with no
// manual "Apply" call anywhere, and confirms the SAME build row picked
// up the new content_revision_id in place, with a real event recorded --
// the exact mechanism TestApplyUpcomingAppliesCommitInPlace exercises by
// hand, just triggered by reconcile itself here.
func TestPushAppliesAnAlreadyDeployedConfiguredBuildAutomatically(t *testing.T) {
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

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 1: status = %d", resp.StatusCode)
	}
	rev1, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}

	// Auto-deploy is on by default and nothing was live, so push 1 already built
	// AND deployed. That auto-deployed build is the live one push 2 will apply
	// onto in place -- no manual trigger/deploy needed (which would only add a
	// second deploying build and race the apply).
	afterPush1, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(afterPush1) != 1 {
		t.Fatalf("expected one build after push 1, got %d", len(afterPush1))
	}
	build := afterPush1[0]
	if build.Status != "deploying" {
		t.Fatalf("build after push 1 = %q, want deploying (auto-deploy on, nothing live)", build.Status)
	}
	if build.ContentRevisionID != rev1.ID {
		t.Fatalf("build content_revision_id = %v, want rev1 %v", build.ContentRevisionID, rev1.ID)
	}

	// --- push 2: a genuinely different commit, onto a now-deploying build,
	// with no manual "apply" call anywhere below ---
	sha2 := pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "AUTO_DEPLOY_TEST.txt", "second commit\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": sha2,
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 2: status = %d", resp.StatusCode)
	}
	rev2, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{RepositoryID: env.repo.ID, CommitSha: sha2})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA (rev2): %v", err)
	}
	if rev2.ID == rev1.ID {
		t.Fatalf("push 2 resolved to the same content_revision as push 1 -- test setup is broken")
	}

	updated, err := env.q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if updated.ContentRevisionID != rev2.ID {
		t.Fatalf("build content_revision_id after push 2 = %v, want rev2 %v -- auto-deploy should have applied it automatically", updated.ContentRevisionID, rev2.ID)
	}
	if updated.Status != "deploying" {
		t.Fatalf("build status after push 2 = %q, want still deploying", updated.Status)
	}

	events, err := env.q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Kind == "build.commit_applied" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a build.commit_applied event from the automatic apply, events = %+v", events)
	}
}

// TestPushDoesNotAutoDeployWhenDisabled confirms the real
// POST /repos/{id}/auto-deploy endpoint actually gates reconcile's
// automatic apply -- turning it off leaves an already-deployed build
// untouched by a later push, same as before item 13 existed at all.
func TestPushDoesNotAutoDeployWhenDisabled(t *testing.T) {
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

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 1: status = %d", resp.StatusCode)
	}
	rev1, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}

	// Auto-deploy is on by default, so push 1 already built AND deployed. Use
	// that live build; we then turn auto-deploy off and confirm a later push
	// does not apply onto it.
	afterPush1, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(afterPush1) != 1 {
		t.Fatalf("expected one build after push 1, got %d", len(afterPush1))
	}
	build := afterPush1[0]
	if build.Status != "deploying" {
		t.Fatalf("build after push 1 = %q, want deploying (auto-deploy on)", build.Status)
	}
	if build.ContentRevisionID != rev1.ID {
		t.Fatalf("build content_revision_id = %v, want rev1 %v", build.ContentRevisionID, rev1.ID)
	}

	// Turn auto-deploy off for this configured build (per branch now, not
	// repo-wide). A later push should then build but not deploy.
	updatedCB, err := env.q.SetConfiguredBuildAutoDeploy(ctx, db.SetConfiguredBuildAutoDeployParams{ID: cb.ID, AutoDeployEnabled: false})
	if err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}
	if updatedCB.AutoDeployEnabled {
		t.Fatalf("auto_deploy_enabled after disabling = true, want false")
	}

	sha2 := pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "NO_AUTO_DEPLOY_TEST.txt", "second commit\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": sha2,
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 2: status = %d", resp.StatusCode)
	}

	stillOnRev1, err := env.q.GetBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("GetBuild: %v", err)
	}
	if stillOnRev1.ContentRevisionID != rev1.ID {
		t.Fatalf("build content_revision_id after push 2 = %v, want still rev1 %v -- auto-deploy is disabled, nothing should apply automatically", stillOnRev1.ContentRevisionID, rev1.ID)
	}
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
