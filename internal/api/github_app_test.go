package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

func testRSAKeyForInstallationTest(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test RSA key: %v", err)
	}
	return key
}

// TestClientIDEndpointLetsCLILoginSkipLocalConfig is the fix for the
// other real UX gap that was closed: `laforge login` used to
// require GITHUB_OAUTH_CLIENT_ID set locally, an internal App
// identifier nobody running the CLI should need to know out of band.
// GET /auth/client-id (not a secret -- GitHub treats client ids as
// public) is what cmd/laforge/github.go's runLogin now fetches instead.
func TestClientIDEndpointLetsCLILoginSkipLocalConfig(t *testing.T) {
	env := setupAPITest(t)
	resp, err := http.Get(env.httpURL + "/auth/client-id")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.ClientID != "test-client-id" {
		t.Fatalf("client_id = %q, want the server's real configured value", out.ClientID)
	}
}

func TestClientIDEndpointWithNoAppConfigured(t *testing.T) {
	env := setupAPITest(t)
	env.server.GitHubClientID = ""
	resp, err := http.Get(env.httpURL + "/auth/client-id")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (no App configured -- an honest answer, not an empty client_id)", resp.StatusCode)
	}
}

// TestGitHubAppInstallApproveAndDeindex is "install on GitHub, approve
// in LaForge" end to end, through the real HTTP webhook path (no
// shortcuts through the DB): a real `installation` "created" event
// records the installation and its one repository; that repository
// shows up as unapproved; an instance admin approves it, creating the
// real `repository` row tagged with its installation; the same
// repository no longer shows up as unapproved once approved. A second
// repository added later via `installation_repositories` "added" is
// covered too, and "removed" makes it disappear again before it's ever
// approved.
func TestGitHubAppInstallApproveAndDeindex(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	env.server.AdminLogins = []string{"exchanged-installer-admin"}

	resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action": "created",
		"installation": map[string]interface{}{
			"id":      918273,
			"account": map[string]string{"login": "some-org", "type": "Organization"},
		},
		"repositories": []map[string]interface{}{
			{"id": 555, "full_name": "some-org/some-repo"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation created: status = %d", resp.StatusCode)
	}

	inst, err := env.q.GetGithubInstallationByInstallationID(ctx, 918273)
	if err != nil {
		t.Fatalf("GetGithubInstallationByInstallationID: %v", err)
	}
	if inst.AccountLogin != "some-org" || inst.AccountType != "Organization" {
		t.Fatalf("installation = %+v, want account_login=some-org account_type=Organization", inst)
	}

	adminClient := signedInClient(t, env, "installer-admin")
	nonAdminClient := signedInClient(t, env, "not-an-admin")

	// A signed-in user who isn't in AdminLogins can't even list what's
	// pending approval.
	resp, err = nonAdminClient.Get(env.httpURL + "/installations/repositories")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("list as non-admin: status = %d, want 403", resp.StatusCode)
	}

	resp, err = adminClient.Get(env.httpURL + "/installations/repositories")
	if err != nil {
		t.Fatal(err)
	}
	var pending []db.ListUnapprovedInstallationRepositoriesRow
	json.NewDecoder(resp.Body).Decode(&pending)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list as admin: status = %d, want 200", resp.StatusCode)
	}
	if len(pending) != 1 || pending[0].GithubOwner != "some-org" || pending[0].GithubRepo != "some-repo" {
		t.Fatalf("pending = %+v, want exactly some-org/some-repo", pending)
	}

	approveBody, _ := json.Marshal(approveInstalledRepositoryRequest{
		Owner: "some-org", Repo: "some-repo", InstallationID: inst.ID.String(),
	})
	resp, err = adminClient.Post(env.httpURL+"/installations/repositories/approve", "application/json", bytes.NewReader(approveBody))
	if err != nil {
		t.Fatal(err)
	}
	var approved db.Repository
	json.NewDecoder(resp.Body).Decode(&approved)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("approve: status = %d, want 201", resp.StatusCode)
	}
	if approved.GithubOwner != "some-org" || approved.GithubRepo != "some-repo" {
		t.Fatalf("approved repository = %+v", approved)
	}
	if approved.InstallationID != inst.ID {
		t.Fatalf("approved.InstallationID = %v, want %v (tagged with the installation it came from)", approved.InstallationID, inst.ID)
	}
	t.Cleanup(func() { env.server.Pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", approved.ID) })

	// Now approved -- it must no longer show up as pending.
	resp, err = adminClient.Get(env.httpURL + "/installations/repositories")
	if err != nil {
		t.Fatal(err)
	}
	var afterApprove []db.ListUnapprovedInstallationRepositoriesRow
	json.NewDecoder(resp.Body).Decode(&afterApprove)
	resp.Body.Close()
	if len(afterApprove) != 0 {
		t.Fatalf("pending after approval = %+v, want empty", afterApprove)
	}

	// A second repository added to the same installation later.
	resp = postWebhook(t, env.httpURL, "installation_repositories", map[string]interface{}{
		"action":       "added",
		"installation": map[string]interface{}{"id": 918273},
		"repositories_added": []map[string]interface{}{
			{"id": 556, "full_name": "some-org/second-repo"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation_repositories added: status = %d", resp.StatusCode)
	}
	resp, err = adminClient.Get(env.httpURL + "/installations/repositories")
	if err != nil {
		t.Fatal(err)
	}
	var afterAdd []db.ListUnapprovedInstallationRepositoriesRow
	json.NewDecoder(resp.Body).Decode(&afterAdd)
	resp.Body.Close()
	if len(afterAdd) != 1 || afterAdd[0].GithubRepo != "second-repo" {
		t.Fatalf("pending after add = %+v, want exactly second-repo", afterAdd)
	}

	// Removed before ever being approved -- disappears entirely, no
	// repository row was ever created for it.
	resp = postWebhook(t, env.httpURL, "installation_repositories", map[string]interface{}{
		"action":       "removed",
		"installation": map[string]interface{}{"id": 918273},
		"repositories_removed": []map[string]interface{}{
			{"id": 556, "full_name": "some-org/second-repo"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation_repositories removed: status = %d", resp.StatusCode)
	}
	resp, err = adminClient.Get(env.httpURL + "/installations/repositories")
	if err != nil {
		t.Fatal(err)
	}
	var afterRemove []db.ListUnapprovedInstallationRepositoriesRow
	json.NewDecoder(resp.Body).Decode(&afterRemove)
	resp.Body.Close()
	if len(afterRemove) != 0 {
		t.Fatalf("pending after remove = %+v, want empty", afterRemove)
	}

	// Uninstalling the App entirely removes the installation and cascades
	// to installation_repository -- the approved repository row itself
	// is untouched (it's LaForge's own tracked content, not GitHub's to
	// delete).
	resp = postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":       "deleted",
		"installation": map[string]interface{}{"id": 918273, "account": map[string]string{"login": "some-org", "type": "Organization"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation deleted: status = %d", resp.StatusCode)
	}
	if _, err := env.q.GetGithubInstallationByInstallationID(ctx, 918273); err == nil {
		t.Fatal("installation still exists after a \"deleted\" event")
	}
	stillThere, err := env.q.GetRepository(ctx, approved.ID)
	if err != nil {
		t.Fatalf("approved repository was deleted along with its installation, want it kept: %v", err)
	}
	if stillThere.GithubOwner != "some-org" {
		t.Fatalf("unexpected repository after installation deletion: %+v", stillThere)
	}
}

// TestListInstallationsShowsBothApprovedAndPendingRepos is the real fix
// for a product gap: the old "Installations" screen (GET
// /installations/repositories) only ever listed unapproved repos, so an
// installation with everything already approved simply vanished from the
// one screen meant to make every installation manageable ("list
// everywhere we have the app installed that we know about"). GET
// /installations lists every real installation with every repo it
// covers, each with its own real approved/pending state -- proven here
// with one installation carrying one of each.
func TestListInstallationsShowsBothApprovedAndPendingRepos(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	env.server.AdminLogins = []string{"exchanged-installations-admin"}

	resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action": "created",
		"installation": map[string]interface{}{
			"id":      918274,
			"account": map[string]string{"login": "another-org", "type": "Organization"},
		},
		"repositories": []map[string]interface{}{
			{"id": 557, "full_name": "another-org/approved-repo"},
			{"id": 558, "full_name": "another-org/pending-repo"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation created: status = %d", resp.StatusCode)
	}
	// DeleteGithubInstallation cascades to installation_repository (ON
	// DELETE CASCADE) -- without this, "pending-repo" would never be
	// approved or otherwise removed, so it'd leak into every other
	// test's own ListUnapprovedInstallationRepositories result for as
	// long as this local Postgres instance lives (found live: it did,
	// breaking TestGitHubAppInstallApproveAndDeindex's own exact-count
	// assertion on a later run).
	t.Cleanup(func() { env.q.DeleteGithubInstallation(context.Background(), 918274) })
	inst, err := env.q.GetGithubInstallationByInstallationID(ctx, 918274)
	if err != nil {
		t.Fatalf("GetGithubInstallationByInstallationID: %v", err)
	}

	adminClient := signedInClient(t, env, "installations-admin")
	nonAdminClient := signedInClient(t, env, "not-an-admin")

	resp, err = nonAdminClient.Get(env.httpURL + "/installations")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("list as non-admin: status = %d, want 403", resp.StatusCode)
	}

	approveBody, _ := json.Marshal(approveInstalledRepositoryRequest{
		Owner: "another-org", Repo: "approved-repo", InstallationID: inst.ID.String(),
	})
	resp, err = adminClient.Post(env.httpURL+"/installations/repositories/approve", "application/json", bytes.NewReader(approveBody))
	if err != nil {
		t.Fatal(err)
	}
	var approved db.Repository
	json.NewDecoder(resp.Body).Decode(&approved)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("approve: status = %d, want 201", resp.StatusCode)
	}
	t.Cleanup(func() { env.server.Pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", approved.ID) })

	resp, err = adminClient.Get(env.httpURL + "/installations")
	if err != nil {
		t.Fatal(err)
	}
	var views []installationView
	json.NewDecoder(resp.Body).Decode(&views)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list installations: status = %d, want 200", resp.StatusCode)
	}

	var found *installationView
	for i := range views {
		if views[i].InstallationID == 918274 {
			found = &views[i]
		}
	}
	if found == nil {
		t.Fatalf("installation 918274 not present in %+v", views)
	}
	if len(found.Repos) != 2 {
		t.Fatalf("expected 2 repos on the installation (one approved, one pending), got %+v", found.Repos)
	}
	byName := map[string]installedRepoView{}
	for _, r := range found.Repos {
		byName[r.GithubRepo] = r
	}
	if !byName["approved-repo"].Approved {
		t.Errorf("approved-repo.Approved = false, want true: %+v", byName["approved-repo"])
	}
	if byName["pending-repo"].Approved {
		t.Errorf("pending-repo.Approved = true, want false (never approved): %+v", byName["pending-repo"])
	}
}

// TestPushEventFetchesCloneURLThroughInstallationToken is the
// actual security fix, proven end to end: a push webhook delivered with
// an installation id resolves the repository's real clone URL through a
// freshly minted installation access token (a real RSA-signed JWT,
// exchanged against the fake server's /app/installations/.../access_tokens),
// never from anything the payload itself claims -- the payload in this
// test doesn't even have a clone_url field any more (see pushPayload).
func TestPushEventFetchesCloneURLThroughInstallationToken(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	key := testRSAKeyForInstallationTest(t)
	env.server.AppID = "app-id-for-test"
	env.server.AppPrivateKey = key

	inst, err := env.q.UpsertGithubInstallation(ctx, db.UpsertGithubInstallationParams{
		InstallationID: 4242, AccountLogin: env.owner, AccountType: "Organization",
	})
	if err != nil {
		t.Fatalf("UpsertGithubInstallation: %v", err)
	}
	t.Cleanup(func() { env.q.DeleteGithubInstallation(ctx, 4242) })
	if _, err := env.server.Pool.Exec(ctx, "UPDATE repository SET installation_id = $1 WHERE id = $2", inst.ID, env.repo.ID); err != nil {
		t.Fatalf("tagging test repo with the installation: %v", err)
	}

	resp := postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository":   map[string]string{"full_name": env.owner + "/" + env.name},
		"installation": map[string]interface{}{"id": 4242},
	})
	if resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		t.Fatalf("push via installation token: status = %d: %s", resp.StatusCode, body.String())
	}

	rev, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}
	if !rev.Valid {
		t.Fatalf("content should be valid (real examples/lm-test): %s", rev.ValidationErrors)
	}
}

// TestInstallationEventSyncsAllRepositoriesInstall is the real fix for
// an "all repositories"
// install's own webhook payload carries an empty repositories array (by
// GitHub's own design), which used to mean LaForge silently tracked
// nothing for that installation at all. With a real GitHub App
// configured (AppID/AppPrivateKey), the installation event now mints a
// real installation token and calls GET /installation/repositories
// itself, populating installation_repository from the real list.
func TestInstallationEventSyncsAllRepositoriesInstall(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	env.server.AppID = "app-id-for-test"
	env.server.AppPrivateKey = testRSAKeyForInstallationTest(t)
	env.gh.installationRepos = []ghclient.Repo{
		{ID: 9001, FullName: "all-org/repo-one"},
		{ID: 9002, FullName: "all-org/repo-two"},
	}

	resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":               "created",
		"repository_selection": "all",
		"installation": map[string]interface{}{
			"id":      918275,
			"account": map[string]string{"login": "all-org", "type": "Organization"},
		},
		// Deliberately empty/absent, matching a real "all repositories"
		// payload -- the sync path must not depend on this at all.
		"repositories": []map[string]interface{}{},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation created (all repositories): status = %d", resp.StatusCode)
	}
	t.Cleanup(func() { env.q.DeleteGithubInstallation(context.Background(), 918275) })

	inst, err := env.q.GetGithubInstallationByInstallationID(ctx, 918275)
	if err != nil {
		t.Fatalf("GetGithubInstallationByInstallationID: %v", err)
	}
	pending, err := env.q.ListUnapprovedInstallationRepositories(ctx)
	if err != nil {
		t.Fatalf("ListUnapprovedInstallationRepositories: %v", err)
	}
	var found []string
	for _, p := range pending {
		if p.InstallationID == inst.ID {
			found = append(found, p.GithubOwner+"/"+p.GithubRepo)
		}
	}
	if len(found) != 2 {
		t.Fatalf("synced repos for the all-repositories install = %v, want repo-one and repo-two", found)
	}
}

// TestInstallationEventAllRepositoriesWithNoAppConfiguredFailsHonestly
// proves the real limit: without a GitHub App configured, there's no
// installation token to mint, so an "all repositories" install's sync
// can't happen -- a clear 500 (surfaced to GitHub's own delivery log for
// a real admin to notice) rather than silently tracking nothing and
// looking like it worked.
func TestInstallationEventAllRepositoriesWithNoAppConfiguredFailsHonestly(t *testing.T) {
	env := setupAPITest(t)
	resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":               "created",
		"repository_selection": "all",
		"installation": map[string]interface{}{
			"id":      918276,
			"account": map[string]string{"login": "no-app-org", "type": "Organization"},
		},
	})
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (no GitHub App configured to mint an installation token)", resp.StatusCode)
	}
	t.Cleanup(func() { env.q.DeleteGithubInstallation(context.Background(), 918276) })
}
