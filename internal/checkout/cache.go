// Package checkout is the checkout-cache fix: "a
// checkout cache keyed by {repository, commit}," replacing "one fixed
// -repo/REPO_ROOT path for the whole process" as the answer to "where's
// this build's content on disk" -- internal/orchestrator,
// internal/runner, and internal/api's render endpoint all used to ask
// that question of one process-wide flag/env value regardless of which
// repository or commit a given build actually pointed at.
//
// Wired in additively everywhere it's used: every real caller falls back
// to its old fixed-path behavior when no Cache is configured (no GitHub
// App or service token available), so an existing single-repo dev/
// compose/test setup keeps working completely unchanged -- this package
// only starts mattering once more than one repository or commit is
// genuinely in play in the same process, exactly the case the original
// design never handled.
package checkout

import (
	"bytes"
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// Cache manages a directory of git checkouts, one per {repository,
// commit}, fetched once and reused across every later call for the same
// key -- within this process and, since population is atomic (fetch
// into a scratch dir under BaseDir, then rename into place), safely
// across multiple processes sharing BaseDir too. "Orchestrators are
// stateless replicas... more than one can run at once"
// (internal/orchestrator's own doc comment) applies to this cache the
// same way: whichever replica's rename wins for a given key, every other
// replica's redundant fetch is wasted work, never corruption, and no
// reader ever sees a partially-populated directory.
type Cache struct {
	BaseDir       string
	Q             *db.Queries
	GH            *ghclient.Client
	AppID         string
	AppPrivateKey *rsa.PrivateKey
	ServiceToken  string
}

func (c *Cache) keyDir(repositoryID pgtype.UUID, commitSHA string) string {
	return filepath.Join(c.BaseDir, repositoryID.String(), commitSHA)
}

// Dir returns a local directory containing repositoryID's content at
// commitSHA (reached via ref), fetching it first if not already cached.
// Takes cloneURL directly rather than resolving it itself, so the actual
// fetch-and-cache mechanics here are testable against a local bare
// repository with no GitHub involved at all -- the same scope
// internal/ingest.FetchCommit's own tests already use. See ForBuild for
// the real entry point, which resolves cloneURL first.
//
// Tiered fetch, so a commit still resolves even after its ref's tip has
// moved past it:
//  1. Fetch the recorded commitSHA directly, shallow. GitHub (and any
//     server with uploadpack.allowReachableSHA1InWant) serves any commit
//     reachable from an advertised ref this way, which is exactly the
//     "historical commit on a branch that has since advanced" case.
//  2. If that isn't allowed (an older/!SHA-in-want server), fall back to
//     fetching ref's FULL history and checking the commit out of it.
//  3. Only if both fail -- the commit is genuinely gone (force-pushed away
//     or GC'd), or there's no ref to fall back through -- return a clear
//     error rather than silently caching the wrong content.
// Either way the checked-out SHA is verified against commitSHA before the
// result is published to the cache.
var tokenInURL = regexp.MustCompile(`x-access-token:[^@\s]+@`)

// redactToken hides an embedded git credential in any string bound for a
// log or error.
func redactToken(s string) string {
	return tokenInURL.ReplaceAllString(s, "x-access-token:***@")
}

func (c *Cache) Dir(ctx context.Context, cloneURL string, repositoryID pgtype.UUID, ref, commitSHA string) (string, error) {
	final := c.keyDir(repositoryID, commitSHA)
	if info, err := os.Stat(final); err == nil && info.IsDir() {
		now := time.Now()
		os.Chtimes(final, now, now) // touch -- Evict's own "last used" cutoff reads this
		return final, nil
	}

	if err := os.MkdirAll(c.BaseDir, 0o755); err != nil {
		return "", fmt.Errorf("creating cache base dir: %w", err)
	}
	tmp, err := os.MkdirTemp(c.BaseDir, ".fetch-*")
	if err != nil {
		return "", fmt.Errorf("creating scratch dir: %w", err)
	}
	defer os.RemoveAll(tmp) // no-op once the rename below succeeds

	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = tmp
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			// args and stderr can contain the token embedded in the
			// clone URL (x-access-token:<token>@...) -- never let it reach
			// a log or an error surface.
			return fmt.Errorf("git %s: %w: %s", redactToken(strings.Join(args, " ")), err, redactToken(stderr.String()))
		}
		return nil
	}
	if err := run("init", "-q"); err != nil {
		return "", err
	}
	// 1. Fetch the recorded commit directly (shallow). Works on any server
	// that serves reachable SHAs (GitHub does), even after ref advanced.
	fetched := false
	if err := run("fetch", "-q", "--depth", "1", cloneURL, commitSHA); err == nil {
		if err := run("checkout", "-q", "FETCH_HEAD"); err == nil {
			fetched = true
		}
	}
	// 2. Fall back to ref's full history, then check the commit out of it --
	// for a server that won't serve a bare SHA. Needs a ref to fetch through.
	if !fetched {
		if ref == "" {
			return "", fmt.Errorf("commit %s has no recorded ref and could not be fetched directly", commitSHA)
		}
		if err := run("fetch", "-q", cloneURL, ref); err != nil {
			return "", fmt.Errorf("fetching %s: %w", ref, err)
		}
		if err := run("checkout", "-q", commitSHA); err != nil {
			return "", fmt.Errorf("commit %s not found on %q (it may have been removed by a force-push or garbage-collected): %w", commitSHA, ref, err)
		}
	}
	gotOut, err := exec.CommandContext(ctx, "git", "-C", tmp, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("resolving fetched HEAD: %w", err)
	}
	got := strings.TrimSpace(string(gotOut))
	if got != commitSHA {
		return "", fmt.Errorf("fetched %s, not the recorded commit %s -- content_revision is inconsistent with the repository", got, commitSHA)
	}

	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		if info, statErr := os.Stat(final); statErr == nil && info.IsDir() {
			// Another process (or goroutine) populated the same key
			// first -- see the Cache doc comment's own race note.
			return final, nil
		}
		return "", fmt.Errorf("publishing fetched checkout: %w", err)
	}
	return final, nil
}

