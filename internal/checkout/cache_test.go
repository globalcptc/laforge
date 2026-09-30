package checkout

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// randUUID is a throwaway key for Dir/Evict tests, which only need SOME
// stable repository identifier to namespace the cache directory under --
// they don't touch Postgres at all, unlike ForBuild's own tests.
func randUUID(t *testing.T) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if _, err := rand.Read(u.Bytes[:]); err != nil {
		t.Fatal(err)
	}
	u.Valid = true
	return u
}

// testDBConnString mirrors every other package's own small copy (see
// internal/ingest/ingest_test.go's identical helper and its own doc
// comment on why this isn't shared).
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

func testPool(t *testing.T) (*db.Queries, *pgxpool.Pool) {
	t.Helper()
	q, pool, err := db.Open(context.Background(), testDBConnString(t))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	return q, pool
}

// newTestRepo creates a real bare git repo plus a working checkout,
// commits one file to it on branch "main", and returns the bare repo's
// path (usable as a `git fetch` clone URL with no network or GitHub
// account involved) and the SHA of the commit it made -- the exact
// fixture internal/ingest/fetch_test.go's own newTestRepo uses, copied
// rather than shared (see testDBConnString's own doc comment on why).
func newTestRepo(t *testing.T, files map[string]string) (cloneURL, sha string) {
	t.Helper()
	bareDir := t.TempDir()
	workDir := t.TempDir()

	run := func(dir string, args ...string) string {
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

	run(bareDir, "init", "-q", "--bare", "-b", "main")
	run(workDir, "init", "-q", "-b", "main")
	for path, content := range files {
		full := filepath.Join(workDir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run(workDir, "add", ".")
	run(workDir, "commit", "-q", "-m", "test commit")
	run(workDir, "remote", "add", "origin", bareDir)
	run(workDir, "push", "-q", "origin", "main")
	sha = run(workDir, "rev-parse", "HEAD")
	return bareDir, sha
}

func pushAnotherCommit(t *testing.T, cloneURL, extraFile, extraContent string) (sha string) {
	t.Helper()
	workDir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("clone", "-q", cloneURL, ".")
	if err := os.WriteFile(filepath.Join(workDir, extraFile), []byte(extraContent), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "second commit")
	run("push", "-q", "origin", "main")
	return run("rev-parse", "HEAD")
}

// TestDirFetchesAndCaches is Dir's own core proof: a cache miss fetches
// the real content; a second call for the same key reuses the cached
// directory rather than fetching again (proven by removing the origin
// repo out from under it -- a fresh fetch would fail, so success proves
// reuse).
func TestDirFetchesAndCaches(t *testing.T) {
	cloneURL, sha := newTestRepo(t, map[string]string{"README.md": "hello\n"})
	c := &Cache{BaseDir: t.TempDir()}
	repoID := randUUID(t)

	dir, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha)
	if err != nil {
		t.Fatalf("Dir (miss): %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || string(content) != "hello\n" {
		t.Fatalf("README.md = %q, %v, want %q, nil", content, err, "hello\n")
	}

	// Make the origin unreachable -- a real fetch would now fail.
	if err := os.RemoveAll(cloneURL); err != nil {
		t.Fatal(err)
	}
	dir2, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha)
	if err != nil {
		t.Fatalf("Dir (hit, origin gone): %v -- should have reused the cache, not re-fetched", err)
	}
	if dir2 != dir {
		t.Fatalf("Dir (hit) = %q, want the same path as the miss, %q", dir2, dir)
	}
}

// TestDirTwoCommitsGetTwoDirectories confirms the cache key is really
// {repository, commit}, not just {repository}: two different commits of
// the same repo must each get their own cached directory, both present
// at once.
func TestDirTwoCommitsGetTwoDirectories(t *testing.T) {
	cloneURL, sha1 := newTestRepo(t, map[string]string{"f.txt": "one\n"})
	c := &Cache{BaseDir: t.TempDir()}
	repoID := randUUID(t)

	// Fetch sha1 while it's still main's own tip -- pushing a second
	// commit afterward must not disturb the copy already cached for it.
	dir1, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha1)
	if err != nil {
		t.Fatalf("Dir (sha1): %v", err)
	}
	sha2 := pushAnotherCommit(t, cloneURL, "g.txt", "two\n")
	dir2, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha2)
	if err != nil {
		t.Fatalf("Dir (sha2): %v", err)
	}
	if dir1 == dir2 {
		t.Fatalf("two different commits cached at the same directory %q", dir1)
	}
	if _, err := os.Stat(filepath.Join(dir1, "f.txt")); err != nil {
		t.Fatalf("dir1 should still have f.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir2, "g.txt")); err != nil {
		t.Fatalf("dir2 should have g.txt: %v", err)
	}
}

// TestDirFetchesACommitTheRefMovedPast covers the
// residual gap, now closed: a commit a content_revision recorded is still
// fetchable after ref advanced past it, as long as it remains reachable
// (the ordinary "branch got more commits since this build was captured"
// case). The tiered fetch resolves it and caches exactly that commit's
// content, not the newer tip's.
func TestDirFetchesACommitTheRefMovedPast(t *testing.T) {
	cloneURL, sha1 := newTestRepo(t, map[string]string{"f.txt": "one\n"})
	pushAnotherCommit(t, cloneURL, "g.txt", "two\n") // main now points past sha1
	c := &Cache{BaseDir: t.TempDir()}
	repoID := randUUID(t)

	dir, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha1)
	if err != nil {
		t.Fatalf("expected the historical commit to resolve, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "f.txt")); err != nil {
		t.Fatalf("checkout should be at sha1 (f.txt present): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "g.txt")); err == nil {
		t.Fatal("checkout should be at sha1, not the newer tip -- g.txt must not be present")
	}
}

// TestDirRejectsAnUnreachableCommit is the genuine, still-honest failure: a
// commit that no ref can reach (force-pushed away / GC'd, stood in for here
// by a SHA that was never in the repo) fails clearly rather than silently
// caching the wrong content.
func TestDirRejectsAnUnreachableCommit(t *testing.T) {
	cloneURL, _ := newTestRepo(t, map[string]string{"f.txt": "one\n"})
	c := &Cache{BaseDir: t.TempDir()}
	const bogus = "0123456789012345678901234567890123456789"
	if _, err := c.Dir(context.Background(), cloneURL, randUUID(t), "main", bogus); err == nil {
		t.Fatal("expected an error for a commit no ref can reach")
	}
}

func TestDirRejectsEmptyRef(t *testing.T) {
	c := &Cache{BaseDir: t.TempDir()}
	if _, err := c.Dir(context.Background(), "unused", randUUID(t), "", "deadbeef"); err == nil {
		t.Fatal("expected an error for a commit with no recorded ref")
	}
}

// TestEvictRemovesOnlyStaleEntries proves the age-based retention: an
// entry touched (via a real Dir hit) after cutoff survives; one whose
// mtime predates cutoff is removed.
func TestEvictRemovesOnlyStaleEntries(t *testing.T) {
	cloneURL, sha1 := newTestRepo(t, map[string]string{"f.txt": "one\n"})
	c := &Cache{BaseDir: t.TempDir()}
	repoID := randUUID(t)

	// Fetch sha1 while it's still main's own tip, same reasoning as
	// TestDirTwoCommitsGetTwoDirectories.
	freshDir, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha1)
	if err != nil {
		t.Fatalf("Dir (sha1): %v", err)
	}
	sha2 := pushAnotherCommit(t, cloneURL, "g.txt", "two\n")
	staleDir, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha2)
	if err != nil {
		t.Fatalf("Dir (sha2): %v", err)
	}
	// Backdate the "stale" entry's mtime directly, then re-touch the
	// "fresh" one via a real cache hit, so Evict's cutoff falls cleanly
	// between the two.
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(staleDir, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Dir(context.Background(), cloneURL, repoID, "main", sha1); err != nil {
		t.Fatalf("re-touching fresh dir: %v", err)
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	n, err := c.Evict(cutoff)
	if err != nil {
		t.Fatalf("Evict: %v", err)
	}
	if n != 1 {
		t.Fatalf("Evict removed %d entries, want 1", n)
	}
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatalf("fresh dir should survive Evict: %v", err)
	}
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("stale dir should have been removed by Evict, stat err = %v", err)
	}
}

