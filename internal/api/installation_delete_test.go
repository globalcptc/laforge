package api

import (
	"context"
	"net/http"
	"testing"
)

// createInstallation records one installation the way GitHub's "created"
// webhook does, returning LaForge's id for it.
func createInstallation(t *testing.T, env *apiTestEnv, installationID int64, account string) string {
	t.Helper()
	resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":       "created",
		"installation": map[string]interface{}{"id": installationID, "account": map[string]string{"login": account, "type": "Organization"}},
		"repositories": []map[string]interface{}{{"id": installationID * 10, "full_name": account + "/repo"}},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("installation created: status = %d", resp.StatusCode)
	}
	t.Cleanup(func() { env.q.DeleteGithubInstallation(context.Background(), installationID) })
	inst, err := env.q.GetGithubInstallationByInstallationID(context.Background(), installationID)
	if err != nil {
		t.Fatal(err)
	}
	return inst.ID.String()
}

func deleteInstallation(t *testing.T, env *apiTestEnv, client *http.Client, id string) int {
	t.Helper()
	resp, err := client.Do(mustRequest(t, http.MethodDelete, env.httpURL+"/installations/"+id, ""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestDeleteInstallationUninstallsOnGitHubThenForgets: an instance admin can
// remove a GitHub connection; with the App configured it is uninstalled on
// GitHub first, and an installation GitHub no longer has is still forgotten.
func TestDeleteInstallationUninstallsOnGitHubThenForgets(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	env.server.AdminLogins = []string{"exchanged-del-root"}
	env.server.AppID = "app-id-for-test"
	env.server.AppPrivateKey = testRSAKeyForInstallationTest(t)
	root := signedInClient(t, env, "del-root")
	env.gh.pushTokens["exchanged-del-user"] = true
	user := signedInClient(t, env, "del-user")

	live := createInstallation(t, env, 771001, "del-live-org")
	gone := createInstallation(t, env, 771002, "del-gone-org")
	env.gh.installationsGone = map[string]bool{"771002": true}

	if got := deleteInstallation(t, env, user, live); got != http.StatusForbidden {
		t.Fatalf("non-admin delete: status = %d, want 403", got)
	}
	if got := deleteInstallation(t, env, root, live); got != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200", got)
	}
	if len(env.gh.deletedInstallations) != 1 || env.gh.deletedInstallations[0] != "771001" {
		t.Errorf("uninstall calls on GitHub = %v, want [771001]", env.gh.deletedInstallations)
	}
	if _, err := env.q.GetGithubInstallationByInstallationID(ctx, 771001); err == nil {
		t.Error("installation 771001 should be gone from LaForge")
	}

	// Already uninstalled on GitHub (404): still forgotten here.
	if got := deleteInstallation(t, env, root, gone); got != http.StatusOK {
		t.Fatalf("delete of an installation GitHub no longer has: status = %d, want 200", got)
	}
	if _, err := env.q.GetGithubInstallationByInstallationID(ctx, 771002); err == nil {
		t.Error("installation 771002 should be gone from LaForge")
	}
	// And deleting it again is a no-op.
	if got := deleteInstallation(t, env, root, gone); got != http.StatusNoContent {
		t.Errorf("repeat delete: status = %d, want 204", got)
	}
}

// TestDeleteInstallationWithoutAppOnlyForgets: with no GitHub App configured
// there is nothing to uninstall with, so the connection is only forgotten.
func TestDeleteInstallationWithoutAppOnlyForgets(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-del-root2"}
	root := signedInClient(t, env, "del-root2")
	id := createInstallation(t, env, 771003, "del-noapp-org")

	if got := deleteInstallation(t, env, root, id); got != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200", got)
	}
	if len(env.gh.deletedInstallations) != 0 {
		t.Errorf("no App configured, but GitHub was asked to uninstall: %v", env.gh.deletedInstallations)
	}
	if _, err := env.q.GetGithubInstallationByInstallationID(context.Background(), 771003); err == nil {
		t.Error("installation should be gone from LaForge")
	}
}
