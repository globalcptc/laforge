package ingest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestRepo creates a real bare git repo plus a working checkout, commits
// one file to it on branch "main", and returns the bare repo's path (usable
// as a `git fetch` clone URL with no network or GitHub account involved)
// and the SHA of the commit it made.
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

func TestFetchCommit(t *testing.T) {
	cloneURL, wantSHA := newTestRepo(t, map[string]string{
		"README.md": "hello\n",
	})

	dir, sha, cleanup, err := FetchCommit(context.Background(), cloneURL, "main")
	if err != nil {
		t.Fatalf("FetchCommit: %v", err)
	}
	defer cleanup()

	if sha != wantSHA {
		t.Fatalf("sha = %q, want %q", sha, wantSHA)
	}
	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("reading fetched file: %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("README.md content = %q, want %q", content, "hello\n")
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("fetched dir should exist before cleanup: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("cleanup should have removed %s, stat err = %v", dir, err)
	}
}

func TestFetchCommitUnknownRefFails(t *testing.T) {
	cloneURL, _ := newTestRepo(t, map[string]string{"f.txt": "x"})

	_, _, cleanup, err := FetchCommit(context.Background(), cloneURL, "no-such-branch")
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatal("expected an error fetching a ref that doesn't exist")
	}
}
