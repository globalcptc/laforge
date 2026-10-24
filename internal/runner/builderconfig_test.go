package runner

import (
	"context"
	"testing"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

// TestResolveBuilderFromDBFallsBackForAdHocBuild is the "no
// configured_build at all" case -- every existing test's newTestBuild
// helper, and every real build created before builder_config existed.
func TestResolveBuilderFromDBFallsBackForAdHocBuild(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "resolve-builder-adhoc")

	fallback := fake.New(pool)
	r := &Runner{Pool: pool, Builder: fallback}
	got, err := r.ResolveBuilderFromDB(ctx, db.Task{BuildID: build.ID})
	if err != nil {
		t.Fatalf("ResolveBuilderFromDB: %v", err)
	}
	if got != fallback {
		t.Fatalf("ResolveBuilderFromDB (ad-hoc build) = %v, want the fallback r.Builder", got)
	}
}

// TestResolveBuilderFromDBResolvesRealBuilderConfig is the real path:
// configured_build names a real builder_config by name, and
// ResolveBuilderFromDB must find it -- proving the whole chain
// (task -> build -> configured_build -> builder_config -> a real
// builder.Builder), the actual fix for "what about the MicroCloud
// builder."
func TestResolveBuilderFromDBResolvesRealBuilderConfig(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-test", GithubRepo: "resolve-builder-real"})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "resolve-builder-real-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	cb, err := q.CreateConfiguredBuild(ctx, db.CreateConfiguredBuildParams{
		RepositoryID: repo.ID, Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "resolve-builder-real-config",
	})
	if err != nil {
		t.Fatalf("CreateConfiguredBuild: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ConfiguredBuildID: cb.ID, ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}

	if _, err := q.CreateBuilderConfig(ctx, db.CreateBuilderConfigParams{
		Name: "resolve-builder-real-config", Kind: "fake",
		IncusImages: []byte("{}"), IncusSizes: []byte("{}"), IncusHosts: []byte("[]"),
	}); err != nil {
		t.Fatalf("CreateBuilderConfig: %v", err)
	}
	t.Cleanup(func() {
		q.DeleteBuilderConfig(context.Background(), "resolve-builder-real-config")
	})

	r := &Runner{Pool: pool}
	got, err := r.ResolveBuilderFromDB(ctx, db.Task{BuildID: build.ID})
	if err != nil {
		t.Fatalf("ResolveBuilderFromDB: %v", err)
	}
	if _, ok := got.(*fake.Builder); !ok {
		t.Fatalf("ResolveBuilderFromDB = %T, want *fake.Builder (resolved from the real builder_config row)", got)
	}
}

// TestResolveBuilderFromDBFailsClearlyForAMissingBuilderConfig proves
// the real, named error this closes -- item 13's own "no such builder
// config" case -- instead of silently falling back to anything.
func TestResolveBuilderFromDBFailsClearlyForAMissingBuilderConfig(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-test", GithubRepo: "resolve-builder-missing"})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "resolve-builder-missing-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	cb, err := q.CreateConfiguredBuild(ctx, db.CreateConfiguredBuildParams{
		RepositoryID: repo.ID, Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "no-such-builder-config-at-all",
	})
	if err != nil {
		t.Fatalf("CreateConfiguredBuild: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ConfiguredBuildID: cb.ID, ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}

	r := &Runner{Pool: pool}
	if _, err := r.ResolveBuilderFromDB(ctx, db.Task{BuildID: build.ID}); err == nil {
		t.Fatal("expected a clear error for a builder_config_name with no matching row")
	}
}
