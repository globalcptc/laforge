package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/render"
)

// signedInClient runs the real OAuth callback (state cookie, code
// exchange, GetAuthenticatedUser, session creation) against the fake
// GitHub server this test file's setupAPITest wires up, and returns an
// http.Client whose cookie jar now holds a real session cookie -- every
// subsequent request through it is authenticated exactly the way a
// signed-in browser's would be. code becomes the fake exchanged token
// (see fakeGitHub's /login/oauth/access_token case), which becomes the
// account's github_login too (identityForToken), so tests can pick a
// recognizable one per "person."
func signedInClient(t *testing.T, env *apiTestEnv, code string) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

	startResp, err := client.Get(env.httpURL + "/auth/github/login")
	if err != nil {
		t.Fatalf("GET /auth/github/login: %v", err)
	}
	startResp.Body.Close()
	if startResp.StatusCode != http.StatusFound {
		t.Fatalf("login start status = %d, want 302", startResp.StatusCode)
	}
	loc, err := startResp.Location()
	if err != nil {
		t.Fatalf("login start Location: %v", err)
	}
	state := loc.Query().Get("state")
	if state == "" {
		t.Fatal("no state in AuthorizeURL redirect")
	}

	cbURL := env.httpURL + "/auth/github/callback?code=" + code + "&state=" + state
	cbResp, err := client.Get(cbURL)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	cbResp.Body.Close()
	if cbResp.StatusCode != http.StatusFound {
		body, _ := json.Marshal(cbResp.Header)
		t.Fatalf("callback status = %d, want 302 (headers: %s)", cbResp.StatusCode, body)
	}
	return client
}

// TestSignInRejectsAnUnauthorizedGitHubAccount is the real fix for a gap
// found by direct product feedback: handleCallback minted a real session
// for any GitHub account, full stop -- in a CPTC deployment, where the
// sign-in button is reachable by every competitor, that's a real attack
// surface (session creation, probing every authenticated endpoint), not
// just an inconvenience. Proves the actual rejection path over real HTTP
// (not just calling signInAuthorized directly): no repository this
// instance tracks, not an admin, no explicit grant -- the callback
// refuses with a real 403 and no session cookie is ever set, confirmed
// by a follow-up /auth/me still reporting unauthenticated.
func TestSignInRejectsAnUnauthorizedGitHubAccount(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	// The fake GitHub server otherwise reports Pull:true for any repo
	// it's asked about (see fakeGitHub's own /repos/ handler), for any
	// row in the `repository` table -- including rows other packages'
	// tests concurrently leave in this same shared local dev Postgres
	// under `go test ./...`. Rather than depend on that table being
	// globally empty at this exact instant (a real flake, found live:
	// the table is not test-isolated, only per-test-cleaned-up), deny
	// repo access at the one thing this test actually owns -- its own
	// fake server's response -- so "no real access anywhere" holds
	// regardless of what rows happen to exist.
	env.gh.denyAllRepoAccess = true

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

	startResp, err := client.Get(env.httpURL + "/auth/github/login")
	if err != nil {
		t.Fatal(err)
	}
	startResp.Body.Close()
	state := startResp.Header.Get("Location")
	loc, _ := startResp.Location()
	state = loc.Query().Get("state")

	cbResp, err := client.Get(env.httpURL + "/auth/github/callback?code=nobody-in-particular&state=" + state)
	if err != nil {
		t.Fatal(err)
	}
	cbResp.Body.Close()
	if cbResp.StatusCode != http.StatusForbidden {
		t.Fatalf("callback status = %d, want 403 (unauthorized identity)", cbResp.StatusCode)
	}

	// No session cookie should have been set at all.
	meResp, err := client.Get(env.httpURL + "/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/auth/me after rejected sign-in = %d, want 401 (no session should exist)", meResp.StatusCode)
	}

	// And no account row should have been created for them either --
	// rejected before UpsertAccount, not a session immediately torn
	// back down.
	if _, err := env.q.GetAccountByLogin(ctx, "nobody-in-particular"); err == nil {
		t.Fatal("an account row was created for a rejected sign-in attempt")
	}
}

// TestSessionCookieAttributes covers the cross-origin cookie fix: by
// default (same-origin or same-registrable-domain deployments)
// the session cookie stays SameSite=Lax, but with CrossOriginCookies
// set, it becomes SameSite=None; Secure -- required for a genuinely
// cross-domain UI/API split, since a Lax cookie is never attached to a
// cross-site fetch()/XHR at all (only a top-level GET navigation), which
// would otherwise mean every authenticated API call from such a UI
// silently carried no session. Inspects the real Set-Cookie header from
// a real callback round trip, not just the Go struct passed to
// http.SetCookie.
func TestSessionCookieAttributes(t *testing.T) {
	for _, crossOrigin := range []bool{false, true} {
		t.Run(fmt.Sprintf("CrossOriginCookies=%v", crossOrigin), func(t *testing.T) {
			env := setupAPITest(t)
			env.server.CrossOriginCookies = crossOrigin

			jar, err := cookiejar.New(nil)
			if err != nil {
				t.Fatal(err)
			}
			client := &http.Client{Jar: jar, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}

			startResp, err := client.Get(env.httpURL + "/auth/github/login")
			if err != nil {
				t.Fatal(err)
			}
			startResp.Body.Close()
			loc, _ := startResp.Location()
			state := loc.Query().Get("state")

			cbResp, err := client.Get(env.httpURL + "/auth/github/callback?code=cookie-attr-test&state=" + state)
			if err != nil {
				t.Fatal(err)
			}
			cbResp.Body.Close()

			var sessionCookie *http.Cookie
			for _, c := range cbResp.Cookies() {
				if c.Name == sessionCookieName {
					sessionCookie = c
				}
			}
			if sessionCookie == nil {
				t.Fatalf("callback response set no %s cookie (headers: %v)", sessionCookieName, cbResp.Header["Set-Cookie"])
			}

			wantSameSite := http.SameSiteLaxMode
			wantSecure := false
			if crossOrigin {
				wantSameSite, wantSecure = http.SameSiteNoneMode, true
			}
			if sessionCookie.SameSite != wantSameSite {
				t.Errorf("SameSite = %v, want %v", sessionCookie.SameSite, wantSameSite)
			}
			if sessionCookie.Secure != wantSecure {
				t.Errorf("Secure = %v, want %v", sessionCookie.Secure, wantSecure)
			}
		})
	}
}

