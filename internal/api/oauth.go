// GitHub OAuth web-application login for the browser UI -- "Sign in
// (GitHub). Single GitHub button. No local accounts, no password field."
// Distinct from auth.go's requirePush, which is the
// CLI/webhook's bearer-token path and stays untouched: this is what a
// browser session actually is, start to finish.
//
// Needs a registered GitHub OAuth App (client id + secret) to run for
// real -- not available in this environment (same gap noted in
// ghclient.go's web-flow doc comment for
// the service token). Every piece up to and including the callback
// handler's own logic is real and unit-tested against a fake GitHub
// server; only an actual browser round trip through github.com itself
// couldn't be exercised here.
package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

const (
	sessionCookieName = "laforge_session"
	stateCookieName   = "laforge_oauth_state"
	sessionTTL        = 7 * 24 * time.Hour
	oauthScopes       = "repo read:user"
)

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// cookieSameSite/cookieSecure centralize the CrossOriginCookies decision
// (see Server's own doc comment on that field) so every cookie this
// package sets -- state and session, set and cleared -- agrees, rather
// than each call site making its own judgment call that could drift.
func (s *Server) cookieSameSite() http.SameSite {
	if s.CrossOriginCookies {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func (s *Server) cookieSecure(r *http.Request) bool {
	if s.CrossOriginCookies {
		// SameSite=None is rejected outright by every modern browser
		// unless Secure is also set -- not conditional on r.TLS, which is
		// nil behind any TLS-terminating reverse proxy (Railway, nginx, or
		// another TLS proxy) even though the browser's own connection was HTTPS.
		return true
	}
	return r.TLS != nil
}

// handleClientID hands back the App's client id -- not a secret (GitHub
// itself treats it as public; it's visible in every OAuth redirect URL
// anyway), so there's no reason to make every `laforge login` user
// configure it locally when the server they're already talking to for
// every other CLI command already knows it. Fixes a real UX gap: device-
// flow login used to require GITHUB_OAUTH_CLIENT_ID set in the CLI's own
// environment, which meant knowing an internal App identifier out of
// band instead of just knowing the API's URL, same as every other CLI
// command already only needs.
func (s *Server) handleClientID(w http.ResponseWriter, r *http.Request) {
	if s.GitHubClientID == "" {
		writeError(w, http.StatusNotFound, errors.New("no GitHub App is configured on this server yet"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"client_id": s.GitHubClientID})
}

// handleLoginStart begins the web flow: mint a CSRF state value, stash it
// in a short-lived cookie (avoids needing server-side storage just for
// the state round trip), and send the browser to GitHub.
func (s *Server) handleLoginStart(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookieName, Value: state, Path: "/", HttpOnly: true,
		SameSite: s.cookieSameSite(), Secure: s.cookieSecure(r), MaxAge: 600,
	})
	redirectURI := s.PublicBaseURL + "/auth/github/callback"
	http.Redirect(w, r, s.GH.AuthorizeURL(s.GitHubClientID, redirectURI, state, []string{"repo", "read:user"}), http.StatusFound)
}

// handleCallback completes the flow: verify the state cookie matches
// what GitHub echoed back (the actual CSRF check -- a callback whose
// state doesn't match this browser's own cookie is rejected before any
// token exchange happens), exchange the code for a real access token,
// fetch who that token belongs to, and open a session.
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	code, state := q.Get("code"), q.Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing code or state"))
		return
	}
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != state {
		writeError(w, http.StatusBadRequest, errors.New("state mismatch -- start sign-in again"))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookieName, Value: "", Path: "/", MaxAge: -1,
		SameSite: s.cookieSameSite(), Secure: s.cookieSecure(r),
	})

	redirectURI := s.PublicBaseURL + "/auth/github/callback"
	ghToken, err := s.GH.ExchangeCode(ctx, s.GitHubClientID, s.GitHubClientSecret, code, redirectURI)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	ghUser, err := s.GH.GetAuthenticatedUser(ctx, ghToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	// Found missing entirely by direct product feedback: this handler
	// minted a real session for ANY GitHub account, full stop -- every
	// actual authorization check in this system happens only after a
	// session already exists (requireLevel/requireInstanceAdmin/
	// requirePush, all per-endpoint). In a CPTC deployment, the sign-in
	// button is reachable by every competitor; that's a real attack
	// surface (session creation, probing every authenticated endpoint),
	// not just an inconvenience.
	authorized, err := s.signInAuthorized(ctx, ghToken, ghUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !authorized {
		writeError(w, http.StatusForbidden, errors.New("your GitHub account has no access to anything tracked here -- ask an admin to grant you access to a repository first"))
		return
	}
	account, err := s.Queries.UpsertAccount(ctx, db.UpsertAccountParams{
		GithubID: ghUser.ID, GithubLogin: ghUser.Login, AvatarUrl: db.StrPtr(ghUser.AvatarURL),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	token, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	expiresAt := time.Now().Add(sessionTTL)
	if _, err := s.Queries.CreateSession(ctx, db.CreateSessionParams{
		AccountID: account.ID, TokenHash: hashToken(token), GithubToken: ghToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		SameSite: s.cookieSameSite(), Secure: s.cookieSecure(r), Expires: expiresAt,
	})
	http.Redirect(w, r, s.UIBaseURL+"/", http.StatusFound)
}

// signInAuthorized is the real sign-in gate handleCallback was missing
// entirely: "limit this either to users on the repos or a roster that we
// manage." Three independent ways in, any one is enough --
//   - an instance admin (AdminLogins), the same floor requireInstanceAdmin
//     already uses;
//   - "a roster that we manage": a repository_access grant above 'none'
//     already recorded for this identity on some repository (the repo
//     access page, RepoAccess.tsx, creates these);
//   - "users on the repos": live GitHub read/push/admin permission
//     (Permissions.Pull/Push/Admin -- any real access at all, matching
//     levelRead) on at least one repository this instance tracks --
//     skipping any repository an admin set them to 'none' on.
//
// Deliberately checked BEFORE UpsertAccount/CreateSession in
// handleCallback, not after: an unauthorized identity gets no account
// row and no session, not a session immediately torn back down.
func (s *Server) signInAuthorized(ctx context.Context, ghToken string, ghUser *ghclient.User) (bool, error) {
	for _, login := range s.AdminLogins {
		if strings.EqualFold(login, ghUser.Login) {
			return true, nil
		}
	}

	// Repositories an admin has explicitly shut this person out of don't
	// count, even if GitHub would let them in.
	blocked := map[pgtype.UUID]bool{}
	if acct, err := s.Queries.GetAccountByLogin(ctx, ghUser.Login); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		// No account row yet -- fine, just means no explicit grant could
		// exist for them either. Fall through to the live GitHub check.
	} else {
		grants, err := s.Queries.ListRepositoryAccessByAccount(ctx, acct.ID)
		if err != nil {
			return false, err
		}
		for _, g := range grants {
			if parseLevel(g.Level) > levelNone {
				return true, nil
			}
			blocked[g.RepositoryID] = true
		}
	}

	repos, err := s.Queries.ListRepositories(ctx)
	if err != nil {
		return false, err
	}
	for _, repo := range repos {
		if blocked[repo.ID] {
			continue
		}
		ghRepo, err := s.GH.GetRepo(ctx, ghToken, repo.GithubOwner, repo.GithubRepo)
		if err != nil {
			continue // can't see it on GitHub (404/403 for this token) -- not a match, try the next one
		}
		if ghRepo.Permissions.Pull || ghRepo.Permissions.Push || ghRepo.Permissions.Admin {
			return true, nil
		}
	}
	return false, nil
}

// handleLogout ends the session both places it lives: the row (so the
// token can never be replayed again even if the cookie leaks after this
// point) and the cookie itself.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.Queries.DeleteSession(r.Context(), hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1,
		SameSite: s.cookieSameSite(), Secure: s.cookieSecure(r),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type meResponse struct {
	GithubLogin string `json:"github_login"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	// IsInstanceAdmin lets the UI show instance-admin-only actions (approving
	// installed repositories, builders) only to people who can use them.
	IsInstanceAdmin bool `json:"is_instance_admin"`
	// Timezone is the account's own IANA timezone preference (empty means
	// "follow the viewer's browser"); the UI renders every timestamp in it.
	Timezone string `json:"timezone"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	_, adminErr := s.requireInstanceAdmin(r.Context(), r)
	writeJSON(w, http.StatusOK, meResponse{GithubLogin: sess.GithubLogin, AvatarURL: db.StrOrEmpty(sess.AvatarUrl), IsInstanceAdmin: adminErr == nil, Timezone: sess.Timezone})
}

// handleSetTimezone stores the signed-in account's IANA timezone
// preference, which the UI then renders every timestamp in. A real
// time.LoadLocation check rejects a bogus name (so the UI can't persist
// something that would silently break formatting), and "" is accepted as
// "clear it -- follow the browser."
func (s *Server) handleSetTimezone(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var body struct {
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	if body.Timezone != "" {
		if _, err := time.LoadLocation(body.Timezone); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("unknown timezone %q", body.Timezone))
			return
		}
	}
	if err := s.Queries.SetAccountTimezone(r.Context(), db.SetAccountTimezoneParams{ID: sess.AccountID, Timezone: body.Timezone}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"timezone": body.Timezone})
}

