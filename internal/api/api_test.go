package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

const webhookSecret = "test-webhook-secret"

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

// fakeGitHub simulates just enough of the real GitHub REST API for these
// tests: repo permission lookups keyed by bearer token, and the
// CI-status/check-runs/create-status endpoints the reconciler calls. It's
// a real HTTP server (httptest), so ghclient talks to it exactly as it
// would talk to api.github.com -- nothing about ghclient's own code is
// mocked, only the network endpoint it points at.
type fakeGitHub struct {
	// pushTokens is the set of bearer tokens that get push permission on
	// any repo they ask about.
	pushTokens map[string]bool
	// adminTokens get GitHub admin on any repo they ask about.
	adminTokens   map[string]bool
	combinedState string // returned by GetCombinedStatus; "" -> no statuses at all
	checkRuns     []ghclient.CheckRun
	statusPosts   []map[string]string
	// cloneURL is returned as every repo's clone_url from GetRepo --
	// since webhook.go's resolveCloneURL no longer reads
	// clone_url off the webhook payload at all, this is now the ONLY
	// place a test's real local bare repo path reaches the server;
	// setupAPITest sets it once, right after pushDirAsRepo creates that
	// repo.
	cloneURL string
	// denyAllRepoAccess flips GetRepo's response to Push:false,
	// Pull:false for every repo, regardless of pushTokens -- used by
	// TestSignInRejectsAnUnauthorizedGitHubAccount, which needs "this
	// identity has no real access anywhere" to hold even though the
	// `repository` table is a shared local dev Postgres also touched by
	// other packages' tests running concurrently under `go test ./...`;
	// depending on the table being globally empty at that instant is a
	// real flake (found live), so the test controls the one thing it
	// actually owns -- what its own fake server reports -- instead.
	denyAllRepoAccess bool
	// branches lets a test control what ListBranches/GetBranch return;
	// nil defaults to a single "main" branch, enough for tests that don't
	// care about the real list.
	branches []string
	// installationRepos lets a test control what
	// GET /installation/repositories returns -- an "all repositories"
	// install's real repo list (see webhook.go's
	// listAllRepositoriesForInstallation).
	installationRepos []ghclient.Repo
	// collaborators is what GET /repos/{owner}/{repo}/collaborators
	// returns, keyed by "owner/repo"; a missing key is a 404.
	collaborators map[string][]ghclient.Collaborator
}

