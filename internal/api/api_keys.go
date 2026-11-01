// Personal API keys: what a person's AI agent uses to read their builds through
// LaForge's read-only MCP server (docs/plans/mcp-readonly.md). Each person
// manages their own keys on the API Keys page; instance admins can see and
// revoke everyone's.
//
// These endpoints take the browser session only (sessionFromRequest), never a
// bearer token: an API key can't mint, list or revoke keys, and neither can a
// GitHub token. API keys themselves are refused on every REST route (see
// bearerToken) -- they're for the MCP endpoint alone.
package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

const (
	// apiKeyPrefix marks a LaForge API key, so a leaked one is recognisable
	// (and a REST route can refuse it without asking GitHub about it).
	apiKeyPrefix = "lfk_"
	// apiKeyDisplayLen is how much of a key is kept for display: the prefix
	// and the first 8 characters of the secret.
	apiKeyDisplayLen = len(apiKeyPrefix) + 8
	// maxActiveAPIKeys caps one person's live keys.
	maxActiveAPIKeys = 20
	maxAPIKeyNameLen = 100
)

// apiKeyLifetimes are the expiries a key can be created with. There's no
// "never": every key expires.
var apiKeyLifetimes = map[int]bool{7: true, 30: true, 90: true}

var errAPIKeyNotForREST = errors.New("LaForge API keys only work with the MCP server, not the REST API; sign in with GitHub (laforge login) instead")

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newAPIKey returns a fresh key: the prefix and 32 random bytes in base62.
func newAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := new(big.Int).SetBytes(b)
	base := big.NewInt(62)
	mod := new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, mod)
		out = append(out, base62Alphabet[mod.Int64()])
	}
	// 32 bytes is at most 43 base62 digits; pad so every key is the same length.
	for len(out) < 43 {
		out = append(out, base62Alphabet[0])
	}
	return apiKeyPrefix + string(out), nil
}