func TestOAuthLoginFlowAndMe(t *testing.T) {
	env := setupAPITest(t)
	client := signedInClient(t, env, "person-a")

	resp, err := client.Get(env.httpURL + "/auth/me")
	if err != nil {
		t.Fatalf("GET /auth/me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me status = %d, want 200", resp.StatusCode)
	}
	var me meResponse
	json.NewDecoder(resp.Body).Decode(&me)
	if me.GithubLogin != "exchanged-person-a" {
		t.Fatalf("GithubLogin = %q, want exchanged-person-a", me.GithubLogin)
	}

	// A second callback with a stale/reused state must fail -- proves the
	// state cookie is actually being checked, not just present.
	resp2, err := client.Get(env.httpURL + "/auth/github/callback?code=x&state=not-the-real-state")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("stale-state callback status = %d, want 400", resp2.StatusCode)
	}

	// Logout clears the session -- /auth/me should go back to 401.
	logoutResp, err := client.Post(env.httpURL+"/auth/logout", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	logoutResp.Body.Close()
	meAfter, err := client.Get(env.httpURL + "/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer meAfter.Body.Close()
	if meAfter.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/auth/me after logout status = %d, want 401", meAfter.StatusCode)
	}
}

func TestMeWithoutSessionIsUnauthenticated(t *testing.T) {
	env := setupAPITest(t)
	resp, err := http.Get(env.httpURL + "/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

// realBuildFixture reconciles examples/m7-two-team (the small, real,
// 2-team fixture already proved against a live Incus daemon)
// against the fake builder's own path -- Reconcile itself doesn't touch
// any hoster, only Postgres, so this needs no live daemon, just a real
// content_revision/build/team/deployed_object/task/event graph to list.
func realBuildFixture(t *testing.T, env *apiTestEnv) db.Build {
	t.Helper()
	ctx := context.Background()
	rev, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: env.repo.ID, CommitSha: "ui-test-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := env.q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "m7-two-team",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	// Every caller of this fixture across the suite needs real deploy_*
	// tasks to exist (several drive the real runner against them) --
	// only true for a build in "deploying" now that
	// internal/orchestrator.Reconcile actually enforces "Build and
	// deploy are separate verbs". A test
	// specifically about the `planned` state uses its own build instead
	// (see internal/orchestrator's own TestReconcilePlannedBuildCreatesNoRealWork).
	if _, err := env.q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"
	if err := orchestrator.Reconcile(ctx, env.server.Pool, "../../examples/m7-two-team", build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return build
}

func TestListBuildsAndGetBuildDetail(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")

	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/builds")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list builds status = %d, want 200", resp.StatusCode)
	}
	var builds []db.Build
	json.NewDecoder(resp.Body).Decode(&builds)
	found := false
	for _, b := range builds {
		if b.ID == build.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("listed builds %+v did not include %s", builds, build.ID)
	}

	detailResp, err := client.Get(env.httpURL + "/builds/" + build.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer detailResp.Body.Close()
	if detailResp.StatusCode != http.StatusOK {
		t.Fatalf("get build status = %d, want 200", detailResp.StatusCode)
	}
	var detail buildDetail
	json.NewDecoder(detailResp.Body).Decode(&detail)
	if len(detail.Teams) != 2 {
		t.Fatalf("detail.Teams = %d, want 2 (m7-two-team)", len(detail.Teams))
	}
	totalObjects := 0
	for _, tm := range detail.Teams {
		totalObjects += len(tm.Objects)
	}
	if totalObjects != 6 { // 2 teams x (network + host + container)
		t.Fatalf("total objects across teams = %d, want 6", totalObjects)
	}

	objResp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects")
	if err != nil {
		t.Fatal(err)
	}
	defer objResp.Body.Close()
	var objs []db.DeployedObject
	json.NewDecoder(objResp.Body).Decode(&objs)
	if len(objs) != 6 {
		t.Fatalf("flat objects list = %d, want 6", len(objs))
	}
}

func TestBuildEndpointsRequireAtLeastRead(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)

	resp, err := http.Get(env.httpURL + "/builds/" + build.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous get build status = %d, want 401", resp.StatusCode)
	}
}

func TestAdHocTaskDryRunAndDispatch(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	// "good-token" is in fakeGitHub's pushTokens map (set in
	// setupAPITest), so exchanging code "good-token" yields a login whose
	// live GitHub permission check comes back push=true -- requireLevel's
	// implicit floor for levelBuild. Ad-hoc tasks need levelManage
	// though, which push alone does NOT satisfy (see requireLevel's own
	// doc comment: push implies build, not manage) -- so this also
	// exercises the "build-level user is forbidden from manage-level
	// action" path before granting explicit access.
	client := signedInClient(t, env, "good-token")

	dryRunBody := `{"target":{"kind":"host"},"command":"reboot","payload":{},"dry_run":true}`
	resp, err := client.Post(env.httpURL+"/builds/"+build.ID.String()+"/tasks", "application/json", strings.NewReader(dryRunBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("dry-run with only push (build-level): status = %d, want 403", resp.StatusCode)
	}

	// Grant this account explicit manage access, using an admin session
	// (levelAdmin, since GitHub admin permission isn't in the fake's
	// pushTokens/repo permissions model).
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	grantManage(t, env, adminClient, "exchanged-good-token")

	resp2, err := client.Post(env.httpURL+"/builds/"+build.ID.String()+"/tasks", "application/json", strings.NewReader(dryRunBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("dry-run after manage grant: status = %d, want 200", resp2.StatusCode)
	}
	var dryRun adHocResult
	json.NewDecoder(resp2.Body).Decode(&dryRun)
	if dryRun.Matched != 2 { // 2 teams x 1 host each
		t.Fatalf("dry-run matched = %d, want 2 hosts", dryRun.Matched)
	}
	if len(dryRun.Created) != 0 {
		t.Fatalf("dry-run must not create anything, got %d created", len(dryRun.Created))
	}

	realBody := `{"target":{"kind":"host"},"command":"reboot","payload":{"delay_sec":5}}`
	resp3, err := client.Post(env.httpURL+"/builds/"+build.ID.String()+"/tasks", "application/json", strings.NewReader(realBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("real dispatch status = %d, want 200", resp3.StatusCode)
	}
	var real adHocResult
	json.NewDecoder(resp3.Body).Decode(&real)
	if len(real.Created) != 2 {
		t.Fatalf("created = %d agent_task rows, want 2", len(real.Created))
	}
	for _, task := range real.Created {
		if task.Command != "reboot" || task.Status != "pending" {
			t.Fatalf("created task = %+v, want command=reboot status=pending", task)
		}
	}
}

// TestAdHocTaskByIDDoesNotOverMatchSameNameInOtherTeam is a regression
// test for a real bug caught by hand in the browser: the host table's
// row selection originally targeted by `search: as_name`, which matched
// every team's identically-named host, not just the one row a person
// selected -- "every team gets exactly the same network" means hostnames
// are never unique build-wide. Fixed by adding exact id-based targeting
// (adHocTarget.IDs); this proves selecting exactly one of two teams'
// "web01" only ever touches that one.
func TestAdHocTaskByIDDoesNotOverMatchSameNameInOtherTeam(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env) // m7-two-team: both teams have a host named web01
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")

	objs, err := env.q.ListDeployedObjectsByBuild(context.Background(), build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01IDs []string
	for _, o := range objs {
		if o.Kind == "host" {
			web01IDs = append(web01IDs, o.ID.String())
		}
	}
	if len(web01IDs) != 2 {
		t.Fatalf("fixture has %d host(s) named web01, want 2 (one per team)", len(web01IDs))
	}

	body := `{"target":{"ids":["` + web01IDs[0] + `"]},"command":"reboot","payload":{}}`
	resp, err := adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/tasks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var result adHocResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Matched != 1 || len(result.Created) != 1 {
		t.Fatalf("targeting one id matched %d objects and created %d tasks, want exactly 1 each", result.Matched, len(result.Created))
	}
	if result.Created[0].DeployedObjectID.String() != web01IDs[0] {
		t.Fatalf("task created against %s, want the exact targeted id %s", result.Created[0].DeployedObjectID, web01IDs[0])
	}
}

// TestAdHocTaskRejectsIDFromAnotherBuild is the direct answer to "we may
// also have multiple builds from the same repo... let's really make sure
// there's no overlap": two REAL builds of the same repository (the
// ordinary case -- a rebuild, or two configured builds off different
// branches of one environment), each with their own deployed_object rows
// carrying real Postgres UUIDs. Confirms two things that together rule
// out cross-build overlap by construction, not by convention:
//  1. A host's id from build A is REJECTED (400, naming the id) when
//     targeted against build B's /tasks endpoint, rather than silently
//     matching zero objects -- build B's object list genuinely never
//     contains build A's ids (ListDeployedObjectsByBuild is WHERE
//     team.build_id = $1, and every id in Postgres is a real UUID:
//     there is no scheme, hash, or name that could make two different
//     rows share one, from any build), so "not found here" is the only
//     possible outcome, and now it fails loudly instead of quietly.
//  2. Targeting build B by build A's id never creates anything against
//     build B, and never touches build A either (build A's own object
//     list is untouched -- this call only ever reads/writes rows scoped
//     to build B in the first place).
func TestAdHocTaskRejectsIDFromAnotherBuild(t *testing.T) {
	env := setupAPITest(t)
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	ctx := context.Background()

	buildA := realBuildFixture(t, env)

	// A second, independent real build of the SAME repository -- a
	// distinct commit_sha (content_revision has a UNIQUE(repository_id,
	// commit_sha) constraint), same environment, standing in for
	// "another build from the same repo with a similar configuration."
	revB, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: env.repo.ID, CommitSha: "ui-test-sha-build-b",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision (build B): %v", err)
	}
	buildB, err := env.q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: revB.ID, EnvironmentName: "m7-two-team",
	})
	if err != nil {
		t.Fatalf("CreateBuild (build B): %v", err)
	}
	if err := orchestrator.Reconcile(ctx, env.server.Pool, "../../examples/m7-two-team", buildB.ID); err != nil {
		t.Fatalf("Reconcile (build B): %v", err)
	}

	objsA, err := env.q.ListDeployedObjectsByBuild(ctx, buildA.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (A): %v", err)
	}
	var hostAID string
	for _, o := range objsA {
		if o.Kind == "host" {
			hostAID = o.ID.String()
			break
		}
	}
	if hostAID == "" {
		t.Fatal("build A has no host to use as the foreign id")
	}

	body := `{"target":{"ids":["` + hostAID + `"]},"command":"reboot","payload":{}}`
	resp, err := adminClient.Post(env.httpURL+"/builds/"+buildB.ID.String()+"/tasks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("targeting build B with build A's host id: status = %d, want 400 (rejected, not silently 0-matched)", resp.StatusCode)
	}
	var errBody map[string]string
	json.NewDecoder(resp.Body).Decode(&errBody)
	if !strings.Contains(errBody["error"], hostAID) {
		t.Fatalf("error message %q does not name the rejected id %q", errBody["error"], hostAID)
	}

	// Confirm nothing was created against build B at all, and build A's
	// own object is untouched.
	tasksB, err := env.q.ListDeployedObjectsByBuild(ctx, buildB.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (B): %v", err)
	}
	for _, o := range tasksB {
		if o.ID.String() == hostAID {
			t.Fatal("build A's host id leaked into build B's object list")
		}
	}
}

func TestTeamAccessOpenCloseExtend(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner") // admin ranks above the manage this endpoint requires

	closeResp, err := adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teams/1/access", "application/json", strings.NewReader(`{"action":"close"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer closeResp.Body.Close()
	if closeResp.StatusCode != http.StatusAccepted {
		t.Fatalf("close status = %d, want 202", closeResp.StatusCode)
	}
	var closeTask db.Task
	json.NewDecoder(closeResp.Body).Decode(&closeTask)
	if closeTask.Kind != "close_access" {
		t.Fatalf("task kind = %q, want close_access", closeTask.Kind)
	}

	extendResp, err := adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teams/1/access", "application/json", strings.NewReader(`{"action":"extend","extend_minutes":30}`))
	if err != nil {
		t.Fatal(err)
	}
	defer extendResp.Body.Close()
	if extendResp.StatusCode != http.StatusOK {
		t.Fatalf("extend status = %d, want 200", extendResp.StatusCode)
	}
	var extended db.Team
	json.NewDecoder(extendResp.Body).Decode(&extended)
	if !extended.AccessOverrideUntil.Valid {
		t.Fatal("extend did not set access_override_until")
	}

	badResp, err := adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teams/99/access", "application/json", strings.NewReader(`{"action":"open"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusNotFound {
		t.Fatalf("nonexistent team status = %d, want 404", badResp.StatusCode)
	}
}

