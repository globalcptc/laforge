package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// skipIfNoOVN preflights the one thing this whole test now depends on --
// a real OVN control plane behind the daemon -- with a single throwaway
// network create, so a daemon without one (this session's own
// environment, confirmed live) skips with a clear reason instead of
// failing 40 lease attempts deep with a wall of the same error. Mirrors
// internal/builder/incus's own TestNetworkDeployAdoptDestroy.
func skipIfNoOVN(t *testing.T, b *incus.Builder) {
	t.Helper()
	ref, err := b.DeployNetwork(context.Background(), builder.NetworkSpec{
		ExternalName: fmt.Sprintf("ovn-preflight-%d", time.Now().UnixNano()),
		Team:         "preflight", CIDR: "10.250.250.0/24",
	})
	if err != nil {
		var apiErr *incus.APIError
		if errors.As(err, &apiErr) && apiErr.OVNUnavailable() {
			t.Skipf("OVN isn't available on this daemon (expected in this session's own environment -- see incus.Builder's own doc comment): %v", err)
		}
		t.Fatalf("OVN preflight: %v", err)
	}
	if err := b.DestroyNetwork(context.Background(), "", ref); err != nil {
		t.Fatalf("OVN preflight cleanup: %v", err)
	}
}

// TestRealTwoTeamIncusBuild is the required proof: "first
// real two-team build from a branch." It runs the actual pipeline --
// Reconcile against real content, a real Runner leasing real tasks from
// Postgres, calling the real Incus builder against a live daemon -- for
// examples/m7-two-team (2 teams, deliberately Linux-only so it doesn't
// hit this environment's documented no-KVM limitation). Skips cleanly if
// either a local Postgres or the live Incus test daemon isn't available,
// matching every other real-infrastructure test in this repo
// (chaos_test.go, builder_test.go).
//
// Previously (before this session's own OVN change -- see
// internal/builder/incus.Builder's own doc comment) this test asserted a
// real, live finding: team 1's network deployed and team 2's failed,
// because two plain Incus bridges can't share the identical address
// "every team gets exactly the same network" requires. That's now fixed
// at the design level -- DeployNetwork uses OVN logical networks, which
// natively support the same subnet across isolated networks -- but this
// session's own environment has no OVN control plane to prove it against
// (confirmed live: `type=ovn` network create fails outright, "OVN isn't
// currently available"). So this test preflights that exact condition and
// skips cleanly rather than asserting a since-fixed limitation as if it
// were still correct behavior; a real MicroCloud cluster's own OVN setup
// is what actually proves this now.
func TestRealTwoTeamIncusBuild(t *testing.T) {
	pool := testPool(t)
	client := liveIncusClient(t)
	b := incus.New(client, incus.Config{
		Images: map[string]incus.ImageRef{
			"alpine": {Alias: "alpine/3.21", Server: "https://images.linuxcontainers.org", Protocol: "simplestreams"},
		},
		Sizes:            map[string]incus.SizeSpec{"small": {CPU: "1", Memory: "256MiB"}},
		OVNUplinkNetwork: "UPLINK",
	})
	skipIfNoOVN(t, b)

	ctx := context.Background()
	q := db.New(pool)
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{
		GithubOwner: "laforge-test", GithubRepo: "m7-two-team-e2e",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })

	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "m7-e2e-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "m7-two-team",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	// This is the real two-team DEPLOY proof against a live
	// Incus daemon, which only happens for a build in "deploying" -- see
	// internal/orchestrator.Reconcile's deployTasks gate.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"

	repoRoot := "../../examples/m7-two-team"
	if err := orchestrator.Reconcile(ctx, pool, repoRoot, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("ListTeamsByBuild = %d, want 2", len(teams))
	}

	r := &Runner{
		Pool: pool, Builder: b, RepoRoot: repoRoot, ID: "m7-e2e-runner",
		LeaseDuration: 60 * time.Second, HeartbeatInterval: 5 * time.Second,
		BuildID: &build.ID,
	}

	var team1ID, team2ID pgtype.UUID
	for _, tm := range teams {
		switch tm.TeamNumber {
		case 1:
			team1ID = tm.ID
		case 2:
			team2ID = tm.ID
		}
	}

	// Drive the real task queue until every task reaches a terminal state.
	// LeaseTask picks any pending task, so ordering isn't guaranteed -- a
	// host/container task can genuinely be leased before its own team's
	// network task, fail once with a real "network not found," and
	// requeue via RetryTask, converging once the network task itself has
	// run. Errors are only logged, not fatal; only a queue that never
	// reaches an all-terminal state is.
	attempts, completed := 0, 0
	for attempts < 40 {
		attempts++
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Logf("attempt %d: %v", attempts, err)
			continue
		}
		if !did {
			break
		}
		completed++
	}
	tasksFinal, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	for _, tk := range tasksFinal {
		if tk.Status != "done" && tk.Status != "failed" {
			t.Fatalf("task %s (%s) status = %q after %d lease attempts -- queue did not converge to a terminal state", tk.ID, tk.Kind, tk.Status, attempts)
		}
	}
	t.Logf("queue reached a terminal state after %d lease attempts, %d tasks completed", attempts, completed)

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != 6 {
		t.Fatalf("ListDeployedObjectsByBuild = %d, want 6", len(objs))
	}
	// Both teams fully deploy now -- OVN gives each team's network its own
	// isolated logical router, so team 2's identically-addressed network
	// no longer collides with team 1's (see this test's own doc comment).
	for _, o := range objs {
		if o.Status != "running" {
			t.Fatalf("deployed_object %s (team %v, %s) status = %q, want running", o.ID, o.TeamID, o.Kind, o.Status)
		}
		if db.StrOrEmpty(o.ExternalRef) == "" {
			t.Fatalf("deployed_object %s (%s) is deployed but has no external_ref", o.ID, o.Kind)
		}
	}
	t.Log("both teams fully deployed for real against the live Incus daemon, sharing identical addressing via OVN")

	// --- close_access / open_access, through the real runner, against
	// team 1's real deployed host and container -- and confirmed to leave
	// team 2 untouched, now that team 2 has real resources of its own to
	// check ("Blast radius: failures are per team," the same property
	// this test already held for a builder-level failure, holding here
	// too for an operator action scoped to one team). ---
	closePayload, _ := json.Marshal(map[string]string{"team": "1"})
	if _, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{
		BuildID: build.ID, Kind: "close_access", Payload: closePayload,
	}); err != nil {
		t.Fatalf("CreateTeamTask(close_access): %v", err)
	}
	if did, err := r.LeaseAndExecuteOne(ctx); err != nil || !did {
		t.Fatalf("close_access task: did=%v err=%v", did, err)
	}

	byTeam1, err := q.ListDeployedObjectsByTeam(ctx, team1ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByTeam: %v", err)
	}
	for _, o := range byTeam1 {
		if o.Kind == "network" {
			continue
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if hasWorkingEth0(t, client, ref) {
			t.Fatalf("instance %s (team 1, %s) still has a working eth0 after close_access", ref, o.Kind)
		}
	}
	t.Log("close_access confirmed: team 1's instances have no working eth0")

	byTeam2, err := q.ListDeployedObjectsByTeam(ctx, team2ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByTeam (team 2): %v", err)
	}
	for _, o := range byTeam2 {
		if o.Kind == "network" {
			continue
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if !hasWorkingEth0(t, client, ref) {
			t.Fatalf("instance %s (team 2, %s) lost its eth0 after CloseAccess(\"team 1\") -- affected the wrong team", ref, o.Kind)
		}
	}
	t.Log("close_access(team 1) confirmed to leave team 2 untouched")

	openPayload, _ := json.Marshal(map[string]string{"team": "1"})
	if _, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{
		BuildID: build.ID, Kind: "open_access", Payload: openPayload,
	}); err != nil {
		t.Fatalf("CreateTeamTask(open_access): %v", err)
	}
	if did, err := r.LeaseAndExecuteOne(ctx); err != nil || !did {
		t.Fatalf("open_access task: did=%v err=%v", did, err)
	}
	for _, o := range byTeam1 {
		if o.Kind == "network" {
			continue
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if !hasWorkingEth0(t, client, ref) {
			t.Fatalf("instance %s (team 1, %s) still has no working eth0 after open_access", ref, o.Kind)
		}
	}
	t.Log("open_access confirmed: team 1's instances have their eth0 back")

	// --- teardown: destroy every real resource this test created, through
	// the real builder, so the live daemon is left clean. Both teams left
	// real resources behind now. Hosts/containers first, network last --
	// confirmed live that Incus refuses to delete a network still
	// attached to a running instance ("The network is currently in use"),
	// matching the plan's own "teardown cascades along depends_on."
	for _, kind := range []string{"host", "container", "network"} {
		for _, o := range objs {
			if o.Kind != kind {
				continue
			}
			ref := db.StrOrEmpty(o.ExternalRef)
			var derr error
			switch o.Kind {
			case "host":
				derr = b.DestroyHost(ctx, "", ref) // team unused: a single-endpoint incus.Builder ignores it
			case "container":
				derr = b.DestroyContainer(ctx, "", ref)
			case "network":
				derr = b.DestroyNetwork(ctx, "", ref)
			}
			if derr != nil {
				t.Errorf("teardown: destroying %s %s: %v", o.Kind, ref, derr)
			}
		}
	}
}

// liveIncusClient mirrors internal/builder/incus's own liveClient test
// helper (unexported there) -- see that package's builder_test.go for the
// full rationale. Duplicated rather than exported purely for a test,
// since it's a handful of lines and this is the only other package that
// needs it.
func liveIncusClient(t *testing.T) *incus.Client {
	t.Helper()
	if c, ok, err := incus.DialFromEnv(context.Background()); err != nil {
		t.Fatalf("DialFromEnv: %v", err)
	} else if ok {
		return c
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found, skipping (needs a live Incus daemon, or set LAFORGE_INCUS_TEST_URL for a real cluster)")
	}
	if err := exec.Command("docker", "exec", "incus-test", "incus", "info").Run(); err != nil {
		t.Skip("no running 'incus-test' container with a live Incus daemon")
	}
	ca, err := agentpki.GenerateCA(fmt.Sprintf("incus-runner-e2e-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	certPath := filepath.Join(t.TempDir(), "client.crt")
	if err := os.WriteFile(certPath, ca.CertPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("docker", "cp", certPath, "incus-test:/tmp/go-e2e-client.crt").Run(); err != nil {
		t.Fatalf("docker cp: %v", err)
	}
	trustName := fmt.Sprintf("go-e2e-%d", time.Now().UnixNano())
	if out, err := exec.Command("docker", "exec", "incus-test", "incus", "config", "trust", "add-certificate", "/tmp/go-e2e-client.crt", "--name", trustName).CombinedOutput(); err != nil {
		t.Fatalf("trusting test cert: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		exec.Command("docker", "exec", "incus-test", "incus", "config", "trust", "remove", trustName).Run()
	})
	serverCertPEM, err := incus.FetchServerCertificateInsecure(context.Background(), "localhost:8443")
	if err != nil {
		t.Fatalf("FetchServerCertificateInsecure: %v", err)
	}
	client, err := incus.NewClient("https://localhost:8443", ca.CertPEM, ca.KeyPEM, serverCertPEM, "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// hasWorkingEth0 checks a real instance's expanded_devices via `incus
// query`, the same real API GET path a Go client uses, filtered through
// the CLI since this test file has no direct handle on the incus
// package's unexported Client.get.
// hasWorkingEth0 goes through the real Client via incus.Builder's own
// exported HasWorkingEth0 -- not a local `incus` CLI shelled out to a
// docker container, which only ever worked against this package's own
// local-daemon fallback (liveIncusClient's docker-exec path) and simply
// couldn't check anything real against a genuine remote cluster (a
// `docker exec incus-test ...` call has no idea a real cluster even
// exists). Found while running this test for real against a live,
// OVN-enabled cluster for the first time: this previously untested-live
// path failed outright with "exit status 1" the moment there was no
// local `incus-test` container to shell into.
func hasWorkingEth0(t *testing.T, client *incus.Client, ref string) bool {
	t.Helper()
	b := incus.New(client, incus.Config{})
	ok, err := b.HasWorkingEth0(context.Background(), ref)
	if err != nil {
		t.Fatalf("HasWorkingEth0 for %s: %v", ref, err)
	}
	return ok
}
