package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// A key's owner gets no GitHub token of their own, so their access comes from
// an admin's setting or from GitHub's answer about their login.
func TestMCPAccessReadableRepos(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	n := os.Getpid()
	account := func(login string, id int64) pgtype.UUID {
		a, err := env.q.UpsertAccount(ctx, db.UpsertAccountParams{GithubID: id, GithubLogin: login})
		if err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	collab := account(fmt.Sprintf("mcp-collab-%d", n), int64(800_000_000+n))
	stranger := account(fmt.Sprintf("mcp-stranger-%d", n), int64(810_000_000+n))
	revoked := account(fmt.Sprintf("mcp-revoked-%d", n), int64(820_000_000+n))
	admin := account(fmt.Sprintf("mcp-admin-%d", n), int64(830_000_000+n))
	env.server.AdminLogins = []string{fmt.Sprintf("mcp-admin-%d", n)}

	full := env.owner + "/" + env.name
	env.gh.collaborators = map[string][]ghclient.Collaborator{full: {
		{Login: fmt.Sprintf("mcp-collab-%d", n), Permissions: ghclient.RepoPermissions{Pull: true}},
		{Login: fmt.Sprintf("mcp-revoked-%d", n), Permissions: ghclient.RepoPermissions{Push: true}},
	}}
	// An admin's 'none' beats GitHub's answer.
	if _, err := env.q.UpsertRepositoryAccess(ctx, db.UpsertRepositoryAccessParams{RepositoryID: env.repo.ID, AccountID: revoked, Level: "none"}); err != nil {
		t.Fatal(err)
	}

	access := env.server.MCPAccess()
	has := func(account pgtype.UUID, login string) bool {
		t.Helper()
		repos, err := access.ReadableRepos(ctx, account, login)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range repos {
			if r == env.repo.ID {
				return true
			}
		}
		return false
	}
	if !has(collab, fmt.Sprintf("mcp-collab-%d", n)) {
		t.Error("a read collaborator can't read the repository")
	}
	if has(stranger, fmt.Sprintf("mcp-stranger-%d", n)) {
		t.Error("someone GitHub doesn't list can read the repository")
	}
	if has(revoked, fmt.Sprintf("mcp-revoked-%d", n)) {
		t.Error("an admin's 'none' was ignored")
	}
	if !has(admin, fmt.Sprintf("mcp-admin-%d", n)) {
		t.Error("an instance admin can't read the repository")
	}
}

func TestPermissionLevel(t *testing.T) {
	for perm, want := range map[string]accessLevel{
		"admin": levelAdmin, "maintain": levelBuild, "write": levelBuild, "triage": levelRead, "read": levelRead, "none": levelNone, "": levelNone,
	} {
		if got := permissionLevel(perm); got != want {
			t.Errorf("permissionLevel(%q) = %v, want %v", perm, got, want)
		}
	}
}

// /mcp goes to the MCP handler ahead of every REST route, and says so plainly
// when the MCP server isn't configured.
func TestMCPMount(t *testing.T) {
	env := setupAPITest(t)
	resp, err := http.Post(env.httpURL+"/mcp", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/mcp with no MCP server: status %d, want 404", resp.StatusCode)
	}

	env.server.MCP = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "mcp handler")
	})
	resp, err = http.Post(env.httpURL+"/mcp", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "mcp handler" || resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("/mcp: body %q, CORS %q", body, resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