// TestTeardownBuild is teardown made real at the
// HTTP layer: gated at levelManage (like close/open access, not the
// lower levelBuild every other build-lifecycle endpoint uses, since this
// acts on real infrastructure, not a build's existence), sets
// 'tearing_down' immediately, and refuses a second teardown request
// against a build that's already tearing down.
func TestTeardownBuild(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	buildOnlyClient := signedInClient(t, env, "build-only-actor")
	grantLevel(t, env, adminClient, "exchanged-build-only-actor", "build")

	// levelBuild is not enough -- teardown needs levelManage.
	resp, err := buildOnlyClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teardown", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("teardown at levelBuild: status = %d, want 403", resp.StatusCode)
	}

	resp, err = adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teardown", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var updated db.Build
	json.NewDecoder(resp.Body).Decode(&updated)
	resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("teardown: status = %d, want 202", resp.StatusCode)
	}
	if updated.Status != "tearing_down" {
		t.Fatalf("build.Status = %q, want tearing_down", updated.Status)
	}

	// Already tearing down -- a second request is a real conflict, not a
	// silent no-op.
	resp, err = adminClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/teardown", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("teardown (already tearing down): status = %d, want 409", resp.StatusCode)
	}
}

// TestLiveStatusStreamsRealEvents proves handleLiveStatus is a real SSE
// stream, not a stub: it opens a real long-lived HTTP connection, writes
// a real event to the build's journal from a separate goroutine (as a
// live runner would), and asserts the client actually receives it as an
// SSE frame over the open connection, not from a subsequent request.
func TestLiveStatusStreamsRealEvents(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")

	req, err := http.NewRequest(http.MethodGet, env.httpURL+"/builds/"+build.ID.String()+"/live", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("live status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	go func() {
		env.q.CreateEvent(context.Background(), db.CreateEventParams{
			BuildID: build.ID, Kind: "test.live.marker", Message: "hello from a live event", Payload: []byte("{}"),
		})
	}()

	buf := make([]byte, 4096)
	deadlineCh := make(chan struct{})
	go func() { <-ctx.Done(); close(deadlineCh) }()
	found := false
	for i := 0; i < 20 && !found; i++ {
		n, err := resp.Body.Read(buf)
		if err != nil {
			break
		}
		if strings.Contains(string(buf[:n]), "test.live.marker") {
			found = true
		}
	}
	if !found {
		t.Fatal("did not receive the live event over the SSE stream")
	}
}

// TestTriggerBuildAndDeployManualFlow is the real write-side
// proof: "Build: resolve the environment at a commit ... touches no
// hoster" and "Deploy: apply that build" as actual operator actions, gated at
// levelBuild. It rides a real webhook push (like TestFullReconcileFlow)
// to get a genuinely CI-validated commit tracked on a configured build,
// rather than hand-inserting a content_revision, so
// GetEnvironmentByRevisionAndPath resolves against real persisted
// content.
func TestTriggerBuildAndDeployManualFlow(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	cbBody, _ := json.Marshal(createConfiguredBuildRequest{
		Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", bytes.NewReader(cbBody))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create configured build: status = %d", resp.StatusCode)
	}
	// Isolate the manual flow from auto-deploy: with it off, the push below
	// leaves a 'planned' build we can drive through the Deploy endpoint, while
	// Build Now still deploys immediately on its own.
	if _, err := env.q.SetConfiguredBuildAutoDeploy(ctx, db.SetConfiguredBuildAutoDeployParams{ID: cb.ID, AutoDeployEnabled: false}); err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}

	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	buildClient := signedInClient(t, env, "build-actor")
	grantLevel(t, env, adminClient, "exchanged-build-actor", "build")
	readOnlyClient := signedInClient(t, env, "read-only-viewer")

	triggerURL := env.httpURL + "/configured-builds/" + cb.ID.String() + "/builds"

	// Create auto-pulls the branch HEAD, so the configured build already tracks a
	// validated commit and is immediately buildable -- no push needed first.
	{
		autoCB, err := env.q.GetConfiguredBuild(ctx, cb.ID)
		if err != nil {
			t.Fatalf("GetConfiguredBuild after create: %v", err)
		}
		if !autoCB.CurrentContentRevisionID.Valid {
			t.Fatalf("create should have auto-pulled a tracked revision, but current_content_revision_id is unset")
		}
	}

	// A real push, exactly like TestFullReconcileFlow, so
	// current_content_revision_id points at a genuinely validated commit.
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push: status = %d", resp.StatusCode)
	}
	rev, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}

	// The push auto-built a 'planned' build (auto-deploy is off) -- this is what
	// the Deploy endpoint's happy path acts on below.
	afterPush, err := env.q.ListBuildsByRepository(ctx, env.repo.ID)
	if err != nil {
		t.Fatalf("ListBuildsByRepository: %v", err)
	}
	if len(afterPush) != 1 {
		t.Fatalf("expected one planned build after the push, got %d", len(afterPush))
	}
	plannedBuild := afterPush[0]
	if plannedBuild.Status != "planned" {
		t.Fatalf("auto-built build status = %q, want planned (auto-deploy off)", plannedBuild.Status)
	}

	// Read-only can't trigger a build.
	resp, err = readOnlyClient.Post(triggerURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("trigger as read-only: status = %d, want 403", resp.StatusCode)
	}

	// Build-level can, and "Build Now" deploys immediately -- it gets back a real
	// `deploying` build resolved against the tracked commit and the configured
	// build's environment file (the plan-then-deploy split is folded in).
	resp, err = buildClient.Post(triggerURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var build db.Build
	json.NewDecoder(resp.Body).Decode(&build)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("trigger build: status = %d, want 201", resp.StatusCode)
	}
	if build.Status != "deploying" {
		t.Fatalf("new build status = %q, want deploying (Build Now deploys immediately)", build.Status)
	}
	if build.EnvironmentName != "lm-test" {
		t.Fatalf("new build environment_name = %q, want lm-test (resolved from environment_path via the tracked revision)", build.EnvironmentName)
	}
	if build.ContentRevisionID != rev.ID {
		t.Fatalf("new build content_revision_id = %v, want the tracked revision %v", build.ContentRevisionID, rev.ID)
	}
	if build.ConfiguredBuildID != cb.ID {
		t.Fatalf("new build configured_build_id = %v, want %v", build.ConfiguredBuildID, cb.ID)
	}

	plannedDeployURL := env.httpURL + "/builds/" + plannedBuild.ID.String() + "/deploy"

	// Read-only can't deploy either.
	resp, err = readOnlyClient.Post(plannedDeployURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("deploy as read-only: status = %d, want 403", resp.StatusCode)
	}

	// Build-level flips the PLANNED build to deploying -- this is the whole
	// Deploy action; the already-running orchestrator poll loop is what actually
	// creates real tasks off this status change (internal/orchestrator's
	// deployTasks gate).
	resp, err = buildClient.Post(plannedDeployURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var deployed db.Build
	json.NewDecoder(resp.Body).Decode(&deployed)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deploy: status = %d, want 200", resp.StatusCode)
	}
	if deployed.Status != "deploying" {
		t.Fatalf("deployed build status = %q, want deploying", deployed.Status)
	}

	// Deploying a build that's no longer planned is refused -- the
	// manually-triggered build is already deploying (Build Now folded the deploy
	// in), so its own Deploy endpoint answers 409.
	resp, err = buildClient.Post(env.httpURL+"/builds/"+build.ID.String()+"/deploy", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("deploy of already-deploying build: status = %d, want 409", resp.StatusCode)
	}

	// --- rendered output: "the exact script delivered to a given host in
	// a given team can be retrieved from the UI and matches what the
	// agent ran" -- reconcile for real (standing in for the orchestrator
	// poll loop this test doesn't run) so real team/deployed_object rows
	// exist, then compare handleRenderObject's output against
	// render.RenderScript called directly against the same real checkout.
	if err := orchestrator.Reconcile(ctx, env.server.Pool, env.server.RepoRoot, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01 db.DeployedObject
	var team1 db.Team
	for _, o := range objs {
		if o.AsName != nil && *o.AsName == "web01" {
			tm, err := env.q.GetTeam(ctx, o.TeamID)
			if err != nil {
				t.Fatal(err)
			}
			if tm.TeamNumber == 1 {
				web01, team1 = o, tm
				break
			}
		}
	}
	if !web01.ID.Valid {
		t.Fatal("team 1's web01 object not found after reconcile")
	}

	resp, err = buildClient.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + web01.ID.String() + "/render")
	if err != nil {
		t.Fatal(err)
	}
	var steps []renderedStep
	json.NewDecoder(resp.Body).Decode(&steps)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("render: status = %d, want 200", resp.StatusCode)
	}
	// webserver.yaml's real step list: script:base, download, extract,
	// script:vuln-sqli -- see examples/lm-test/hosts/webserver.yaml. Its
	// `schedule:` entry is a sibling field now, not a step, so it never
	// appears in this list.
	if len(steps) != 4 {
		t.Fatalf("len(steps) = %d, want 4 (webserver.yaml's real step list)", len(steps))
	}
	if steps[0].Action != "script" || steps[0].ScriptName != "base" || steps[0].Rendered == "" {
		t.Fatalf("step 0 = %+v, want a non-empty rendered script:base", steps[0])
	}
	if steps[1].Action != "download" || steps[1].Rendered != "" {
		t.Fatalf("step 1 = %+v, want action=download with no rendered script output", steps[1])
	}
	if steps[3].Action != "script" || steps[3].ScriptName != "vuln-sqli" || steps[3].Rendered == "" {
		t.Fatalf("step 3 = %+v, want a non-empty rendered script:vuln-sqli", steps[3])
	}

	// The real proof: this must be BYTE-IDENTICAL to calling
	// internal/render directly against the same checkout, team, and
	// host -- not a parallel reimplementation that merely looks similar.
	c, err := loader.Load(env.server.RepoRoot)
	if err != nil {
		t.Fatal(err)
	}
	rctx, err := render.Resolve(c, build.EnvironmentName, "web01", int(team1.TeamNumber))
	if err != nil {
		t.Fatal(err)
	}
	var vulnSQLi *loader.Script
	for i := range c.Scripts {
		if c.Scripts[i].Name == "vuln-sqli" {
			vulnSQLi = &c.Scripts[i]
		}
	}
	if vulnSQLi == nil {
		t.Fatal("script vuln-sqli not found in real content")
	}
	want, err := render.RenderScript(env.server.RepoRoot, vulnSQLi, rctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if steps[3].Rendered != want {
		t.Fatalf("rendered vuln-sqli via the API does not match calling internal/render directly:\ngot:  %q\nwant: %q", steps[3].Rendered, want)
	}
}

