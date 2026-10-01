package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
)

// deployToConvergence drives Reconcile + a fake-builder runner to a fixed
// point: loop (create deploy tasks, then execute them, bringing objects up)
// until no object is still pending or deploying. Boxes deploy ahead of their
// dependencies now, so this typically converges in a single pass.
func deployToConvergence(t *testing.T, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	r := &runner.Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: repoRoot,
		ID: "convergence-runner", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &buildID,
	}
	for pass := 0; pass < 10; pass++ {
		if err := Reconcile(ctx, pool, repoRoot, buildID); err != nil {
			t.Fatalf("Reconcile (pass %d): %v", pass, err)
		}
		for {
			worked, err := r.LeaseAndExecuteOne(ctx)
			if err != nil {
				t.Fatalf("LeaseAndExecuteOne: %v", err)
			}
			if !worked {
				break
			}
		}
		objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
		if err != nil {
			t.Fatalf("ListDeployedObjectsByBuild: %v", err)
		}
		pending := 0
		for _, o := range objs {
			if o.Status == "pending" || o.Status == "deploying" {
				pending++
			}
		}
		if pending == 0 {
			return
		}
	}
	t.Fatal("deploy did not converge within 10 passes")
}

// newTestBuildWithFakeBuilder mirrors newTestBuild (reconcile_test.go) but also
// wires a real configured_build -> builder_config(kind=fake) chain, so
// checkBuilderCompatibility/resolveBuilderImages resolve a real builder instead
// of taking the "no configured_build, nothing to check" ad-hoc path every other
// reconcile_test.go helper uses. Used by tests that need a resolvable builder.
func newTestBuildWithFakeBuilder(t *testing.T, pool *pgxpool.Pool, name string) db.Build {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)

	bc, err := q.CreateBuilderConfig(ctx, db.CreateBuilderConfigParams{
		Name: name + "-builder", Kind: "fake", IncusImages: []byte("{}"), IncusSizes: []byte("{}"), IncusHosts: []byte("[]"),
	})
	if err != nil {
		t.Fatalf("CreateBuilderConfig: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = $1", bc.Name) })

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-test", GithubRepo: name})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })

	cb, err := q.CreateConfiguredBuild(ctx, db.CreateConfiguredBuildParams{
		RepositoryID: repo.ID, Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: bc.Name,
	})
	if err != nil {
		t.Fatalf("CreateConfiguredBuild: %v", err)
	}

	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: "orchestrator-fakebuild-test-sha",
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ConfiguredBuildID: cb.ID, ContentRevisionID: rev.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"
	return build
}
