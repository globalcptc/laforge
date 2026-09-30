package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ingest"
)

// errCompetitionStarted is returned when an action is refused because the
// configured build's competition-started lock freezes its tracked commit.
var errCompetitionStarted = errors.New("competition started: the tracked commit is locked; unlock to sync a new commit")

type createConfiguredBuildRequest struct {
	Branch            string `json:"branch"`
	EnvironmentPath   string `json:"environment_path"`
	BuilderConfigName string `json:"builder_config_name"`
}

// handleCreateConfiguredBuild is "a build is always configured explicitly
// ... branch, environment file, builder config." follow_enabled and
// competition_started take their DEFAULTs from the migration (true,
// false) -- a freshly configured build always starts following and
// unlocked, matching "auto-build is on by default, always."
func (s *Server) handleCreateConfiguredBuild(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := s.requirePush(r.Context(), r, repo.GithubOwner, repo.GithubRepo); err != nil {
		writeAuthError(w, err)
		return
	}
	var req createConfiguredBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Branch == "" || req.EnvironmentPath == "" || req.BuilderConfigName == "" {
		writeError(w, http.StatusBadRequest, errors.New("branch, environment_path, and builder_config_name are all required"))
		return
	}
	cb, err := s.Queries.CreateConfiguredBuild(r.Context(), db.CreateConfiguredBuildParams{
		RepositoryID: repo.ID, Branch: req.Branch, EnvironmentPath: req.EnvironmentPath, BuilderConfigName: req.BuilderConfigName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Auto-pull on create: ingest the branch HEAD now so a just-added build is
	// immediately runnable (its content is committed -- the picker reads a live
	// checkout -- but ingestion is otherwise webhook-only). Best-effort: the build
	// is created regardless; if the content doesn't validate or the fetch fails,
	// the operator uses Force Pull (which surfaces the reason) or fixes + pushes.
	if updated, _, err := s.syncConfiguredBuildRevision(r.Context(), repo, cb); err != nil {
		log.Printf("configured build %s: auto-pull on create failed: %v", cb.ID, err)
	} else {
		cb = updated
	}
	writeJSON(w, http.StatusCreated, cb)
}

// syncConfiguredBuildRevision ingests the configured build's branch HEAD (the
// same ingest the push webhook runs -- "Force Pull") and, if it's buildable,
// points the configured build at that fresh content revision. Returns the
// (possibly) updated row and, when the content DOESN'T validate, the validation
// issues so the caller can tell the operator exactly why it isn't buildable
// (instead of silently doing nothing). Shared by create-time auto-pull and the
// manual Force Pull endpoint.
func (s *Server) syncConfiguredBuildRevision(ctx context.Context, repo db.Repository, cb db.ConfiguredBuild) (db.ConfiguredBuild, []ingest.ValidationIssue, error) {
	if cb.CompetitionStarted {
		// The lock freezes the tracked commit deliberately -- the same reason the
		// push webhook skips a locked build -- so a manual sync must not advance it.
		return cb, nil, errCompetitionStarted
	}
	ref := "refs/heads/" + cb.Branch
	result, ciPassed, err := s.ingestBranch(ctx, repo, ref, s.installationIDForRepo(ctx, repo), false)
	if err != nil {
		return cb, nil, err
	}
	if !ciPassed {
		// Ingested, but the content doesn't validate (or CI failed): leave the
		// tracked revision untouched (an unbuildable commit can't be built) and
		// hand back the reasons.
		return cb, result.Issues, nil
	}
	updated, err := s.Queries.SetConfiguredBuildCurrentRevision(ctx, db.SetConfiguredBuildCurrentRevisionParams{
		ID: cb.ID, CurrentContentRevisionID: result.Revision.ID,
	})
	if err != nil {
		return cb, nil, fmt.Errorf("advancing current revision: %w", err)
	}
	return updated, nil, nil
}

// handleSyncConfiguredBuild is the manual "Force Pull" action: ingest the branch
// HEAD on demand and point the configured build at it, for content that's
// committed but not yet ingested (a new build, or a commit the webhook hasn't
// processed). Reuses the webhook's exact ingest path. When the content doesn't
// validate, it returns 422 with the actual errors so the operator sees WHY the
// build didn't become runnable -- not a silent no-op.
func (s *Server) handleSyncConfiguredBuild(w http.ResponseWriter, r *http.Request) {
	cb, repo, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := s.requirePush(r.Context(), r, repo.GithubOwner, repo.GithubRepo); err != nil {
		writeAuthError(w, err)
		return
	}
	updated, issues, err := s.syncConfiguredBuildRevision(r.Context(), repo, cb)
	if err != nil {
		if errors.Is(err, errCompetitionStarted) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if len(issues) > 0 {
		writeError(w, http.StatusUnprocessableEntity, errors.New(summarizeIssues(cb.Branch, issues)))
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// summarizeIssues turns validation issues into a one-line, operator-readable
// message (the first few, with a count), for the Force Pull 422 response.
func summarizeIssues(branch string, issues []ingest.ValidationIssue) string {
	parts := make([]string, 0, 3)
	for i, iss := range issues {
		if i >= 3 {
			break
		}
		loc := iss.File
		if iss.Line > 0 {
			loc = fmt.Sprintf("%s:%d", iss.File, iss.Line)
		}
		if loc != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", loc, iss.Message))
		} else {
			parts = append(parts, iss.Message)
		}
	}
	more := ""
	if len(issues) > len(parts) {
		more = fmt.Sprintf(" (+%d more)", len(issues)-len(parts))
	}
	return fmt.Sprintf("content on %s doesn't validate (%d issue(s)): %s%s", branch, len(issues), strings.Join(parts, "; "), more)
}

// handleDeleteConfiguredBuild removes a configured build. Refused while it has
// any live build (deploying/building/finished/tearing_down) so a config is never
// detached from a running deployment -- tear those down first. Existing builds
// survive as detached history (the FK is ON DELETE SET NULL).
func (s *Server) handleDeleteConfiguredBuild(w http.ResponseWriter, r *http.Request) {
	cb, repo, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := s.requirePush(r.Context(), r, repo.GithubOwner, repo.GithubRepo); err != nil {
		writeAuthError(w, err)
		return
	}
	live, err := s.Queries.ConfiguredBuildHasLiveBuilds(r.Context(), cb.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if live {
		writeError(w, http.StatusConflict, errors.New("this configured build has live builds (deploying, building, finished, or tearing down) -- tear them down before removing it"))
		return
	}
	if err := s.Queries.DeleteConfiguredBuild(r.Context(), cb.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListConfiguredBuilds(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	builds, err := s.Queries.ListConfiguredBuildsByRepository(r.Context(), repo.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, builds)
}

// buildAndOwningRepo resolves a configured_build id to the row plus its
// owning repository -- every mutating handler below needs both: the build
// itself, and the repository to run requirePush against.
func (s *Server) buildAndOwningRepo(r *http.Request) (db.ConfiguredBuild, db.Repository, error) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		return db.ConfiguredBuild{}, db.Repository{}, err
	}
	cb, err := s.Queries.GetConfiguredBuild(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ConfiguredBuild{}, db.Repository{}, errors.New("no such configured build")
		}
		return db.ConfiguredBuild{}, db.Repository{}, err
	}
	repo, err := s.Queries.GetRepository(r.Context(), cb.RepositoryID)
	if err != nil {
		return db.ConfiguredBuild{}, db.Repository{}, err
	}
	return cb, repo, nil
}

func (s *Server) handleGetConfiguredBuild(w http.ResponseWriter, r *http.Request) {
	cb, _, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, cb)
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// handleSetFollow toggles "follow on/off" -- see server.go's package
// comment for why this currently gates the whole auto-build step (there's
// no separate auto-deploy step yet to gate instead).
func (s *Server) handleSetConfiguredBuildAutoDeploy(w http.ResponseWriter, r *http.Request) {
	cb, repo, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := s.requirePush(r.Context(), r, repo.GithubOwner, repo.GithubRepo); err != nil {
		writeAuthError(w, err)
		return
	}
	var req setEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.Queries.SetConfiguredBuildAutoDeploy(r.Context(), db.SetConfiguredBuildAutoDeployParams{
		ID: cb.ID, AutoDeployEnabled: req.Enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type setLockRequest struct {
	Started bool `json:"started"`
}

// handleSetLock is "marking the competition started" -- see this handler's
// counterpart in webhook.go's reconcile for exactly what it prevents.
func (s *Server) handleSetLock(w http.ResponseWriter, r *http.Request) {
	cb, repo, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := s.requirePush(r.Context(), r, repo.GithubOwner, repo.GithubRepo); err != nil {
		writeAuthError(w, err)
		return
	}
	var req setLockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	updated, err := s.Queries.SetConfiguredBuildCompetitionStarted(r.Context(), db.SetConfiguredBuildCompetitionStartedParams{
		ID: cb.ID, CompetitionStarted: req.Started,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
