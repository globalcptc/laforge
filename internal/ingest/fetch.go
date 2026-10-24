// Package ingest is the "per-commit validation": given a git
// remote and a ref, fetch exactly that commit, run it through the same
// loader/schema/render engine `laforge check` uses, and persist the
// result to Postgres. It's the one piece of code both the webhook handler
// and a future `laforge validate-commit`-style CLI command call, so the
// server and a local checkout can never disagree about what a commit
// means -- the plan's own promise for `laforge check`, extended to cover
// the server side too.
package ingest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// tokenInURL matches the x-access-token credential a private-repo clone URL
// embeds, so it's never printed in an error or log. Mirrors the same
// redaction internal/checkout applies to its own git command errors.
var tokenInURL = regexp.MustCompile(`x-access-token:[^@\s]+@`)

func redactToken(s string) string {
	return tokenInURL.ReplaceAllString(s, "x-access-token:***@")
}

// FetchCommit shallow-fetches ref from cloneURL into a fresh temporary
// directory and checks it out, returning the directory, the commit SHA
// actually checked out, and a cleanup function the caller must run.
//
// It fetches by ref rather than by raw SHA on purpose: GitHub (and most
// git servers) only allow fetching an arbitrary SHA that isn't currently a
// ref tip when `uploadpack.allowReachableSHA1InWant` is enabled, which
// isn't guaranteed, while fetching a branch/tag ref always works. A push
// webhook always names both the ref and the SHA it just moved that ref to,
// so fetching the ref and then checking the resulting SHA against what the
// caller expected (see ValidateAndStore) gets the same guarantee -- "this
// is really the commit that was pushed" -- without needing SHA-addressed
// fetch support from the remote.
//
// cloneURL is whatever `git fetch` accepts: a plain https:// GitHub URL
// for a public repo, an https://x-access-token:<token>@... URL for a
// private one, or a local path/file:// URL, which is exactly how this is
// exercised in fetch_test.go without needing a live GitHub account.
func FetchCommit(ctx context.Context, cloneURL, ref string) (dir, sha string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "laforge-ingest-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(dir) }

	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %s: %w: %s", redactToken(strings.Join(args, " ")), err, redactToken(strings.TrimSpace(stderr.String())))
		}
		return nil
	}

	if err := run("init", "-q"); err != nil {
		cleanup()
		return "", "", nil, err
	}
	if err := run("fetch", "--depth", "1", "--quiet", cloneURL, ref); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("fetching %s from %s: %w", ref, redactToken(cloneURL), err)
	}
	if err := run("checkout", "-q", "FETCH_HEAD"); err != nil {
		cleanup()
		return "", "", nil, err
	}

	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "FETCH_HEAD").Output()
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("resolving FETCH_HEAD: %w", err)
	}
	sha = strings.TrimSpace(string(out))
	return dir, sha, cleanup, nil
}
