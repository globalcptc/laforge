package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
	"github.com/globalcptc/laforge/internal/ingest"
)

type pushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Deleted    bool   `json:"deleted"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	// Installation is present whenever this event was delivered by a
	// GitHub App installation (every real delivery, once one is
	// configured) -- its id is what resolveCloneURL uses to mint a
	// scoped installation token. Deliberately the only thing read off
	// `repository` in this struct any more: this payload no longer
	// carries a clone_url field at all, so there is nothing here left to
	// blindly trust -- see resolveCloneURL's own doc comment for why.
	Installation *struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

type installationRepoRef struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type installationPayload struct {
	Action       string `json:"action"` // created | deleted | suspend | unsuspend | new_permissions_accepted
	Installation struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"account"`
	} `json:"installation"`
	// Repositories is only ever populated on "created"/"deleted" when
	// RepositorySelection is "selected" -- an "all repositories" install
	// carries an empty array here regardless of how many repos it
	// actually covers; see RepositorySelection's own doc comment for the
	// real fix.
	Repositories []installationRepoRef `json:"repositories"`
	// RepositorySelection is "all" or "selected": "all" means
	// Repositories above is empty by
	// GitHub's own design, so handleInstallationEvent syncs the real
	// list itself via GET /installation/repositories instead of trusting
	// an empty payload array.
	RepositorySelection string `json:"repository_selection"`
}

type installationRepositoriesPayload struct {
	Action       string `json:"action"` // added | removed
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	RepositoriesAdded   []installationRepoRef `json:"repositories_added"`
	RepositoriesRemoved []installationRepoRef `json:"repositories_removed"`
}

// verifySignature checks GitHub's X-Hub-Signature-256 header, an
// HMAC-SHA256 of the raw request body keyed on the webhook secret both
// sides were configured with. hmac.Equal (not ==) specifically, so this
// doesn't leak timing information about how much of the signature
// matched.
func verifySignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	sig, ok := strings.CutPrefix(header, prefix)
	if !ok {
		return false
	}
	want, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	got := mac.Sum(nil)
	return hmac.Equal(got, want)
}

