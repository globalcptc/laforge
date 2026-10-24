package orchestrator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// countNetAccessTasks returns the build's configure_network_access tasks and
// their payload fingerprints.
func netAccessTasks(t *testing.T, q *db.Queries, buildID db.Build) []db.Task {
	t.Helper()
	all, err := q.ListTasksByBuild(context.Background(), buildID.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	var out []db.Task
	for _, tk := range all {
		if tk.Kind == "configure_network_access" {
			out = append(out, tk)
		}
	}
	return out
}

func fpOf(t *testing.T, task db.Task) string {
	t.Helper()
	var p struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.Unmarshal(task.Payload, &p); err != nil {
		t.Fatalf("decoding task payload: %v", err)
	}
	return p.Fingerprint
}

// TestReconcileNetworkAccessGating is the issue regression for the reconciler
// half: it must NOT enqueue the ACL task while a guest is still being created,
// must enqueue once every guest has settled, must re-enqueue when a guest's
// instance reference changes (a rebuild replacement), and must re-enqueue after
// a failed attempt -- so the policy converges instead of staying half-applied.
func TestReconcileNetworkAccessGating(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "netaccess-gating")

	if _, err := q.CreateNetwork(ctx, db.CreateNetworkParams{
		ContentRevisionID: build.ContentRevisionID, Path: "networks/lan.yaml", Name: "lan", Cidr: "10.0.0.0/24",
		VisibleFrom: []byte("[]"), Vars: []byte("{}"), Tags: []byte("{}"), Findings: []byte("[]"),
	}); err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	netObj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "network", ObjectName: "lan", Fingerprint: "n"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='finished', external_ref='net-lan' WHERE id=$1", netObj.ID); err != nil {
		t.Fatal(err)
	}
	host, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "box", AsName: db.StrPtr("box"), NetworkName: db.StrPtr("lan"), Fingerprint: "h",
	})
	if err != nil {
		t.Fatal(err)
	}
	setHost := func(status, ref string) {
		if ref == "" {
			if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status=$2, external_ref=NULL WHERE id=$1", host.ID, status); err != nil {
				t.Fatal(err)
			}
			return
		}
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status=$2, external_ref=$3 WHERE id=$1", host.ID, status, ref); err != nil {
			t.Fatal(err)
		}
	}

	// 1) Guest still being created -> no task.
	setHost("pending", "")
	if err := ReconcileNetworkAccess(ctx, pool, build.ID); err != nil {
		t.Fatalf("ReconcileNetworkAccess: %v", err)
	}
	if n := len(netAccessTasks(t, q, build)); n != 0 {
		t.Fatalf("created %d task(s) while a guest was still pending, want 0", n)
	}

	// 2) All settled -> exactly one task.
	setHost("running", "ref-1")
	if err := ReconcileNetworkAccess(ctx, pool, build.ID); err != nil {
		t.Fatalf("ReconcileNetworkAccess: %v", err)
	}
	tasks := netAccessTasks(t, q, build)
	if len(tasks) != 1 {
		t.Fatalf("after settling, task count = %d, want 1", len(tasks))
	}
	fp1 := fpOf(t, tasks[0])

	// 3) Idempotent: same guest set -> no new task.
	if err := ReconcileNetworkAccess(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(netAccessTasks(t, q, build)); n != 1 {
		t.Fatalf("re-ran reconcile with an unchanged guest set, task count = %d, want 1", n)
	}

	// 4) Instance replacement (new ref) -> a fresh task with a different fingerprint.
	setHost("running", "ref-2")
	if err := ReconcileNetworkAccess(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	tasks = netAccessTasks(t, q, build)
	if len(tasks) != 2 {
		t.Fatalf("after a guest's ref changed, task count = %d, want 2", len(tasks))
	}
	var fp2 string
	for _, tk := range tasks {
		if f := fpOf(t, tk); f != fp1 {
			fp2 = f
		}
	}
	if fp2 == "" {
		t.Fatal("replacement produced no new fingerprint -- the guest ref isn't in the fingerprint")
	}

	// 5) A failed task does not dedup -> reconcile re-enqueues to converge.
	// Mark every current task failed, then reconcile with the same guest set.
	if _, err := pool.Exec(ctx, "UPDATE task SET status='failed' WHERE build_id=$1 AND kind='configure_network_access'", build.ID); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileNetworkAccess(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(netAccessTasks(t, q, build)); n != 3 {
		t.Fatalf("after failing the tasks, reconcile task count = %d, want 3 (a fresh retry)", n)
	}
}
