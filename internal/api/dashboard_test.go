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
	"github.com/jackc/pgx/v5/pgtype"
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

// TestHeartbeatChartsAveragesAndSkipsNil pins the resource-graph math, as the
// database does it: a metric present on some heartbeats is averaged over just
// those (a NULL is skipped, not counted as 0), activity counts distinct
// objects rather than heartbeats, and a build with no heartbeats has empty
// (non-nil) series.
func TestHeartbeatChartsAveragesAndSkipsNil(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	ctx := context.Background()

	empty, emptyRes, err := env.server.heartbeatCharts(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || emptyRes == nil || len(empty) != 0 || len(emptyRes) != 0 {
		t.Fatalf("no heartbeats: got %v / %v, want empty non-nil slices", empty, emptyRes)
	}

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil || len(objs) < 2 {
		t.Fatalf("fixture objects: %d, %v", len(objs), err)
	}
	at := time.Now().Add(-time.Minute).Truncate(time.Second)
	insert := func(obj pgtype.UUID, at time.Time, cpu, mem any, rx, tx float64) {
		if _, err := env.server.Pool.Exec(ctx,
			`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, created_at, cpu_pct, mem_pct, net_rx_bps, net_tx_bps) VALUES ($1,'fp',$2,$3,$4,$5,$6)`,
			obj, at, cpu, mem, rx, tx); err != nil {
			t.Fatal(err)
		}
	}
	// Same instant: one bucket. Two heartbeats from one object, one from another.
	insert(objs[0].ID, at, 10.0, 50.0, 100, 200)
	insert(objs[0].ID, at, 30.0, nil, 300, 400)
	insert(objs[1].ID, at, 20.0, nil, 200, 300)

	activity, res, err := env.server.heartbeatCharts(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	// A second read inside the TTL comes from the cache, unchanged, even after
	// new heartbeats arrive.
	insert(objs[1].ID, at.Add(time.Second), 90.0, nil, 0, 0)
	again, _, err := env.server.heartbeatCharts(ctx, build.ID)
	if err != nil || len(again) != len(activity) || again[0] != activity[0] {
		t.Fatalf("cached read: %v / %v, want the first result again", again, err)
	}
	if len(res) == 0 || len(activity) != len(res) {
		t.Fatalf("got %d activity / %d resource buckets, want the same non-zero count", len(activity), len(res))
	}
	b := res[0]
	if b.Cpu != 20 {
		t.Errorf("cpu avg = %v, want 20", b.Cpu)
	}
	if b.Mem != 50 {
		t.Errorf("mem avg = %v, want 50 (the NULLs are skipped, not averaged as 0)", b.Mem)
	}
	if b.NetRx != 200 || b.NetTx != 300 {
		t.Errorf("net avg = rx %v tx %v, want 200 / 300", b.NetRx, b.NetTx)
	}
	if activity[0].Active != 2 {
		t.Errorf("active = %d, want 2 (distinct objects, not 3 heartbeats)", activity[0].Active)
	}
}

// TestChartBucketsLayout: the history spreads over at most maxChartBuckets even
// buckets, every bucket gets a labelled point (empty ones read 0), and a row
// past the last bucket is ignored rather than indexing out of range.
func TestChartBucketsLayout(t *testing.T) {
	first := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	width, count := bucketLayout(first, first.Add(80*time.Minute))
	if count != maxChartBuckets || width != 2*time.Minute {
		t.Fatalf("80 minutes: width %v count %d, want 2m and %d", width, count, maxChartBuckets)
	}
	// A single instant is stretched to one second, as before: still a full set
	// of buckets with a positive width, everything landing in the first.
	if w, c := bucketLayout(first, first); c != maxChartBuckets || w <= 0 {
		t.Fatalf("one instant: width %v count %d, want %d buckets of positive width", w, c, maxChartBuckets)
	}
	rows := []db.AgentHeartbeatBucketsByBuildRow{{Bucket: 3, Active: 7, Cpu: 42}, {Bucket: 99, Active: 1}}
	activity, res := chartBuckets(first, width, count, rows)
	if len(activity) != count || len(res) != count {
		t.Fatalf("got %d/%d points, want %d each", len(activity), len(res), count)
	}
	if activity[3].Active != 7 || res[3].Cpu != 42 || activity[0].Active != 0 {
		t.Fatalf("bucket 3 = %+v / %+v, bucket 0 = %+v", activity[3], res[3], activity[0])
	}
	if want := first.Add(3 * width).Format(time.RFC3339); activity[3].At != want {
		t.Fatalf("bucket 3 labelled %s, want %s", activity[3].At, want)
	}
}