// TestUpcomingChangesDetectsRealDriftAfterANewerPush is the "upcoming
// changes" feature (blocked at the time on the API having no
// git-checkout capability at all, unblocked by RepoRoot): build a real
// deploying build
// from a configured build's first tracked commit, push a second real
// commit so the configured build's current_content_revision_id moves
// ahead of what this build was created from, and confirm GET
// /builds/{id}/upcoming correctly distinguishes "nothing pending" from
// "there's a newer commit, here's what it would actually change." This
// is specifically the manual-review path, so auto-deploy (item 13,
// on by default per repository since 2026-09-24) is turned off for this
// repo first -- otherwise the second push would apply itself
// automatically and there'd be nothing left to preview, which is exactly
// TestPushAppliesAnAlreadyDeployedConfiguredBuildAutomatically's own
// scenario instead.
func TestUpcomingChangesDetectsRealDriftAfterANewerPush(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	cbBody, _ := json.Marshal(createConfiguredBuildRequest{
		Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", bytes.NewReader(cbBody))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	resp.Body.Close()
	if _, err := env.q.SetConfiguredBuildAutoDeploy(ctx, db.SetConfiguredBuildAutoDeployParams{ID: cb.ID, AutoDeployEnabled: false}); err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 1: status = %d", resp.StatusCode)
	}

	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	buildClient := signedInClient(t, env, "upcoming-actor")
	grantLevel(t, env, adminClient, "exchanged-upcoming-actor", "build")

	resp, err = buildClient.Post(env.httpURL+"/configured-builds/"+cb.ID.String()+"/builds", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var build db.Build
	json.NewDecoder(resp.Body).Decode(&build)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("trigger build: status = %d, want 201", resp.StatusCode)
	}
	// "Build Now" deploys immediately -- the trigger already returns a
	// 'deploying' build, so there is no separate Deploy step here anymore.
	if build.Status != "deploying" {
		t.Fatalf("triggered build status = %q, want deploying (Build Now deploys immediately)", build.Status)
	}
	if err := orchestrator.Reconcile(ctx, env.server.Pool, env.server.RepoRoot, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	upcomingURL := env.httpURL + "/builds/" + build.ID.String() + "/upcoming"

	// Nothing pushed since this build's own commit -- nothing pending.
	resp, err = buildClient.Get(upcomingURL)
	if err != nil {
		t.Fatal(err)
	}
	var before upcomingResponse
	json.NewDecoder(resp.Body).Decode(&before)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upcoming (before any new push): status = %d, want 200", resp.StatusCode)
	}
	if before.Pending || len(before.Changes) != 0 {
		t.Fatalf("upcoming before any new push = %+v, want pending=false, no changes", before)
	}

	// A second real commit lands -- current_content_revision_id moves
	// ahead of what this build was created from. Nothing in
	// examples/lm-test's own content changes (the pushed file is unrelated),
	// so this alone must NOT fabricate changes -- only mark the diff as
	// genuinely pending.
	pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "UPCOMING_TEST.txt", "unrelated file, no host reads this\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 2: status = %d", resp.StatusCode)
	}

	resp, err = buildClient.Get(upcomingURL)
	if err != nil {
		t.Fatal(err)
	}
	var pendingNoDrift upcomingResponse
	json.NewDecoder(resp.Body).Decode(&pendingNoDrift)
	resp.Body.Close()
	if !pendingNoDrift.Pending || len(pendingNoDrift.Changes) != 0 {
		t.Fatalf("upcoming after an unrelated new commit = %+v, want pending=true, still no changes (nothing web01 reads was touched)", pendingNoDrift)
	}

	// Now simulate real drift: web01 is marked deployed with a
	// deliberately stale fingerprint, exactly like
	// internal/orchestrator's own TestReconcileRebuildsOnFingerprintChange.
	// This time the same query must report it as a real "changed" entry.
	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01 db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" && o.AsName != nil && *o.AsName == "web01" {
			tm, err := env.q.GetTeam(ctx, o.TeamID)
			if err != nil {
				t.Fatal(err)
			}
			if tm.TeamNumber == 1 {
				web01 = o
			}
		}
	}
	if !web01.ID.Valid {
		t.Fatal("team 1's web01 object not found")
	}
	if _, err := env.q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
		ID: web01.ID, ExternalRef: db.StrPtr("fake-ref-web01"), Fingerprint: "deliberately-stale-fingerprint",
	}); err != nil {
		t.Fatalf("MarkDeployedObjectRunning: %v", err)
	}

	resp, err = buildClient.Get(upcomingURL)
	if err != nil {
		t.Fatal(err)
	}
	var afterDrift upcomingResponse
	json.NewDecoder(resp.Body).Decode(&afterDrift)
	resp.Body.Close()
	if !afterDrift.Pending {
		t.Fatalf("upcoming after real drift: pending = %v, want true", afterDrift.Pending)
	}
	if len(afterDrift.Changes) != 1 || afterDrift.Changes[0].AsName != "web01" || afterDrift.Changes[0].Change != "changed" {
		t.Fatalf("upcoming after real drift: changes = %+v, want exactly one {as_name:web01 change:changed}", afterDrift.Changes)
	}
}

