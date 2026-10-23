package api

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// authCache keeps the answers every request's access check needs -- who the
// session is, what level they have on a repository, which repository a build
// belongs to -- in memory for a few seconds. Without it every request, every
// poll from every open tab, made several database lookups just to check
// access, so the number of viewers still set the database's load even once
// the data itself was shared.
//
// Anything that changes state (any request other than a read) clears the
// session and level answers at once, so a logout or an access change takes
// effect on the very next request; otherwise they're re-checked at least
// every authCacheTTL.
type authCache struct {
	mu       sync.Mutex
	sessions map[string]cachedSession // token hash ->
	levels   map[levelKey]cachedLevel
	repos    map[pgtype.UUID]cachedRepo // build id -> its repository
}

const (
	authCacheTTL  = 15 * time.Second
	buildRepoTTL  = time.Minute
	authCacheSize = 100_000 // entries per map before it's cleared outright
)

type cachedSession struct {
	row   db.GetSessionByTokenHashRow
	until time.Time
}

type levelKey struct{ account, repo pgtype.UUID }

type cachedLevel struct {
	level accessLevel
	until time.Time
}

type cachedRepo struct {
	repo  db.Repository
	until time.Time
}

func (c *authCache) session(hash string) (db.GetSessionByTokenHashRow, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.sessions[hash]
	if !ok || time.Now().After(e.until) {
		return db.GetSessionByTokenHashRow{}, false
	}
	return e.row, true
}

func (c *authCache) putSession(hash string, row db.GetSessionByTokenHashRow) {
	until := time.Now().Add(authCacheTTL)
	// Never past the session's own expiry, nor past when its GitHub token is
	// due a refresh (sessionFromRequest refreshes it on the next lookup).
	if row.ExpiresAt.Valid && row.ExpiresAt.Time.Before(until) {
		until = row.ExpiresAt.Time
	}
	if row.GithubTokenExpiresAt.Valid {
		if t := row.GithubTokenExpiresAt.Time.Add(-githubTokenSkew); t.Before(until) {
			until = t
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessions == nil || len(c.sessions) > authCacheSize {
		c.sessions = map[string]cachedSession{}
	}
	c.sessions[hash] = cachedSession{row: row, until: until}
}

func (c *authCache) level(account, repo pgtype.UUID) (accessLevel, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.levels[levelKey{account, repo}]
	if !ok || time.Now().After(e.until) {
		return levelNone, false
	}
	return e.level, true
}

func (c *authCache) putLevel(account, repo pgtype.UUID, l accessLevel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.levels == nil || len(c.levels) > authCacheSize {
		c.levels = map[levelKey]cachedLevel{}
	}
	c.levels[levelKey{account, repo}] = cachedLevel{level: l, until: time.Now().Add(authCacheTTL)}
}

// forgetAccess drops every cached session and level answer -- called after
// any request that changes state.
func (c *authCache) forgetAccess() {
	c.mu.Lock()
	c.sessions = nil
	c.levels = nil
	c.mu.Unlock()
}

// buildRepository is which repository a build belongs to, for the access
// check on a read -- without reading the build row itself, which a shared
// read (sharedread.go) reads inside its one computation instead.
func (s *Server) buildRepository(r *http.Request) (pgtype.UUID, db.Repository, error) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		return pgtype.UUID{}, db.Repository{}, err
	}
	c := &s.auth
	c.mu.Lock()
	e, ok := c.repos[id]
	c.mu.Unlock()
	if ok && time.Now().Before(e.until) {
		return id, e.repo, nil
	}
	repo, err := s.Queries.GetBuildRepository(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, db.Repository{}, errors.New("no such build")
		}
		return pgtype.UUID{}, db.Repository{}, err
	}
	c.mu.Lock()
	if c.repos == nil || len(c.repos) > authCacheSize {
		c.repos = map[pgtype.UUID]cachedRepo{}
	}
	c.repos[id] = cachedRepo{repo: repo, until: time.Now().Add(buildRepoTTL)}
	c.mu.Unlock()
	return id, repo, nil
}
