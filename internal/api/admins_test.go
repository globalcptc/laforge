package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// emptyInstanceAdmins starts the test from "no admin has ever been
// recorded", putting back whatever the shared dev database held afterwards.
func emptyInstanceAdmins(t *testing.T, env *apiTestEnv) {
	t.Helper()
	ctx := context.Background()
	before, err := env.server.Queries.ListInstanceAdmins(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.server.Pool.Exec(ctx, "DELETE FROM instance_admin"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		env.server.Pool.Exec(ctx, "DELETE FROM instance_admin")
		for _, row := range before {
			env.server.Queries.AddInstanceAdmin(ctx, db.AddInstanceAdminParams{GithubLogin: row.GithubLogin, AddedBy: row.AddedBy})
		}
	})
}

func adminRequest(t *testing.T, client *http.Client, method, url, body string) int {
	t.Helper()
	resp, err := client.Do(mustRequest(t, method, url, body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func listedAdmins(t *testing.T, env *apiTestEnv, client *http.Client) []string {
	t.Helper()
	resp, err := client.Get(env.httpURL + "/instance-admins")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /instance-admins: status = %d, want 200", resp.StatusCode)
	}
	var rows []db.ListInstanceAdminsRow
	json.NewDecoder(resp.Body).Decode(&rows)
	var logins []string
	for _, row := range rows {
		logins = append(logins, row.GithubLogin)
	}
	return logins
}

// TestInstanceAdminsSeedOnceThenManagedInUI: LAFORGE_ADMIN_LOGINS only fills
// an empty list; after that admins are added and removed through the API,
// and the last one can't be removed.
func TestInstanceAdminsSeedOnceThenManagedInUI(t *testing.T) {
	env := setupAPITest(t)
	emptyInstanceAdmins(t, env)
	ctx := context.Background()
	url := env.httpURL + "/instance-admins"

	if err := env.server.SeedInstanceAdmins(ctx, []string{"exchanged-im-root"}); err != nil {
		t.Fatal(err)
	}
	// A later start with a different env var changes nothing.
	if err := env.server.SeedInstanceAdmins(ctx, []string{"exchanged-im-late"}); err != nil {
		t.Fatal(err)
	}
	root := signedInClient(t, env, "im-root")
	if got := listedAdmins(t, env, root); len(got) != 1 || got[0] != "exchanged-im-root" {
		t.Fatalf("after seeding twice: admins = %v, want only exchanged-im-root", got)
	}

	// Someone who can sign in but isn't an admin can't see or change the list.
	env.gh.pushTokens["exchanged-im-second"] = true
	second := signedInClient(t, env, "im-second")
	if got := getStatus(t, second, url); got != http.StatusForbidden {
		t.Fatalf("non-admin list: status = %d, want 403", got)
	}
	if got := adminRequest(t, second, http.MethodPost, url, `{"login":"exchanged-im-second"}`); got != http.StatusForbidden {
		t.Fatalf("non-admin add: status = %d, want 403", got)
	}

	if got := adminRequest(t, root, http.MethodDelete, url+"/exchanged-im-root", ""); got != http.StatusConflict {
		t.Fatalf("removing the last admin: status = %d, want 409", got)
	}
	if got := adminRequest(t, root, http.MethodPost, url, `{"login":"not a login!"}`); got != http.StatusBadRequest {
		t.Fatalf("adding a non-username: status = %d, want 400", got)
	}

	// Added: admin on their very next request, no restart or new sign-in.
	if got := adminRequest(t, root, http.MethodPost, url, `{"login":"@Exchanged-IM-Second"}`); got != http.StatusOK {
		t.Fatalf("add: status = %d, want 200", got)
	}
	if got := listedAdmins(t, env, second); len(got) != 2 {
		t.Fatalf("after add: admins = %v, want 2", got)
	}

	// Removed: same, and removing someone who isn't an admin is a no-op.
	if got := adminRequest(t, root, http.MethodDelete, url+"/exchanged-im-second", ""); got != http.StatusNoContent {
		t.Fatalf("remove: status = %d, want 204", got)
	}
	if got := getStatus(t, second, url); got != http.StatusForbidden {
		t.Fatalf("removed admin list: status = %d, want 403", got)
	}
	if got := adminRequest(t, root, http.MethodDelete, url+"/exchanged-im-second", ""); got != http.StatusNoContent {
		t.Fatalf("remove again: status = %d, want 204", got)
	}
}