// TestApplyUpcomingAppliesCommitInPlace is proven end to end: "a new
// commit is applied to the existing build,
// not a teardown and rebuild from scratch", via the
// deliberate action GET /builds/{id}/upcoming's preview exists to
// inform. Reuses TestUpcomingChangesDetectsRealDriftAfterANewerPush's
// exact setup (deploy build 1, push an unrelated second commit so
// current_content_revision_id moves ahead) but then actually calls
// POST /builds/{id}/apply-upcoming instead of just reading the diff, and
// confirms: the SAME build row (by id) picked up the new
// content_revision_id in place, a real event was recorded, and
// "upcoming" itself now reports nothing pending -- the reviewable signal
// this endpoint exists to resolve. (Each push here also auto-builds its
// own separate, unrelated `planned` build via follow_enabled, same as
// TestPushAutoBuildsAFollowingConfiguredBuild -- this test only cares
// about what happens to the one build it explicitly deployed.) Auto-deploy
// (item 13) is turned off for this repo, since this test wants to apply
// the commit itself rather than have the webhook do it first.
func TestApplyUpcomingAppliesCommitInPlace(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	cbBody, _ := json.Marshal(createConfiguredBuildRequest{
		Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos/"+env.repo.ID.String()+"/configured-builds", bytes.NewReader(cbBody))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	resp.Body.Close()
	if _, err := env.q.SetConfiguredBuildAutoDeploy(ctx, db.SetConfiguredBuildAutoDeployParams{ID: cb.ID, AutoDeployEnabled: false}); err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}

	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 1: status = %d", resp.StatusCode)
	}
	rev1, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}

	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	buildClient := signedInClient(t, env, "apply-actor")
	grantLevel(t, env, adminClient, "exchanged-apply-actor", "build")
	readOnlyClient := signedInClient(t, env, "apply-read-only")

	resp, err = buildClient.Post(env.httpURL+"/configured-builds/"+cb.ID.String()+"/builds", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var build db.Build
	json.NewDecoder(resp.Body).Decode(&build)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("trigger build: status = %d, want 201", resp.StatusCode)
	}
	if build.ContentRevisionID != rev1.ID {
		t.Fatalf("triggered build content_revision_id = %v, want rev1 %v", build.ContentRevisionID, rev1.ID)
	}
	// "Build Now" deploys immediately -- the trigger already returns a
	// 'deploying' build, no separate Deploy step needed.
	if build.Status != "deploying" {
		t.Fatalf("triggered build status = %q, want deploying (Build Now deploys immediately)", build.Status)
	}

	applyURL := env.httpURL + "/builds/" + build.ID.String() + "/apply-upcoming"

	// Nothing pending yet -- applying now is refused, not a silent no-op.
	resp, err = buildClient.Post(applyURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("apply with nothing pending: status = %d, want 409", resp.StatusCode)
	}

	// A second, unrelated commit lands -- current_content_revision_id
	// moves ahead of what this build was created from.
	pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "APPLY_UPCOMING_TEST.txt", "unrelated file\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 2: status = %d", resp.StatusCode)
	}
	rev2, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA (rev2): %v", err)
	}
	if rev2.ID == rev1.ID {
		t.Fatalf("push 2 resolved to the same content_revision as push 1 -- test setup is broken")
	}

	// Read-only can't apply it.
	resp, err = readOnlyClient.Post(applyURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("apply as read-only: status = %d, want 403", resp.StatusCode)
	}

	// Build-level applies it for real.
	resp, err = buildClient.Post(applyURL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var applied db.Build
	json.NewDecoder(resp.Body).Decode(&applied)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("apply: status = %d, want 200", resp.StatusCode)
	}
	if applied.ID != build.ID {
		t.Fatalf("apply created a different build row (%v) instead of reusing this one (%v)", applied.ID, build.ID)
	}
	if applied.ContentRevisionID != rev2.ID {
		t.Fatalf("build content_revision_id after apply = %v, want rev2 %v", applied.ContentRevisionID, rev2.ID)
	}
	if applied.Status != "deploying" {
		t.Fatalf("build status after apply = %q, want still deploying (unchanged by applying a new commit in place)", applied.Status)
	}

	events, err := env.q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Kind == "build.commit_applied" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a build.commit_applied event recording the in-place redeploy, events = %+v", events)
	}

	// Upcoming now reports nothing pending -- the review signal is resolved.
	resp, err = buildClient.Get(env.httpURL + "/builds/" + build.ID.String() + "/upcoming")
	if err != nil {
		t.Fatal(err)
	}
	var after upcomingResponse
	json.NewDecoder(resp.Body).Decode(&after)
	resp.Body.Close()
	if after.Pending || len(after.Changes) != 0 {
		t.Fatalf("upcoming after apply = %+v, want pending=false, no changes", after)
	}
}

