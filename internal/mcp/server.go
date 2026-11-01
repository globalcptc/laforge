// Package mcp is LaForge's MCP (Model Context Protocol) server: the endpoint
// (/mcp on laforge-api) a volunteer's AI agent connects to with a personal
// API key to inspect builds -- and, for someone with manage access, to power
// hosts on, off or through a reboot. See docs/plans/mcp-readonly.md.
//
// Everything but power is read-only, enforced structurally, not by convention:
//
//   - Every query runs through Config.Queries, which must sit on a pool
//     connected as the laforge_mcp Postgres role (migrations/00081). That role
//     can SELECT only listed, non-secret columns and write only its own audit
//     rows and a key's last-used time; Postgres refuses anything else.
//   - This package imports no builder, gateway, runner or API code, so it has
//     no way to reach a hypervisor or a mutating handler (dependency_test.go).
//   - It accepts only LaForge API keys (lfk_...), which the REST API in turn
//     refuses.
//   - Pages come through the API's read bridge, which holds only read routes
//     and caps the caller at read level. Power goes through a separate action
//     bridge holding that one route, at the owner's real level.
package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/globalcptc/laforge/internal/db"
)

// Access is what the API does for the MCP server on its own connection
// (api.Server.MCPAccess): which repositories an account can read right now,
// and its read pages, fetched as the key's owner and capped at read level.
type Access interface {
	ReadableRepos(ctx context.Context, accountID pgtype.UUID, login string) ([]pgtype.UUID, error)
	// Get fetches one of the API's read pages (GET path?query) and returns
	// its HTTP status and JSON body.
	Get(ctx context.Context, accountID pgtype.UUID, login, path string, query url.Values) (int, []byte, error)
	// Act runs the API's one MCP action, power (POST path with a JSON body),
	// as the key's owner at their real level -- manage, as in the UI.
	Act(ctx context.Context, accountID pgtype.UUID, login, path string, body []byte) (int, []byte, error)
}

// Config is everything the server needs. Zero limits take the defaults.
type Config struct {
	// Queries MUST be on a pool connected as laforge_mcp.
	Queries *db.Queries
	Access  Access
	// Version is reported to clients.
	Version string

	// PerAccountRate is tool calls per second per key owner, with
	// PerAccountBurst calls allowed at once.
	PerAccountRate  float64
	PerAccountBurst int
	// MaxConcurrent is how many tool calls run at once across everyone;
	// QueueWait is how long a call waits for a slot before failing.
	MaxConcurrent int
	QueueWait     time.Duration
	// CallTimeout bounds one tool call.
	CallTimeout time.Duration
}

const (
	defaultRate          = 2
	defaultBurst         = 10
	defaultMaxConcurrent = 3
	defaultQueueWait     = 2 * time.Second
	defaultCallTimeout   = 10 * time.Second

	keyPrefix   = "lfk_"
	keyCacheTTL = 15 * time.Second
	// accessCacheTTL matches the API's own access cache, so losing access
	// takes effect on the same schedule here as in the UI.
	accessCacheTTL = 15 * time.Second
	// maxResultBytes caps one tool result; tools stay under it with their row
	// limits and max_field_chars, and a result over it is refused, not cut.
	maxResultBytes = 256 << 10
)

// Server is the MCP server. Build it with New and mount Handler.
type Server struct {
	cfg Config
	sdk *sdk.Server
	sem chan struct{}
	lim *limiter
	// powerLim is power's own, tighter limit, on top of lim.
	powerLim *limiter
	keys     sync.Map // token hash -> cachedKey
	acc      sync.Map // account id -> cachedAccess
}

type cachedKey struct {
	key   apiKey
	until time.Time
}

type cachedAccess struct {
	repos []pgtype.UUID
	until time.Time
}

// apiKey is a verified key, carried to tool calls in the request's TokenInfo.
type apiKey struct {
	ID        pgtype.UUID
	AccountID pgtype.UUID
	Login     string
	Name      string
	Prefix    string
	Scope     string
	// Repos limits the key to these repositories; empty means every
	// repository its owner can read.
	Repos     []pgtype.UUID
	ExpiresAt time.Time
}

const tokenInfoKey = "laforge_api_key"

// New builds the server and registers every tool.
func New(cfg Config) *Server {
	if cfg.PerAccountRate <= 0 {
		cfg.PerAccountRate = defaultRate
	}
	if cfg.PerAccountBurst <= 0 {
		cfg.PerAccountBurst = defaultBurst
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent
	}
	if cfg.QueueWait <= 0 {
		cfg.QueueWait = defaultQueueWait
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = defaultCallTimeout
	}
	if cfg.Version == "" {
		cfg.Version = "dev"
	}
	s := &Server{
		cfg: cfg,
		sem: make(chan struct{}, cfg.MaxConcurrent),
		lim: newLimiter(cfg.PerAccountRate, cfg.PerAccountBurst),
		// Power: 3 calls at once, then one every 10 seconds per owner.
		powerLim: newLimiter(0.1, 3),
	}
	s.sdk = sdk.NewServer(&sdk.Implementation{Name: "laforge", Title: "LaForge", Version: cfg.Version}, &sdk.ServerOptions{
		Instructions: instructions,
	})
	s.registerTools()
	return s
}

