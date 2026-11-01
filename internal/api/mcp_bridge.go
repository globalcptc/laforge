package api

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// The MCP read bridge: how the read-only MCP server (internal/mcp) shows
// exactly what the API shows -- rendered scripts, host passwords, step output,
// findings, topology -- without a second copy of every page's logic.
//
// internal/mcp calls MCPAccess.Get with a GET path. The request runs, in
// process, through readMux: a router holding ONLY the read routes listed in
// readRoutes, never the REST router itself. It carries the API key's owner as
// its identity (mcpIdentity, a context value no HTTP client can set), and an
// identity that arrives that way is capped at read level:
//
//   - requireLevel refuses anything needing more than levelRead, so a route
//     gated at build/manage/admin answers 403 even if it were listed;
//   - only GET is ever issued, and readMux has nothing else registered;
//   - streams (live updates, terminals) and routes that call a hoster or
//     GitHub live are not listed.
//
// An instance admin's key sees the admin-only read pages (builders' details,
// image build logs, installations), as their session would.
//
// The one exception to read-only is power (start, stop, reboot, forced or
// not), through MCPAccess.Act and actionMux, which holds that single route
// and nothing else. An action runs at the owner's REAL level, so it needs
// manage on the repository exactly as the UI's power buttons do. Rebuild,
// deploy, teardown and every other action stay in the UI and CLI.

type mcpIdentityKey struct{}

type mcpIdentity struct {
	accountID pgtype.UUID
	login     string
	// action marks a request from the action bridge (MCPAccess.Act): it runs
	// at the owner's real level instead of being capped at read.
	action bool
}

func mcpIdentityFrom(ctx context.Context) (mcpIdentity, bool) {
	id, ok := ctx.Value(mcpIdentityKey{}).(mcpIdentity)
	return id, ok
}

// readRoutes are the API's read pages the MCP server can reach. Each is a GET
// gated at levelRead (or instance admin, or signed-in), takes no action, and
// returns JSON.
func (s *Server) readRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	for pattern, h := range map[string]http.HandlerFunc{
		"GET /home":                                s.handleGetHome,
		"GET /repos":                               s.handleListRepos,
		"GET /repos/{id}":                          s.handleGetRepo,
		"GET /repos/{id}/builds":                   s.handleListBuilds,
		"GET /repos/{id}/configured-builds":        s.handleListConfiguredBuilds,
		"GET /repos/{id}/people":                   s.handleListPeople,
		"GET /repos/{id}/access":                   s.handleListRepositoryAccess,
		"GET /repos/{id}/my-access":                s.handleGetMyRepositoryAccess,
		"GET /configured-builds/{id}":              s.handleGetConfiguredBuild,
		"GET /content-revisions/{id}/environments": s.handleListRevisionEnvironments,

		"GET /builds/{id}":                                    s.handleGetBuild,
		"GET /builds/{id}/objects":                            s.handleListDeployedObjects,
		"GET /builds/{id}/objects/{objectId}/events":          s.handleListObjectEvents,
		"GET /builds/{id}/objects/{objectId}/heartbeats":      s.handleListObjectHeartbeats,
		"GET /builds/{id}/objects/{objectId}/render":          s.handleRenderObject,
		"GET /builds/{id}/objects/{objectId}/cloud-init":      s.handleObjectCloudInit,
		"GET /builds/{id}/objects/{objectId}/steps":           s.handleListObjectSteps,
		"GET /builds/{id}/objects/{objectId}/infra":           s.handleObjectInfra,
		"GET /builds/{id}/objects/{objectId}/config":          s.handleObjectConfig,
		"GET /builds/{id}/objects/{objectId}/validation-runs": s.handleListObjectValidationRuns,
		"GET /builds/{id}/events":                             s.handleListEvents,
		"GET /builds/{id}/events/summary":                     s.handleEventsSummary,
		"GET /builds/{id}/validation-reports":                 s.handleListValidationReports,
		"GET /builds/{id}/validation-reports/{reportId}":      s.handleGetValidationReport,
		"GET /builds/{id}/scheduled-tasks":                    s.handleListScheduledTasks,
		"GET /builds/{id}/scheduled-tasks/{taskId}/runs":      s.handleListScheduledTaskRuns,
		"GET /builds/{id}/findings":                           s.handleListFindings,
		"GET /builds/{id}/dashboard":                          s.handleGetDashboard,
		"GET /builds/{id}/topology":                           s.handleGetTopology,
		"GET /builds/{id}/external-access":                    s.handleListExternalAccess,
		"GET /builds/{id}/external-access/export":             s.handleExportExternalAccess,
		"GET /builds/{id}/credentials":                        s.handleListCredentials,
		"GET /builds/{id}/artifacts":                          s.handleGetArtifacts,
		"GET /builds/{id}/upcoming":                           s.handleGetUpcoming,

		"GET /builders":                            s.handleListBuilders,
		"GET /builder-configs":                     s.handleListBuilderConfigs,
		"GET /builder-configs/{name}":              s.handleGetBuilderConfig,
		"GET /builder-configs/{name}/stats":        s.handleGetBuilderConfigStats,
		"GET /builder-configs/{name}/usage":        s.handleGetBuilderConfigUsage,
		"GET /builder-configs/{name}/image-builds": s.handleListBuilderImageBuilds,
		"GET /image-builds/{id}/log":               s.handleImageBuildLog,
		"GET /instance-admins":                     s.handleListInstanceAdmins,
		"GET /installations":                       s.handleListInstallations,
		"GET /installations/repositories":          s.handleListUnapprovedInstalledRepositories,
		"GET /registry-credentials":                s.handleListRegistryCredentials,
		"GET /cert-status":                         s.handleCertStatus,
	} {
		mux.HandleFunc(pattern, h)
	}
	return mux
}

