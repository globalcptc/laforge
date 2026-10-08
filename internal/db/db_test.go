package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbTestConnString returns the connection string for a real local Postgres
// to test against, or "" to skip -- this package talks to Postgres and
// nothing about that is meaningfully testable without one. Defaults to the
// same laforge_dev database used throughout development
// (goose-migrated, see internal/db/schema-note.md for how schema.sql
// itself was produced from it) so `go test ./...` works out of the box on
// a machine that already has it, matching how the loader tests
// run straight against examples/lm-test with no flags needed.
func dbTestConnString() string {
	if v := os.Getenv("LAFORGE_TEST_DATABASE_URL"); v != "" {
		return v
	}
	if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err == nil {
		return "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
	}
	return ""
}

// TestRoundTrip exercises every sqlc-generated query this package has
// against a real database: repository -> content_revision -> the six
// child tables -> configured_build, then every read path, then verifies
// ON DELETE CASCADE actually cleans up when the repository is removed
// (rather than assuming the migration's cascade clauses do what they say).
func TestRoundTrip(t *testing.T) {
	conn := dbTestConnString()
	if conn == "" {
		t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	}
	ctx := context.Background()
	q, pool, err := Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Registered first, so among t.Cleanup callbacks (which all run after
	// every plain `defer` in this function has already fired) this one
	// runs last -- the cascade-check cleanup registered further down
	// needs the pool still open when it runs.
	t.Cleanup(func() { pool.Close() })

	repo, err := q.CreateRepository(ctx, CreateRepositoryParams{
		GithubOwner: "laforge-test-owner",
		GithubRepo:  "laforge-test-repo-roundtrip",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	defer func() {
		// Deleting the repository must cascade away every row this test
		// creates -- if it doesn't, later runs collide on the UNIQUE
		// constraints and fail loudly rather than silently leaking rows.
		if _, err := pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", repo.ID); err != nil {
			t.Errorf("cleanup: delete repository: %v", err)
		}
	}()

	got, err := q.GetRepositoryByOwnerRepo(ctx, GetRepositoryByOwnerRepoParams{
		GithubOwner: "laforge-test-owner",
		GithubRepo:  "laforge-test-repo-roundtrip",
	})
	if err != nil {
		t.Fatalf("GetRepositoryByOwnerRepo: %v", err)
	}
	if got.ID != repo.ID {
		t.Fatalf("GetRepositoryByOwnerRepo returned a different row: %v vs %v", got.ID, repo.ID)
	}

	list, err := q.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	found := false
	for _, r := range list {
		if r.ID == repo.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("ListRepositories did not include the repository just created")
	}

	rev, err := q.CreateContentRevision(ctx, CreateContentRevisionParams{
		RepositoryID: repo.ID,
		CommitSha:    "deadbeef",
		Ref:          strPtr("refs/heads/main"),
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}

	emptyArr := []byte(`[]`)
	emptyObj := []byte(`{}`)

	env, err := q.CreateEnvironment(ctx, CreateEnvironmentParams{
		ContentRevisionID: rev.ID,
		Path:              "lm-test.yaml",
		Name:              "lm-test",
		Teams:             1,
		Access:            emptyArr,
		Vars:              emptyObj,
		Tags:              emptyObj,
		Findings:          emptyArr,
	})
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	if _, err := q.CreatePlacement(ctx, CreatePlacementParams{
		EnvironmentID: env.ID,
		NetworkName:   "prod",
		ObjectKind:    "host",
		ObjectName:    "web01",
		AsName:        "web01",
		LastOctet:     12,
	}); err != nil {
		t.Fatalf("CreatePlacement: %v", err)
	}

	if _, err := q.CreateNetwork(ctx, CreateNetworkParams{
		ContentRevisionID: rev.ID, Path: "networks/prod.yaml", Name: "prod", Cidr: "10.0.1.0/24",
		VisibleFrom: emptyArr, Vars: emptyObj, Tags: emptyObj, Findings: emptyArr,
	}); err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}

	if _, err := q.CreateHost(ctx, CreateHostParams{
		ContentRevisionID: rev.ID, Path: "hosts/web01.yaml", Name: "web01",
		Ports: emptyObj, DependsOn: emptyArr, Steps: emptyArr, Schedule: emptyArr, Vars: emptyObj,
		Tags: emptyObj, Findings: emptyArr, People: emptyArr,
	}); err != nil {
		t.Fatalf("CreateHost: %v", err)
	}

	if _, err := q.CreateContainer(ctx, CreateContainerParams{
		ContentRevisionID: rev.ID, Path: "hosts/scoreboard.yaml", Name: "scoreboard",
		Ports: emptyObj, DependsOn: emptyArr, Steps: emptyArr, Schedule: emptyArr, Vars: emptyObj,
		Tags: emptyObj, Findings: emptyArr, People: emptyArr,
	}); err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}

	if _, err := q.CreateScript(ctx, CreateScriptParams{
		ContentRevisionID: rev.ID, Path: "scripts/base.yaml", Name: "base",
		Args: emptyArr, Tags: emptyObj, Findings: emptyArr, People: emptyArr, Validate: emptyArr,
	}); err != nil {
		t.Fatalf("CreateScript: %v", err)
	}

	ps, err := q.CreatePeopleSource(ctx, CreatePeopleSourceParams{
		ContentRevisionID: rev.ID, Path: "people/employees.csv", Name: "employees",
	})
	if err != nil {
		t.Fatalf("CreatePeopleSource: %v", err)
	}
	attrs, _ := json.Marshal(map[string]string{"first_name": "Aaron"})
	if _, err := q.CreatePerson(ctx, CreatePersonParams{
		PeopleSourceID: ps.ID, Username: "arivera3", Attributes: attrs,
	}); err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	envs, err := q.ListEnvironmentsByRevision(ctx, rev.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentsByRevision: %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "lm-test" {
		t.Fatalf("ListEnvironmentsByRevision = %+v, want one environment named lm-test", envs)
	}

	updated, err := q.SetContentRevisionValidation(ctx, SetContentRevisionValidationParams{
		ID: rev.ID, Valid: true, ValidationErrors: emptyArr,
	})
	if err != nil {
		t.Fatalf("SetContentRevisionValidation: %v", err)
	}
	if !updated.Valid || !updated.ValidatedAt.Valid {
		t.Fatalf("SetContentRevisionValidation didn't stick: %+v", updated)
	}

	cb, err := q.CreateConfiguredBuild(ctx, CreateConfiguredBuildParams{
		RepositoryID: repo.ID, Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	if err != nil {
		t.Fatalf("CreateConfiguredBuild: %v", err)
	}
	if !cb.AutoDeployEnabled || cb.CompetitionStarted {
		t.Fatalf("CreateConfiguredBuild defaults wrong: auto_deploy=%v started=%v, want auto_deploy=true started=false", cb.AutoDeployEnabled, cb.CompetitionStarted)
	}

	byBranch, err := q.ListConfiguredBuildsByRepositoryAndBranch(ctx, ListConfiguredBuildsByRepositoryAndBranchParams{
		RepositoryID: repo.ID, Branch: "main",
	})
	if err != nil {
		t.Fatalf("ListConfiguredBuildsByRepositoryAndBranch: %v", err)
	}
	if len(byBranch) != 1 {
		t.Fatalf("ListConfiguredBuildsByRepositoryAndBranch = %d rows, want 1", len(byBranch))
	}

	cb2, err := q.SetConfiguredBuildCurrentRevision(ctx, SetConfiguredBuildCurrentRevisionParams{
		ID: cb.ID, CurrentContentRevisionID: rev.ID,
	})
	if err != nil {
		t.Fatalf("SetConfiguredBuildCurrentRevision: %v", err)
	}
	if cb2.CurrentContentRevisionID != rev.ID {
		t.Fatalf("SetConfiguredBuildCurrentRevision didn't stick: %+v", cb2)
	}

	locked, err := q.SetConfiguredBuildCompetitionStarted(ctx, SetConfiguredBuildCompetitionStartedParams{
		ID: cb.ID, CompetitionStarted: true,
	})
	if err != nil {
		t.Fatalf("SetConfiguredBuildCompetitionStarted: %v", err)
	}
	if !locked.CompetitionStarted {
		t.Fatal("SetConfiguredBuildCompetitionStarted didn't stick")
	}

	noAutoDeploy, err := q.SetConfiguredBuildAutoDeploy(ctx, SetConfiguredBuildAutoDeployParams{
		ID: cb.ID, AutoDeployEnabled: false,
	})
	if err != nil {
		t.Fatalf("SetConfiguredBuildAutoDeploy: %v", err)
	}
	if noAutoDeploy.AutoDeployEnabled {
		t.Fatal("SetConfiguredBuildAutoDeploy didn't stick")
	}

	// Cascade check: deleting the repository (in the deferred cleanup)
	// must take content_revision, environment, placement, network, host,
	// container, script, people_source, person, and configured_build with
	// it. Verified after the deferred delete runs, below.
	t.Cleanup(func() {
		var count int
		row := pool.QueryRow(ctx, "SELECT count(*) FROM content_revision WHERE repository_id = $1", repo.ID)
		if err := row.Scan(&count); err != nil {
			t.Errorf("post-cleanup cascade check query: %v", err)
			return
		}
		if count != 0 {
			t.Errorf("ON DELETE CASCADE did not clean up content_revision rows: %d left", count)
		}
	})
}

func strPtr(s string) *string { return &s }

var _ = pgtype.UUID{} // keep the pgtype import honest if the above ever stops using it directly

// TestRuntimeTablesRoundTrip exercises the runtime schema: build, team,
// deployed_object, task (lease/heartbeat/complete), event, and
// fake_hoster_resource. The one thing worth a dedicated test rather than
// folding into TestRoundTrip: deployed_object_identity_idx has to allow
// TWO deployed_object rows for the SAME object_name in the SAME team --
// "listing a host twice puts two copies of it on the network" -- while
// still treating a second EnsureDeployedObject call for the SAME as_name
// as an update-in-place, not a third row.
func TestRuntimeTablesRoundTrip(t *testing.T) {
	conn := dbTestConnString()
	if conn == "" {
		t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	}
	ctx := context.Background()
	q, pool, err := Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	repo, err := q.CreateRepository(ctx, CreateRepositoryParams{
		GithubOwner: "laforge-test-owner", GithubRepo: "laforge-test-repo-runtime",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", repo.ID); err != nil {
			t.Errorf("cleanup: delete repository: %v", err)
		}
	}()
	rev, err := q.CreateContentRevision(ctx, CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "runtime-test-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}

	build, err := q.CreateBuild(ctx, CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	if build.Status != "planned" {
		t.Fatalf("Status = %q, want planned (the column default)", build.Status)
	}

	team, err := q.EnsureTeam(ctx, EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	team2, err := q.EnsureTeam(ctx, EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam (re-ensure): %v", err)
	}
	if team2.ID != team.ID {
		t.Fatalf("EnsureTeam created a second row for the same (build, team_number)")
	}

	// Two copies of the "kali" host definition, as kali01 and kali02 --
	// must land as two distinct deployed_object rows.
	kali01, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "kali", AsName: strPtr("kali01"), NetworkName: strPtr("vdi"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject(kali01): %v", err)
	}
	kali02, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "kali", AsName: strPtr("kali02"), NetworkName: strPtr("vdi"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject(kali02): %v", err)
	}
	if kali01.ID == kali02.ID {
		t.Fatal("two copies of the same host definition (kali01, kali02) collapsed into one deployed_object row")
	}

	// Re-ensuring kali01 again (same as_name) must update in place, not
	// create a third row.
	kali01Again, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "kali", AsName: strPtr("kali01"), NetworkName: strPtr("vdi"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject(kali01, again): %v", err)
	}
	if kali01Again.ID != kali01.ID {
		t.Fatal("re-ensuring kali01 created a new row instead of updating in place")
	}

	all, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ListDeployedObjectsByBuild = %d rows, want exactly 2 (kali01, kali02)", len(all))
	}

	// A network has no as_name, so its identity falls back to
	// object_name -- prove that path separately.
	net1, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "network", ObjectName: "vdi",
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject(network vdi): %v", err)
	}
	net1Again, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "network", ObjectName: "vdi",
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject(network vdi, again): %v", err)
	}
	if net1.ID != net1Again.ID {
		t.Fatal("re-ensuring the vdi network created a second row")
	}

	// --- task lease/heartbeat/complete, and the one-open-task guarantee ---

	task1, err := q.CreateTaskIfNoneOpen(ctx, CreateTaskIfNoneOpenParams{
		BuildID: build.ID, DeployedObjectID: kali01.ID, Kind: "deploy_host", Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateTaskIfNoneOpen: %v", err)
	}

	// A second task for the SAME object while the first is still open must
	// be refused (zero rows, not an error) -- the whole point of the
	// partial unique index this query relies on.
	_, err = q.CreateTaskIfNoneOpen(ctx, CreateTaskIfNoneOpenParams{
		BuildID: build.ID, DeployedObjectID: kali01.ID, Kind: "deploy_host", Payload: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("CreateTaskIfNoneOpen should have returned an error (no rows) for a second open task on the same object")
	} else if !strings.Contains(err.Error(), "no rows") {
		t.Fatalf("CreateTaskIfNoneOpen second call: got %v, want a \"no rows\" error", err)
	}

	leased, err := q.LeaseTask(ctx, LeaseTaskParams{LeaseOwner: strPtr("runner-test"), Column2: Interval(30 * time.Second)})
	if err != nil {
		t.Fatalf("LeaseTask: %v", err)
	}
	if leased.ID != task1.ID {
		t.Fatalf("LeaseTask leased %v, want %v", leased.ID, task1.ID)
	}
	if leased.Attempts != 1 {
		t.Fatalf("Attempts = %d, want 1 after first lease", leased.Attempts)
	}

	affected, err := q.HeartbeatTask(ctx, HeartbeatTaskParams{
		ID: task1.ID, LeaseOwner: strPtr("runner-test"), Column3: Interval(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("HeartbeatTask: %v", err)
	}
	if affected != 1 {
		t.Fatalf("HeartbeatTask affected %d rows, want 1", affected)
	}

	// A heartbeat from a runner that doesn't hold the lease must affect
	// nothing -- this is the exact signal a real runner uses to know its
	// lease was reclaimed out from under it and stop working.
	affected, err = q.HeartbeatTask(ctx, HeartbeatTaskParams{
		ID: task1.ID, LeaseOwner: strPtr("someone-else"), Column3: Interval(30 * time.Second),
	})
	if err != nil {
		t.Fatalf("HeartbeatTask (wrong owner): %v", err)
	}
	if affected != 0 {
		t.Fatalf("HeartbeatTask (wrong owner) affected %d rows, want 0", affected)
	}

	completed, err := q.CompleteTask(ctx, CompleteTaskParams{ID: task1.ID, LeaseOwner: strPtr("runner-test")})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if completed.Status != "done" {
		t.Fatalf("Status = %q, want done", completed.Status)
	}

	// Now that the task is done (no longer open), a fresh task for the
	// same object must be allowed again.
	task2, err := q.CreateTaskIfNoneOpen(ctx, CreateTaskIfNoneOpenParams{
		BuildID: build.ID, DeployedObjectID: kali01.ID, Kind: "deploy_host", Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateTaskIfNoneOpen after completion: %v", err)
	}
	if task2.ID == task1.ID {
		t.Fatal("expected a new task row, got the same one back")
	}

	// --- event journal ---

	ev, err := q.CreateEvent(ctx, CreateEventParams{
		BuildID: build.ID, TaskID: task1.ID, Kind: "task.completed", Message: "deploy_host ok", Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	events, err := q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	if len(events) != 1 || events[0].ID != ev.ID {
		t.Fatalf("ListEventsByBuild = %+v, want exactly the one event just created", events)
	}

	// --- fake_hoster_resource: ensure is idempotent, destroy is safe twice ---
	//
	// fake_hoster_resource has no foreign key back to repository/build (by
	// design -- see migrations/00003: it deliberately outlives any single
	// runner or build), so nothing about deleting the repository above
	// cleans it up. Using a ref unique to this test run (rather than a
	// fixed "test-kali01") and deleting it explicitly afterward is what
	// keeps a second run of this same test -- the exact thing `go test
	// ./...` does every time -- from finding a row already at
	// ensure_count=2 and failing on that leftover state instead of what
	// this run itself did.
	ref := "test-" + build.ID.String()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM fake_hoster_resource WHERE external_ref = $1", ref); err != nil {
			t.Errorf("cleanup: delete fake_hoster_resource: %v", err)
		}
	})

	res1, err := q.EnsureFakeHosterResource(ctx, EnsureFakeHosterResourceParams{ExternalRef: ref, Kind: "host"})
	if err != nil {
		t.Fatalf("EnsureFakeHosterResource: %v", err)
	}
	res2, err := q.EnsureFakeHosterResource(ctx, EnsureFakeHosterResourceParams{ExternalRef: ref, Kind: "host"})
	if err != nil {
		t.Fatalf("EnsureFakeHosterResource (again): %v", err)
	}
	if res2.EnsureCount != 2 {
		t.Fatalf("EnsureCount = %d, want 2 after two ensures", res2.EnsureCount)
	}
	count, err := q.CountFakeHosterResourcesByRef(ctx, ref)
	if err != nil {
		t.Fatalf("CountFakeHosterResourcesByRef: %v", err)
	}
	if count != 1 {
		t.Fatalf("CountFakeHosterResourcesByRef = %d, want exactly 1 despite two ensures", count)
	}
	_ = res1

	if _, err := q.DestroyFakeHosterResource(ctx, ref); err != nil {
		t.Fatalf("DestroyFakeHosterResource: %v", err)
	}
	// Destroying an unknown ref (never ensured) is a real ErrNoRows from
	// this :one query -- internal/builder/fake treats that as a no-op
	// success; here we just confirm the query itself behaves that way.
	if _, err := q.DestroyFakeHosterResource(ctx, "never-existed"); err == nil {
		t.Fatal("expected pgx.ErrNoRows destroying a ref that was never ensured")
	}
}

// TestEnvironmentAgentDebugForObject exercises the object -> team -> build ->
// environment join the runner uses to decide whether to bake the agent-debug
// flag into a host's agent binary. It builds the full chain with agent_debug
// on, and confirms a deployed object in that build resolves to true.
func TestEnvironmentAgentDebugForObject(t *testing.T) {
	conn := dbTestConnString()
	if conn == "" {
		t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	}
	ctx := context.Background()
	q, pool, err := Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	repo, err := q.CreateRepository(ctx, CreateRepositoryParams{
		GithubOwner: "laforge-test-owner", GithubRepo: "laforge-test-repo-agentdebug",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", repo.ID); err != nil {
			t.Errorf("cleanup: delete repository: %v", err)
		}
	})
	rev, err := q.CreateContentRevision(ctx, CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "agentdebug-test-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	if _, err := q.CreateEnvironment(ctx, CreateEnvironmentParams{
		ContentRevisionID: rev.ID, Path: "dbg.yaml", Name: "dbg", Teams: 1,
		Access: []byte(`[]`), Vars: []byte(`{}`), Tags: []byte(`{}`), Findings: []byte(`[]`),
		AgentDebug: true,
	}); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}
	build, err := q.CreateBuild(ctx, CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "dbg"})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	team, err := q.EnsureTeam(ctx, EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "box", AsName: strPtr("box01"), NetworkName: strPtr("lan"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}

	debug, err := q.GetEnvironmentAgentDebugForObject(ctx, obj.ID)
	if err != nil {
		t.Fatalf("GetEnvironmentAgentDebugForObject: %v", err)
	}
	if !debug {
		t.Fatal("agent_debug = false, want true -- the flag did not flow object -> team -> build -> environment")
	}
}

// TestAdHocTaskRunsPastFailedStep is the regression for "a stuck/failed build
// blocks ad-hoc commands." A terminally-failed authored step must block later
// authored steps (the deploy stops), but an ad-hoc task must still run so an
// operator can debug the box.
func TestAdHocTaskRunsPastFailedStep(t *testing.T) {
	conn := dbTestConnString()
	if conn == "" {
		t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	}
	ctx := context.Background()
	q, pool, err := Open(ctx, conn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	repo, err := q.CreateRepository(ctx, CreateRepositoryParams{
		GithubOwner: "laforge-test-owner", GithubRepo: "laforge-test-repo-adhoc",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, "DELETE FROM repository WHERE id = $1", repo.ID); err != nil {
			t.Errorf("cleanup: delete repository: %v", err)
		}
	})
	rev, err := q.CreateContentRevision(ctx, CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "adhoc-test-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "adhoc"})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	team, err := q.EnsureTeam(ctx, EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "box", AsName: strPtr("box01"), NetworkName: strPtr("lan"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}

	emptyObj := []byte(`{}`)
	// Authored step 0, then mark it terminally failed (retries exhausted).
	step0, err := q.CreateAgentTask(ctx, CreateAgentTaskParams{DeployedObjectID: obj.ID, StepIndex: 0, Command: "execute", Payload: emptyObj})
	if err != nil {
		t.Fatalf("CreateAgentTask(step0): %v", err)
	}
	if _, err := q.FailAgentTask(ctx, FailAgentTaskParams{ID: step0.ID, Status: "failed", LastError: strPtr("boom")}); err != nil {
		t.Fatalf("FailAgentTask: %v", err)
	}
	// Authored step 1 (pending) -- must stay blocked behind the failed step 0.
	if _, err := q.CreateAgentTask(ctx, CreateAgentTaskParams{DeployedObjectID: obj.ID, StepIndex: 1, Command: "execute", Payload: emptyObj}); err != nil {
		t.Fatalf("CreateAgentTask(step1): %v", err)
	}

	lease := NextAgentTaskForHostParams{DeployedObjectID: obj.ID, Column2: Interval(time.Minute)}
	if _, err := q.NextAgentTaskForHost(ctx, lease); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("with only a failed step 0 + pending step 1, get-task = %v, want ErrNoRows (the deploy is blocked)", err)
	}

	// Now an operator fires an ad-hoc command (appended at the next index).
	adhoc, err := q.CreateAdHocAgentTask(ctx, CreateAdHocAgentTaskParams{DeployedObjectID: obj.ID, StepIndex: 2, Command: "execute", Payload: emptyObj})
	if err != nil {
		t.Fatalf("CreateAdHocAgentTask: %v", err)
	}
	got, err := q.NextAgentTaskForHost(ctx, lease)
	if err != nil {
		t.Fatalf("get-task after ad-hoc dispatch: %v (want the ad-hoc task, not blocked by the failed step)", err)
	}
	if got.ID != adhoc.ID {
		t.Fatalf("get-task returned task %s, want the ad-hoc task %s -- ad-hoc must run past a failed step", got.ID, adhoc.ID)
	}
}