func (f *fakeGitHub) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Count(r.URL.Path, "/") == 3 && strings.HasPrefix(r.URL.Path, "/repos/"):
			// exactly "/repos/{owner}/{repo}" (3 slashes) -- the more
			// specific suffix-matched cases below (status/check-runs/
			// statuses) also start with "/repos/" but have more path
			// segments, so they're distinguished by slash count here
			// rather than needing to be listed before this case.
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			admin := f.adminTokens[tok] && !f.denyAllRepoAccess
			push := (f.pushTokens[tok] || admin) && !f.denyAllRepoAccess
			pull := !f.denyAllRepoAccess
			json.NewEncoder(w).Encode(ghclient.Repo{
				FullName:      strings.TrimPrefix(r.URL.Path, "/repos/"),
				DefaultBranch: "main",
				CloneURL:      f.cloneURL,
				Permissions:   ghclient.RepoPermissions{Admin: admin, Push: push, Pull: pull},
			})
		case strings.HasSuffix(r.URL.Path, "/collaborators") && strings.HasPrefix(r.URL.Path, "/repos/"):
			full := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/repos/"), "/collaborators")
			collabs, ok := f.collaborators[full]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(collabs)
		case r.URL.Path == "/installation/repositories":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": len(f.installationRepos), "repositories": f.installationRepos,
			})
		case strings.Contains(r.URL.Path, "/branches/"):
			// GetBranch: /repos/{owner}/{repo}/branches/{branch} -- a
			// deterministic fake sha keyed by name, so a test can assert
			// on it without needing a real commit.
			name := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": name, "commit": map[string]string{"sha": "fake-sha-" + name},
			})
		case strings.HasSuffix(r.URL.Path, "/branches"):
			// ListBranches
			names := f.branches
			if names == nil {
				names = []string{"main"}
			}
			out := make([]map[string]interface{}, 0, len(names))
			for _, n := range names {
				out = append(out, map[string]interface{}{"name": n, "commit": map[string]string{"sha": "fake-sha-" + n}})
			}
			json.NewEncoder(w).Encode(out)
		case strings.HasSuffix(r.URL.Path, "/access_tokens") && strings.HasPrefix(r.URL.Path, "/app/installations/"):
			// CreateInstallationToken -- doesn't check the app JWT's
			// signature (that's ghclient's own responsibility, proven
			// for real in internal/ghclient/app_test.go); this only
			// needs to hand back a token GetRepo can then be called
			// with, so resolveCloneURL's installation-token path can be
			// exercised end to end against this same fake server.
			json.NewEncoder(w).Encode(map[string]string{
				"token": "fake-installation-token", "expires_at": "2099-01-01T00:00:00Z",
			})
		case strings.HasSuffix(r.URL.Path, "/status"):
			state := f.combinedState
			total := 1
			if state == "" {
				state, total = "pending", 0
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"state": state, "total_count": total})
		case strings.HasSuffix(r.URL.Path, "/check-runs"):
			json.NewEncoder(w).Encode(map[string]interface{}{"total_count": len(f.checkRuns), "check_runs": f.checkRuns})
		case strings.Contains(r.URL.Path, "/statuses/"):
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			f.statusPosts = append(f.statusPosts, body)
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/user":
			// GetAuthenticatedUser -- who a token belongs to, keyed by
			// the same pushTokens map so "logged in as" matches whichever
			// fake identity a test's token represents.
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			id, login := f.identityForToken(tok)
			json.NewEncoder(w).Encode(ghclient.User{ID: id, Login: login, AvatarURL: "https://avatars.example/" + login})
		case strings.HasPrefix(r.URL.Path, "/users/"):
			// GetUserByLogin -- resolving an arbitrary username to an id,
			// for granting repository_access by login.
			login := strings.TrimPrefix(r.URL.Path, "/users/")
			json.NewEncoder(w).Encode(ghclient.User{ID: f.idForLogin(login), Login: login, AvatarURL: "https://avatars.example/" + login})
		case r.URL.Path == "/login/oauth/access_token":
			// ExchangeCode -- the web flow's code-for-token swap. The
			// fake "code" this test suite uses IS the token it wants
			// back, so a test can assert on a value it chose itself.
			r.ParseForm()
			json.NewEncoder(w).Encode(map[string]string{"access_token": "exchanged-" + r.PostForm.Get("code")})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// identityForToken/idForLogin give oauth.go's UpsertAccount something
// deterministic and collision-free to key on across a whole test run,
// without needing a real GitHub account anywhere.
func (f *fakeGitHub) identityForToken(tok string) (id int64, login string) {
	if tok == "" {
		return 0, "anonymous"
	}
	return f.idForLogin(tok), tok
}
func (f *fakeGitHub) idForLogin(login string) int64 {
	var sum int64
	for _, c := range login {
		sum = sum*31 + int64(c)
	}
	if sum < 0 {
		sum = -sum
	}
	return sum + 1
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

// pushDirAsRepo copies srcDir's files into a fresh working tree, commits,
// and pushes to a fresh bare repo -- a real git remote, no GitHub account
// needed, reused (and re-pushed to) across a test's several "commits."
func pushDirAsRepo(t *testing.T, srcDir string) (cloneURL, sha string) {
	t.Helper()
	bareDir := t.TempDir()
	workDir := t.TempDir()
	runGit(t, bareDir, "init", "-q", "--bare", "-b", "main")
	runGit(t, workDir, "init", "-q", "-b", "main")
	copyTree(t, srcDir, workDir)
	runGit(t, workDir, "add", ".")
	runGit(t, workDir, "commit", "-q", "-m", "initial")
	runGit(t, workDir, "remote", "add", "origin", bareDir)
	runGit(t, workDir, "push", "-q", "origin", "main")
	sha = runGit(t, workDir, "rev-parse", "HEAD")
	return bareDir, sha
}

// pushAnotherCommit adds one more real commit (a trivial content tweak)
// to an existing bare repo made by pushDirAsRepo, for testing follow-mode
// reconciliation across more than one push.
func pushAnotherCommit(t *testing.T, cloneURL, srcDir, extraFile, extraContent string) (sha string) {
	t.Helper()
	workDir := t.TempDir()
	runGit(t, workDir, "clone", "-q", cloneURL, ".")
	if err := os.WriteFile(filepath.Join(workDir, extraFile), []byte(extraContent), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, workDir, "add", ".")
	runGit(t, workDir, "commit", "-q", "-m", "follow-up commit")
	runGit(t, workDir, "push", "-q", "origin", "main")
	return runGit(t, workDir, "rev-parse", "HEAD")
}

func copyTree(t *testing.T, srcDir, dstDir string) {
	t.Helper()
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
		dst := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, content, info.Mode())
	}); err != nil {
		t.Fatalf("copying %s: %v", srcDir, err)
	}
}

func sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func postWebhook(t *testing.T, apiURL, event string, payload interface{}) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, apiURL+"/webhook/github", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature-256", sign(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

type apiTestEnv struct {
	server   *Server
	httpURL  string
	q        *db.Queries
	repo     db.Repository
	owner    string
	name     string
	gh       *fakeGitHub
	cloneURL string
}

func setupAPITest(t *testing.T) *apiTestEnv {
	t.Helper()
	ctx := context.Background()
	q, pool, err := db.Open(ctx, testDBConnString(t))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	gh := &fakeGitHub{pushTokens: map[string]bool{"good-token": true}, combinedState: "success"}
	ghSrv := httptest.NewServer(gh.handler())
	t.Cleanup(ghSrv.Close)

	ghc := ghclient.New()
	ghc.APIBaseURL = ghSrv.URL
	ghc.AuthBaseURL = ghSrv.URL // the same fake server serves both -- see its /login/oauth/access_token case

	s := New(q, pool, ghc, webhookSecret, "service-token")
	s.GitHubClientID, s.GitHubClientSecret = "test-client-id", "test-client-secret"
	httpSrv := httptest.NewServer(s)
	t.Cleanup(httpSrv.Close)
	s.PublicBaseURL = httpSrv.URL
	s.UIBaseURL = "https://ui.example"
	// examples/lm-test is also what env.cloneURL below is pushed from
	// (pushDirAsRepo), so a build resolved through the real webhook flow
	// (TestFullReconcileFlow, TestTriggerBuildAndDeployManualFlow) has
	// content on disk here that matches -- needed for
	// handleRenderObject, the one handler that reads from a checkout
	// rather than Postgres.
	s.RepoRoot = "../../examples/lm-test"

	// Every test in this suite that creates a configured build uses the
	// literal name "microcloud" (createConfiguredBuildRequest's own
	// convention, long predating internal/orchestrator's real
	// checkBuilderCompatibility check). A real
	// build now genuinely resolves that name against a real
	// builder_config row before Reconcile does any work, so one has to
	// exist for these tests to mean what they always meant -- "fake"
	// kind, matching every other builder these tests actually deploy
	// against. ON CONFLICT DO NOTHING: this is process-wide, shared,
	// harmless fixture state, not scoped to this one test's own cleanup.
	pool.Exec(ctx, `INSERT INTO builder_config (name, kind, incus_images, incus_sizes) VALUES ('microcloud', 'fake', '{}', '{}') ON CONFLICT (name) DO NOTHING`)

	owner := "laforge-test-owner"
	name := fmt.Sprintf("api-test-repo-%d", os.Getpid())

	cloneURL, _ := pushDirAsRepo(t, "../../examples/lm-test")
	gh.cloneURL = cloneURL // see fakeGitHub.cloneURL's own doc comment

	env := &apiTestEnv{server: s, httpURL: httpSrv.URL, q: q, owner: owner, name: name, gh: gh, cloneURL: cloneURL}

	// Register the repo directly through the DB rather than the API's own
	// POST /repos here, since that's exercised as its own explicit test
	// below (TestCreateRepoAuth) -- every other test just needs a
	// registered repo to already exist.
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: owner, GithubRepo: name})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	// A fresh connection dedicated to cleanup, opened and closed on its
	// own, so it works regardless of whether `pool` above has already
	// been closed by the time this runs.
	t.Cleanup(func() {
		ctx := context.Background()
		_, cleanupPool, err := db.Open(ctx, testDBConnString(t))
		if err != nil {
			return
		}
		defer cleanupPool.Close()
		cleanupPool.Exec(ctx, "DELETE FROM repository WHERE id = $1", repo.ID)
	})
	env.repo = repo
	return env
}

