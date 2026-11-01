package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
	"github.com/globalcptc/laforge/internal/mcp"
)

func bridgeAccount(t *testing.T, env *apiTestEnv, who string) (pgtype.UUID, string) {
	t.Helper()
	login := fmt.Sprintf("%s-%d", who, os.Getpid())
	sum := sha256.Sum256([]byte(login))
	id := int64(sum[0])<<24 | int64(sum[1])<<16 | int64(sum[2])<<8 | int64(sum[3])
	a, err := env.q.UpsertAccount(context.Background(), db.UpsertAccountParams{GithubID: 700_000_000 + id, GithubLogin: login})
	if err != nil {
		t.Fatal(err)
	}
	return a.ID, login
}

// The bridge reaches only its read pages, as the key's owner, capped at read.
func TestMCPBridgeReadsOnlyAndCapsAtRead(t *testing.T) {
	env := setupAPITest(t)
	build := stepsBuildFixture(t, env)
	ctx := context.Background()
	access := env.server.MCPAccess()

	reader, readerLogin := bridgeAccount(t, env, "bridge-reader")
	repoAdmin, repoAdminLogin := bridgeAccount(t, env, "bridge-repo-admin")
	stranger, strangerLogin := bridgeAccount(t, env, "bridge-stranger")
	admin, adminLogin := bridgeAccount(t, env, "bridge-admin")
	env.server.AdminLogins = []string{adminLogin}
	env.gh.collaborators = map[string][]ghclient.Collaborator{env.owner + "/" + env.name: {
		{Login: readerLogin, Permissions: ghclient.RepoPermissions{Pull: true}},
		{Login: repoAdminLogin, Permissions: ghclient.RepoPermissions{Admin: true, Push: true, Pull: true}},
	}}

	get := func(account pgtype.UUID, login, path string) (int, string) {
		t.Helper()
		status, body, err := access.Get(ctx, account, login, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		return status, string(body)
	}
	b := "/builds/" + build.ID.String()
	objs, _ := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	var host db.DeployedObject
	for _, o := range objs {
		if o.ObjectName == "database" {
			host = o
			break
		}
	}

	// A reader sees every read page, secrets included.
	for _, path := range []string{b, b + "/objects", b + "/events", b + "/credentials", b + "/external-access/export",
		b + "/scheduled-tasks", b + "/topology", b + "/findings",
		b + "/objects/" + host.ID.String() + "/steps", b + "/objects/" + host.ID.String() + "/infra",
		b + "/objects/" + host.ID.String() + "/render", b + "/objects/" + host.ID.String() + "/config",
		"/repos/" + env.repo.ID.String() + "/people", "/builders"} {
		if status, body := get(reader, readerLogin, path); status != http.StatusOK {
			t.Errorf("reader GET %s: %d %s", path, status, body)
		}
	}
	if _, body := get(reader, readerLogin, b+"/objects/"+host.ID.String()+"/render"); !strings.Contains(body, "rendered") {
		t.Errorf("render page has no rendered steps: %.300s", body)
	}

	// Someone who can't read the repository sees nothing.
	if status, _ := get(stranger, strangerLogin, b); status != http.StatusForbidden {
		t.Errorf("stranger GET build: %d, want 403", status)
	}

	// Pages that need more than read refuse even a repository admin...
	if status, _ := get(repoAdmin, repoAdminLogin, "/repos/"+env.repo.ID.String()+"/access"); status != http.StatusForbidden {
		t.Errorf("repo admin GET access list through the bridge: %d, want 403 (capped at read)", status)
	}
	// ...while an instance admin's read-only admin pages work.
	for _, path := range []string{"/repos/" + env.repo.ID.String() + "/access", "/instance-admins", "/builder-configs"} {
		if status, body := get(admin, adminLogin, path); status != http.StatusOK {
			t.Errorf("admin GET %s: %d %s", path, status, body)
		}
	}
	if status, _ := get(reader, readerLogin, "/instance-admins"); status != http.StatusForbidden {
		t.Errorf("reader GET instance admins: %d, want 403", status)
	}

	// Nothing but the listed read pages is there at all: no actions, no live
	// hoster calls, no streams -- even for an instance admin.
	for _, path := range []string{b + "/deploy", b + "/teardown", b + "/readiness", b + "/drift", b + "/live",
		b + "/objects/" + host.ID.String() + "/terminal", "/builder-configs/microcloud/available-images", "/auth/me"} {
		if status, _ := get(admin, adminLogin, path); status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
			t.Errorf("admin GET %s through the bridge: %d, want not found", path, status)
		}
	}
	after, _ := env.q.GetBuild(ctx, build.ID)
	if after.Status != build.Status {
		t.Errorf("build status changed from %s to %s", build.Status, after.Status)
	}
}

