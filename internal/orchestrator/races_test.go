package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

func taskRows(t *testing.T, pool *pgxpool.Pool, o db.DeployedObject, kind, status string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM task WHERE deployed_object_id = $1 AND kind = $2 AND status = $3", o.ID, kind, status).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func openTask(t *testing.T, pool *pgxpool.Pool, build db.Build, o db.DeployedObject, kind, status string, leasedFor time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO task (build_id, deployed_object_id, kind, payload, status, attempts, lease_owner, lease_expires_at)
		VALUES ($1, $2, $3, '{}', $4, CASE WHEN $4 = 'leased' THEN 1 ELSE 0 END, CASE WHEN $4 = 'leased' THEN 'r1' END, CASE WHEN $4 = 'leased' THEN now() + $5::interval END)`,
		build.ID, o.ID, kind, status, leasedFor.String()); err != nil {
		t.Fatal(err)
	}
}

// TestRebuildWhileBusyIsCarriedOutLater: a rebuild asked for while the host's
// deploy is being created can't queue its destroy. It used to reset the
// object anyway -- losing the hoster reference -- and the request was then
// forgotten. Now the object is left as it is with the request recorded, and
// the reconciler queues the destroy (with the reference) once the deploy ends.
// A deploy that was only queued, not started, is withdrawn instead.
func TestRebuildWhileBusyIsCarriedOutLater(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "rebuild-busy")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	mk := func(as string) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		pool.Exec(ctx, "UPDATE deployed_object SET status = 'deploying', external_ref = $2 WHERE id = $1", o.ID, "lf-"+as)
		o, _ = q.GetDeployedObject(ctx, o.ID)
		return o
	}

	// A deploy in flight (leased, lease live).
	busy := mk("web-busy")
	openTask(t, pool, build, busy, "deploy_host", "leased", time.Minute)
	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{busy.ID.String()}}, false, false); err != nil {
		t.Fatal(err)
	}
	got, _ := q.GetDeployedObject(ctx, busy.ID)
	if !got.RebuildRequestedAt.Valid || db.StrOrEmpty(got.ExternalRef) != "lf-web-busy" {
		t.Fatalf("busy host after rebuild: requested=%v ref=%q; want the request recorded and the reference kept", got.RebuildRequestedAt.Valid, db.StrOrEmpty(got.ExternalRef))
	}
	// The deploy finishes; the next pass carries the rebuild out.
	pool.Exec(ctx, "UPDATE task SET status = 'done' WHERE deployed_object_id = $1", busy.ID)
	pool.Exec(ctx, "UPDATE deployed_object SET status = 'running' WHERE id = $1", busy.ID)
	got, _ = q.GetDeployedObject(ctx, busy.ID)
	if _, err := reconcileObject(ctx, q, build.ID, got, got.Fingerprint, got.InfraFingerprint, false, 0, true); err != nil {
		t.Fatal(err)
	}
	if n := taskRows(t, pool, busy, "destroy_host", "pending"); n != 1 {
		t.Fatalf("recorded rebuild not carried out: %d pending destroys", n)
	}
	var ref string
	pool.QueryRow(ctx, "SELECT payload->>'external_ref' FROM task WHERE deployed_object_id = $1 AND kind = 'destroy_host'", busy.ID).Scan(&ref)
	if ref != "lf-web-busy" {
		t.Fatalf("destroy payload ref = %q, want the host's reference", ref)
	}
	got, _ = q.GetDeployedObject(ctx, busy.ID)
	if got.RebuildRequestedAt.Valid {
		t.Fatal("rebuild request not cleared once its destroy was queued")
	}

	// A deploy only queued (pending): withdrawn, and the destroy queued now.
	queued := mk("web-queued")
	openTask(t, pool, build, queued, "deploy_host", "pending", 0)
	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{queued.ID.String()}}, false, false); err != nil {
		t.Fatal(err)
	}
	if taskRows(t, pool, queued, "deploy_host", "failed") != 1 || taskRows(t, pool, queued, "destroy_host", "pending") != 1 {
		t.Fatal("a queued deploy wasn't withdrawn in favour of the rebuild's destroy")
	}
}

// TestTeardownWithdrawsQueuedDeploys: a teardown doesn't wait for a deploy
// that hasn't started (behind the whole deploy queue) only to destroy what it
// creates; it withdraws it and queues the destroy.
func TestTeardownWithdrawsQueuedDeploys(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "teardown-queued")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	o, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr("web-1")})
	pool.Exec(ctx, "UPDATE deployed_object SET status = 'pending' WHERE id = $1", o.ID)
	openTask(t, pool, build, o, "deploy_host", "pending", 0)
	pool.Exec(ctx, "UPDATE build SET status = 'tearing_down' WHERE id = $1", build.ID)
	if err := Teardown(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	if taskRows(t, pool, o, "deploy_host", "failed") != 1 || taskRows(t, pool, o, "destroy_host", "pending") != 1 {
		t.Fatal("teardown didn't withdraw the queued deploy and queue the destroy")
	}
}

// TestFailedDeploysBackOffAndStop: a deploy_failed host is retried after a
// backoff, not on every pass, and not at all after enough failures in a row
// or a permanent failure.
func TestFailedDeploysBackOffAndStop(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "deploy-backoff")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	mk := func(as string, failures int, ago time.Duration, permanent bool) db.DeployedObject {
		o, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		pool.Exec(ctx, "UPDATE deployed_object SET status = 'deploy_failed' WHERE id = $1", o.ID)
		for i := 0; i < failures; i++ {
			pool.Exec(ctx, `INSERT INTO task (build_id, deployed_object_id, kind, payload, status, attempts, last_error, permanent, created_at, updated_at)
				VALUES ($1, $2, 'deploy_host', '{}', 'failed', 3, 'cluster said no', $3, now() - $4::interval + ($5 * interval '1 second'), now() - $4::interval + ($5 * interval '1 second'))`,
				build.ID, o.ID, permanent, ago.String(), i)
		}
		o, _ = q.GetDeployedObject(ctx, o.ID)
		return o
	}
	deploys := func(o db.DeployedObject) int { return taskRows(t, pool, o, "deploy_host", "pending") }
	run := func(o db.DeployedObject) {
		if _, err := reconcileObject(ctx, q, build.ID, o, "fp", "infra", false, 0, true); err != nil {
			t.Fatal(err)
		}
	}
	soon := mk("web-soon", 1, 5*time.Second, false)
	run(soon)
	if deploys(soon) != 0 {
		t.Fatal("a failed deploy was retried inside its backoff")
	}
	later := mk("web-later", 1, time.Hour, false)
	run(later)
	if deploys(later) != 1 {
		t.Fatal("a failed deploy wasn't retried after its backoff")
	}
	many := mk("web-many", maxConsecutiveDeployFailures, time.Hour, false)
	run(many)
	if deploys(many) != 0 {
		t.Fatal("a deploy that failed too many times in a row was retried")
	}
	perm := mk("web-perm", 1, time.Hour, true)
	run(perm)
	if deploys(perm) != 0 {
		t.Fatal("a permanently failed deploy was retried")
	}
}
