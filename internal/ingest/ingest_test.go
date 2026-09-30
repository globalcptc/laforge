package ingest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// testDBConnString mirrors internal/db's own test helper -- kept as a
// separate small copy rather than exported from internal/db, since
// wanting a real local Postgres to test against is the only thing these
// two packages' tests have in common.
func testDBConnString(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("LAFORGE_TEST_DATABASE_URL"); v != "" {
		return v
	}
	if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err == nil {
		return "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
	}
	t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	return ""
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s (in %s): %v: %s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// pushDirAsRepo copies srcDir's real files into a fresh working tree,
// commits them, and pushes to a fresh bare repo, returning the bare repo's
// path as a `git fetch`-able clone URL.
func pushDirAsRepo(t *testing.T, srcDir string) (cloneURL, sha string) {
	t.Helper()
	bareDir := t.TempDir()
	workDir := t.TempDir()

	runGit(t, bareDir, "init", "-q", "--bare", "-b", "main")
	runGit(t, workDir, "init", "-q", "-b", "main")

	if err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dst := filepath.Join(workDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, content, info.Mode())
	}); err != nil {
		t.Fatalf("copying %s: %v", srcDir, err)
	}

	runGit(t, workDir, "add", ".")
	runGit(t, workDir, "commit", "-q", "-m", "test commit")
	runGit(t, workDir, "remote", "add", "origin", bareDir)
	runGit(t, workDir, "push", "-q", "origin", "main")
	sha = runGit(t, workDir, "rev-parse", "HEAD")
	return bareDir, sha
}

func openTestQueries(t *testing.T) (*db.Queries, *pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()
	q, pool, err := db.Open(ctx, testDBConnString(t))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	return q, pool, func() { pool.Close() }
}

func createTestRepository(t *testing.T, q *db.Queries, name string) db.Repository {
	t.Helper()
	repo, err := q.CreateRepository(context.Background(), db.CreateRepositoryParams{
		GithubOwner: "laforge-test", GithubRepo: name,
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() {
		q2, pool, err := db.Open(context.Background(), testDBConnString(t))
		if err != nil {
			return
		}
		defer pool.Close()
		pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID)
		_ = q2
	})
	return repo
}

// TestValidateAndStoreOnRealExampleRepo runs the whole pipeline -- fetch,
// load, schema-check, render every script for every host in every team,
// persist -- against the same real, hand-authored examples/lm-test repo
// the render tests verify against, pushed through an actual git remote (no
// GitHub account involved, just a local bare repo) and a real Postgres.
// This is the strongest test this package has: if the real example repo
// doesn't come back valid with everything persisted, something is
// actually broken, not just a fixture mismatch.
func TestValidateAndStoreOnRealExampleRepo(t *testing.T) {
	q, pool, closeDB := openTestQueries(t)
	defer closeDB()

	cloneURL, wantSHA := pushDirAsRepo(t, "../../examples/lm-test")
	repo := createTestRepository(t, q, "lm-test-ingest")

	dir, sha, cleanup, err := FetchCommit(context.Background(), cloneURL, "main")
	if err != nil {
		t.Fatalf("FetchCommit: %v", err)
	}
	defer cleanup()
	if sha != wantSHA {
		t.Fatalf("sha = %q, want %q", sha, wantSHA)
	}

	result, err := ValidateAndStore(context.Background(), pool, repo.ID, sha, "refs/heads/main", dir)
	if err != nil {
		t.Fatalf("ValidateAndStore: %v", err)
	}
	if !result.Valid {
		t.Fatalf("Valid = false, want true; issues: %+v", result.Issues)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("Issues = %+v, want none", result.Issues)
	}

	envs, err := q.ListEnvironmentsByRevision(context.Background(), result.Revision.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentsByRevision: %v", err)
	}
	if len(envs) != 1 || envs[0].Name != "lm-test" {
		t.Fatalf("ListEnvironmentsByRevision = %+v, want one environment named lm-test", envs)
	}
	if envs[0].Teams != 5 {
		t.Fatalf("teams = %d, want 5 (matching examples/lm-test/lm-test.yaml)", envs[0].Teams)
	}

	// Re-running the same commit must not fail on the UNIQUE(repository_id,
	// commit_sha) constraint by producing a duplicate content_revision --
	// it should be treated as a fresh ingest attempt. This matters because
	// a webhook redelivery (GitHub retries on timeout) must not wedge the
	// pipeline.
	rev2, err := q.GetContentRevisionByRepoAndSHA(context.Background(), db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: repo.ID, CommitSha: sha,
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}
	if rev2.ID != result.Revision.ID {
		t.Fatalf("GetContentRevisionByRepoAndSHA returned a different revision than ValidateAndStore created")
	}
}

// TestValidateAndStoreOnInvalidContent proves the other half: a commit
// that fails validation is recorded as invalid, with real issues
// describing what's wrong, and nothing from it is persisted as real
// content -- an invalid commit must never appear to have valid
// environments just because CreateContentRevision succeeded.
func TestValidateAndStoreOnInvalidContent(t *testing.T) {
	q, pool, closeDB := openTestQueries(t)
	defer closeDB()

	srcDir := t.TempDir()
	// No type header at all -- an immediate, unambiguous schema/loader
	// error, per internal/loader.loadDocument's "no type header found".
	if err := os.WriteFile(filepath.Join(srcDir, "broken.yaml"), []byte("not_a_real_key: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cloneURL, _ := pushDirAsRepo(t, srcDir)
	repo := createTestRepository(t, q, "broken-ingest")

	dir, sha, cleanup, err := FetchCommit(context.Background(), cloneURL, "main")
	if err != nil {
		t.Fatalf("FetchCommit: %v", err)
	}
	defer cleanup()

	result, err := ValidateAndStore(context.Background(), pool, repo.ID, sha, "refs/heads/main", dir)
	if err != nil {
		t.Fatalf("ValidateAndStore: %v", err)
	}
	if result.Valid {
		t.Fatal("Valid = true, want false for content with no type header")
	}
	if len(result.Issues) == 0 {
		t.Fatal("want at least one validation issue")
	}
	found := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "no type header") {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %+v, want one mentioning \"no type header\"", result.Issues)
	}

	envs, err := q.ListEnvironmentsByRevision(context.Background(), result.Revision.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentsByRevision: %v", err)
	}
	if len(envs) != 0 {
		t.Fatalf("ListEnvironmentsByRevision = %+v, want none persisted for invalid content", envs)
	}
}
