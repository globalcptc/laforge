package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/ghclient"
)

// fetchRepoAccess fetches env.repo's access list as client sees it.
func fetchRepoAccess(t *testing.T, env *apiTestEnv, client *http.Client) repoAccessView {
	t.Helper()
	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/access")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET access: status = %d, want 200", resp.StatusCode)
	}
	var view repoAccessView
	json.NewDecoder(resp.Body).Decode(&view)
	return view
}

// repoAccessLevel is the admin's setting for login on env.repo ("" when
// they follow GitHub or aren't listed).
func repoAccessLevel(t *testing.T, env *apiTestEnv, client *http.Client, login string) string {
	t.Helper()
	for _, p := range fetchRepoAccess(t, env, client).People {
		if strings.EqualFold(p.Login, login) {
			return p.Level
		}
	}
	return ""
}

func myAccess(t *testing.T, env *apiTestEnv, client *http.Client) string {
	t.Helper()
	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/my-access")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Level string `json:"level"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Level
}

func deleteAccess(t *testing.T, env *apiTestEnv, client *http.Client, login string) int {
	t.Helper()
	resp, err := client.Do(mustRequest(t, http.MethodDelete, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/"+login, ""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestRepositoryAccessStartsFromGitHubAndAdminDecides is the model: a
// person's level on a repository starts from their GitHub role there, and
// an admin's setting replaces it -- raising it, lowering it, or taking
// access away -- until it's reset back to GitHub.
func TestRepositoryAccessStartsFromGitHubAndAdminDecides(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-ra-root"}
	root := signedInClient(t, env, "ra-root")
	env.gh.pushTokens["exchanged-ra-writer"] = true
	writer := signedInClient(t, env, "ra-writer")
	repoPath := env.httpURL + "/repos/" + env.repo.ID.String() + "/builds"

	// GitHub write -> build.
	if got := myAccess(t, env, writer); got != "build" {
		t.Fatalf("from GitHub: level = %q, want build", got)
	}

	// Raised above GitHub.
	grantLevel(t, env, root, "exchanged-ra-writer", "manage")
	if got := myAccess(t, env, writer); got != "manage" {
		t.Fatalf("raised: level = %q, want manage", got)
	}

	// Lowered below GitHub.
	grantLevel(t, env, root, "exchanged-ra-writer", "read")
	if got := myAccess(t, env, writer); got != "read" {
		t.Fatalf("lowered: level = %q, want read", got)
	}

	// Taken away entirely, even though GitHub still lets them see it.
	grantLevel(t, env, root, "exchanged-ra-writer", "none")
	if got := myAccess(t, env, writer); got != "none" {
		t.Fatalf("none: level = %q, want none", got)
	}
	if got := getStatus(t, writer, repoPath); got != http.StatusForbidden {
		t.Fatalf("none: listing builds status = %d, want 403", got)
	}

	// Reset: back to their GitHub role.
	if got := deleteAccess(t, env, root, "exchanged-ra-writer"); got != http.StatusNoContent {
		t.Fatalf("reset: status = %d, want 204", got)
	}
	if got := myAccess(t, env, writer); got != "build" {
		t.Fatalf("after reset: level = %q, want build", got)
	}
}

// TestRepositoryAccessListsGitHubCollaborators: the list is GitHub's
// collaborators on this one repository, each with their role and the level
// it gives, merged with anyone an admin has set a level for.
func TestRepositoryAccessListsGitHubCollaborators(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-rl-root"}
	root := signedInClient(t, env, "rl-root")
	env.gh.collaborators = map[string][]ghclient.Collaborator{
		env.repo.GithubOwner + "/" + env.repo.GithubRepo: {
			{ID: 11, Login: "alice", RoleName: "admin", Permissions: ghclient.RepoPermissions{Admin: true, Push: true, Pull: true}},
			{ID: 12, Login: "bob", RoleName: "write", Permissions: ghclient.RepoPermissions{Push: true, Pull: true}},
		},
	}
	grantLevel(t, env, root, "alice", "read")
	grantLevel(t, env, root, "outsider", "build")

	view := fetchRepoAccess(t, env, root)
	if view.GithubError != "" {
		t.Fatalf("github_error = %q, want none", view.GithubError)
	}
	got := map[string]repoAccessPerson{}
	for _, p := range view.People {
		got[p.Login] = p
	}
	if a := got["alice"]; a.GithubRole != "admin" || a.GithubLevel != "admin" || a.Level != "read" || a.Effective != "read" {
		t.Fatalf("alice = %+v, want GitHub admin lowered to read", a)
	}
	if b := got["bob"]; b.GithubRole != "write" || b.GithubLevel != "build" || b.Level != "" || b.Effective != "build" {
		t.Fatalf("bob = %+v, want GitHub write -> build, no setting", b)
	}
	if o := got["outsider"]; o.GithubRole != "" || o.GithubLevel != "none" || o.Effective != "build" {
		t.Fatalf("outsider = %+v, want not a collaborator, build from the admin's setting", o)
	}

	// GitHub failing to list collaborators is reported, and the people with
	// a setting are still there.
	env.gh.collaborators = map[string][]ghclient.Collaborator{}
	view = fetchRepoAccess(t, env, root)
	if view.GithubError == "" || len(view.People) < 2 {
		t.Fatalf("with GitHub failing: github_error = %q, %d people; want an error and the two settings", view.GithubError, len(view.People))
	}
}

// TestRepositoryAdminCantLockThemselvesOut: a repository admin who isn't an
// instance admin can manage the repository's access but can't lower their
// own access below admin (or reset themselves to a lower GitHub role).
func TestRepositoryAdminCantLockThemselvesOut(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-lo-root"}
	root := signedInClient(t, env, "lo-root")
	repoAdmin := signedInClient(t, env, "lo-repo-admin")
	grantLevel(t, env, root, "exchanged-lo-repo-admin", "admin")

	// They can manage others.
	grantLevel(t, env, repoAdmin, "lo-someone", "build")

	if got := putJSON(t, repoAdmin, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/exchanged-lo-repo-admin", `{"level":"build"}`); got != http.StatusBadRequest {
		t.Fatalf("lowering self: status = %d, want 400", got)
	}
	if got := deleteAccess(t, env, repoAdmin, "exchanged-lo-repo-admin"); got != http.StatusBadRequest {
		t.Fatalf("resetting self to a GitHub read role: status = %d, want 400", got)
	}
	// An instance admin can change it.
	grantLevel(t, env, root, "exchanged-lo-repo-admin", "build")

	// A GitHub admin needs no setting to manage access.
	env.gh.adminTokens = map[string]bool{"exchanged-lo-gh-admin": true}
	ghAdmin := signedInClient(t, env, "lo-gh-admin")
	if got := getStatus(t, ghAdmin, env.httpURL+"/repos/"+env.repo.ID.String()+"/access"); got != http.StatusOK {
		t.Fatalf("GitHub admin: status = %d, want 200", got)
	}
	// Someone with only read can't.
	reader := signedInClient(t, env, "lo-reader")
	if got := getStatus(t, reader, env.httpURL+"/repos/"+env.repo.ID.String()+"/access"); got != http.StatusForbidden {
		t.Fatalf("reader: status = %d, want 403", got)
	}
	// And only instance admins see the installations list.
	if got := getStatus(t, ghAdmin, env.httpURL+"/installations"); got != http.StatusForbidden {
		t.Fatalf("installations as a repository admin: status = %d, want 403", got)
	}
}

func putJSON(t *testing.T, client *http.Client, url, body string) int {
	t.Helper()
	resp, err := client.Do(mustRequest(t, http.MethodPut, url, body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func getStatus(t *testing.T, client *http.Client, url string) int {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestRepoListShowsOnlyAccessibleRepositories: the sidebar's list is the
// repositories this person can access -- one they've been set to "none" on
// drops out, and an instance admin sees everything.
func TestRepoListShowsOnlyAccessibleRepositories(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-rv-root"}
	root := signedInClient(t, env, "rv-root")
	member := signedInClient(t, env, "rv-member")

	listed := func(client *http.Client) bool {
		t.Helper()
		resp, err := client.Get(env.httpURL + "/repos")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var repos []struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&repos)
		for _, r := range repos {
			if r.ID == env.repo.ID.String() {
				return true
			}
		}
		return false
	}

	if !listed(member) {
		t.Fatal("a GitHub collaborator should see the repository")
	}
	grantLevel(t, env, root, "exchanged-rv-member", "none")
	if listed(member) {
		t.Fatal("a person set to none on the repository should not see it")
	}
	if !listed(root) {
		t.Fatal("an instance admin should see every repository")
	}
}

// TestInstanceAdminReachesRepoEndpointsWithoutGitHubAccess: an instance
// admin whose own GitHub token can't see a repository still reaches its
// requireLevel-gated endpoints (the builds list 403'd before -- it checked
// the personal token and ignored instance-admin, while other endpoints
// honored it, so an admin saw configured builds but no builds).
func TestInstanceAdminReachesRepoEndpointsWithoutGitHubAccess(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-ia-admin"}
	// denyAllRepoAccess makes GetRepo report no access for every token, so
	// the only thing that can grant access here is instance-admin.
	env.gh.denyAllRepoAccess = true
	admin := signedInClient(t, env, "ia-admin")

	if got := getStatus(t, admin, env.httpURL+"/repos/"+env.repo.ID.String()+"/builds"); got != http.StatusOK {
		t.Fatalf("instance admin GET builds: status = %d, want 200", got)
	}
	// ...and the configured-builds list, which already worked, still does.
	if got := getStatus(t, admin, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds"); got != http.StatusOK {
		t.Fatalf("instance admin GET configured-builds: status = %d, want 200", got)
	}
}