func TestCreateRepoAuth(t *testing.T) {
	env := setupAPITest(t)

	// No token at all.
	resp, err := http.Post(env.httpURL+"/repos", "application/json", strings.NewReader(`{"owner":"o","repo":"r"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want 401", resp.StatusCode)
	}

	// Token with no push access.
	req, _ := http.NewRequest(http.MethodPost, env.httpURL+"/repos", strings.NewReader(`{"owner":"o","repo":"r"}`))
	req.Header.Set("Authorization", "Bearer bad-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bad token: status = %d, want 403", resp.StatusCode)
	}

	// Token with push access.
	req, _ = http.NewRequest(http.MethodPost, env.httpURL+"/repos", strings.NewReader(`{"owner":"newowner","repo":"newrepo"}`))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("good token: status = %d, want 201", resp.StatusCode)
	}
	var created db.Repository
	json.NewDecoder(resp.Body).Decode(&created)
	defer func() {
		ctxb := context.Background()
		_, pool, err := db.Open(ctxb, testDBConnString(t))
		if err == nil {
			defer pool.Close()
			pool.Exec(ctxb, "DELETE FROM repository WHERE id = $1", created.ID)
		}
	}()
	if created.GithubOwner != "newowner" {
		t.Fatalf("created.GithubOwner = %q, want newowner", created.GithubOwner)
	}
}

// TestWebhookRejectsBadSignature needs no database at all -- signature
// verification happens before anything else in handleWebhook.
func TestWebhookRejectsBadSignature(t *testing.T) {
	gh := &fakeGitHub{}
	ghSrv := httptest.NewServer(gh.handler())
	defer ghSrv.Close()
	ghc := ghclient.New()
	ghc.APIBaseURL = ghSrv.URL

	s := New(nil, nil, ghc, webhookSecret, "")
	httpSrv := httptest.NewServer(s)
	defer httpSrv.Close()

	body := []byte(`{"ref":"refs/heads/main"}`)
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/webhook/github", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

// TestWebhookIgnoresUnregisteredRepo and non-push events -- both are
// "recognized but nothing to do," and must answer 200 rather than erroring
// (a non-2xx makes GitHub retry, which would never change the outcome).
func TestWebhookIgnoresUnregisteredRepoAndNonPushEvents(t *testing.T) {
	env := setupAPITest(t)

	resp := postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "deadbeef",
		"repository": map[string]string{"full_name": "nobody/nothing", "clone_url": "file:///dev/null"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unregistered repo: status = %d, want 200", resp.StatusCode)
	}

	resp = postWebhook(t, env.httpURL, "ping", map[string]interface{}{"zen": "hi"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ping event: status = %d, want 200", resp.StatusCode)
	}
}

// TestFullReconcileFlow is the capstone: register a configured build,
// deliver a real push webhook for a real commit, and confirm the build's
// current_content_revision_id actually advances -- then confirm toggling
// follow off, and locking the competition, both correctly stop it from
// advancing on later real commits. This exercises every piece the api
// built, wired together exactly as GitHub would drive them.
func TestFullReconcileFlow(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()

	// Create a configured build via the real API, authenticated.
	cbBody, _ := json.Marshal(createConfiguredBuildRequest{
		Branch: "main", EnvironmentPath: "lm-test.yaml", BuilderConfigName: "microcloud",
	})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/repos/%s/configured-builds", env.httpURL, env.repo.ID.String()), bytes.NewReader(cbBody))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create configured build: status = %d", resp.StatusCode)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	resp.Body.Close()
	if !cb.AutoDeployEnabled || cb.CompetitionStarted {
		t.Fatalf("defaults wrong: auto_deploy=%v started=%v", cb.AutoDeployEnabled, cb.CompetitionStarted)
	}

	// --- push 1: should advance current_content_revision_id ---
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": "ignored-in-favor-of-real-fetch",
		"repository": map[string]string{
			"full_name": env.owner + "/" + env.name,
			"clone_url": env.cloneURL,
		},
	})
	if resp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		t.Fatalf("webhook push 1: status = %d: %s", resp.StatusCode, body.String())
	}

	rev1, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{
		RepositoryID: env.repo.ID, CommitSha: headSHA(t, env.cloneURL),
	})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA: %v", err)
	}
	if !rev1.Valid {
		t.Fatalf("first commit should be valid (it's the real examples/lm-test repo); validation_errors = %s", rev1.ValidationErrors)
	}

	cbAfter1 := getConfiguredBuild(t, env.httpURL, cb.ID.String())
	if cbAfter1.CurrentContentRevisionID != rev1.ID {
		t.Fatalf("after push 1: current_content_revision_id = %v, want %v (rev1)", cbAfter1.CurrentContentRevisionID, rev1.ID)
	}
	if len(env.gh.statusPosts) != 1 || env.gh.statusPosts[0]["state"] != "success" {
		t.Fatalf("statusPosts = %+v, want one success status posted back to GitHub", env.gh.statusPosts)
	}

	// --- disable auto-deploy, then push again: auto-build is always on, so
	// the revision STILL advances (it just wouldn't be deployed) ---
	setConfiguredBuildAutoDeploy(t, env.httpURL, cb.ID.String(), false)
	sha2 := pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "FOLLOW_TEST.txt", "first follow-up\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": sha2,
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 2: status = %d", resp.StatusCode)
	}
	rev2, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{RepositoryID: env.repo.ID, CommitSha: sha2})
	if err != nil {
		t.Fatalf("GetContentRevisionByRepoAndSHA(sha2): %v", err)
	}
	cbAfter2 := getConfiguredBuild(t, env.httpURL, cb.ID.String())
	if cbAfter2.CurrentContentRevisionID != rev2.ID {
		t.Fatalf("after push 2 (auto-deploy off): revision should still advance to rev2 %v (auto-build is always on), got %v", rev2.ID, cbAfter2.CurrentContentRevisionID)
	}

	// --- lock the competition, then push again: now it must NOT advance ---
	setConfiguredBuildAutoDeploy(t, env.httpURL, cb.ID.String(), true)
	setLock(t, env.httpURL, cb.ID.String(), true)
	sha3 := pushAnotherCommit(t, env.cloneURL, "../../examples/lm-test", "FOLLOW_TEST.txt", "second follow-up\n")
	resp = postWebhook(t, env.httpURL, "push", map[string]interface{}{
		"ref": "refs/heads/main", "after": sha3,
		"repository": map[string]string{"full_name": env.owner + "/" + env.name, "clone_url": env.cloneURL},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook push 3: status = %d", resp.StatusCode)
	}
	cbAfter3 := getConfiguredBuild(t, env.httpURL, cb.ID.String())
	if cbAfter3.CurrentContentRevisionID != rev2.ID {
		t.Fatalf("after push 3 (competition started): current_content_revision_id changed to %v, want it to stay locked at rev2 %v", cbAfter3.CurrentContentRevisionID, rev2.ID)
	}

	// But the commit was still validated and recorded -- "commits still
	// build, so work carries on and stays reviewable."
	rev3, err := env.q.GetContentRevisionByRepoAndSHA(ctx, db.GetContentRevisionByRepoAndSHAParams{RepositoryID: env.repo.ID, CommitSha: sha3})
	if err != nil {
		t.Fatalf("commit 3 should still have been recorded even though locked: %v", err)
	}
	if !rev3.Valid {
		t.Fatalf("commit 3 should still validate cleanly: %s", rev3.ValidationErrors)
	}
}

func headSHA(t *testing.T, cloneURL string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "clone", "-q", cloneURL, ".")
	return runGit(t, dir, "rev-parse", "HEAD")
}

func getConfiguredBuild(t *testing.T, apiURL, id string) db.ConfiguredBuild {
	t.Helper()
	resp, err := http.Get(apiURL + "/configured-builds/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET configured build: status = %d", resp.StatusCode)
	}
	var cb db.ConfiguredBuild
	json.NewDecoder(resp.Body).Decode(&cb)
	return cb
}

func setConfiguredBuildAutoDeploy(t *testing.T, apiURL, id string, enabled bool) {
	t.Helper()
	body, _ := json.Marshal(setEnabledRequest{Enabled: enabled})
	req, _ := http.NewRequest(http.MethodPost, apiURL+"/configured-builds/"+id+"/auto-deploy", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setConfiguredBuildAutoDeploy(%v): status = %d", enabled, resp.StatusCode)
	}
}

func setLock(t *testing.T, apiURL, id string, started bool) {
	t.Helper()
	body, _ := json.Marshal(setLockRequest{Started: started})
	req, _ := http.NewRequest(http.MethodPost, apiURL+"/configured-builds/"+id+"/lock", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer good-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setLock(%v): status = %d", started, resp.StatusCode)
	}
}
