package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/checkout"
)

// TestListBranchesRequiresGitHubCredentials proves the degraded-but-
// honest fallback: no GitHub App and no service token configured (the
// default in setupAPITest) means real branches genuinely can't be
// listed, and the handler says so clearly instead of returning an empty
// list that looks like "this repo has no branches."
func TestListBranchesRequiresGitHubCredentials(t *testing.T) {
	env := setupAPITest(t)
	client := signedInClient(t, env, "good-token")

	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/branches")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (no GitHub credentials configured)", resp.StatusCode)
	}
}

// TestListBranchesReturnsRealBranches configures a service token (the
// simplest of the two credential paths checkout.Cache.resolveToken
// supports) and proves the real list -- including the repository's own
// default_branch -- comes back through the fake GitHub server exactly as
// ListBranches/GetRepo would report it live.
func TestListBranchesReturnsRealBranches(t *testing.T) {
	env := setupAPITest(t)
	env.gh.branches = []string{"main", "dev", "feature/x"}
	env.server.Checkouts = &checkout.Cache{Q: env.q, GH: env.server.GH, ServiceToken: "good-token"}

	client := signedInClient(t, env, "good-token")
	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/branches")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out listBranchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.DefaultBranch != "main" {
		t.Fatalf("default_branch = %q, want main", out.DefaultBranch)
	}
	if len(out.Branches) != 3 {
		t.Fatalf("branches = %+v, want 3 real branches", out.Branches)
	}
}

// TestListEnvironmentFilesFallsBackToRepoRoot covers the same degraded-
// but-working path handleRenderObject already relies on: with no
// Checkouts configured, this reads the fixed s.RepoRoot checkout
// directly (examples/lm-test, set by setupAPITest) and finds its one
// real environment file, ignoring the branch query param entirely --
// same single-repo fallback as every other Checkouts-aware handler.
func TestListEnvironmentFilesFallsBackToRepoRoot(t *testing.T) {
	env := setupAPITest(t)
	client := signedInClient(t, env, "good-token")

	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/environment-files")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out listEnvironmentFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.HasErrors {
		t.Fatalf("examples/lm-test should be valid content, got has_errors=true")
	}
	found := false
	for _, f := range out.Files {
		if f.Path == "lm-test.yaml" {
			found = true
		}
	}
	if !found {
		t.Fatalf("files = %+v, want lm-test.yaml among them", out.Files)
	}

	// The file picker's listing: every YAML file on the branch, the
	// environment file included, nested paths relative to the root.
	yamls := map[string]bool{}
	nested := false
	for _, p := range out.YAMLFiles {
		yamls[p] = true
		if strings.Contains(p, "/") {
			nested = true
		}
	}
	if !yamls["lm-test.yaml"] || !nested {
		t.Fatalf("yaml_files = %v, want lm-test.yaml and nested files", out.YAMLFiles)
	}
}

// TestListBuildersForAnySignedInPerson: the New Build picker's builder list
// (names and types only) doesn't need instance admin.
func TestListBuildersForAnySignedInPerson(t *testing.T) {
	env := setupAPITest(t)
	client := signedInClient(t, env, "builders-reader")
	if got := getStatus(t, client, env.httpURL+"/builders"); got != http.StatusOK {
		t.Fatalf("GET /builders: status = %d, want 200", got)
	}
	if got := getStatus(t, &http.Client{}, env.httpURL+"/builders"); got != http.StatusUnauthorized {
		t.Fatalf("signed out: status = %d, want 401", got)
	}
}

func TestListEnvironmentFilesRequiresBranchWhenCheckoutsConfigured(t *testing.T) {
	env := setupAPITest(t)
	env.server.Checkouts = &checkout.Cache{Q: env.q, GH: env.server.GH, ServiceToken: "good-token"}
	client := signedInClient(t, env, "good-token")

	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/environment-files")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (branch is required once Checkouts is configured)", resp.StatusCode)
	}
}
