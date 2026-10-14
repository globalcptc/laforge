package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestLeaseTaskRespectsBuilderConcurrencyCap verifies the per-builder soft cap
// in LeaseTask: a builder_config with max_concurrency=N never has more than N of
// its tasks leased-and-live at once, while a builder with max_concurrency=0 is
// unbounded. This guards the join + correlated-subquery cap logic in the hot
// lease query against regressions.
func TestLeaseTaskRespectsBuilderConcurrencyCap(t *testing.T) {
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

	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	capped := "captest-capped-" + sfx
	unlimited := "captest-unlimited-" + sfx

	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	scanID := func(sql string, args ...any) pgtype.UUID {
		t.Helper()
		var id pgtype.UUID
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		return id
	}

	repo, err := q.CreateRepository(ctx, CreateRepositoryParams{GithubOwner: "captest", GithubRepo: "repo-" + sfx})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	rev, err := q.CreateContentRevision(ctx, CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "cap" + sfx})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}

	// Two builders: one capped at 1, one unlimited. The jsonb map columns carry
	// their table defaults, so a name/kind/max_concurrency insert is enough.
	exec(`INSERT INTO builder_config (name, kind, max_concurrency) VALUES ($1,'microcloud',1), ($2,'microcloud',0)`, capped, unlimited)

	const taskCount = 3
	buildFor := map[string]pgtype.UUID{}
	for _, bc := range []string{capped, unlimited} {
		cbID := scanID(`INSERT INTO configured_build (repository_id, branch, environment_path, builder_config_name) VALUES ($1,'main',$2,$3) RETURNING id`, repo.ID, "env-"+bc, bc)
		bID := scanID(`INSERT INTO build (configured_build_id, content_revision_id, environment_name) VALUES ($1,$2,'e') RETURNING id`, cbID, rev.ID)
		buildFor[bc] = bID
		exec(`INSERT INTO task (build_id, kind, status) SELECT $1,'deploy_host','pending' FROM generate_series(1,$2)`, bID, taskCount)
	}

	t.Cleanup(func() {
		// Explicit teardown in FK order so repeated runs don't collide on the
		// UNIQUE builder_config name.
		for _, bID := range buildFor {
			_, _ = pool.Exec(ctx, `DELETE FROM task WHERE build_id=$1`, bID)
			_, _ = pool.Exec(ctx, `DELETE FROM build WHERE id=$1`, bID)
		}
		_, _ = pool.Exec(ctx, `DELETE FROM configured_build WHERE repository_id=$1`, repo.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM content_revision WHERE id=$1`, rev.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM repository WHERE id=$1`, repo.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM builder_config WHERE name IN ($1,$2)`, capped, unlimited)
	})

	owner := "captest-worker"
	lease := LeaseTaskParams{LeaseOwner: &owner, Column2: pgtype.Interval{Microseconds: int64(5 * time.Minute / time.Microsecond), Valid: true}}

	// Drain the queue. The capped builder must contribute exactly one leased
	// task (it's never re-advertised while its one lease is live); the unlimited
	// builder must contribute all of its.
	leasedBy := map[string]int{}
	for i := 0; i < taskCount*2+2; i++ {
		task, err := q.LeaseTask(ctx, lease)
		if err != nil {
			break // no eligible task left (capped at cap, unlimited drained)
		}
		switch task.BuildID {
		case buildFor[capped]:
			leasedBy[capped]++
		case buildFor[unlimited]:
			leasedBy[unlimited]++
		default:
			t.Fatalf("leased a task for an unexpected build %v", task.BuildID)
		}
	}

	if leasedBy[capped] != 1 {
		t.Errorf("capped builder: leased %d tasks, want exactly 1 (max_concurrency=1)", leasedBy[capped])
	}
	if leasedBy[unlimited] != taskCount {
		t.Errorf("unlimited builder: leased %d tasks, want all %d (max_concurrency=0)", leasedBy[unlimited], taskCount)
	}
}
