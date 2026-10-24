package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/db"
)

var (
	errUnauthenticated = errors.New("missing bearer token: send \"Authorization: Bearer <github-token>\"")
	errForbidden       = errors.New("your github token does not have push access to this repository")
)

func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(h, "Bearer ")
	if !ok || tok == "" {
		return "", errUnauthenticated
	}
	return tok, nil
}

// requirePush is "GitHub-based authorization" end to end: GitHub itself
// decides whether the caller's own token can push to owner/repo, and
// hands that answer back as a `permissions` object on the repository it
// returns. There's deliberately no separate permission table on our side
// to keep in sync -- "if you can push to the repo, you can build its
// environments," checked fresh, every call, against GitHub itself.
//
// This collapses the plan's four access levels (read/build/manage/admin)
// down to what GitHub's own token permissions actually distinguish
// (pull/push/admin): push covers build+manage here, since nothing yet
// needs the finer distinction the real four-level model
// wants eventually.
//
// The four endpoints still gated by this (create repository, create
// configured build, set follow, set lock) can't use requireLevel instead
// -- requireLevel takes an existing db.Repository, and the first of those
// four is exactly the call that creates one; there's nothing to look up
// repository_access against yet. So this stays GitHub-permission-only,
// but resolves its token from EITHER the real browser session cookie or
// an explicit Bearer header, not Bearer alone. Found live, this session:
// the actual UI (ui/src/api/client.ts) sends only `credentials: 'include'`,
// no Authorization header at all, so every one of these four endpoints
// was unreachable from a real signed-in browser -- a genuine 401 on
// "Create configured build," confirmed with a real OAuth session cookie
// before this fix, not a hypothetical. Bearer-only still works unchanged
// (the CLI's own device-flow token, and every existing test using
// "Authorization: Bearer <token>" directly) -- this only adds the second
// path, tried first since a browser sending both would mean the session
// is the one actually representing "this browser tab, signed in."
func (s *Server) requirePush(ctx context.Context, r *http.Request, owner, repo string) error {
	// An instance admin manages everything, regardless of their own GitHub
	// permissions -- the same rule requireLevel/effectiveLevel and home.go
	// already apply. Without it an admin whose token can't see the repo is
	// wrongly refused (the "sees configured builds but not builds" class of
	// bug, here for the configured-build management endpoints).
	if sess, err := s.sessionFromRequest(r); err == nil && s.isInstanceAdmin(sess) {
		return nil
	}
	tok, err := s.pushToken(r)
	if err != nil {
		return err
	}
	ghRepo, err := s.GH.GetRepo(ctx, tok, owner, repo)
	if err != nil {
		return err
	}
	if !ghRepo.Permissions.Push {
		return errForbidden
	}
	return nil
}

// pushToken resolves the real GitHub token to check push permission
// against -- see requirePush's own doc comment for why both a session
// cookie and a bearer header have to work.
func (s *Server) pushToken(r *http.Request) (string, error) {
	if sess, err := s.sessionFromRequest(r); err == nil {
		return sess.GithubToken, nil
	}
	return bearerToken(r)
}

// authSessionForRequest resolves the caller to an authSession from EITHER the
// browser session cookie OR a Bearer GitHub token (the CLI's device-flow
// token) -- so the four-level, repository-scoped endpoints (requireLevel) work
// for a CLI client the same way requirePush/pushToken already let the
// GitHub-permission endpoints take a bearer token. A bearer identity is
// resolved live against GitHub and reconciled to its account row (looked up by
// login, upserted the first time it's seen), so the admin-login list and any
// repository_access grants apply to it identically to a browser session. The
// cookie path is tried first and is unchanged; this only adds the bearer path
// when there is no session.
func (s *Server) authSessionForRequest(ctx context.Context, r *http.Request) (authSession, error) {
	if sess, err := s.sessionFromRequest(r); err == nil {
		return sess, nil
	}
	tok, err := bearerToken(r)
	if err != nil {
		return authSession{}, err
	}
	ghUser, err := s.GH.GetAuthenticatedUser(ctx, tok)
	if err != nil {
		return authSession{}, errUnauthenticated
	}
	account, err := s.Queries.GetAccountByLogin(ctx, ghUser.Login)
	if errors.Is(err, pgx.ErrNoRows) {
		account, err = s.Queries.UpsertAccount(ctx, db.UpsertAccountParams{
			GithubID: ghUser.ID, GithubLogin: ghUser.Login, AvatarUrl: db.StrPtr(ghUser.AvatarURL),
		})
	}
	if err != nil {
		return authSession{}, err
	}
	return authSession{db.GetSessionByTokenHashRow{
		AccountID:   account.ID,
		GithubToken: tok,
		GithubID:    ghUser.ID,
		GithubLogin: ghUser.Login,
		AvatarUrl:   db.StrPtr(ghUser.AvatarURL),
	}}, nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errUnauthenticated):
		writeError(w, http.StatusUnauthorized, err)
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err)
	default:
		// A GetRepo failure (network error, or GitHub 404ing a repo the
		// token can't even see) -- surfaced as 403 rather than 500, since
		// from the caller's point of view "I can't see whether I have
		// access" and "I don't have access" are the same non-answer, and
		// a 500 would wrongly suggest it's our bug rather than theirs.
		writeError(w, http.StatusForbidden, err)
	}
}
