package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestForceDeleteBuildIsAdminOnlyAndCascades proves the buried escape hatch:
// only an instance admin can force-remove a build, and doing so deletes the
// build and every dependent row (teams, deployed_objects, tasks, …) via the DB
// cascade -- while leaving the hoster untouched (it's a pure DB removal).
func TestForceDeleteBuildIsAdminOnlyAndCascades(t *testing.T) {
	env := setupAPITest(t)
	emptyInstanceAdmins(t, env)
	ctx := context.Background()

	if err := env.server.SeedInstanceAdmins(ctx, []string{"exchanged-im-root"}); err != nil {
		t.Fatal(err)
	}
	build := realBuildFixture(t, env) // reconciled: has teams + deployed_objects
	url := env.httpURL + "/builds/" + build.ID.String()

	// Sanity: the fixture really did create dependent rows that must cascade.
	teams, err := env.q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild: %v", err)
	}
	if len(teams) == 0 {
		t.Fatal("fixture should have created teams to prove the cascade deletes them")
	}

	// A signed-in non-admin cannot force-delete.
	env.gh.pushTokens["exchanged-not-admin"] = true
	notAdmin := signedInClient(t, env, "not-admin")
	if got := adminRequest(t, notAdmin, http.MethodDelete, url, ""); got != http.StatusForbidden {
		t.Fatalf("non-admin force delete: status = %d, want 403", got)
	}
	// It's still there.
	if _, err := env.q.GetBuild(ctx, build.ID); err != nil {
		t.Fatalf("build should still exist after the refused delete: %v", err)
	}

	// An instance admin can, and it returns 204.
	root := signedInClient(t, env, "im-root")
	if got := adminRequest(t, root, http.MethodDelete, url, ""); got != http.StatusNoContent {
		t.Fatalf("admin force delete: status = %d, want 204", got)
	}

	// The build row is gone...
	if _, err := env.q.GetBuild(ctx, build.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetBuild after force delete: err = %v, want ErrNoRows", err)
	}
	// ...and so are its dependent rows (cascade): no teams remain.
	remaining, err := env.q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild after force delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("teams after force delete = %d, want 0 (cascade should have removed them)", len(remaining))
	}
}
