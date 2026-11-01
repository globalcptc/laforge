package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/globalcptc/laforge/internal/db"
)

// adminDSN is the test database as its owner; mcpDSN is the same database as
// the laforge_mcp role (migrations/00081's dev password).
func testDSNs(t *testing.T) (adminDSN, mcpDSN string) {
	t.Helper()
	adminDSN = os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Skip("set LAFORGE_TEST_DATABASE_URL to a database migrated through 00081")
	}
	if v := os.Getenv("LAFORGE_TEST_MCP_DATABASE_URL"); v != "" {
		return adminDSN, v
	}
	cfg, err := pgx.ParseConfig(adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	mcpDSN = fmt.Sprintf("host=%s port=%d dbname=%s user=laforge_mcp password=laforge_mcp_dev_only_do_not_use_in_prod sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database)
	return adminDSN, mcpDSN
}

// fixture is one account with keys, two repositories (only the first
// readable), and a build in each.
type fixture struct {
	admin               *pgxpool.Pool
	q                   *db.Queries
	account             pgtype.UUID
	login               string
	token, revokedToken string
	expiredToken        string
	repo, otherRepo     pgtype.UUID
	repoName            string
	build, otherBuild   pgtype.UUID
	acted               []string
	denyActs            bool
}

func hash(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	adminDSN, mcpDSN := testDSNs(t)
	_, admin, err := db.Open(ctx, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	q, pool, err := OpenPool(ctx, mcpDSN)
	if err != nil {
		t.Fatalf("OpenPool: %v", err)
	}
	t.Cleanup(pool.Close)

	n := time.Now().UnixNano()
	f := &fixture{admin: admin, q: q, login: fmt.Sprintf("mcp-tester-%d", n)}
	f.token = fmt.Sprintf("lfk_test%dAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", n)
	f.revokedToken = f.token + "r"
	f.expiredToken = f.token + "x"
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	row := func(sql string, args ...any) pgtype.UUID {
		t.Helper()
		var id pgtype.UUID
		must(admin.QueryRow(ctx, sql, args...).Scan(&id))
		return id
	}
	f.account = row(`INSERT INTO account (github_id, github_login) VALUES ($1, $2) RETURNING id`, n%1_000_000_000, f.login)
	key := func(tok, extra string) {
		_, err := admin.Exec(ctx, `INSERT INTO api_key (account_id, name, prefix, token_hash, expires_at, revoked_at)
			VALUES ($1, 'test key', $2, $3, `+extra+`)`, f.account, tok[:12], hash(tok))
		must(err)
	}
	key(f.token, "now() + interval '1 day', NULL")
	key(f.revokedToken, "now() + interval '1 day', now()")
	key(f.expiredToken, "now() - interval '1 minute', NULL")

	f.repoName = fmt.Sprintf("mcp-test-%d", n)
	f.repo = row(`INSERT INTO repository (github_owner, github_repo) VALUES ('globalcptc', $1) RETURNING id`, f.repoName)
	f.otherRepo = row(`INSERT INTO repository (github_owner, github_repo) VALUES ('globalcptc', $1) RETURNING id`, f.repoName+"-hidden")
	t.Cleanup(func() {
		admin.Exec(context.Background(), `DELETE FROM repository WHERE id = ANY($1)`, []pgtype.UUID{f.repo, f.otherRepo})
		admin.Exec(context.Background(), `DELETE FROM account WHERE id = $1`, f.account)
	})

	makeBuild := func(repo pgtype.UUID) pgtype.UUID {
		rev := row(`INSERT INTO content_revision (repository_id, commit_sha, ref, valid, commit_message)
			VALUES ($1, 'abcdef0123456789', 'main', true, 'Fix the DC') RETURNING id`, repo)
		_, err := admin.Exec(ctx, `INSERT INTO environment (content_revision_id, path, name, teams, root_password)
			VALUES ($1, 'central.yaml', 'central', 2, 'SuperSecretRootPw!')`, rev)
		must(err)
		b := row(`INSERT INTO build (content_revision_id, environment_name, status) VALUES ($1, 'central', 'building') RETURNING id`, rev)
		for team := 1; team <= 2; team++ {
			tid := row(`INSERT INTO team (build_id, team_number) VALUES ($1, $2) RETURNING id`, b, team)
			for i, spec := range []struct{ name, status, loc, hb string }{
				{"dc01", "finished", "member-a", "now() - interval '10 seconds'"},
				{"web01", "finished", "member-b", "now() - interval '20 minutes'"},
				{"ws01", "build_failed", "member-b", ""},
			} {
				oid := row(`INSERT INTO deployed_object (team_id, kind, object_name, status, instance_location, deployed_at, last_error)
					VALUES ($1, 'host', $2, $3, $4, now() - interval '1 hour', $5) RETURNING id`,
					tid, spec.name, spec.status, spec.loc, map[bool]any{true: "step 3 failed", false: nil}[spec.status == "build_failed"])
				if spec.hb != "" {
					_, err := admin.Exec(ctx, `INSERT INTO agent_session (deployed_object_id, cert_fingerprint, last_heartbeat_at)
						VALUES ($1, $2, `+spec.hb+`)`, oid, fmt.Sprintf("fp-%d-%d-%d", n, team, i))
					must(err)
				}
			}
		}
		return b
	}
	f.build = makeBuild(f.repo)
	f.otherBuild = makeBuild(f.otherRepo)
	return f
}

// fakeAccess says the account can read exactly one repository, and records
// every action asked of it.
type fakeAccess struct {
	repos    []pgtype.UUID
	acted    *[]string
	denyActs bool
}

func (a fakeAccess) Act(_ context.Context, _ pgtype.UUID, _, path string, body []byte) (int, []byte, error) {
	if a.denyActs {
		return http.StatusForbidden, []byte(`{"error":"your github token does not have push access to this repository"}`), nil
	}
	if a.acted != nil {
		*a.acted = append(*a.acted, path+" "+string(body))
	}
	return http.StatusAccepted, []byte(`{"started":1,"failed":0,"results":[{"object_id":"x","name":"web01"}]}`), nil
}

func (a fakeAccess) ReadableRepos(context.Context, pgtype.UUID, string) ([]pgtype.UUID, error) {
	return a.repos, nil
}

// Get answers every page with its path, so page tools can be checked for
// which page they asked for.
func (a fakeAccess) Get(_ context.Context, _ pgtype.UUID, _, path string, q url.Values) (int, []byte, error) {
	if strings.HasSuffix(path, "/forbidden") {
		return http.StatusForbidden, []byte(`{"error":"nope"}`), nil
	}
	b, _ := json.Marshal([]map[string]string{{"path": path, "query": q.Encode(), "long": strings.Repeat("x", 5000)}})
	return http.StatusOK, b, nil
}

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func connect(t *testing.T, url, token string) (*sdk.ClientSession, error) {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	return client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint: url, HTTPClient: &http.Client{Transport: bearer{token}}, MaxRetries: -1,
	}, nil)
}