// TestForBuildResolvesAndFetchesRealContent is ForBuild's own end-to-end
// proof: a real repository/content_revision/build row in Postgres, a
// fake GitHub server standing in for the real API (GET /repos/{owner}/{repo}
// returning a clone_url that's actually the local bare repo from
// newTestRepo -- no real GitHub account needed, same technique
// internal/api's own webhook tests use), and GITHUB_SERVICE_TOKEN as the
// credential path (the simpler of the two ResolveCloneURL supports --
// the installation-token path reuses the exact same ghclient calls
// internal/ghclient's own tests already cover, so it isn't re-proven
// here).
func TestForBuildResolvesAndFetchesRealContent(t *testing.T) {
	q, pool := testPool(t)
	ctx := context.Background()
	cloneURL, sha := newTestRepo(t, map[string]string{"README.md": "hello\n"})

	gh := ghclient.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/laforge-test/checkout-cache-test" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"clone_url": cloneURL})
	}))
	defer srv.Close()
	gh.APIBaseURL = srv.URL

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{
		GithubOwner: "laforge-test", GithubRepo: "checkout-cache-test",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		RepositoryID: repo.ID, CommitSha: sha, Ref: strPtr("main"),
	})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: rev.ID, EnvironmentName: "checkout-cache-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}

	c := &Cache{BaseDir: t.TempDir(), Q: q, GH: gh, ServiceToken: "test-token"}
	dir, err := c.ForBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ForBuild: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || string(content) != "hello\n" {
		t.Fatalf("README.md = %q, %v, want %q, nil", content, err, "hello\n")
	}
}

func strPtr(s string) *string { return &s }
