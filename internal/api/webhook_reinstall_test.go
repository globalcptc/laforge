package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestReinstallKeepsRepositoriesAttached is the regression test for the
// production outage: an org uninstalled + reinstalled the App (new
// installation_id), and because the installation upsert was keyed on
// installation_id it created a SECOND row, stranding the repository on
// the dead old row -- every build then 404'd minting a token. With the
// upsert keyed on account_login, a reinstall updates the one row in place
// (uuid stable), and AttachOwnerRepositoriesToInstallation re-points any
// repository left detached.
func TestReinstallKeepsRepositoriesAttached(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	pool := env.server.Pool
	const org = "reinstall-org"

	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM repository WHERE github_owner = $1", org)
		pool.Exec(ctx, "DELETE FROM github_installation WHERE account_login = $1", org)
	})

	// Initial install, id 111.
	if resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":       "created",
		"installation": map[string]interface{}{"id": 111, "account": map[string]string{"login": org, "type": "Organization"}},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("install 111: status = %d", resp.StatusCode)
	}
	inst, err := env.q.GetGithubInstallationByInstallationID(ctx, 111)
	if err != nil {
		t.Fatalf("GetGithubInstallationByInstallationID(111): %v", err)
	}
	// A tracked repository owned by the org, tied to that installation row.
	if _, err := pool.Exec(ctx, "INSERT INTO repository (github_owner, github_repo, installation_id) VALUES ($1,$2,$3)", org, "alpha", inst.ID); err != nil {
		t.Fatalf("insert repository: %v", err)
	}

	// Reinstall: SAME org, NEW installation_id 222.
	if resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":       "created",
		"installation": map[string]interface{}{"id": 222, "account": map[string]string{"login": org, "type": "Organization"}},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("reinstall 222: status = %d", resp.StatusCode)
	}

	// Exactly one installation row for the org, now carrying id 222.
	var rows int
	var curID int64
	if err := pool.QueryRow(ctx, "SELECT count(*), max(installation_id) FROM github_installation WHERE account_login = $1", org).Scan(&rows, &curID); err != nil {
		t.Fatalf("counting installations: %v", err)
	}
	if rows != 1 || curID != 222 {
		t.Fatalf("after reinstall: rows=%d installation_id=%d, want exactly 1 row with id 222 (updated in place, not a second row)", rows, curID)
	}
	// The repository still resolves to the live installation -- the whole
	// point: token minting would work again.
	assertRepoAttachedTo(t, pool, org, 222)

	// Now the uninstall-then-reinstall ordering: an uninstall event NULLs
	// the repo's installation_id (ON DELETE SET NULL). A subsequent install
	// must re-attach it.
	if _, err := pool.Exec(ctx, "UPDATE repository SET installation_id = NULL WHERE github_owner = $1", org); err != nil {
		t.Fatalf("null repo installation: %v", err)
	}
	if resp := postWebhook(t, env.httpURL, "installation", map[string]interface{}{
		"action":       "created",
		"installation": map[string]interface{}{"id": 222, "account": map[string]string{"login": org, "type": "Organization"}},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("re-install after null: status = %d", resp.StatusCode)
	}
	assertRepoAttachedTo(t, pool, org, 222)
}

// TestInstallationRepositoriesBootstrapsMissingInstallation covers the
// other half of the outage: the one-time `installation` created event was
// never delivered, so when `installation_repositories` arrived LaForge had
// no installation row and used to drop it. It must now bootstrap the row
// from the event's own installation object.
func TestInstallationRepositoriesBootstrapsMissingInstallation(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	pool := env.server.Pool
	const org = "bootstrap-org"

	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM github_installation WHERE account_login = $1", org)
	})

	// No prior `installation` event -- straight to installation_repositories.
	if resp := postWebhook(t, env.httpURL, "installation_repositories", map[string]interface{}{
		"action":       "added",
		"installation": map[string]interface{}{"id": 333, "account": map[string]string{"login": org, "type": "Organization"}},
		"repositories_added": []map[string]interface{}{
			{"id": 999, "full_name": org + "/thing"},
		},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("installation_repositories: status = %d", resp.StatusCode)
	}

	inst, err := env.q.GetGithubInstallationByInstallationID(ctx, 333)
	if err != nil {
		t.Fatalf("installation was not bootstrapped from installation_repositories: %v", err)
	}
	if inst.AccountLogin != org {
		t.Fatalf("bootstrapped installation account = %q, want %q", inst.AccountLogin, org)
	}
}

func assertRepoAttachedTo(t *testing.T, pool *pgxpool.Pool, org string, wantInstallationID int64) {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM repository r JOIN github_installation gi ON gi.id = r.installation_id WHERE r.github_owner = $1 AND gi.installation_id = $2",
		org, wantInstallationID).Scan(&n); err != nil {
		t.Fatalf("checking repo attachment: %v", err)
	}
	if n != 1 {
		t.Fatalf("repository for %q attached to installation %d: got %d, want 1", org, wantInstallationID, n)
	}
}
