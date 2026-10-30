package orchestrator

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestScopedRebuildDoesNotRecreateTheEnvironment is the CPTC12 SoCal regression:
// rebuilding a handful of hosts took down every host in every team. Each host's
// computed fingerprint had stopped matching what it was deployed with (a cache
// bug), with no content change -- the build's revision was the same -- and the
// rebuild's flip back to deploying let reconcile recreate them all. Under an
// unchanged revision a fingerprint mismatch must never destroy anything: only
// the hosts asked for are rebuilt, and the rest are reported, not touched.
func TestScopedRebuildDoesNotRecreateTheEnvironment(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "scoped-rebuild-drift")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	// Every up host now "disagrees" with its fingerprint, same revision.
	if _, err := pool.Exec(ctx, `UPDATE deployed_object SET fingerprint = 'drifted'
		WHERE kind = 'host' AND team_id IN (SELECT id FROM team WHERE build_id = $1)`, build.ID); err != nil {
		t.Fatal(err)
	}
	target := firstHost(t, q, build)
	if !target.ContentRevisionID.Valid {
		t.Fatal("a deployed host should record the revision it was deployed from")
	}
	if _, err := Rebuild(ctx, pool, build, nil, AdHocTarget{IDs: []string{target.ID.String()}}, false, false); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID, nil); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var destroys int
	var others []string
	pool.QueryRow(ctx, `SELECT count(*) FROM task WHERE build_id = $1 AND kind = 'destroy_host' AND status = 'pending'`, build.ID).Scan(&destroys)
	rows, _ := pool.Query(ctx, `SELECT DISTINCT o.as_name FROM task t JOIN deployed_object o ON o.id = t.deployed_object_id
		WHERE t.build_id = $1 AND t.kind = 'destroy_host' AND t.status = 'pending' AND o.id <> $2`, build.ID, target.ID)
	for rows.Next() {
		var n string
		rows.Scan(&n)
		others = append(others, n)
	}
	rows.Close()
	if len(others) > 0 || destroys != 1 {
		t.Fatalf("rebuilding one host queued %d destroy(s); also destroying %v -- want only the rebuilt host", destroys, others)
	}
	var drift int
	pool.QueryRow(ctx, `SELECT count(*) FROM event WHERE build_id = $1 AND kind = 'reconcile.drift'`, build.ID).Scan(&drift)
	if drift == 0 {
		t.Fatal("the drifted hosts weren't reported (want a reconcile.drift event)")
	}
}

// TestNewRevisionStillRecreatesChangedHosts: applying new content (the build
// moves to another revision) is the real content change, and still recreates a
// host whose fingerprint differs.
func TestNewRevisionStillRecreatesChangedHosts(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "new-revision-recreates")
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)

	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: mustRepoOf(t, q, build), CommitSha: "next-commit"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.ApplyContentRevisionToBuild(ctx, db.ApplyContentRevisionToBuildParams{ID: build.ID, ContentRevisionID: rev.ID, EnvironmentName: build.EnvironmentName}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatal(err)
	}
	target := firstHost(t, q, build)
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET fingerprint = 'old-content' WHERE id = $1", target.ID); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(ctx, pool, "../../examples/lm-test", build.ID, nil); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM task WHERE deployed_object_id = $1 AND kind = 'destroy_host' AND status = 'pending'`, target.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("changed host under a new revision: %d destroy task(s), want 1", n)
	}
}

func mustRepoOf(t *testing.T, q *db.Queries, b db.Build) pgtype.UUID {
	t.Helper()
	rev, err := q.GetContentRevision(context.Background(), b.ContentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	return rev.RepositoryID
}
