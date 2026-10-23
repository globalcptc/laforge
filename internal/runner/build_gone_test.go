package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

// TestTasksOfAGoneBuildAreCancelled: a deploy or access task whose build has
// been torn down is cancelled -- failed for good, with the reason logged --
// instead of run. Access tasks deferred while hosts were rebuilt used to keep
// waking every 30 seconds for a build torn down days earlier.
func TestTasksOfAGoneBuildAreCancelled(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "gone-build-tasks")
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "torn_down"}); err != nil {
		t.Fatal(err)
	}
	// A deferred access task: leased by an earlier runner, its lease expired.
	var taskID string
	if err := pool.QueryRow(ctx, `INSERT INTO task (build_id, kind, payload, status, lease_owner, lease_expires_at, last_error)
		VALUES ($1, 'configure_external_access', '{"team":"1","fingerprint":"x"}', 'leased', 'old-runner', now() - interval '1 second',
		        'instance is not running yet: public target vdi-windows has not completed deployment') RETURNING id::text`, build.ID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	r := &Runner{Pool: pool, Builder: fake.New(pool), ID: "gone-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: time.Second, BuildID: &build.ID}
	worked, err := r.LeaseAndExecuteOne(ctx)
	if err != nil || !worked {
		t.Fatalf("LeaseAndExecuteOne = %v, %v; want the task leased and handled", worked, err)
	}
	var status, lastErr string
	var permanent bool
	pool.QueryRow(ctx, "SELECT status, last_error, permanent FROM task WHERE id = $1", taskID).Scan(&status, &lastErr, &permanent)
	if status != "failed" || !permanent {
		t.Fatalf("task after its build was torn down: status %s permanent %v (%s); want cancelled for good", status, permanent, lastErr)
	}
	var logged int
	pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE task_id = $1 AND kind = 'task.cancelled'", taskID).Scan(&logged)
	if logged != 1 {
		t.Fatalf("cancellation logged %d time(s), want 1", logged)
	}
	if worked, _ := r.LeaseAndExecuteOne(ctx); worked {
		t.Fatal("a cancelled task was leased again")
	}
}

func TestAccessWaitBacksOff(t *testing.T) {
	first := 30 * time.Second
	for _, c := range []struct{ waited, want time.Duration }{
		{0, first},
		{time.Minute, first},
		{20 * time.Minute, 2 * time.Minute},
		{8 * 24 * time.Hour, 10 * time.Minute},
	} {
		if got := accessWaitBackoff(first, c.waited); got != c.want {
			t.Errorf("after waiting %s: next check in %s, want %s", c.waited, got, c.want)
		}
	}
}