// ForBuild resolves buildID's own content_revision (repository, ref,
// commit) and returns Dir for it -- the one call a real caller
// (orchestrator.Reconcile, the runner, handleRenderObject...) actually
// needs.
func (c *Cache) ForBuild(ctx context.Context, buildID pgtype.UUID) (string, error) {
	build, err := c.Q.GetBuild(ctx, buildID)
	if err != nil {
		return "", fmt.Errorf("loading build: %w", err)
	}
	rev, err := c.Q.GetContentRevision(ctx, build.ContentRevisionID)
	if err != nil {
		return "", fmt.Errorf("loading content revision: %w", err)
	}
	repo, err := c.Q.GetRepository(ctx, rev.RepositoryID)
	if err != nil {
		return "", fmt.Errorf("loading repository: %w", err)
	}
	cloneURL, err := c.resolveCloneURL(ctx, repo)
	if err != nil {
		return "", err
	}
	return c.Dir(ctx, cloneURL, repo.ID, db.StrOrEmpty(rev.Ref), rev.CommitSha)
}

// resolveCloneURL mirrors internal/api/webhook.go's own resolveCloneURL
// -- same two credentials, same order of preference (installation token,
// then GITHUB_SERVICE_TOKEN) -- but resolves which installation to use
// FROM repo.ID (GetRepositoryInstallation, written when the GitHub App
// work landed but never actually called anywhere until now) instead of
// a webhook payload's own installation id, since a background process
// has no webhook delivery to read one from. Left as its own small
// duplication of webhook.go's logic rather than a shared refactor of
// that already-tested, production credential path.
func (c *Cache) resolveCloneURL(ctx context.Context, repo db.Repository) (string, error) {
	tok, err := c.resolveToken(ctx, repo)
	if err != nil {
		return "", err
	}
	ghRepo, err := c.GH.GetRepo(ctx, tok, repo.GithubOwner, repo.GithubRepo)
	if err != nil {
		return "", fmt.Errorf("fetching %s/%s: %w", repo.GithubOwner, repo.GithubRepo, err)
	}
	return authenticatedCloneURL(ghRepo.CloneURL, tok)
}

// authenticatedCloneURL embeds a token into an https clone URL as
// x-access-token (GitHub's scheme for both installation tokens and PATs),
// so a private repository's git fetch authenticates -- resolving the token
// isn't enough on its own, the fetch has to actually present it. A non-token
// or non-https URL is returned unchanged.
func authenticatedCloneURL(cloneURL, token string) (string, error) {
	if token == "" {
		return cloneURL, nil
	}
	u, err := url.Parse(cloneURL)
	if err != nil {
		return "", fmt.Errorf("parsing clone URL: %w", err)
	}
	if u.Scheme != "https" {
		return cloneURL, nil
	}
	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
}