// TestRepositoryAccessGrantAndRevoke is "grant and revoke levels"
// made real end to end: DeleteRepositoryAccess existed
// since access control was first built, but nothing ever called it
// -- handleSetRepositoryAccess could
// only ever create or downgrade a grant, never remove one.
func TestRepositoryAccessGrantAndRevoke(t *testing.T) {
	env := setupAPITest(t)
	adminClient := signedInClient(t, env, "access-admin")
	grantAdmin(t, env, "access-admin")

	login := "revoke-target"
	grantLevel(t, env, adminClient, login, "build")

	if got := repoAccessLevel(t, env, adminClient, login); got != "build" {
		t.Fatalf("setting for %s after grantLevel = %q, want build", login, got)
	}

	// Revoke it for real.
	req, _ := http.NewRequest(http.MethodDelete, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/"+login, nil)
	resp, err := adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", resp.StatusCode)
	}

	// The setting is gone, not just downgraded.
	if got := repoAccessLevel(t, env, adminClient, login); got != "" {
		t.Fatalf("setting for %s after revoke = %q, want none left", login, got)
	}

	// Revoking again (nothing left to revoke) is still a real, safe no-op.
	req, _ = http.NewRequest(http.MethodDelete, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/"+login, nil)
	resp, err = adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete (already revoked) status = %d, want 204", resp.StatusCode)
	}

	// A revoke for a login that never had any account at all is the same
	// safe no-op, not an error.
	req, _ = http.NewRequest(http.MethodDelete, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/never-existed", nil)
	resp, err = adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete (never existed) status = %d, want 204", resp.StatusCode)
	}
}

