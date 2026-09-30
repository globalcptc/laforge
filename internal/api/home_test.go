package api

import (
	"context"
	"encoding/json"
	"net/http"
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

	// A failed object puts the build on the attention list.
	if _, err := env.server.Pool.Exec(context.Background(), `
		UPDATE deployed_object SET status = 'deploy_failed'
		WHERE id = (SELECT d.id FROM deployed_object d JOIN team t ON t.id = d.team_id WHERE t.build_id = $1 LIMIT 1)`, build.ID); err != nil {
		t.Fatal(err)
	}
	view := fetchHome(t, env, member)
	flagged := false
	for _, a := range view.Attention {
		if a.BuildID == build.ID.String() && a.Reason == "1 object failed" {
			flagged = true
		}
	}
	if !flagged {
		t.Fatalf("attention = %+v, want the failed object flagged", view.Attention)
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
