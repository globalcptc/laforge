package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestDiscardStalePlannedBuilds covers the sweep that heals a backlog of
// planned builds: a planned build superseded by a newer planned build of the
// same configured build is discarded; a planned build whose configured build
// was removed is discarded; the newest planned build, and anything that is
// not planned, are untouched. Each discarded build gets a build.discarded event.
func TestDiscardStalePlannedBuilds(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	bc, err := q.CreateBuilderConfig(ctx, db.CreateBuilderConfigParams{
		Name: "planned-sweep-builder", Kind: "fake", IncusImages: []byte("{}"), IncusSizes: []byte("{}"), IncusHosts: []byte("[]"),
	})
	if err != nil {
		t.Fatalf("CreateBuilderConfig: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = $1", bc.Name) })
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-test", GithubRepo: "planned-sweep"})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })

	newCB := func(path string) db.ConfiguredBuild {
		cb, err := q.CreateConfiguredBuild(ctx, db.CreateConfiguredBuildParams{
			RepositoryID: repo.ID, Branch: "main", EnvironmentPath: path, BuilderConfigName: bc.Name,
		})
		if err != nil {
			t.Fatalf("CreateConfiguredBuild: %v", err)
		}
		return cb
	}
	newBuild := func(cb db.ConfiguredBuild, sha, status string) db.Build {
		rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: sha})
		if err != nil {
			t.Fatalf("CreateContentRevision: %v", err)
		}
		b, err := q.CreateBuild(ctx, db.CreateBuildParams{ConfiguredBuildID: cb.ID, ContentRevisionID: rev.ID, EnvironmentName: "lm-test"})
		if err != nil {
			t.Fatalf("CreateBuild: %v", err)
		}
		if status != "planned" {
			if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: b.ID, Status: status}); err != nil {
				t.Fatalf("SetBuildStatus: %v", err)
			}
		}
		time.Sleep(2 * time.Millisecond) // distinct created_at ordering
		return b
	}

	cb1 := newCB("lm-test.yaml")
	older := newBuild(cb1, "sweep-sha-1", "planned")
	live := newBuild(cb1, "sweep-sha-2", "deploying")
	newest := newBuild(cb1, "sweep-sha-3", "planned")
	cb2 := newCB("other.yaml")
	orphan := newBuild(cb2, "sweep-sha-4", "planned")
	if err := q.DeleteConfiguredBuild(ctx, cb2.ID); err != nil { // FK sets configured_build_id NULL
		t.Fatalf("DeleteConfiguredBuild: %v", err)
	}
	cb3 := newCB("third.yaml")
	lone := newBuild(cb3, "sweep-sha-5", "planned") // the only planned build of its configured build: kept

	n, err := DiscardStalePlannedBuilds(ctx, pool)
	if err != nil {
		t.Fatalf("DiscardStalePlannedBuilds: %v", err)
	}
	// The sweep is instance-wide and the test database is shared with other
	// packages' tests running at the same time, so assert on THIS test's builds
	// rather than an exact total.
	if n < 2 {
		t.Fatalf("discarded %d builds, want at least 2 (the superseded one and the orphan)", n)
	}
	want := map[pgtype.UUID]string{
		older.ID: "discarded", live.ID: "deploying", newest.ID: "planned", orphan.ID: "discarded", lone.ID: "planned",
	}
	for id, status := range want {
		b, err := q.GetBuild(ctx, id)
		if err != nil {
			t.Fatalf("GetBuild %s: %v", id, err)
		}
		if b.Status != status {
			t.Errorf("build %s status = %q, want %q", id, b.Status, status)
		}
	}
	for _, id := range []pgtype.UUID{older.ID, orphan.ID} {
		events, err := q.ListEventsByBuild(ctx, id)
		if err != nil {
			t.Fatalf("ListEventsByBuild: %v", err)
		}
		found := false
		for _, e := range events {
			if e.Kind == "build.discarded" {
				found = true
			}
		}
		if !found {
			t.Errorf("build %s: no build.discarded event recorded", id)
		}
	}

	// Idempotent: a second sweep leaves this test's builds exactly as they are.
	if _, err := DiscardStalePlannedBuilds(ctx, pool); err != nil {
		t.Fatalf("second DiscardStalePlannedBuilds: %v", err)
	}
	for id, status := range want {
		b, err := q.GetBuild(ctx, id)
		if err != nil {
			t.Fatalf("GetBuild %s: %v", id, err)
		}
		if b.Status != status {
			t.Errorf("after second sweep, build %s status = %q, want %q", id, b.Status, status)
		}
	}
	if !IsTerminalBuildStatus("discarded") || IsTerminalBuildStatus("planned") || IsTerminalBuildStatus("tearing_down") {
		t.Fatal("IsTerminalBuildStatus: discarded must be terminal; planned and tearing_down must not")
	}
}
