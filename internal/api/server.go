// Package api is the one service humans, the CLI, and GitHub talk to:
// register a repository, configure a build (branch, environment file,
// builder config, follow mode, the competition-started lock), receive
// GitHub's push webhook (per-commit validation and follow-mode
// reconciliation, webhook.go), and the two
// operator write actions that turn a tracked commit into a running
// build: handleTriggerBuild (build_triggers.go) resolves a configured
// build's tracked commit into a real `planned` build, and
// handleDeployBuild flips it to `deploying`.
//
// Nothing here talks to a hypervisor -- that's internal/orchestrator and
// internal/runner. "Auto-build" and "build" remain distinct concepts (a push
// advancing configured_build.current_content_revision_id vs. turning that into
// a real build row), but both paths exist now: handleTriggerBuild is the manual
// one, and webhook.go's reconcile auto-builds on a passing push (triggerBuild)
// and, when AutoDeployEnabled, auto-deploys it (applyUpcoming) -- see
// webhook.go's reconcileConfiguredBuilds.
package api

import (
	"crypto/rsa"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

type Server struct {
	Queries *db.Queries
	// Pool is used only where a transaction spans more than one query --
	// today, that's internal/ingest.ValidateAndStore, called from
	// webhook.go's reconcile. Every other handler goes through Queries.
	Pool          *pgxpool.Pool
	GH            *ghclient.Client
	WebhookSecret string
	// ServiceToken is used for the server's OWN calls to GitHub (reading
	// check-runs/status on a commit, posting LaForge's own validation
	// result back) -- separate from a signed-in person's own bearer token,
	// which is only ever used to check THEIR permission on a repository,
	// never stored.
	ServiceToken string

	// GitHubClientID/Secret are the registered OAuth App's credentials for
	// the browser sign-in flow (see oauth.go) -- separate from
	// ServiceToken, which is the server's OWN token, not an OAuth App.
	GitHubClientID     string
	GitHubClientSecret string
	// PublicBaseURL is this API's own externally-reachable origin, used
	// to build the OAuth callback's redirect_uri (must exactly match what
	// the GitHub OAuth App has registered). UIBaseURL is where the
	// browser is sent after a successful login -- the React app's own
	// origin, which is not necessarily the same origin as the API.
	PublicBaseURL string
	UIBaseURL     string
	// CrossOriginCookies is the real fix for a cross-domain deployment:
	// the session cookie defaults to SameSite=Lax, correct for local dev
	// and for a same-registrable-domain production split (api.laforge.example
	// + laforge.example), but a genuinely cross-domain UI/API deployment
	// (two unrelated origins) needs SameSite=None; Secure -- Lax cookies
	// are simply never attached to a cross-site fetch()/XHR at all (only
	// a top-level GET navigation), so every authenticated API call from
	// such a UI would otherwise silently arrive with no session cookie.
	// Off by default; set true only when PublicBaseURL and UIBaseURL are
	// genuinely different origins and both serve HTTPS (SameSite=None
	// without Secure is rejected by every modern browser outright).
	CrossOriginCookies bool

	// RepoRoot is a real git checkout path content can be loaded from --
	// the same pattern cmd/laforge-orchestrator and cmd/laforge-runner
	// already use (their own `-repo` flag). Used directly when Checkouts
	// is nil (the original, still-supported single-repository mode); once
	// Checkouts is configured, handleRenderObject resolves each build's
	// own repository/commit through it instead, and RepoRoot stops
	// mattering. Only handleRenderObject
	// (render.go) reads either field; every other handler reads
	// persisted content from Postgres instead.
	RepoRoot  string
	Checkouts *checkout.Cache

	// GitHub App identity, for the installation-token flow (see
	// internal/ghclient/app.go and webhook.go's resolveCloneURL) --
	// separate from GitHubClientID/Secret above, which is the SAME
	// App's user-to-server OAuth credentials (a GitHub App exposes both
	// under one App registration; GitHubClientID/Secret didn't need to
	// change when this was added). AppPrivateKey being nil is a
	// supported, real configuration: every content-fetch path falls
	// back to ServiceToken, so a deployment can run with no App
	// configured at all.
	AppID         string
	AppPrivateKey *rsa.PrivateKey
	// AppSlug builds "Install on GitHub" links in the admin UI
	// (github.com/apps/<slug>/installations/new) -- cosmetic, never
	// used for an auth decision.
	AppSlug string

	// AdminLogins are GitHub logins with instance-wide admin. Approving
	// an installed repository into LaForge's own tracking
	// (installations.go) is the one action with no repository yet to
	// check repository_access against, so it needs a floor that isn't
	// scoped to any one repo. Deliberately narrow: this grants nothing
	// else. Every other admin action stays the existing per-repository
	// levelAdmin.
	AdminLogins []string

	mux *http.ServeMux
}

func New(q *db.Queries, pool *pgxpool.Pool, gh *ghclient.Client, webhookSecret, serviceToken string) *Server {
	s := &Server{Queries: q, Pool: pool, GH: gh, WebhookSecret: webhookSecret, ServiceToken: serviceToken}
	s.routes()
	return s
}

// ServeHTTP wraps the route table with CORS. The browser UI and this API
// are two different origins in every real deployment (a Vite dev server
// on one port, the built app served from wherever it's hosted, against
// this API on its own) -- and it has to be credentialed CORS
// specifically, not "allow *", because session auth is a cookie
// (oauth.go): a wildcard origin is incompatible with
// Access-Control-Allow-Credentials by the CORS spec itself, so the
// allowed origin has to be echoed back exactly, checked against
// s.UIBaseURL.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && s.UIBaseURL != "" && origin == s.UIBaseURL {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Vary", "Origin")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /config", s.handleGetConfig)

	mux.HandleFunc("POST /webhook/github", s.handleWebhook)

	mux.HandleFunc("POST /repos", s.handleCreateRepo)
	mux.HandleFunc("GET /repos", s.handleListRepos)
	mux.HandleFunc("GET /home", s.handleGetHome)
	mux.HandleFunc("GET /agent-binary/{id}", s.handleGetAgentBinary)
	mux.HandleFunc("GET /builders", s.handleListBuilders)
	mux.HandleFunc("GET /repos/{id}", s.handleGetRepo)

	mux.HandleFunc("POST /repos/{id}/configured-builds", s.handleCreateConfiguredBuild)
	mux.HandleFunc("GET /repos/{id}/configured-builds", s.handleListConfiguredBuilds)

	mux.HandleFunc("GET /configured-builds/{id}", s.handleGetConfiguredBuild)
	mux.HandleFunc("DELETE /configured-builds/{id}", s.handleDeleteConfiguredBuild)
	mux.HandleFunc("POST /configured-builds/{id}/auto-deploy", s.handleSetConfiguredBuildAutoDeploy)
	mux.HandleFunc("POST /configured-builds/{id}/lock", s.handleSetLock)
	mux.HandleFunc("POST /configured-builds/{id}/sync", s.handleSyncConfiguredBuild)
	mux.HandleFunc("POST /configured-builds/{id}/builds", s.handleTriggerBuild)

	mux.HandleFunc("GET /content-revisions/{id}/environments", s.handleListRevisionEnvironments)

	mux.HandleFunc("GET /installations", s.handleListInstallations)
	mux.HandleFunc("GET /installations/repositories", s.handleListUnapprovedInstalledRepositories)
	mux.HandleFunc("POST /installations/repositories/approve", s.handleApproveInstalledRepository)

	mux.HandleFunc("GET /builder-configs", s.handleListBuilderConfigs)
	mux.HandleFunc("GET /builder-configs/{name}", s.handleGetBuilderConfig)
	mux.HandleFunc("POST /builder-configs/{name}", s.handleCreateBuilderConfig)
	mux.HandleFunc("PUT /builder-configs/{name}", s.handleUpdateBuilderConfig)
	mux.HandleFunc("DELETE /builder-configs/{name}", s.handleDeleteBuilderConfig)
	mux.HandleFunc("POST /builder-connections", s.handleConnectBuilder)
	mux.HandleFunc("GET /builder-connections/{id}", s.handleGetBuilderConnection)

	mux.HandleFunc("POST /builder-configs/{name}/image-builds", s.handleRebuildBuilderImage)
	mux.HandleFunc("GET /builder-configs/{name}/image-builds", s.handleListBuilderImageBuilds)
	mux.HandleFunc("GET /image-builds/{id}/log", s.handleImageBuildLog)

	mux.HandleFunc("GET /registry-credentials", s.handleListRegistryCredentials)
	mux.HandleFunc("PUT /registry-credentials", s.handleUpsertRegistryCredential)
	mux.HandleFunc("DELETE /registry-credentials/{host}", s.handleDeleteRegistryCredential)

	mux.HandleFunc("GET /auth/client-id", s.handleClientID)
	mux.HandleFunc("GET /auth/github/login", s.handleLoginStart)
	mux.HandleFunc("GET /auth/github/callback", s.handleCallback)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
	mux.HandleFunc("GET /auth/me", s.handleMe)
	mux.HandleFunc("POST /auth/me/timezone", s.handleSetTimezone)

	mux.HandleFunc("GET /repos/{id}/builds", s.handleListBuilds)
	mux.HandleFunc("GET /repos/{id}/access", s.handleListRepositoryAccess)
	mux.HandleFunc("GET /repos/{id}/my-access", s.handleGetMyRepositoryAccess)
	mux.HandleFunc("PUT /repos/{id}/access/{login}", s.handleSetRepositoryAccess)
	mux.HandleFunc("DELETE /repos/{id}/access/{login}", s.handleDeleteRepositoryAccess)
	mux.HandleFunc("GET /repos/{id}/people", s.handleListPeople)
	mux.HandleFunc("GET /repos/{id}/branches", s.handleListBranches)
	mux.HandleFunc("GET /repos/{id}/environment-files", s.handleListEnvironmentFiles)

	mux.HandleFunc("GET /builds/{id}", s.handleGetBuild)
	mux.HandleFunc("POST /builds/{id}/deploy", s.handleDeployBuild)
	mux.HandleFunc("POST /builds/{id}/teardown", s.handleTeardownBuild)
	mux.HandleFunc("GET /builds/{id}/drift", s.handleDetectDrift)
	mux.HandleFunc("GET /builds/{id}/objects", s.handleListDeployedObjects)
	mux.HandleFunc("GET /builds/{id}/objects/{objectId}/events", s.handleListObjectEvents)
	mux.HandleFunc("GET /builds/{id}/objects/{objectId}/heartbeats", s.handleListObjectHeartbeats)
	mux.HandleFunc("GET /builds/{id}/objects/{objectId}/render", s.handleRenderObject)
	mux.HandleFunc("GET /builds/{id}/objects/{objectId}/steps", s.handleListObjectSteps)
	mux.HandleFunc("GET /builds/{id}/objects/{objectId}/infra", s.handleObjectInfra)
	mux.HandleFunc("GET /builds/{id}/events", s.handleListEvents)
	mux.HandleFunc("GET /builds/{id}/live", s.handleLiveStatus)
	mux.HandleFunc("POST /builds/{id}/tasks", s.handleCreateAdHocTask)
	mux.HandleFunc("POST /builds/{id}/power", s.handlePowerAction)
	mux.HandleFunc("POST /builds/{id}/scheduled-tasks", s.handleCreateScheduledTask)
	mux.HandleFunc("POST /builds/{id}/scheduled-tasks/preview", s.handlePreviewScheduledTask)
	mux.HandleFunc("GET /builds/{id}/scheduled-tasks", s.handleListScheduledTasks)
	mux.HandleFunc("POST /builds/{id}/scheduled-tasks/{taskId}/cancel", s.handleCancelScheduledTask)
	mux.HandleFunc("POST /builds/{id}/teams/{team}/access", s.handleSetTeamAccess)
	mux.HandleFunc("GET /builds/{id}/findings", s.handleListFindings)
	mux.HandleFunc("GET /builds/{id}/dashboard", s.handleGetDashboard)
	mux.HandleFunc("GET /builds/{id}/topology", s.handleGetTopology)
	mux.HandleFunc("GET /builds/{id}/artifacts", s.handleGetArtifacts)
	mux.HandleFunc("POST /builds/{id}/artifacts/purge", s.handlePurgeArtifacts)
	mux.HandleFunc("GET /builds/{id}/upcoming", s.handleGetUpcoming)
	mux.HandleFunc("POST /builds/{id}/apply-upcoming", s.handleApplyUpcoming)

	s.mux = mux
}
