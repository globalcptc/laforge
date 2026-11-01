package api

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// MCPAccess answers, for the read-only MCP server (internal/mcp), which
// repositories a personal API key's owner can read right now. It's the same
// rule requireLevel applies to a signed-in person -- instance admin, else an
// admin's repository_access setting, else their GitHub permission -- with one
// difference: a key carries no GitHub token of its own, so GitHub is asked
// about the owner's login with the repository's installation token instead
// (falling back to the service token when no GitHub App is configured).
//
// It lives here, on the privileged connection, because the MCP server's own
// database role can't read access grants and has no GitHub credentials.
type MCPAccess struct{ s *Server }

// MCPAccess returns the MCP server's view of repository access.
func (s *Server) MCPAccess() MCPAccess { return MCPAccess{s} }

// ReadableRepos lists every repository the account can read.
func (a MCPAccess) ReadableRepos(ctx context.Context, accountID pgtype.UUID, login string) ([]pgtype.UUID, error) {
	s := a.s
	repos, err := s.Queries.ListRepositories(ctx)
	if err != nil {
		return nil, err
	}
	if s.isAdminLogin(login) {
		out := make([]pgtype.UUID, len(repos))
		for i, r := range repos {
			out[i] = r.ID
		}
		return out, nil
	}
	readable := make([]bool, len(repos))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, repo := range repos {
		wg.Add(1)
		go func(i int, repo db.Repository) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			l, err := a.level(ctx, accountID, login, repo)
			readable[i] = err == nil && l >= levelRead
		}(i, repo)
	}
	wg.Wait()
	out := []pgtype.UUID{}
	for i, ok := range readable {
		if ok {
			out = append(out, repos[i].ID)
		}
	}
	return out, nil
}

// level is effectiveLevel for someone who isn't making the request.
func (a MCPAccess) level(ctx context.Context, accountID pgtype.UUID, login string, repo db.Repository) (accessLevel, error) {
	s := a.s
	if l, ok := s.auth.level(accountID, repo.ID); ok {
		return l, nil
	}
	grant, err := s.Queries.GetRepositoryAccess(ctx, db.GetRepositoryAccessParams{RepositoryID: repo.ID, AccountID: accountID})
	if err == nil {
		l := parseLevel(grant.Level)
		s.auth.putLevel(accountID, repo.ID, l)
		return l, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return levelNone, err
	}
	token, err := a.repoToken(ctx, repo)
	if err != nil || token == "" {
		return levelNone, err // no way to ask GitHub: grants only
	}
	perm, err := s.GH.GetCollaboratorPermission(ctx, token, repo.GithubOwner, repo.GithubRepo, login)
	if err != nil {
		return levelNone, err // not cached: may be a passing GitHub failure
	}
	l := permissionLevel(perm)
	s.auth.putLevel(accountID, repo.ID, l)
	return l, nil
}

// repoToken is a token that can ask GitHub about any collaborator on repo.
func (a MCPAccess) repoToken(ctx context.Context, repo db.Repository) (string, error) {
	s := a.s
	if repo.InstallationID.Valid && s.AppID != "" && s.AppPrivateKey != nil {
		inst, err := s.Queries.GetGithubInstallation(ctx, repo.InstallationID)
		if err != nil {
			return "", err
		}
		tok, err := s.GH.InstallationToken(ctx, s.AppID, s.AppPrivateKey, inst.InstallationID)
		if err != nil {
			return "", err
		}
		return tok.Token, nil
	}
	return s.ServiceToken, nil
}

// permissionLevel maps GitHub's collaborator permission onto LaForge's levels
// the same way githubRoleLevel does for a repository's permissions object.
func permissionLevel(p string) accessLevel {
	switch p {
	case "admin":
		return levelAdmin
	case "maintain", "write":
		return levelBuild
	case "triage", "read":
		return levelRead
	}
	return levelNone
}