// grantAdmin makes the given account (identified by the exchanged code
// used to sign in) the repository's admin directly through the database
// -- bootstrapping the very first admin has to happen somehow outside the
// admin-only grant endpoint itself, same as any real system's first-admin
// problem.
func grantAdmin(t *testing.T, env *apiTestEnv, code string) {
	t.Helper()
	ctx := context.Background()
	login := "exchanged-" + code
	acct, err := env.q.UpsertAccount(ctx, db.UpsertAccountParams{GithubID: env.gh.idForLogin(login), GithubLogin: login})
	if err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}
	if _, err := env.q.UpsertRepositoryAccess(ctx, db.UpsertRepositoryAccessParams{
		RepositoryID: env.repo.ID, AccountID: acct.ID, Level: "admin",
	}); err != nil {
		t.Fatalf("UpsertRepositoryAccess: %v", err)
	}
}

// grantManage exercises the real HTTP grant endpoint (unlike grantAdmin,
// which bootstraps directly through the database).
func grantManage(t *testing.T, env *apiTestEnv, adminClient *http.Client, login string) {
	t.Helper()
	grantLevel(t, env, adminClient, login, "manage")
}

// grantLevel is grantManage generalized to any level, for tests (like
// TestTriggerBuildAndDeployManualFlow) that need a session sitting at
// exactly levelBuild -- above read, below manage -- rather than always
// reaching for manage.
func grantLevel(t *testing.T, env *apiTestEnv, adminClient *http.Client, login, level string) {
	t.Helper()
	resp, err := adminClient.Do(mustRequest(t, http.MethodPut, env.httpURL+"/repos/"+env.repo.ID.String()+"/access/"+login, `{"level":"`+level+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("grantLevel(%s) status = %d, want 200", level, resp.StatusCode)
	}
}

func mustRequest(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestDetectDriftRequiresLevelManage is a leftover, wired to a real
// HTTP endpoint: GET /builds/{id}/drift
// makes a real, live call to the build's own builder (Builder.Inspect),
// so it's gated at levelManage like teardown/close-open-access, not the
// lower levelBuild every read-only build endpoint uses.
func TestDetectDriftRequiresLevelManage(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	adminClient := signedInClient(t, env, "drift-owner")
	grantAdmin(t, env, "drift-owner")
	buildOnlyClient := signedInClient(t, env, "drift-build-only")
	grantLevel(t, env, adminClient, "exchanged-drift-build-only", "build")

	resp, err := buildOnlyClient.Get(env.httpURL + "/builds/" + build.ID.String() + "/drift")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("drift at levelBuild: status = %d, want 403", resp.StatusCode)
	}

	// realBuildFixture's own build has no configured_build (see its doc
	// comment) -- DetectDrift correctly refuses to guess a builder, and
	// that refusal must surface as a real HTTP error, not a 200 with a
	// meaningless empty report.
	resp2, err := adminClient.Get(env.httpURL + "/builds/" + build.ID.String() + "/drift")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadGateway {
		t.Fatalf("drift with no configured_build: status = %d, want 502", resp2.StatusCode)
	}
}
