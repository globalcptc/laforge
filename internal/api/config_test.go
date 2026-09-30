package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestGetConfigReturnsTheConfiguredAppSlug proves GET /config is real and
// unauthenticated (no signed-in client used here at all) -- the UI's
// "Add Repository" flow needs this before there's ever a session.
func TestGetConfigReturnsTheConfiguredAppSlug(t *testing.T) {
	env := setupAPITest(t)
	env.server.AppSlug = "test-laforge-app"

	resp, err := http.Get(env.httpURL + "/config")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got publicConfig
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.GithubAppSlug != "test-laforge-app" {
		t.Fatalf("github_app_slug = %q, want test-laforge-app", got.GithubAppSlug)
	}
}

// TestGetConfigReturnsEmptyAppSlugWhenUnconfigured proves the "no App at
// all" case is real too -- Server.AppPrivateKey's own doc comment ("a
// deployment can run with no App configured at all") applies to the slug
// too, and the UI's own fallback to direct registration keys off this
// exact empty string, not a missing field.
func TestGetConfigReturnsEmptyAppSlugWhenUnconfigured(t *testing.T) {
	env := setupAPITest(t)
	env.server.AppSlug = ""

	resp, err := http.Get(env.httpURL + "/config")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got publicConfig
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.GithubAppSlug != "" {
		t.Fatalf("github_app_slug = %q, want empty", got.GithubAppSlug)
	}
}