// apiKeyView is a key as the API returns it. The secret and its hash are
// never part of it.
type apiKeyView struct {
	ID            pgtype.UUID        `json:"id"`
	Name          string             `json:"name"`
	Prefix        string             `json:"prefix"`
	Scope         string             `json:"scope"`
	RepositoryIDs []pgtype.UUID      `json:"repository_ids"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	ExpiresAt     pgtype.Timestamptz `json:"expires_at"`
	LastUsedAt    pgtype.Timestamptz `json:"last_used_at"`
	RevokedAt     pgtype.Timestamptz `json:"revoked_at"`
	// Status is "active", "expired" or "revoked".
	Status string `json:"status"`
	// Kind is "key" for a pasted key, or "app" for a connected app: a key an
	// app got by OAuth sign-in (oauth_server.go). ClientHost is where that
	// app lives (its redirect host).
	Kind       string `json:"kind"`
	ClientHost string `json:"client_host,omitempty"`
	// Owner fields are set only on the admin listing.
	OwnerLogin     string `json:"owner_login,omitempty"`
	OwnerAvatarURL string `json:"owner_avatar_url,omitempty"`
}

func newAPIKeyView(id pgtype.UUID, name, prefix, scope string, repos []pgtype.UUID, created, expires, lastUsed, revoked pgtype.Timestamptz) apiKeyView {
	if repos == nil {
		repos = []pgtype.UUID{}
	}
	status := "active"
	switch {
	case revoked.Valid:
		status = "revoked"
	case expires.Valid && !expires.Time.After(time.Now()):
		status = "expired"
	}
	return apiKeyView{
		ID: id, Name: name, Prefix: prefix, Scope: scope, RepositoryIDs: repos,
		CreatedAt: created, ExpiresAt: expires, LastUsedAt: lastUsed, RevokedAt: revoked, Status: status, Kind: "key",
	}
}

// withClient marks a key as a connected app when it came from OAuth sign-in.
func (v apiKeyView) withClient(clientID pgtype.UUID, redirectURIs []string) apiKeyView {
	if clientID.Valid {
		v.Kind, v.ClientHost = "app", redirectHost(redirectURIs)
	}
	return v
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListAPIKeysByAccount(r.Context(), sess.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]apiKeyView, 0, len(rows))
	for _, k := range rows {
		out = append(out, newAPIKeyView(k.ID, k.Name, k.Prefix, k.Scope, k.RepositoryIds, k.CreatedAt, k.ExpiresAt, k.LastUsedAt, k.RevokedAt).
			withClient(k.OauthClientID, k.OauthRedirectUris))
	}
	writeJSON(w, http.StatusOK, out)
}

type createAPIKeyRequest struct {
	Name          string   `json:"name"`
	ExpiresInDays int      `json:"expires_in_days"`
	RepositoryIDs []string `json:"repository_ids"`
}

// createdAPIKey is the one response that carries the key itself.
type createdAPIKey struct {
	Key   apiKeyView `json:"key"`
	Token string     `json:"token"`
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("a key needs a name"))
		return
	}
	if utf8.RuneCountInString(name) > maxAPIKeyNameLen {
		writeError(w, http.StatusBadRequest, fmt.Errorf("a key's name can be at most %d characters", maxAPIKeyNameLen))
		return
	}
	if !apiKeyLifetimes[req.ExpiresInDays] {
		writeError(w, http.StatusBadRequest, errors.New("expires_in_days must be 7, 30 or 90"))
		return
	}

	// A repository limit may only name repositories the person can read now.
	// (The key is still checked against their live access on every use.)
	var repoIDs []pgtype.UUID
	seen := map[pgtype.UUID]bool{}
	for _, raw := range req.RepositoryIDs {
		id, err := parseUUID(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid repository id %q", raw))
			return
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		repo, err := s.Queries.GetRepository(r.Context(), id)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, fmt.Errorf("no such repository %q", raw))
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		level, err := s.effectiveLevel(r.Context(), sess, repo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if level < levelRead {
			writeError(w, http.StatusBadRequest, fmt.Errorf("you don't have access to %s/%s", repo.GithubOwner, repo.GithubRepo))
			return
		}
		repoIDs = append(repoIDs, id)
	}

	active, err := s.Queries.CountActiveAPIKeysByAccount(r.Context(), sess.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if active >= maxActiveAPIKeys {
		writeError(w, http.StatusConflict, fmt.Errorf("you already have %d active keys; revoke one first", maxActiveAPIKeys))
		return
	}

	token, err := newAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	k, err := s.Queries.CreateAPIKey(r.Context(), db.CreateAPIKeyParams{
		AccountID:     sess.AccountID,
		Name:          name,
		Prefix:        token[:apiKeyDisplayLen],
		TokenHash:     hashToken(token),
		RepositoryIds: repoIDs,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, createdAPIKey{
		Key:   newAPIKeyView(k.ID, k.Name, k.Prefix, k.Scope, k.RepositoryIds, k.CreatedAt, k.ExpiresAt, k.LastUsedAt, k.RevokedAt),
		Token: token,
	})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid key id"))
		return
	}
	k, err := s.Queries.RevokeOwnAPIKey(r.Context(), db.RevokeOwnAPIKeyParams{ID: id, AccountID: sess.AccountID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("no such active key"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.MCPKeysChanged != nil {
		s.MCPKeysChanged()
	}
	writeJSON(w, http.StatusOK, newAPIKeyView(k.ID, k.Name, k.Prefix, k.Scope, k.RepositoryIds, k.CreatedAt, k.ExpiresAt, k.LastUsedAt, k.RevokedAt))
}

func (s *Server) handleAdminListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListAPIKeysWithOwner(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]apiKeyView, 0, len(rows))
	for _, k := range rows {
		v := newAPIKeyView(k.ID, k.Name, k.Prefix, k.Scope, k.RepositoryIds, k.CreatedAt, k.ExpiresAt, k.LastUsedAt, k.RevokedAt).
			withClient(k.OauthClientID, k.OauthRedirectUris)
		v.OwnerLogin, v.OwnerAvatarURL = k.OwnerLogin, db.StrOrEmpty(k.OwnerAvatarUrl)
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdminRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid key id"))
		return
	}
	k, err := s.Queries.RevokeAnyAPIKey(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("no such active key"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.MCPKeysChanged != nil {
		s.MCPKeysChanged()
	}
	writeJSON(w, http.StatusOK, newAPIKeyView(k.ID, k.Name, k.Prefix, k.Scope, k.RepositoryIds, k.CreatedAt, k.ExpiresAt, k.LastUsedAt, k.RevokedAt))
}

// activityLimit is how many recent MCP calls an activity listing returns:
// ?limit=N, default 50, at most 200.
func activityLimit(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return int32(n)
}

// activityPage is one page of MCP calls, newest first. Next is the cursor
// for the following (older) page, absent on the last page.
type activityPage[Row any] struct {
	Items []Row   `json:"items"`
	Total int64   `json:"total"`
	Next  *string `json:"next,omitempty"`
}

// activityCursor reads ?before=, the "<created_at>_<id>" of the last row of
// the page before. Empty means the first page.
func activityCursor(r *http.Request) (pgtype.Timestamptz, pgtype.UUID, error) {
	raw := r.URL.Query().Get("before")
	if raw == "" {
		return pgtype.Timestamptz{}, pgtype.UUID{}, nil
	}
	at, id, ok := strings.Cut(raw, "_")
	t, err := time.Parse(time.RFC3339Nano, at)
	if !ok || err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, errors.New("invalid before cursor")
	}
	uid, err := parseUUID(id)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, errors.New("invalid before cursor")
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, uid, nil
}

func nextCursor(n int, limit int32, at pgtype.Timestamptz, id pgtype.UUID) *string {
	if n < int(limit) {
		return nil
	}
	c := at.Time.UTC().Format(time.RFC3339Nano) + "_" + id.String()
	return &c
}

// handleListAPIKeyActivity is the API Keys page's activity log: the MCP calls
// made with the signed-in person's own keys (mcp_audit), newest first, a page
// at a time.
func (s *Server) handleListAPIKeyActivity(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	beforeAt, beforeID, err := activityCursor(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit := activityLimit(r)
	rows, err := s.Queries.ListMcpActivityByAccount(r.Context(), db.ListMcpActivityByAccountParams{AccountID: sess.AccountID, BeforeAt: beforeAt, BeforeID: beforeID, PageSize: limit})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	total, err := s.Queries.CountMcpActivityByAccount(r.Context(), sess.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	page := activityPage[db.ListMcpActivityByAccountRow]{Items: rows, Total: total}
	if page.Items == nil {
		page.Items = []db.ListMcpActivityByAccountRow{}
	} else {
		last := rows[len(rows)-1]
		page.Next = nextCursor(len(rows), limit, last.CreatedAt, last.ID)
	}
	writeJSON(w, http.StatusOK, page)
}

// handleAdminListAPIKeyActivity is everyone's MCP calls, for admins, a page
// at a time.
func (s *Server) handleAdminListAPIKeyActivity(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	beforeAt, beforeID, err := activityCursor(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit := activityLimit(r)
	rows, err := s.Queries.ListMcpActivity(r.Context(), db.ListMcpActivityParams{BeforeAt: beforeAt, BeforeID: beforeID, PageSize: limit})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	total, err := s.Queries.CountMcpActivity(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	page := activityPage[db.ListMcpActivityRow]{Items: rows, Total: total}
	if page.Items == nil {
		page.Items = []db.ListMcpActivityRow{}
	} else {
		last := rows[len(rows)-1]
		page.Next = nextCursor(len(rows), limit, last.CreatedAt, last.ID)
	}
	writeJSON(w, http.StatusOK, page)
}

// handleGetAPIKeyActivity is one of the signed-in person's own MCP calls with
// its full detail: what the activity log shows when a row is opened.
func (s *Server) handleGetAPIKeyActivity(w http.ResponseWriter, r *http.Request) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	s.writeActivity(w, r, &sess.AccountID)
}

// handleAdminGetAPIKeyActivity is any MCP call with its full detail, for admins.
func (s *Server) handleAdminGetAPIKeyActivity(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	s.writeActivity(w, r, nil)
}

// writeActivity writes one mcp_audit row; owner, when set, must own it.
func (s *Server) writeActivity(w http.ResponseWriter, r *http.Request, owner *pgtype.UUID) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid activity id"))
		return
	}
	row, err := s.Queries.GetMcpActivity(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != nil && row.AccountID != *owner) {
		writeError(w, http.StatusNotFound, errors.New("no such call"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}
