package orchestrator

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// TestReconcileSettlesAnUnfinishedDestroyBeforeRedeploying: an object reset
// for a rebuild (pending, its destroy task carrying the real hoster ref) is
// not redeployed over the old instance. An open destroy is waited for; a
// transiently failed one is queued again, with the same payload, once the
// backoff has passed; a permanently failed one stops the object with the
// error; a done one lets the deploy go ahead.
func TestReconcileSettlesAnUnfinishedDestroyBeforeRedeploying(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "settle-destroy")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	mk := func(as string) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	tasks := func(o db.DeployedObject, kind string) []db.Task {
		rows, _ := pool.Query(ctx, "SELECT id, build_id, deployed_object_id, kind, payload, status, attempts, lease_owner, lease_expires_at, last_error, created_at, updated_at, permanent FROM task WHERE deployed_object_id = $1 AND kind = $2 ORDER BY created_at", o.ID, kind)
		defer rows.Close()
		var out []db.Task
		for rows.Next() {
			var tk db.Task
			if err := rows.Scan(&tk.ID, &tk.BuildID, &tk.DeployedObjectID, &tk.Kind, &tk.Payload, &tk.Status, &tk.Attempts, &tk.LeaseOwner, &tk.LeaseExpiresAt, &tk.LastError, &tk.CreatedAt, &tk.UpdatedAt, &tk.Permanent); err != nil {
				t.Fatal(err)
			}
			out = append(out, tk)
		}
		return out
	}
	deploy := func(o db.DeployedObject) {
		if _, err := reconcileObject(ctx, q, build.ID, pgtype.UUID{}, o, "fp", "infra", false, 0, true); err != nil {
			t.Fatal(err)
		}
	}
	destroyTask := func(o db.DeployedObject, status string, permanent bool, updatedAgo time.Duration) {
		if _, err := pool.Exec(ctx, `INSERT INTO task (build_id, deployed_object_id, kind, payload, status, attempts, last_error, permanent, updated_at, created_at)
			VALUES ($1, $2, 'destroy_host', '{"remove":false,"external_ref":"lf-old"}', $3, 3, 'cluster said no', $4, now() - $5::interval, now() - $5::interval)`,
			build.ID, o.ID, status, permanent, updatedAgo.String()); err != nil {
			t.Fatal(err)
		}
	}

	// Open destroy: nothing deployed, and the wait is said.
	open := mk("web-open")
	destroyTask(open, "pending", false, time.Minute)
	deploy(open)
	if n := len(tasks(open, "deploy_host")); n != 0 {
		t.Fatalf("deploy queued over an open destroy (%d deploy tasks)", n)
	}
	var waits int
	pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE build_id = $1 AND deployed_object_id = $2 AND kind = 'reconcile.waiting'", build.ID, open.ID).Scan(&waits)
	if waits != 1 {
		t.Fatalf("waiting events for an open destroy = %d, want 1", waits)
	}

	// Failed, transient, backoff passed: destroyed again with the same payload; no deploy.
	retry := mk("web-retry")
	destroyTask(retry, "failed", false, time.Hour)
	deploy(retry)
	ds := tasks(retry, "destroy_host")
	if len(ds) != 2 || ds[1].Status != "pending" || string(ds[1].Payload) != string(ds[0].Payload) {
		t.Fatalf("after a failed destroy: %d destroy task(s), want a second pending one with the same payload: %+v", len(ds), ds)
	}
	if n := len(tasks(retry, "deploy_host")); n != 0 {
		t.Fatalf("deploy queued while the old instance may still exist (%d deploy tasks)", n)
	}

	// Failed, transient, still in backoff: nothing yet.
	soon := mk("web-soon")
	destroyTask(soon, "failed", false, time.Second)
	deploy(soon)
	if n := len(tasks(soon, "destroy_host")); n != 1 {
		t.Fatalf("destroy re-queued inside the backoff (%d destroy tasks)", n)
	}

	// Failed, permanent: the object is stopped with the error, nothing queued.
	perm := mk("web-perm")
	destroyTask(perm, "failed", true, time.Hour)
	deploy(perm)
	got, _ := q.GetDeployedObject(ctx, perm.ID)
	if got.Status != "deploy_failed" || got.LastError == nil || len(tasks(perm, "destroy_host")) != 1 || len(tasks(perm, "deploy_host")) != 0 {
		t.Fatalf("after a permanent destroy failure: status %s, error %v, tasks destroy=%d deploy=%d; want deploy_failed with the error and nothing queued", got.Status, got.LastError, len(tasks(perm, "destroy_host")), len(tasks(perm, "deploy_host")))
	}

	// Done: the deploy goes ahead.
	clear := mk("web-clear")
	destroyTask(clear, "done", false, time.Minute)
	deploy(clear)
	if n := len(tasks(clear, "deploy_host")); n != 1 {
		t.Fatalf("deploy after a done destroy: %d deploy tasks, want 1", n)
	}
}