// resolveToken is resolveCloneURL's own credential half, split out so a
// caller that needs a real token for other GitHub API calls against this
// repository (listing branches, for instance -- see internal/api's
// handleListBranches) doesn't have to duplicate this same installation-
// token-then-service-token precedence.
func (c *Cache) resolveToken(ctx context.Context, repo db.Repository) (string, error) {
	if c.AppID != "" && c.AppPrivateKey != nil {
		inst, err := c.Q.GetRepositoryInstallation(ctx, repo.ID)
		switch {
		case err == nil:
			jwt, err := ghclient.GenerateAppJWT(c.AppID, c.AppPrivateKey, time.Now())
			if err != nil {
				return "", fmt.Errorf("signing app jwt: %w", err)
			}
			tok, err := c.GH.CreateInstallationToken(ctx, jwt, inst.InstallationID)
			if err != nil {
				return "", fmt.Errorf("minting installation token: %w", err)
			}
			return tok.Token, nil
		case !errors.Is(err, pgx.ErrNoRows):
			return "", fmt.Errorf("looking up installation for %s/%s: %w", repo.GithubOwner, repo.GithubRepo, err)
		}
		// pgx.ErrNoRows: no installation on record -- fall through.
	}
	if c.ServiceToken != "" {
		return c.ServiceToken, nil
	}
	return "", fmt.Errorf("cannot resolve a token for %s/%s: no installation on record and no service token configured", repo.GithubOwner, repo.GithubRepo)
}

// ResolveToken exposes resolveToken to callers outside this package that
// need a real token for a repository this cache already knows how to
// authenticate for, but aren't asking for a checkout directory (e.g.
// listing a repository's real branches through the GitHub API).
func (c *Cache) ResolveToken(ctx context.Context, repo db.Repository) (string, error) {
	return c.resolveToken(ctx, repo)
}

// ForRef resolves repositoryID's current clone URL/credentials and
// fetches ref's current tip into a real checkout -- the same clone-URL/
// credential precedence ForBuild uses, but for "this repository's given
// branch, whatever commit it's at right now" rather than one already-
// recorded content_revision. Used by handlers that need real content
// before any content_revision exists for it yet -- listing a branch's
// available environment files, when configuring a new build.
func (c *Cache) ForRef(ctx context.Context, repositoryID pgtype.UUID, ref string) (string, error) {
	repo, err := c.Q.GetRepository(ctx, repositoryID)
	if err != nil {
		return "", fmt.Errorf("loading repository: %w", err)
	}
	tok, err := c.resolveToken(ctx, repo)
	if err != nil {
		return "", err
	}
	ghRepo, err := c.GH.GetRepo(ctx, tok, repo.GithubOwner, repo.GithubRepo)
	if err != nil {
		return "", fmt.Errorf("fetching %s/%s: %w", repo.GithubOwner, repo.GithubRepo, err)
	}
	branch, err := c.GH.GetBranch(ctx, tok, repo.GithubOwner, repo.GithubRepo, ref)
	if err != nil {
		return "", fmt.Errorf("fetching branch %s: %w", ref, err)
	}
	authURL, err := authenticatedCloneURL(ghRepo.CloneURL, tok)
	if err != nil {
		return "", err
	}
	return c.Dir(ctx, authURL, repositoryID, ref, branch.Commit.SHA)
}

// Evict removes cached checkout directories under BaseDir that haven't
// been asked for (Dir's own os.Chtimes touch on every hit) since before
// cutoff -- the same age-based retention style
// DeleteAgentHeartbeatsOlderThan already uses for agent_heartbeat,
// deliberately no fancier LRU/size accounting. Returns how many were
// removed.
func (c *Cache) Evict(cutoff time.Time) (int, error) {
	repoDirs, err := os.ReadDir(c.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, rd := range repoDirs {
		if !rd.IsDir() || strings.HasPrefix(rd.Name(), ".") {
			continue
		}
		repoPath := filepath.Join(c.BaseDir, rd.Name())
		commitDirs, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}
		for _, cd := range commitDirs {
			info, err := cd.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				if err := os.RemoveAll(filepath.Join(repoPath, cd.Name())); err == nil {
					removed++
				}
			}
		}
		os.Remove(repoPath) // best-effort: only succeeds once actually empty
	}
	return removed, nil
}