// Get runs one read page as the key's owner and returns its status and body.
// path is the page's path (each segment already escaped); query its query.
func (a MCPAccess) Get(ctx context.Context, accountID pgtype.UUID, login, path string, query url.Values) (int, []byte, error) {
	ctx = context.WithValue(ctx, mcpIdentityKey{}, mcpIdentity{accountID: accountID, login: login})
	u := &url.URL{Path: path, RawQuery: query.Encode()}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	w := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
	a.s.readMux.ServeHTTP(w, r)
	return w.status, w.body.Bytes(), nil
}

// actionRoutes are the only actions the MCP server can take: power.
func (s *Server) actionRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /builds/{id}/power", s.handlePowerAction)
	return mux
}

// Act runs one action route (POST path with a JSON body) as the key's owner,
// at their real level, and returns its status and body.
func (a MCPAccess) Act(ctx context.Context, accountID pgtype.UUID, login, path string, body []byte) (int, []byte, error) {
	s := a.s
	ctx = context.WithValue(ctx, mcpIdentityKey{}, mcpIdentity{accountID: accountID, login: login, action: true})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	started := time.Now()
	w := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
	s.actionMux.ServeHTTP(w, r)
	// The same bookkeeping ServeHTTP does after any change.
	s.auth.forgetAccess()
	if rest, ok := strings.CutPrefix(path, "/builds/"); ok {
		id, _, _ := strings.Cut(rest, "/")
		s.shared.forgetBuild(id)
	}
	log.Printf("api: POST %s -> %d in %s (by %s via MCP)", path, w.status, time.Since(started).Round(time.Millisecond), login)
	return w.status, w.body.Bytes(), nil
}

// mcpSession is the authSession a bridged read runs as: the key owner's
// account, with no GitHub token (a key never carries one).
func mcpSession(id mcpIdentity) authSession {
	return authSession{db.GetSessionByTokenHashRow{AccountID: id.accountID, GithubLogin: id.login}}
}
