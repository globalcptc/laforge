package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func fetchHome(t *testing.T, env *apiTestEnv, client *http.Client) homeView {
	t.Helper()
	resp, err := client.Get(env.httpURL + "/home")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /home: status = %d, want 200", resp.StatusCode)
	}
	var view homeView
	json.NewDecoder(resp.Body).Decode(&view)
	return view
}

func findHomeBuild(view homeView, repoID, buildID string) (homeBuild, bool) {
	for _, r := range view.Repositories {
		if r.ID != repoID {
			continue
		}
		for _, b := range r.Builds {
			if b.ID == buildID {
				return b, true
			}
		}
	}
	return homeBuild{}, false
}

// TestHomeShowsActiveBuildsGroupedByRepository: Home lists a live build
// under its repository with its teams and what it has deployed, flags a
// failed object in the attention list, and leaves out repositories the
// person can't access.
func TestHomeShowsActiveBuildsGroupedByRepository(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	env.server.AdminLogins = []string{"exchanged-home-root"}
	root := signedInClient(t, env, "home-root")
	member := signedInClient(t, env, "home-member")

	hb, ok := findHomeBuild(fetchHome(t, env, member), env.repo.ID.String(), build.ID.String())
	if !ok {
		t.Fatal("the deploying build should be on Home under its repository")
	}
	if hb.Status != "deploying" || hb.Teams != 2 || hb.Counts.Hosts+hb.Counts.Containers == 0 || hb.Counts.Networks == 0 {
		t.Fatalf("build card = %+v, want deploying, 2 teams, some hosts/containers and networks", hb)
	}

	// Needs-attention is scoped to the viewer's OWN builds, so make member the
	// build's owner before asserting it shows up there.
	memberAcct, err := env.q.GetAccountByLogin(context.Background(), "exchanged-home-member")
	if err != nil {
		t.Fatalf("GetAccountByLogin: %v", err)
	}
	if _, err := env.server.Pool.Exec(context.Background(),
		`UPDATE build SET created_by_account_id = $1 WHERE id = $2`, memberAcct.ID, build.ID); err != nil {
		t.Fatal(err)
	}

	// A failed object puts the build on the owner's attention list.
	if _, err := env.server.Pool.Exec(context.Background(), `
		UPDATE deployed_object SET status = 'deploy_failed'
		WHERE id = (SELECT d.id FROM deployed_object d JOIN team t ON t.id = d.team_id WHERE t.build_id = $1 LIMIT 1)`, build.ID); err != nil {
		t.Fatal(err)
	}
	view := fetchHome(t, env, member)
	flagged := false
	for _, a := range view.Attention {
		if a.BuildID == build.ID.String() && a.Reason == "1 object failed" && a.Category == attnObjectsFailed {
			flagged = true
		}
	}
	if !flagged {
		t.Fatalf("attention = %+v, want the failed object flagged", view.Attention)
	}

	// Someone who does not own the build never sees it in their attention list,
	// even an instance admin (root) who can see the build card itself.
	for _, a := range fetchHome(t, env, root).Attention {
		if a.BuildID == build.ID.String() {
			t.Fatalf("root does not own this build; it must not be in root's attention list: %+v", a)
		}
	}

	// Set to no access: the repository's builds drop off their Home.
	grantLevel(t, env, root, "exchanged-home-member", "none")
	if _, ok := findHomeBuild(fetchHome(t, env, member), env.repo.ID.String(), build.ID.String()); ok {
		t.Fatal("a person with no access to the repository should not see its builds on Home")
	}

	if got := getStatus(t, &http.Client{}, env.httpURL+"/home"); got != http.StatusUnauthorized {
		t.Fatalf("signed out: status = %d, want 401", got)
	}
}

// TestHomeAttentionDismiss: a person can close one of their own attention
// items, which removes it from their Home list (and only theirs), and reopen it.
func TestHomeAttentionDismiss(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	owner := signedInClient(t, env, "attn-owner")
	ctx := context.Background()

	ownerAcct, err := env.q.GetAccountByLogin(ctx, "exchanged-attn-owner")
	if err != nil {
		t.Fatalf("GetAccountByLogin: %v", err)
	}
	if _, err := env.server.Pool.Exec(ctx,
		`UPDATE build SET created_by_account_id = $1 WHERE id = $2`, ownerAcct.ID, build.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.server.Pool.Exec(ctx, `
		UPDATE deployed_object SET status = 'deploy_failed'
		WHERE id = (SELECT d.id FROM deployed_object d JOIN team t ON t.id = d.team_id WHERE t.build_id = $1 LIMIT 1)`, build.ID); err != nil {
		t.Fatal(err)
	}

	attnCount := func() int {
		n := 0
		for _, a := range fetchHome(t, env, owner).Attention {
			if a.BuildID == build.ID.String() {
				n++
			}
		}
		return n
	}
	if attnCount() != 1 {
		t.Fatalf("before dismiss: attention items for build = %d, want 1", attnCount())
	}

	dismiss := func(path, category string) int {
		body := `{"build_id":"` + build.ID.String() + `","category":"` + category + `"}`
		resp, err := owner.Post(env.httpURL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	if got := dismiss("/home/attention/dismiss", attnObjectsFailed); got != http.StatusNoContent {
		t.Fatalf("dismiss: status = %d, want 204", got)
	}
	if attnCount() != 0 {
		t.Fatalf("after dismiss: attention items for build = %d, want 0", attnCount())
	}

	// An unknown category is rejected, not silently stored.
	if got := dismiss("/home/attention/dismiss", "not-a-category"); got != http.StatusBadRequest {
		t.Fatalf("dismiss bad category: status = %d, want 400", got)
	}

	// Reopening restores it.
	if got := dismiss("/home/attention/undismiss", attnObjectsFailed); got != http.StatusNoContent {
		t.Fatalf("undismiss: status = %d, want 204", got)
	}
	if attnCount() != 1 {
		t.Fatalf("after undismiss: attention items for build = %d, want 1", attnCount())
	}
}