func callTool[T any](t *testing.T, cs *sdk.ClientSession, name string, args map[string]any) (T, *sdk.CallToolResult) {
	t.Helper()
	var out T
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !res.IsError {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("%s: decoding %s: %v", name, b, err)
		}
	}
	return out, res
}

func errText(res *sdk.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, " ")
}

func newTestServer(t *testing.T, f *fixture, cfg Config) (*Server, string) {
	t.Helper()
	cfg.Queries = f.q
	cfg.Access = fakeAccess{repos: []pgtype.UUID{f.repo}, acted: &f.acted, denyActs: f.denyActs}
	s := New(cfg)
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return s, srv.URL
}

func TestToolsEndToEnd(t *testing.T) {
	f := newFixture(t)
	_, url := newTestServer(t, f, Config{PerAccountBurst: 100}) // more calls than the default burst
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 50 {
		t.Errorf("got %d tools, want every read page", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		a := tool.Annotations
		if tool.Name == "power" {
			continue // the one action; TestPowerTool checks it's marked destructive
		}
		if a == nil || !a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint {
			t.Errorf("tool %s isn't annotated read-only: %+v", tool.Name, a)
		}
	}

	who, _ := callTool[whoamiOut](t, cs, "whoami", nil)
	if who.Login != f.login || len(who.Repositories) != 1 || who.Repositories[0].Name != "globalcptc/"+f.repoName {
		t.Fatalf("whoami = %+v", who)
	}

	repos, _ := callTool[listReposOut](t, cs, "list_repos", nil)
	if len(repos.Repositories) != 1 {
		t.Fatalf("list_repos = %+v, want only the readable repository", repos)
	}

	builds, _ := callTool[listBuildsOut](t, cs, "list_builds", map[string]any{"repository": "globalcptc/" + f.repoName})
	if len(builds.Builds) != 1 || builds.Builds[0].ID != f.build.String() || builds.Builds[0].Teams != 2 {
		t.Fatalf("list_builds = %+v", builds)
	}

	// By a short prefix, the way people paste build ids.
	build, res := callTool[getBuildOut](t, cs, "get_build", map[string]any{"build_id": f.build.String()[:8]})
	if res.IsError {
		t.Fatalf("get_build: %s", errText(res))
	}
	if build.Objects != 6 || build.ByStatus["build_failed"] != 2 || build.ByAgentStatus["healthy"] != 2 || build.ByAgentStatus["missing"] != 2 || build.ByAgentStatus["booting"] != 2 {
		t.Fatalf("get_build counts = %+v", build)
	}
	if len(build.ByLocation) != 2 || build.ByLocation[1].Location != "member-b" || build.ByLocation[1].Failed != 2 || build.ByLocation[1].AgentsMissing != 2 || build.ByLocation[1].AgentsBooting != 2 {
		t.Fatalf("get_build by_location = %+v", build.ByLocation)
	}
	if len(build.Teams) != 2 {
		t.Fatalf("get_build teams = %+v", build.Teams)
	}

	// web01 went quiet 20 minutes ago: missing, with how long since it checked in.
	objs, _ := callTool[listObjectsOut](t, cs, "list_objects", map[string]any{"build_id": f.build.String(), "location": "member-b", "agent_status": "missing"})
	if objs.Matched != 2 || len(objs.Objects) != 2 || objs.Objects[0].Name != "web01" || objs.Objects[0].SecondsSinceCheck == nil || *objs.Objects[0].SecondsSinceCheck < 1100 {
		t.Fatalf("list_objects member-b missing = %+v", objs)
	}
	failed, _ := callTool[listObjectsOut](t, cs, "list_objects", map[string]any{"build_id": f.build.String(), "status": "failed", "team": 1})
	if failed.Matched != 1 || failed.Objects[0].Name != "ws01" || failed.Objects[0].LastError != "step 3 failed" {
		t.Fatalf("list_objects failed team 1 = %+v", failed)
	}
	limited, _ := callTool[listObjectsOut](t, cs, "list_objects", map[string]any{"build_id": f.build.String(), "limit": 2})
	if limited.Matched != 6 || len(limited.Objects) != 2 || !limited.Truncated {
		t.Fatalf("list_objects limit 2 = %+v", limited)
	}

	// Nothing a key can't read comes back, and a hidden build looks the same
	// as a missing one.
	for _, ref := range []string{f.otherBuild.String(), f.otherBuild.String()[:8], "00000000-0000-0000-0000-000000000000"} {
		_, res := callTool[getBuildOut](t, cs, "get_build", map[string]any{"build_id": ref})
		if !res.IsError || !strings.Contains(errText(res), "can't read it") {
			t.Fatalf("get_build %s: want the not-found-or-no-access error, got %v", ref, res.Content)
		}
	}
	if _, res := callTool[listBuildsOut](t, cs, "list_builds", map[string]any{"repository": "globalcptc/" + f.repoName + "-hidden"}); !res.IsError {
		t.Fatal("list_builds on an unreadable repository succeeded")
	}

	// The calls were audited, denials included.
	var ok, denied int
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'ok'), count(*) FILTER (WHERE status = 'denied')
		FROM mcp_audit WHERE account_id = $1`, f.account).Scan(&ok, &denied); err != nil {
		t.Fatal(err)
	}
	if ok != 7 || denied != 4 {
		t.Fatalf("audit rows: ok=%d denied=%d, want 7 and 4", ok, denied)
	}
	var lastUsed pgtype.Timestamptz
	f.admin.QueryRow(ctx, `SELECT last_used_at FROM api_key WHERE token_hash = $1`, hash(f.token)).Scan(&lastUsed)
	if !lastUsed.Valid {
		t.Fatal("last_used_at wasn't recorded")
	}
}

func TestKeyChecks(t *testing.T) {
	f := newFixture(t)
	s, url := newTestServer(t, f, Config{})
	for name, tok := range map[string]string{
		"revoked": f.revokedToken, "expired": f.expiredToken, "unknown": "lfk_nope",
		"github token": "gho_abc", "empty": "",
	} {
		if cs, err := connect(t, url, tok); err == nil {
			cs.Close()
			t.Errorf("%s key: connected, want refused", name)
		}
	}
	// Plain HTTP: no key is a 401 with a WWW-Authenticate challenge.
	resp, err := http.Post(url, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") == "" {
		t.Fatalf("no key: status %d, WWW-Authenticate %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}

	// A key revoked while cached stops working once the API says so.
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	cs.Close()
	f.admin.Exec(context.Background(), `UPDATE api_key SET revoked_at = now() WHERE token_hash = $1`, hash(f.token))
	s.ForgetKeys()
	if cs, err := connect(t, url, f.token); err == nil {
		cs.Close()
		t.Fatal("revoked key still connects after ForgetKeys")
	}
}

func TestRateLimit(t *testing.T) {
	f := newFixture(t)
	_, url := newTestServer(t, f, Config{PerAccountRate: 0.01, PerAccountBurst: 2})
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	for i := 0; i < 2; i++ {
		if _, res := callTool[whoamiOut](t, cs, "whoami", nil); res.IsError {
			t.Fatalf("call %d: %s", i, errText(res))
		}
	}
	_, res := callTool[whoamiOut](t, cs, "whoami", nil)
	if !res.IsError || !strings.Contains(errText(res), "rate limited") {
		t.Fatalf("third call: want rate limited, got %v", res.Content)
	}
}

// The role is the read-only guarantee: every query this package runs works
// as laforge_mcp, and Postgres refuses writes and secret columns outright.
func TestRoleGrants(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	q := f.q

	// Every Mcp* query runs under the role's grants.
	if _, err := q.McpGetAPIKeyByHash(ctx, hash(f.token)); err != nil {
		t.Errorf("McpGetAPIKeyByHash: %v", err)
	}
	var keyID pgtype.UUID
	f.admin.QueryRow(ctx, `SELECT id FROM api_key WHERE token_hash = $1`, hash(f.token)).Scan(&keyID)
	if err := q.McpTouchAPIKey(ctx, keyID); err != nil {
		t.Errorf("McpTouchAPIKey: %v", err)
	}
	if err := q.McpInsertAudit(ctx, db.McpInsertAuditParams{ApiKeyID: keyID, AccountID: f.account, Tool: "t", Args: []byte(`{}`), Status: "ok"}); err != nil {
		t.Errorf("McpInsertAudit: %v", err)
	}
	if _, err := q.McpListRepositories(ctx, []pgtype.UUID{f.repo}); err != nil {
		t.Errorf("McpListRepositories: %v", err)
	}
	if _, err := q.McpGetBuildRepository(ctx, f.build); err != nil {
		t.Errorf("McpGetBuildRepository: %v", err)
	}
	if _, err := q.McpFindBuildsByIDPrefix(ctx, f.build.String()[:8]); err != nil {
		t.Errorf("McpFindBuildsByIDPrefix: %v", err)
	}
	if _, err := q.McpListBuildsByRepository(ctx, db.McpListBuildsByRepositoryParams{RepositoryID: f.repo, MaxRows: 5}); err != nil {
		t.Errorf("McpListBuildsByRepository: %v", err)
	}
	b, err := q.McpGetBuild(ctx, f.build)
	if err != nil {
		t.Errorf("McpGetBuild: %v", err)
	}
	if _, err := q.McpGetBuildEnvironment(ctx, db.McpGetBuildEnvironmentParams{ContentRevisionID: b.ContentRevisionID, Name: b.EnvironmentName}); err != nil {
		t.Errorf("McpGetBuildEnvironment: %v", err)
	}
	if _, err := q.McpListTeamsByBuild(ctx, f.build); err != nil {
		t.Errorf("McpListTeamsByBuild: %v", err)
	}
	if _, err := q.McpListObjectsByBuild(ctx, f.build); err != nil {
		t.Errorf("McpListObjectsByBuild: %v", err)
	}

	// And everything else is refused by Postgres itself.
	_, mcpDSN := testDSNs(t)
	conn, err := pgx.Connect(ctx, mcpDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, sql := range []string{
		`SELECT root_password FROM environment`,
		`SELECT * FROM environment`,
		`SELECT vars FROM environment`,
		`SELECT payload FROM task`,
		`SELECT steps FROM host`,
		`SELECT secret FROM build_render_secret`,
		`SELECT secret FROM build_render_secret`,
		`SELECT client_key_pem FROM builder_credential`,
		`SELECT aws_secret_access_key FROM builder_credential`,
		`SELECT secret FROM registry_credential`,
		`SELECT github_token FROM session`,
		`SELECT token FROM agent_artifact`,
		`SELECT attributes FROM person`,
		`SELECT incus_client_key_path FROM builder_config`,
		`SELECT cert_fingerprint FROM agent_session`,
		`SELECT * FROM api_key`,
		`INSERT INTO team (build_id, team_number) VALUES (gen_random_uuid(), 99)`,
		`UPDATE deployed_object SET status = 'destroyed'`,
		`UPDATE build SET status = 'tearing_down'`,
		`DELETE FROM build`,
		`UPDATE api_key SET revoked_at = NULL`,
		`UPDATE api_key SET expires_at = now() + interval '10 years'`,
		`INSERT INTO api_key (account_id, name, prefix, token_hash, expires_at) VALUES (gen_random_uuid(), 'x', 'x', 'x', now())`,
		`DELETE FROM mcp_audit`,
		`UPDATE mcp_audit SET status = 'ok'`,
		`TRUNCATE mcp_audit`,
		`CREATE TABLE mcp_sneaky (id int)`,
	} {
		if _, err := conn.Exec(ctx, sql); err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("%s: err = %v, want permission denied", sql, err)
		}
	}

	// OpenPool refuses a connection that isn't the read-only role.
	adminDSN, _ := testDSNs(t)
	if _, p, err := OpenPool(ctx, adminDSN); err == nil {
		p.Close()
		t.Error("OpenPool accepted the main database role")
	}
}

func TestDiagnosticTools(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, url := newTestServer(t, f, Config{PerAccountBurst: 100})
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	// dc01 (member-a) checks in steadily; web01 (member-b) went quiet for 10
	// minutes 30 minutes ago.
	var dc, web pgtype.UUID
	f.admin.QueryRow(ctx, `SELECT o.id FROM deployed_object o JOIN team t ON t.id = o.team_id WHERE t.build_id = $1 AND t.team_number = 1 AND o.object_name = 'dc01'`, f.build).Scan(&dc)
	f.admin.QueryRow(ctx, `SELECT o.id FROM deployed_object o JOIN team t ON t.id = o.team_id WHERE t.build_id = $1 AND t.team_number = 1 AND o.object_name = 'web01'`, f.build).Scan(&web)
	for _, beat := range []struct {
		obj pgtype.UUID
		ago string
	}{{dc, "50 minutes"}, {dc, "49 minutes"}, {dc, "48 minutes"}, {web, "41 minutes"}, {web, "31 minutes"}, {web, "30 minutes"}} {
		if _, err := f.admin.Exec(ctx, `INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, created_at) VALUES ($1, 'fp', now() - $2::interval)`, beat.obj, beat.ago); err != nil {
			t.Fatal(err)
		}
	}
	gaps, res := callTool[heartbeatGapsOut](t, cs, "heartbeat_gaps", map[string]any{"build_id": f.build.String()})
	if res.IsError {
		t.Fatal(errText(res))
	}
	if len(gaps.Gaps) != 1 || gaps.Gaps[0].Object != "web01" || gaps.Gaps[0].Location != "member-b" || gaps.Gaps[0].GapSeconds < 590 {
		t.Fatalf("gaps = %+v", gaps.Gaps)
	}
	if len(gaps.ByLocation) == 0 || gaps.ByLocation[0].Location != "member-b" || gaps.ByLocation[0].SilentNow != 2 {
		t.Fatalf("by_location = %+v", gaps.ByLocation)
	}

	// A failed ad-hoc command with a payload and long output.
	long := strings.Repeat("line of output\n", 5000)
	var taskID pgtype.UUID
	if err := f.admin.QueryRow(ctx, `INSERT INTO agent_task (deployed_object_id, step_index, command, payload, status, output, last_error, ad_hoc)
		VALUES ($1, 1000, 'exec', '{"command":"systemctl restart dns"}', 'failed', $2, 'exit status 1', true) RETURNING id`, web, long).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	tasks, _ := callTool[searchTasksOut](t, cs, "search_agent_tasks", map[string]any{"build_id": f.build.String()[:8], "ad_hoc": true})
	if len(tasks.Tasks) != 1 || tasks.Tasks[0].LastError != "exit status 1" || len(tasks.Tasks[0].OutputTail) != 1000 {
		t.Fatalf("search_agent_tasks = %+v", tasks)
	}
	if p, _ := tasks.Tasks[0].Payload.(map[string]any); p["command"] != "systemctl restart dns" {
		t.Fatalf("payload = %v", tasks.Tasks[0].Payload)
	}
	page, _ := callTool[stepOutputOut](t, cs, "get_step_output", map[string]any{"task_id": taskID.String(), "max_bytes": 1000})
	if page.OutputBytes != len(long) || len(page.Output) != 1000 || !page.More {
		t.Fatalf("get_step_output page 1 = bytes %d, len %d, more %v", page.OutputBytes, len(page.Output), page.More)
	}
	last, _ := callTool[stepOutputOut](t, cs, "get_step_output", map[string]any{"task_id": taskID.String(), "offset": len(long) - 10})
	if last.Output != long[len(long)-10:] || last.More {
		t.Fatalf("get_step_output last page = %q more=%v", last.Output, last.More)
	}

	// A task in a build the key can't read is refused like a missing one.
	var hidden pgtype.UUID
	f.admin.QueryRow(ctx, `INSERT INTO agent_task (deployed_object_id, step_index, command, status)
		SELECT o.id, 0, 'exec', 'done' FROM deployed_object o JOIN team t ON t.id = o.team_id WHERE t.build_id = $1 LIMIT 1 RETURNING id`, f.otherBuild).Scan(&hidden)
	if _, res := callTool[stepOutputOut](t, cs, "get_step_output", map[string]any{"task_id": hidden.String()}); !res.IsError {
		t.Fatal("get_step_output read a task in an unreadable build")
	}
}

func TestPageTools(t *testing.T) {
	f := newFixture(t)
	_, url := newTestServer(t, f, Config{PerAccountBurst: 100})
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	type page struct {
		Items []map[string]string `json:"items"`
		Count int                 `json:"count"`
	}
	out, res := callTool[page](t, cs, "list_steps", map[string]any{"build_id": f.build.String()[:8], "object": "dc01", "team": 2})
	if res.IsError {
		t.Fatal(errText(res))
	}
	var dc pgtype.UUID
	f.admin.QueryRow(context.Background(), `SELECT o.id FROM deployed_object o JOIN team t ON t.id = o.team_id WHERE t.build_id = $1 AND t.team_number = 2 AND o.object_name = 'dc01'`, f.build).Scan(&dc)
	want := "/builds/" + f.build.String() + "/objects/" + dc.String() + "/steps"
	if out.Count != 1 || out.Items[0]["path"] != want {
		t.Fatalf("list_steps asked for %v, want %s", out.Items, want)
	}
	if !strings.Contains(out.Items[0]["long"], "ask again with a larger max_field_chars") || len(out.Items[0]["long"]) > 4100 {
		t.Fatalf("long string wasn't cut at the default: %d bytes", len(out.Items[0]["long"]))
	}
	full, _ := callTool[page](t, cs, "list_steps", map[string]any{"build_id": f.build.String(), "object": "dc01", "team": 2, "max_field_chars": 10000})
	if len(full.Items[0]["long"]) != 5000 {
		t.Fatalf("max_field_chars 10000 still cut: %d", len(full.Items[0]["long"]))
	}
	events, _ := callTool[page](t, cs, "list_build_events", map[string]any{"build_id": f.build.String(), "limit": 5, "before": "2026-01-01T00:00:00Z"})
	if events.Items[0]["query"] != "before=2026-01-01T00%3A00%3A00Z&limit=5" {
		t.Fatalf("list_build_events query = %q", events.Items[0]["query"])
	}

	// The key's own access is checked before the page is asked for.
	if _, res := callTool[page](t, cs, "list_credentials", map[string]any{"build_id": f.otherBuild.String()}); !res.IsError || !strings.Contains(errText(res), "can't read it") {
		t.Fatalf("list_credentials on an unreadable build: %v", res.Content)
	}
	// An object name in several teams needs a team.
	if _, res := callTool[page](t, cs, "list_steps", map[string]any{"build_id": f.build.String(), "object": "dc01"}); !res.IsError || !strings.Contains(errText(res), "matches 2 objects") {
		t.Fatalf("ambiguous object: %v", res.Content)
	}
}

func TestPowerTool(t *testing.T) {
	f := newFixture(t)
	_, url := newTestServer(t, f, Config{PerAccountBurst: 100})
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	ctx := context.Background()

	tools, _ := cs.ListTools(ctx, nil)
	for _, tool := range tools.Tools {
		a := tool.Annotations
		if tool.Name == "power" {
			if a.ReadOnlyHint || a.DestructiveHint == nil || !*a.DestructiveHint {
				t.Errorf("power isn't marked destructive: %+v", a)
			}
		} else if !a.ReadOnlyHint {
			t.Errorf("%s isn't read-only", tool.Name)
		}
	}

	var web pgtype.UUID
	f.admin.QueryRow(ctx, `SELECT o.id FROM deployed_object o JOIN team t ON t.id = o.team_id WHERE t.build_id = $1 AND t.team_number = 1 AND o.object_name = 'web01'`, f.build).Scan(&web)
	out, res := callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String()[:8], "action": "reboot", "force": true, "objects": []string{"web01"}, "team": 1})
	if res.IsError {
		t.Fatal(errText(res))
	}
	want := "/builds/" + f.build.String() + `/power {"action":"reboot","force":true,"target":{"ids":["` + web.String() + `"]}}`
	if len(f.acted) != 1 || f.acted[0] != want || out.Started != 1 {
		t.Fatalf("acted %v, want %s; out %+v", f.acted, want, out)
	}

	// No target at all needs all=true; a filter alone is a target.
	if _, res := callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String(), "action": "stop"}); !res.IsError || !strings.Contains(errText(res), "all=true") {
		t.Fatalf("power with no target: %v", res.Content)
	}
	callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String(), "action": "start", "team": 2, "kind": "host"})
	if last := f.acted[len(f.acted)-1]; !strings.HasSuffix(last, `{"action":"start","force":false,"target":{"kind":"host","team":2}}`) {
		t.Fatalf("filtered power sent %s", last)
	}
	callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String(), "action": "start", "all": true})
	if last := f.acted[len(f.acted)-1]; !strings.HasSuffix(last, `"target":{}}`) {
		t.Fatalf("all=true sent %s", last)
	}

	// Networks, bad actions and unreadable builds never reach the API.
	before := len(f.acted)
	for _, args := range []map[string]any{
		{"build_id": f.build.String(), "action": "start", "kind": "network"},
		{"build_id": f.build.String(), "action": "rebuild", "all": true},
		{"build_id": f.otherBuild.String(), "action": "stop", "all": true},
	} {
		if _, res := callTool[powerOut](t, cs, "power", args); !res.IsError {
			t.Errorf("power %v succeeded", args)
		}
	}
	if len(f.acted) != before {
		t.Fatalf("refused calls reached the API: %v", f.acted[before:])
	}

	// One power call every 10 seconds after a burst of 3.
	if _, res := callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String(), "action": "start", "all": true}); !res.IsError || !strings.Contains(errText(res), "one every 10 seconds") {
		t.Fatalf("fourth power call: want rate limited, got %v", res.Content)
	}
}

// Someone without manage access is refused by the API itself.
func TestPowerToolDenied(t *testing.T) {
	f := newFixture(t)
	f.denyActs = true
	_, url := newTestServer(t, f, Config{})
	cs, err := connect(t, url, f.token)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	_, res := callTool[powerOut](t, cs, "power", map[string]any{"build_id": f.build.String(), "action": "stop", "all": true})
	if !res.IsError || !strings.Contains(errText(res), "needs manage access") {
		t.Fatalf("power without manage: %v", res.Content)
	}
}
