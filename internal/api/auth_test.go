package api

import (
	"bytes"
	"net/http"
	"testing"
)

// TestRequirePushAcceptsARealBrowserSession is a real, previously-broken
// path found while wiring up the follow/lock UI buttons: the actual
// browser client (ui/src/api/client.ts) authenticates every request with
// `credentials: 'include'` alone -- a session cookie, never an
// `Authorization` header -- but handleCreateConfiguredBuild (and
// handleCreateRepository, handleSetConfiguredBuildAutoDeploy, handleSetLock) were gated by
// requirePush, which only ever looked for a Bearer token. A real signed-in
// session hit these with a genuine 401/403, confirmed live before the fix
// (requirePush's own doc comment). This proves the fix: a real OAuth
// session cookie, with push access on the fake GitHub server exactly the
// way GitHub's own permissions would report it, reaches
// handleCreateConfiguredBuild successfully -- no Authorization header
// anywhere in this test, matching the real UI exactly.
func TestRequirePushAcceptsARealBrowserSession(t *testing.T) {
	env := setupAPITest(t)
	env.gh.pushTokens["exchanged-session-owner"] = true // the fake server's own token-for-code convention -- see its own doc comment
	client := signedInClient(t, env, "session-owner")

	body := []byte(`{"branch":"main","environment_path":"lm-test.yaml","builder_config_name":"microcloud"}`)
	resp, err := client.Post(env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create configured build via session cookie alone: status = %d, want 201", resp.StatusCode)
	}
}

// TestRequirePushRejectsASessionWithoutRealPushAccess confirms the fix
// above didn't quietly turn requirePush into "signed in is enough" --
// GitHub's own permissions on the session's token are still what decides,
// same as the pre-existing Bearer path.
func TestRequirePushRejectsASessionWithoutRealPushAccess(t *testing.T) {
	env := setupAPITest(t)
	client := signedInClient(t, env, "no-push-owner") // never added to env.gh.pushTokens

	body := []byte(`{"branch":"main","environment_path":"lm-test.yaml","builder_config_name":"microcloud"}`)
	resp, err := client.Post(env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create configured build with a session lacking real push access: status = %d, want 403", resp.StatusCode)
	}
}

// TestRequirePushStillAcceptsBearerToken confirms the CLI's own
// device-flow-token path (and every pre-existing test using
// "Authorization: Bearer <token>" directly) still works unchanged --
// the session-cookie path added above is additive, not a replacement.
func TestRequirePushStillAcceptsBearerToken(t *testing.T) {
	env := setupAPITest(t)

	body := []byte(`{"branch":"main","environment_path":"lm-test.yaml","builder_config_name":"microcloud"}`)
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create configured build via bearer token: status = %d, want 201", resp.StatusCode)
	}
}