// authSession is what every session-authenticated handler gets after
// sessionFromRequest: the account's identity plus the live GitHub token
// needed to check that account's real permissions on a specific repo,
// exactly the same live-check discipline requirePush already uses.
type authSession struct {
	db.GetSessionByTokenHashRow
}

// sessionFromRequest resolves the session cookie to a live row, erroring
// the same way bearerToken does (errUnauthenticated) if there's no cookie
// or it's expired/unknown -- callers use writeAuthError exactly like the
// bearer-token path already does, one error vocabulary either way.
func (s *Server) sessionFromRequest(r *http.Request) (authSession, error) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return authSession{}, errUnauthenticated
	}
	row, err := s.Queries.GetSessionByTokenHash(r.Context(), hashToken(c.Value))
	if err != nil {
		return authSession{}, errUnauthenticated
	}
	return authSession{row}, nil
}

// accessLevel ranks the four levels so "each including the ones before
// it" is an ordinary integer comparison.
type accessLevel int

const (
	levelNone accessLevel = iota
	levelRead
	levelBuild
	levelManage
	levelAdmin
)

func (l accessLevel) String() string {
	return [...]string{"none", "read", "build", "manage", "admin"}[l]
}

// githubRoleLevel is the level a person's GitHub permissions on a
// repository give them there before any admin decision: admin -> admin,
// push (write/maintain) -> build, anything else they can see -> read.
func githubRoleLevel(p ghclient.RepoPermissions) accessLevel {
	switch {
	case p.Admin:
		return levelAdmin
	case p.Push:
		return levelBuild
	default:
		return levelRead
	}
}