// A bridged identity can't come from outside: the context key is unexported,
// and a request with no session is still anonymous.
func TestMCPBridgeIdentityIsInProcessOnly(t *testing.T) {
	env := setupAPITest(t)
	resp, err := http.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/builds")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous GET: %d, want 401", resp.StatusCode)
	}
}

func mcpTestDSN(t *testing.T) string {
	t.Helper()
	cfg, err := pgx.ParseConfig(testDBConnString(t))
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=laforge_mcp password=laforge_mcp_dev_only_do_not_use_in_prod sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database)
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// The whole path: an MCP client with an API key, the MCP server on its own
// role, and the API's read pages behind it.
func TestMCPEndToEndThroughAPI(t *testing.T) {
	env := setupAPITest(t)
	build := stepsBuildFixture(t, env)
	ctx := context.Background()
	mq, mpool, err := mcp.OpenPool(ctx, mcpTestDSN(t))
	if err != nil {
		t.Skipf("no laforge_mcp role on the test database: %v", err)
	}
	defer mpool.Close()

	account, login := bridgeAccount(t, env, "mcp-e2e")
	env.gh.collaborators = map[string][]ghclient.Collaborator{env.owner + "/" + env.name: {
		{Login: login, Permissions: ghclient.RepoPermissions{Pull: true}},
	}}
	token := fmt.Sprintf("lfk_e2e%dAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", os.Getpid())
	if _, err := env.q.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		AccountID: account, Name: "e2e", Prefix: token[:12], TokenHash: hashToken(token),
		ExpiresAt: pgtype.Timestamptz{Time: build.CreatedAt.Time.Add(24 * 3600e9), Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), `DELETE FROM api_key WHERE account_id = $1`, account)
	})

	srv := mcp.New(mcp.Config{Queries: mq, Access: env.server.MCPAccess(), PerAccountBurst: 1000})
	env.server.MCP = srv.Handler()
	client := sdk.NewClient(&sdk.Implementation{Name: "e2e", Version: "1"}, nil)
	cs, err := client.Connect(ctx, &sdk.StreamableClientTransport{
		Endpoint: env.httpURL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{token}}, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
		if tool.Name != "power" && (tool.Annotations == nil || !tool.Annotations.ReadOnlyHint) {
			t.Errorf("%s isn't marked read-only", tool.Name)
		}
	}
	t.Logf("%d tools", len(tools.Tools))

	short := build.ID.String()[:8]
	// One team-1 database host, by its exact deployed name.
	var dbHost string
	if err := env.server.Pool.QueryRow(ctx, `SELECT COALESCE(o.as_name, o.object_name) FROM deployed_object o JOIN team t ON t.id = o.team_id
		WHERE t.build_id = $1 AND t.team_number = 1 AND o.object_name = 'database' ORDER BY 1 LIMIT 1`, build.ID).Scan(&dbHost); err != nil {
		t.Fatal(err)
	}
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		if !names[name] {
			t.Fatalf("no tool %s", name)
		}
		res, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var text string
		for _, c := range res.Content {
			if tc, ok := c.(*sdk.TextContent); ok {
				text += tc.Text
			}
		}
		if res.IsError {
			t.Fatalf("%s(%v): %s", name, args, text)
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(text), &out); err != nil {
			t.Fatalf("%s: decoding %.200s: %v", name, text, err)
		}
		return out
	}

	call("home", nil)
	call("get_build", map[string]any{"build_id": short})
	call("get_build_detail", map[string]any{"build_id": short})
	call("get_topology", map[string]any{"build_id": short})
	call("list_credentials", map[string]any{"build_id": short})
	call("export_external_access", map[string]any{"build_id": short})
	call("list_scheduled_tasks", map[string]any{"build_id": short})
	call("list_findings", map[string]any{"build_id": short})
	if ev := call("list_build_events", map[string]any{"build_id": short, "limit": 20}); ev["count"] == nil {
		t.Errorf("list_build_events: %v", ev)
	}
	steps := call("list_steps", map[string]any{"build_id": short, "object": dbHost, "team": 1})
	if n, _ := steps["count"].(float64); n == 0 {
		t.Errorf("list_steps database: no steps: %v", steps)
	}
	call("get_object_infra", map[string]any{"build_id": short, "object": dbHost, "team": 1})
	call("get_object_config", map[string]any{"build_id": short, "object": dbHost, "team": 1})
	rendered := call("render_object", map[string]any{"build_id": short, "object": dbHost, "team": 1, "max_field_chars": 50})
	if b, _ := json.Marshal(rendered); !strings.Contains(string(b), "ask again with a larger max_field_chars") {
		t.Errorf("render_object with max_field_chars 50 cut nothing: %.300s", b)
	}
	call("list_object_events", map[string]any{"build_id": short, "object": dbHost, "team": 1})
	call("list_people", map[string]any{"repository": env.owner + "/" + env.name})
	call("search_agent_tasks", map[string]any{"build_id": short, "statuses": []string{"pending", "leased", "blocked", "done"}})
	call("list_runner_tasks", map[string]any{"build_id": short})
	call("heartbeat_gaps", map[string]any{"build_id": short})
	call("list_builders", nil)

	// An object name that several teams share needs a team.
	res, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "list_steps", Arguments: map[string]any{"build_id": short, "object": "database"}})
	if err != nil || !res.IsError {
		t.Errorf("ambiguous object: err=%v isError=%v", err, res != nil && res.IsError)
	}
}

