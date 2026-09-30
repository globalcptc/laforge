package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/runner"
)

// TestDashboardAggregatesRealData drives the real task queue (the actual
// runner against the fake builder, exactly the same proof
// pattern) so provisioning counts and check-ins reflect genuine
// convergence, then directly fails two real objects with the identical
// error message via the real MarkDeployedObjectFailed query -- proving
// "40 hosts: X as one entry, not forty" grouping against something that
// actually went through the failure path, not synthetic aggregate rows.
func TestDashboardAggregatesRealData(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env) // m7-two-team: 2 teams x (network + host + container)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	b := fake.New(env.server.Pool)
	r := &runner.Runner{
		Pool: env.server.Pool, Builder: b, RepoRoot: "../../examples/m7-two-team", ID: "dashboard-test-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: 5 * time.Second, BuildID: &build.ID,
	}
	for i := 0; i < 20; i++ {
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !did {
			break
		}
	}

	// Advance the object lifecycle the way the orchestrator poll would:
	// with infra up and no agent checked in, networks (which have no agent)
	// converge to finished, while hosts/containers stay running. This is
	// what makes a deployed network read as "completed" health below.
	if err := orchestrator.AdvanceObjectLifecycle(ctx, env.server.Pool, build.ID); err != nil {
		t.Fatalf("AdvanceObjectLifecycle: %v", err)
	}

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var hosts []db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			hosts = append(hosts, o)
		}
	}
	if len(hosts) != 2 {
		t.Fatalf("fixture has %d host(s), want 2 (one per team)", len(hosts))
	}
	const sameError = "install-mysql: exit status 1"
	for _, h := range hosts {
		if _, err := env.q.MarkDeployedObjectDeployFailed(ctx, db.MarkDeployedObjectDeployFailedParams{ID: h.ID, LastError: db.StrPtr(sameError)}); err != nil {
			t.Fatalf("MarkDeployedObjectDeployFailed: %v", err)
		}
	}

	// Two real objects' heartbeat histories, staggered over real
	// (backdated) timestamps -- object A keeps checking in the whole
	// window, object B goes quiet halfway through, standing in for a
	// real agent that died mid-event. Direct SQL, not CreateAgentHeartbeat
	// (which always stamps now()) -- the real write path over a live mTLS
	// connection is already proven in
	// internal/gateway's TestGatewayEndToEnd; this test's job is proving
	// the aggregation reads real historical rows correctly, including
	// detecting exactly this kind of cliff.
	var containers []db.DeployedObject
	for _, o := range objs {
		if o.Kind == "container" {
			containers = append(containers, o)
		}
	}
	if len(containers) != 2 {
		t.Fatalf("fixture has %d container(s), want 2 (one per team)", len(containers))
	}
	steady, quiets := containers[0], containers[1]
	base := time.Now().Add(-4 * time.Minute)
	for i := 0; i < 8; i++ { // steady: checks in across the whole window
		at := base.Add(time.Duration(i) * 30 * time.Second)
		if _, err := env.server.Pool.Exec(ctx,
			`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr, created_at) VALUES ($1,$2,$3,$4)`,
			steady.ID, "fp-steady", "10.0.0.1:4433", at,
		); err != nil {
			t.Fatalf("inserting steady heartbeat %d: %v", i, err)
		}
	}
	for i := 0; i < 4; i++ { // quiets: only checks in for the first half
		at := base.Add(time.Duration(i) * 30 * time.Second)
		if _, err := env.server.Pool.Exec(ctx,
			`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr, created_at) VALUES ($1,$2,$3,$4)`,
			quiets.ID, "fp-quiets", "10.0.0.2:4433", at,
		); err != nil {
			t.Fatalf("inserting quiets heartbeat %d: %v", i, err)
		}
	}

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dash dashboardData
	json.NewDecoder(resp.Body).Decode(&dash)

	if dash.ProvisioningByStatus["deploy_failed"] != 2 {
		t.Fatalf("ProvisioningByStatus[deploy_failed] = %d, want 2", dash.ProvisioningByStatus["deploy_failed"])
	}
	if dash.ProvisioningByStatus["finished"] < 2 { // both networks converge to finished
		t.Fatalf("ProvisioningByStatus[finished] = %d, want at least 2", dash.ProvisioningByStatus["finished"])
	}

	if len(dash.ByTeam) != 2 {
		t.Fatalf("ByTeam has %d entries, want 2", len(dash.ByTeam))
	}
	for _, tm := range dash.ByTeam {
		if tm.ByStatus["deploy_failed"] != 1 {
			t.Fatalf("team %d ByStatus[deploy_failed] = %d, want 1 (exactly its own host)", tm.TeamNumber, tm.ByStatus["deploy_failed"])
		}
		// Per-team health breakdown (the Overview "state by team" colours):
		// each team's failed host is classified as an infra failure, and its
		// finished network -- which never has an agent -- as completed.
		if tm.ByHealth["failed_infra"] != 1 {
			t.Fatalf("team %d ByHealth[failed_infra] = %d, want 1 (its failed host)", tm.TeamNumber, tm.ByHealth["failed_infra"])
		}
		if tm.ByHealth["completed"] < 1 {
			t.Fatalf("team %d ByHealth[completed] = %d, want >=1 (its deployed network)", tm.TeamNumber, tm.ByHealth["completed"])
		}
	}

	if len(dash.FailuresByCause) != 1 {
		t.Fatalf("FailuresByCause has %d group(s), want 1 (both hosts share the same error)", len(dash.FailuresByCause))
	}
	if dash.FailuresByCause[0].Message != sameError || dash.FailuresByCause[0].Count != 2 || len(dash.FailuresByCause[0].Objects) != 2 {
		t.Fatalf("FailuresByCause[0] = %+v, want message=%q count=2 with 2 objects", dash.FailuresByCause[0], sameError)
	}

	// The real cliff: the last bucket must show fewer active agents than
	// an earlier bucket did, because `quiets` genuinely stopped
	// heartbeating -- proving this is live ongoing activity, not a
	// cumulative "first ever seen" curve that could only ever go up.
	if len(dash.AgentActivity) < 2 {
		t.Fatalf("AgentActivity has %d bucket(s), want at least 2 to show a trend", len(dash.AgentActivity))
	}
	maxActive := 0
	for _, b := range dash.AgentActivity {
		if b.Active > maxActive {
			maxActive = b.Active
		}
	}
	if maxActive != 2 {
		t.Fatalf("max Active across buckets = %d, want 2 (both objects heartbeating at some point)", maxActive)
	}
	last := dash.AgentActivity[len(dash.AgentActivity)-1]
	if last.Active != 1 {
		t.Fatalf("last bucket Active = %d, want 1 (only `steady` still checking in by the end -- the cliff)", last.Active)
	}
}

func TestDashboardRequiresAtLeastRead(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	resp, err := http.Get(env.httpURL + "/builds/" + build.ID.String() + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