// handleWebhook verifies the signature once, then dispatches by
// X-GitHub-Event: "push" (per-commit validation and follow-mode
// reconciliation), "installation" and "installation_repositories" (the
// GitHub-side half of "install on GitHub, approve in LaForge"), or
// anything else
// (acknowledged and ignored). It always answers 200 for anything it
// recognizes-but-ignores -- a non-2xx tells GitHub to retry, and none of
// those cases become true by retrying.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	event := r.Header.Get("X-GitHub-Event")
	delivery := r.Header.Get("X-GitHub-Delivery")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook: %s (delivery %s): reading body: %v", event, delivery, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !verifySignature(s.WebhookSecret, body, r.Header.Get("X-Hub-Signature-256")) {
		// Found live: this handler logged
		// nothing at all, success or rejection, so verifying a real
		// delivery actually arrived meant querying Postgres directly
		// instead of reading this container's own logs, which stayed
		// silent throughout. A rejection especially deserves a real log
		// line -- it's either a misconfigured secret or someone probing
		// the endpoint, and either way "nothing happened" isn't true.
		log.Printf("webhook: %s (delivery %s): rejected, invalid signature", event, delivery)
		writeError(w, http.StatusUnauthorized, errors.New("invalid webhook signature"))
		return
	}

	log.Printf("webhook: %s (delivery %s): accepted", event, delivery)
	switch event {
	case "push":
		s.handlePushEvent(w, r.Context(), body)
	case "installation":
		s.handleInstallationEvent(w, r.Context(), body)
	case "installation_repositories":
		s.handleInstallationRepositoriesEvent(w, r.Context(), body)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handlePushEvent(w http.ResponseWriter, ctx context.Context, body []byte) {
	var payload pushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if payload.Deleted {
		w.WriteHeader(http.StatusOK)
		return
	}

	owner, repoName, ok := strings.Cut(payload.Repository.FullName, "/")
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unexpected repository.full_name %q", payload.Repository.FullName))
		return
	}
	repo, err := s.Queries.GetRepositoryByOwnerRepo(ctx, db.GetRepositoryByOwnerRepoParams{
		GithubOwner: owner, GithubRepo: repoName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Not a repository LaForge tracks -- quietly ignore rather
			// than erroring, so an org-wide webhook (or one left over
			// after a repo is deregistered) doesn't spam retries.
			w.WriteHeader(http.StatusOK)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	branch, ok := strings.CutPrefix(payload.Ref, "refs/heads/")
	if !ok {
		// A tag push or some other ref kind -- configured builds only
		// ever track branches, so there's nothing to reconcile.
		w.WriteHeader(http.StatusOK)
		return
	}

	var installationID int64
	if payload.Installation != nil {
		installationID = payload.Installation.ID
	}
	if err := s.reconcile(ctx, repo, branch, payload.Ref, installationID); err != nil {
		log.Printf("webhook: reconcile %s/%s@%s: %v", owner, repoName, branch, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleInstallationEvent records the GitHub-side half of "install on
// GitHub, approve in LaForge": which installations exist, and (when the
// installation is scoped to specific repositories, not "all") which
// repositories each currently covers. Nothing here touches the
// `repository` table -- that only ever happens through an explicit
// admin approval (installations.go), never automatically from this
// event.
func (s *Server) handleInstallationEvent(w http.ResponseWriter, ctx context.Context, body []byte) {
	var p installationPayload
	if err := json.Unmarshal(body, &p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch p.Action {
	case "created", "unsuspend":
		inst, err := s.Queries.UpsertGithubInstallation(ctx, db.UpsertGithubInstallationParams{
			InstallationID: p.Installation.ID, AccountLogin: p.Installation.Account.Login, AccountType: p.Installation.Account.Type,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if p.RepositorySelection == "all" {
			// The payload's own Repositories array is empty by GitHub's
			// design for an "all repositories" install -- the real list
			// only exists behind a real API call, authenticated as this
			// installation.
			repos, err := s.listAllRepositoriesForInstallation(ctx, p.Installation.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			for _, r := range repos {
				owner, name, ok := strings.Cut(r.FullName, "/")
				if !ok {
					continue
				}
				if _, err := s.Queries.UpsertInstallationRepository(ctx, db.UpsertInstallationRepositoryParams{
					InstallationID: inst.ID, GithubOwner: owner, GithubRepo: name, GithubRepoID: r.ID,
				}); err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
		for _, r := range p.Repositories {
			owner, name, ok := strings.Cut(r.FullName, "/")
			if !ok {
				continue
			}
			if _, err := s.Queries.UpsertInstallationRepository(ctx, db.UpsertInstallationRepositoryParams{
				InstallationID: inst.ID, GithubOwner: owner, GithubRepo: name, GithubRepoID: r.ID,
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		}
	case "deleted":
		if err := s.Queries.DeleteGithubInstallation(ctx, p.Installation.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	case "suspend":
		if _, err := s.Queries.SetGithubInstallationSuspended(ctx, db.SetGithubInstallationSuspendedParams{
			InstallationID: p.Installation.ID, Suspended: true,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	default:
		// "new_permissions_accepted" and anything else GitHub adds
		// later -- nothing for LaForge to record.
	}
	w.WriteHeader(http.StatusOK)
}

// handleInstallationRepositoriesEvent keeps installation_repository in
// sync as an org adds or removes repositories from an existing
// installation's selection, without touching the installation itself.
func (s *Server) handleInstallationRepositoriesEvent(w http.ResponseWriter, ctx context.Context, body []byte) {
	var p installationRepositoriesPayload
	if err := json.Unmarshal(body, &p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	inst, err := s.Queries.GetGithubInstallationByInstallationID(ctx, p.Installation.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// An installation_repositories event for an installation
			// LaForge never recorded (e.g. its own "installation"
			// event was missed) -- nothing to attach these to yet.
			w.WriteHeader(http.StatusOK)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, r := range p.RepositoriesAdded {
		owner, name, ok := strings.Cut(r.FullName, "/")
		if !ok {
			continue
		}
		if _, err := s.Queries.UpsertInstallationRepository(ctx, db.UpsertInstallationRepositoryParams{
			InstallationID: inst.ID, GithubOwner: owner, GithubRepo: name, GithubRepoID: r.ID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	for _, r := range p.RepositoriesRemoved {
		if err := s.Queries.DeleteInstallationRepository(ctx, db.DeleteInstallationRepositoryParams{
			InstallationID: inst.ID, GithubRepoID: r.ID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// resolveCloneURL fixes a real vulnerability: a push webhook payload used
// to carry its own
// `repository.clone_url`, trusted with no cross-check against which
// repository it actually claimed to be -- anyone who could produce a
// validly-signed payload could point LaForge at a git server of their
// choosing while claiming a real repository's identity. This never reads
// anything off the payload at all. It asks GitHub directly, using the
// most narrowly scoped credential available:
//
//  1. An installation-scoped token, when the event carried an
//     installation id (every real delivery, once a GitHub App is
//     configured) -- minted fresh via ghclient's app-JWT exchange,
//     valid for exactly this installation's own repositories.
//  2. GITHUB_SERVICE_TOKEN, when no App is configured (or this
//     particular delivery had no installation, e.g. a manually
//     configured legacy webhook) -- broader-scoped than an installation
//     token, but still a real, authoritative call to GitHub, not
//     anything client-supplied.
//  3. An error, naming exactly what's missing, if neither is available
//     -- refusing to fall back to trusting the payload.
//
// authenticatedCloneURL embeds an x-access-token credential into an https
// GitHub clone URL so the ingest git fetch authenticates for a private repo
// -- the same scheme internal/checkout uses for the runner's own clones. A
// non-token or non-https URL is returned unchanged. FetchCommit redacts this
// token from any error it logs.
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

func (s *Server) resolveCloneURL(ctx context.Context, owner, repoName string, installationID int64) (string, error) {
	if installationID != 0 && s.AppID != "" && s.AppPrivateKey != nil {
		jwt, err := ghclient.GenerateAppJWT(s.AppID, s.AppPrivateKey, time.Now())
		if err != nil {
			return "", fmt.Errorf("signing app jwt: %w", err)
		}
		tok, err := s.GH.CreateInstallationToken(ctx, jwt, installationID)
		if err != nil {
			return "", fmt.Errorf("minting installation token: %w", err)
		}
		ghRepo, err := s.GH.GetRepo(ctx, tok.Token, owner, repoName)
		if err != nil {
			return "", fmt.Errorf("fetching %s/%s via installation token: %w", owner, repoName, err)
		}
		// Embed the token so the ingest fetch actually authenticates -- a
		// private repo's plain clone URL fails with "could not read Username".
		// FetchCommit is documented to accept exactly this form.
		return authenticatedCloneURL(ghRepo.CloneURL, tok.Token)
	}
	if s.ServiceToken != "" {
		ghRepo, err := s.GH.GetRepo(ctx, s.ServiceToken, owner, repoName)
		if err != nil {
			return "", fmt.Errorf("fetching %s/%s via service token: %w", owner, repoName, err)
		}
		return authenticatedCloneURL(ghRepo.CloneURL, s.ServiceToken)
	}
	return "", fmt.Errorf("cannot verify %s/%s's real clone URL: no installation token available (no GitHub App configured or this event carried no installation) and GITHUB_SERVICE_TOKEN is not set", owner, repoName)
}

// listAllRepositoriesForInstallation is the real fix for an "all
// repositories" install: mints a real installation access token (the same minting
// resolveCloneURL already does) and calls GitHub's own
// GET /installation/repositories -- authenticated AS the installation,
// the one real way to discover an "all repositories" install's actual
// coverage, since the webhook payload never carries it. Requires a real
// GitHub App to be configured (installationID alone isn't enough to mint
// a token) -- an installation event for an "all repositories" install
// with no App configured is a real, honestly-reported gap, same
// reasoning resolveCloneURL's own error already uses.
func (s *Server) listAllRepositoriesForInstallation(ctx context.Context, installationID int64) ([]ghclient.Repo, error) {
	if s.AppID == "" || s.AppPrivateKey == nil {
		return nil, fmt.Errorf("cannot list an \"all repositories\" installation's real repositories: no GitHub App configured")
	}
	jwt, err := ghclient.GenerateAppJWT(s.AppID, s.AppPrivateKey, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing app jwt: %w", err)
	}
	tok, err := s.GH.CreateInstallationToken(ctx, jwt, installationID)
	if err != nil {
		return nil, fmt.Errorf("minting installation token: %w", err)
	}
	repos, err := s.GH.ListInstallationRepositories(ctx, tok.Token)
	if err != nil {
		return nil, fmt.Errorf("listing installation repositories: %w", err)
	}
	return repos, nil
}

// reconcile is the whole "content lifecycle" loop for one push: resolve
// the repository's real clone URL (see resolveCloneURL), fetch the
// commit, run it through internal/ingest (schema validation + full
// render-check, matching `laforge check`), fold in GitHub's own CI
// result, report LaForge's own validation back as a commit status ("PR
// check style"), and -- for every configured build tracking this branch
// that is following and not locked -- advance its current_content_revision_id
// and auto-build it: "auto-build is on by default, always... produces a
// resolved, rendered build. It costs nothing and touches no hoster, so
// there is no reason not to." Gated the same way current_content_revision_id
// already was (follow_enabled, not competition_started) -- there's no
// separate per-repository auto-build toggle yet, so follow_enabled is
// the one switch.
//
// Auto-*deploy* -- "if it is already deployed, the commit is applied to
// it" -- is real now too (2026-09-24),
// gated by repo.AutoDeployEnabled, a real repository-wide column
// (separate from follow_enabled, which only ever gated auto-*build*).
// When it's on and a configured build already has a live build
// (status = 'deploying' -- see GetLiveDeployingBuildForConfiguredBuild),
// this calls applyUpcoming directly on that build, the exact mechanism
// handleApplyUpcoming exposes for a human's deliberate click. When it's
// off (the repository-wide default an admin sets, matching "both can be
// turned off per repository, for an event where every deploy should be a
// decision someone makes deliberately"), or when nothing is deployed yet
// ("if nothing is deployed, it stops at the build and waits for someone
// to deploy"), this only ever auto-builds via triggerBuild above, exactly
// as before -- GET /builds/{id}/upcoming's "pending" signal
// (`cb.CurrentContentRevisionID != build.ContentRevisionID`) still means
// something for any repository that keeps auto-deploy off.
//
// competition_started still blocks the whole thing, current_content_revision_id
// included: a live event's tracked commit not moving without a
// deliberate action is "marking the competition started locks automatic
// deploys" read at its most conservative.
func (s *Server) reconcile(ctx context.Context, repo db.Repository, branch, ref string, installationID int64) error {
	result, ciPassed, err := s.ingestBranch(ctx, repo, ref, installationID, true)
	if err != nil {
		return err
	}

	builds, err := s.Queries.ListConfiguredBuildsByRepositoryAndBranch(ctx, db.ListConfiguredBuildsByRepositoryAndBranchParams{
		RepositoryID: repo.ID, Branch: branch,
	})
	if err != nil {
		return fmt.Errorf("listing configured builds: %w", err)
	}
	if !ciPassed {
		return nil
	}
	for _, cb := range builds {
		if cb.CompetitionStarted {
			continue
		}
		updated, err := s.Queries.SetConfiguredBuildCurrentRevision(ctx, db.SetConfiguredBuildCurrentRevisionParams{
			ID: cb.ID, CurrentContentRevisionID: result.Revision.ID,
		})
		if err != nil {
			return fmt.Errorf("advancing configured build %s: %w", cb.ID, err)
		}

		build, err := s.triggerBuild(ctx, updated, true)
		if err != nil {
			// Not fatal to the webhook as a whole -- current_content_revision_id
			// already moved for real, and a build that failed to resolve
			// (e.g. the environment file itself is broken in a way schema
			// validation didn't catch) shouldn't block every other
			// configured build tracking this same branch from building.
			log.Printf("webhook: auto-build for configured build %s: %v", cb.ID, err)
			continue
		}

		if !cb.AutoDeployEnabled {
			continue // planned build sits, waiting for a manual Deploy
		}

		live, err := s.Queries.GetLiveDeployingBuildForConfiguredBuild(ctx, cb.ID)
		if err == nil {
			// Something is already live: apply the new commit to it in place
			// rather than deploying a second copy (the fresh planned build
			// above is left as a record of the commit).
			if _, err := s.applyUpcoming(ctx, live, updated); err != nil && !errors.Is(err, errNothingPending) {
				log.Printf("webhook: auto-deploy for configured build %s: %v", cb.ID, err)
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("webhook: checking for a live build to auto-deploy for configured build %s: %v", cb.ID, err)
			continue
		}
		// Nothing is live yet (first build, or everything was torn down). With
		// auto-deploy on, "deploy" means deploy: take the fresh build straight
		// to 'deploying' rather than leaving it 'planned' for a manual click.
		if _, err := s.Queries.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
			log.Printf("webhook: auto-deploying fresh build for configured build %s: %v", cb.ID, err)
		}
	}
	return nil
}

// ingestBranch fetches a repo's branch HEAD, validates + stores it as a content
// revision, and reports LaForge's validation status. Extracted from reconcile so
// the push webhook and on-demand ingestion (configured-build create / sync --
// see internal/api/builds.go) run the IDENTICAL ingest and can never drift.
// Returns the ingest result and whether the commit is buildable (LaForge-valid
// and, when GitHub CI is wired, CI-passing).
func (s *Server) ingestBranch(ctx context.Context, repo db.Repository, ref string, installationID int64, postStatus bool) (*ingest.Result, bool, error) {
	cloneURL, err := s.resolveCloneURL(ctx, repo.GithubOwner, repo.GithubRepo, installationID)
	if err != nil {
		return nil, false, fmt.Errorf("resolving clone url: %w", err)
	}
	dir, sha, cleanup, err := ingest.FetchCommit(ctx, cloneURL, ref)
	if err != nil {
		return nil, false, fmt.Errorf("fetching commit: %w", err)
	}
	defer cleanup()

	result, err := ingest.ValidateAndStore(ctx, s.Pool, repo.ID, sha, ref, dir)
	if err != nil {
		return nil, false, fmt.Errorf("validating commit: %w", err)
	}
	// The push webhook posts a "laforge/validate" commit status; on-demand
	// ingestion just computes pass/fail without posting.
	if postStatus {
		return result, s.reportStatus(ctx, repo, sha, result), nil
	}
	return result, s.computeCIPassed(ctx, repo, sha, result), nil
}

// installationIDForRepo resolves a repository's GitHub App installation to the
// numeric id resolveCloneURL needs, or 0 when there's no installation (the
// ServiceToken path then applies). Best-effort: a lookup miss yields 0.
func (s *Server) installationIDForRepo(ctx context.Context, repo db.Repository) int64 {
	if repo.InstallationID.Valid {
		if inst, err := s.Queries.GetGithubInstallation(ctx, repo.InstallationID); err == nil {
			return inst.InstallationID
		}
	}
	return 0
}

// reportStatus folds LaForge's own validation together with whatever else
// is already reporting on this commit (GitHub Actions checks, the legacy
// status API) into one pass/fail, matching "CI passes" meaning every check
// on the commit is green, not just LaForge's own. It also posts LaForge's
// own result back as a commit status -- "reports pass/fail (PR check
// style) before anyone can build off it." Both the read and the post are
// best-effort: a GitHub API error here (including, today, simply having
// no ServiceToken configured) degrades
// to "judge CI purely on LaForge's own validation" rather than blocking
// the whole webhook on GitHub's API being reachable.
func (s *Server) reportStatus(ctx context.Context, repo db.Repository, sha string, result *ingest.Result) bool {
	ciPassed := s.computeCIPassed(ctx, repo, sha, result)
	if s.GH != nil && s.ServiceToken != "" {
		state := "success"
		desc := "laforge validation passed"
		if !result.Valid {
			state = "failure"
			desc = fmt.Sprintf("%d validation issue(s)", len(result.Issues))
			if len(desc) > 140 {
				desc = desc[:140]
			}
		}
		if err := s.GH.CreateStatus(ctx, s.ServiceToken, repo.GithubOwner, repo.GithubRepo, sha, state, desc, "laforge/validate"); err != nil {
			log.Printf("reportStatus: CreateStatus: %v", err)
		}
	}
	return ciPassed
}

// computeCIPassed folds LaForge's own validation with GitHub's CI (when a
// ServiceToken is configured) into one pass/fail, WITHOUT posting anything back
// to GitHub. On-demand ingestion (Force Pull / configured-build create) uses
// this so a re-pull doesn't spam commit statuses; only the push webhook, via
// reportStatus, posts a "laforge/validate" status.
func (s *Server) computeCIPassed(ctx context.Context, repo db.Repository, sha string, result *ingest.Result) bool {
	ciPassed := result.Valid
	if s.GH == nil || s.ServiceToken == "" {
		return ciPassed
	}
	if combined, err := s.GH.GetCombinedStatus(ctx, s.ServiceToken, repo.GithubOwner, repo.GithubRepo, sha); err == nil {
		if combined.TotalCount > 0 && combined.State != "success" {
			ciPassed = false
		}
	} else {
		log.Printf("computeCIPassed: GetCombinedStatus: %v", err)
	}
	if runs, err := s.GH.ListCheckRuns(ctx, s.ServiceToken, repo.GithubOwner, repo.GithubRepo, sha); err == nil {
		for _, run := range runs {
			if run.Status == "completed" && !isPassingConclusion(run.Conclusion) {
				ciPassed = false
			}
		}
	} else {
		log.Printf("computeCIPassed: ListCheckRuns: %v", err)
	}
	return ciPassed
}

func isPassingConclusion(c string) bool {
	switch c {
	case "success", "neutral", "skipped":
		return true
	default:
		return false
	}
}