const instructions = `LaForge deploys CPTC competition environments. This server shows everything a read-level user sees in LaForge -- builds, hosts, steps and their output, rendered scripts, logs, schedules, access, passwords, builders. The one thing it can change is power: the power tool starts, stops or reboots hosts and containers (force for a hard power operation), and only for someone with manage access to the build's repository. It cannot rebuild, deploy, tear down or run commands; those stay in the LaForge UI and CLI.

Start with home or list_repos, then list_builds, then get_build for a one-build summary (counts by status, agent health and cluster location). From there:
- hosts: list_objects (filter by status, agent status, location), then list_steps, list_object_events, get_object_infra, render_object for one host
- failures: search_agent_tasks (failed steps across the build), get_step_output for full output, list_runner_tasks for stuck deploys
- agents: heartbeat_gaps (silences, rolled up per cluster member), list_object_heartbeats
- logs: list_build_events; schedules: list_scheduled_tasks; access: list_external_access, export_external_access, list_credentials
Builds can be named by their full id or a unique prefix such as "9531b5b4"; hosts by id or name (with team).`

// Handler is the /mcp endpoint: bearer-key auth in front of a stateless
// streamable-HTTP MCP handler that answers in plain JSON.
func (s *Server) Handler() http.Handler {
	h := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s.sdk }, &sdk.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          64 << 10,
	})
	return auth.RequireBearerToken(s.verify, &auth.RequireBearerTokenOptions{Scopes: []string{"read"}})(h)
}

// verify turns a bearer key into the key it names, or refuses it.
func (s *Server) verify(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
	if !strings.HasPrefix(token, keyPrefix) {
		return nil, fmt.Errorf("%w: only LaForge API keys (lfk_...) work here; create one on the API Keys page", auth.ErrInvalidToken)
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	now := time.Now()
	var key apiKey
	if c, ok := s.keys.Load(hash); ok && now.Before(c.(cachedKey).until) {
		key = c.(cachedKey).key
	} else {
		row, err := s.cfg.Queries.McpGetAPIKeyByHash(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: unknown API key", auth.ErrInvalidToken)
		}
		if err != nil {
			return nil, err
		}
		if row.RevokedAt.Valid {
			return nil, fmt.Errorf("%w: this API key was revoked", auth.ErrInvalidToken)
		}
		key = apiKey{
			ID: row.ID, AccountID: row.AccountID, Login: row.GithubLogin, Name: row.Name, Prefix: row.Prefix,
			Scope: row.Scope, Repos: row.RepositoryIds, ExpiresAt: row.ExpiresAt.Time,
		}
		s.keys.Store(hash, cachedKey{key: key, until: now.Add(keyCacheTTL)})
		go s.touch(key.ID)
	}
	if !now.Before(key.ExpiresAt) {
		return nil, fmt.Errorf("%w: this API key has expired", auth.ErrInvalidToken)
	}
	return &auth.TokenInfo{
		Scopes:     []string{key.Scope},
		Expiration: key.ExpiresAt,
		UserID:     key.AccountID.String(),
		Extra:      map[string]any{tokenInfoKey: key},
	}, nil
}

// ForgetKeys drops cached keys, so a revoke takes effect on the next request
// instead of within keyCacheTTL. The API calls it after any key change.
func (s *Server) ForgetKeys() {
	s.keys.Clear()
}

func (s *Server) touch(id pgtype.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.cfg.Queries.McpTouchAPIKey(ctx, id); err != nil {
		log.Printf("mcp: recording key use: %v", err)
	}
}

// readableRepos is the repositories this key can read: its owner's access,
// narrowed to the key's own repository list when it has one.
func (s *Server) readableRepos(ctx context.Context, key apiKey) (map[pgtype.UUID]bool, error) {
	now := time.Now()
	var repos []pgtype.UUID
	if c, ok := s.acc.Load(key.AccountID); ok && now.Before(c.(cachedAccess).until) {
		repos = c.(cachedAccess).repos
	} else {
		var err error
		repos, err = s.cfg.Access.ReadableRepos(ctx, key.AccountID, key.Login)
		if err != nil {
			return nil, err
		}
		s.acc.Store(key.AccountID, cachedAccess{repos: repos, until: now.Add(accessCacheTTL)})
	}
	out := make(map[pgtype.UUID]bool, len(repos))
	for _, r := range repos {
		out[r] = true
	}
	if len(key.Repos) > 0 {
		limited := make(map[pgtype.UUID]bool, len(key.Repos))
		for _, r := range key.Repos {
			if out[r] {
				limited[r] = true
			}
		}
		out = limited
	}
	return out, nil
}