// Power is the MCP server's one action: it runs at the owner's real level, so
// it needs manage, and nothing but power is reachable that way.
func TestMCPActionBridgePowerOnly(t *testing.T) {
	env := setupAPITest(t)
	build := stepsBuildFixture(t, env)
	ctx := context.Background()
	// Power resolves the builder through the build's configured build.
	if _, err := env.server.Pool.Exec(ctx, `WITH cb AS (
		INSERT INTO configured_build (repository_id, branch, environment_path, builder_config_name)
		VALUES ($1, 'main', 'lm-test.yaml', 'microcloud') RETURNING id)
		UPDATE build SET configured_build_id = (SELECT id FROM cb) WHERE id = $2`, env.repo.ID, build.ID); err != nil {
		t.Fatal(err)
	}
	access := env.server.MCPAccess()
	reader, readerLogin := bridgeAccount(t, env, "power-reader")
	builder, builderLogin := bridgeAccount(t, env, "power-builder")
	manager, managerLogin := bridgeAccount(t, env, "power-manager")
	env.gh.collaborators = map[string][]ghclient.Collaborator{env.owner + "/" + env.name: {
		{Login: readerLogin, Permissions: ghclient.RepoPermissions{Pull: true}},
		{Login: builderLogin, Permissions: ghclient.RepoPermissions{Push: true, Pull: true}},
	}}
	if _, err := env.q.UpsertRepositoryAccess(ctx, db.UpsertRepositoryAccessParams{RepositoryID: env.repo.ID, AccountID: manager, Level: "manage"}); err != nil {
		t.Fatal(err)
	}
	b := "/builds/" + build.ID.String()
	body := []byte(`{"target":{"team":1,"kind":"host"},"action":"reboot","force":true}`)

	for _, who := range []struct {
		id    pgtype.UUID
		login string
	}{{reader, readerLogin}, {builder, builderLogin}} {
		if status, resp, _ := access.Act(ctx, who.id, who.login, b+"/power", body); status != http.StatusForbidden {
			t.Errorf("%s (below manage) power: %d %s, want 403", who.login, status, resp)
		}
	}
	status, resp, err := access.Act(ctx, manager, managerLogin, b+"/power", body)
	if err != nil || status != http.StatusAccepted {
		t.Fatalf("manager power: %d %s %v", status, resp, err)
	}
	var out struct {
		Started int `json:"started"`
	}
	json.Unmarshal(resp, &out)
	if out.Started == 0 {
		t.Fatalf("manager power started nothing: %s", resp)
	}
	var events int
	env.server.Pool.QueryRow(ctx, `SELECT count(*) FROM event WHERE build_id = $1 AND kind = 'power.force_reboot.started' AND message LIKE '%via MCP%'`, build.ID).Scan(&events)
	if events != 1 {
		t.Errorf("power.force_reboot.started events naming the MCP caller: %d, want 1", events)
	}

	// Nothing else can be done through the action bridge, even by a manager
	// or an instance admin.
	env.server.AdminLogins = []string{managerLogin}
	for _, path := range []string{b + "/rebuild", b + "/deploy", b + "/teardown", b + "/tasks", b + "/restart-steps", b + "/revalidate", b + "/teams/1/access", "/repos"} {
		if status, _, _ := access.Act(ctx, manager, managerLogin, path, []byte(`{}`)); status != http.StatusNotFound && status != http.StatusMethodNotAllowed {
			t.Errorf("action bridge POST %s: %d, want not found", path, status)
		}
	}
	// And the read bridge still won't let a manager past read.
	if status, _, _ := access.Get(ctx, manager, managerLogin, b+"/readiness", nil); status != http.StatusNotFound {
		t.Errorf("read bridge GET readiness: %d", status)
	}
}