func parseLevel(s string) accessLevel {
	switch s {
	case "read":
		return levelRead
	case "build":
		return levelBuild
	case "manage":
		return levelManage
	case "admin":
		return levelAdmin
	default:
		return levelNone
	}
}

// requireLevel is the session-based counterpart to requirePush, and the
// real implementation of the plan's four-level model. Every repository is
// independent. A person's level on it starts from their live GitHub
// permissions there (admin -> admin, push -> build, can see it -> read);
// an explicit repository_access row is an admin's decision for that
// person on that repository and REPLACES the GitHub-derived level --
// raising it, lowering it, or ('none') removing access (migration 00018).
func (s *Server) requireLevel(ctx context.Context, r *http.Request, repo db.Repository, min accessLevel) (authSession, error) {
	sess, err := s.authSessionForRequest(ctx, r)
	if err != nil {
		return authSession{}, err
	}
	effective, err := s.effectiveLevel(ctx, sess, repo)
	if err != nil {
		return authSession{}, err
	}
	if effective < min {
		return authSession{}, errForbidden
	}
	return sess, nil
}

// effectiveLevel is requireLevel's rule on its own: an admin's setting for
// this person on this repository if there is one, else what their GitHub
// permissions there give them.
func (s *Server) effectiveLevel(ctx context.Context, sess authSession, repo db.Repository) (accessLevel, error) {
	// An instance admin (LAFORGE_ADMIN_LOGINS) has full access to every
	// repository, independent of their own GitHub permissions or any
	// per-repo grant -- the same "admins see everything" rule handleList
	// Installations and home.go's visibleRepositories already apply. Kept
	// here so every requireLevel-gated endpoint agrees, rather than each
	// one re-checking instance-admin on its own (the inconsistency that
	// let an admin see configured builds but not the builds list).
	if s.isInstanceAdmin(sess) {
		return levelAdmin, nil
	}
	grant, err := s.Queries.GetRepositoryAccess(ctx, db.GetRepositoryAccessParams{RepositoryID: repo.ID, AccountID: sess.AccountID})
	if err == nil {
		return parseLevel(grant.Level), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return levelNone, err
	}
	ghRepo, err := s.GH.GetRepo(ctx, sess.GithubToken, repo.GithubOwner, repo.GithubRepo)
	if err != nil {
		return levelNone, nil // can't see the repo on GitHub -- no access unless an admin grants it
	}
	return githubRoleLevel(ghRepo.Permissions), nil
}
