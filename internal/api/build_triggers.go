// The write side of "Build and deploy are separate verbs": the two
// operator actions the
// UI's Build button and Deploy button call, plus the internal versions of
// both webhook.go's reconcile drives automatically for a following,
// CI-passed commit ("auto-build is on by default, always... auto-deploy
// follows the environment's state by default"). The HTTP handlers are
// gated at levelBuild -- "Create builds from branches, and deploy them"
// -- distinct from levelManage, which
// governs a live build's ad-hoc commands and access windows, not its
// existence. The internal versions run with no request/auth at all: a
// webhook delivery isn't an operator acting through the API, it's the
// content lifecycle itself advancing, the same way GitHub's own CI
// gating already isn't a person clicking anything.
package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/db"
)

// triggerBuild is "Build: resolve the environment at a commit ... touches
// no hoster" itself -- creates a `build` row at the schema default status
// ("planned") for cb's current_content_revision_id; internal/orchestrator's
// poll loop picks up any planned build on its own and resolves it (real
// teams, real fingerprints) without creating a single task, per the
// deployTasks gate in internal/orchestrator/reconcile.go. Requires
// cb.CurrentContentRevisionID to already be set -- the caller's job,
// whether that's an operator (handleTriggerBuild, after checking the
// field is valid) or reconcile (which just advanced it).
//
// Deliberately always creates a new build, even when cb already has a
// live one -- see applyUpcoming for the reuse-in-place counterpart ("a
// new commit is applied to the existing build, not a teardown and
// rebuild from scratch"). Folding that reuse into triggerBuild itself
// was tried and reverted: reconcile (webhook.go) already calls
// triggerBuild unconditionally on every CI-passing push, and GET
// /builds/{id}/upcoming's whole "pending" signal is
// `cb.CurrentContentRevisionID != build.ContentRevisionID` -- if
// triggerBuild silently applied the newer commit to the live build in
// the same call that advances current_content_revision_id, "upcoming
// changes" would never have anything left to show for any
// follow_enabled configured build regardless of the repository's own
// auto-deploy setting, defeating its own review purpose (caught by
// TestUpcomingChangesDetectsRealDriftAfterANewerPush). reconcile now
// calls applyUpcoming separately, and only when the repository's own
// auto_deploy_enabled is on.
// auto records whether this build was created automatically off a CI-passing
// push (reconcile) or by a deliberate operator "Build Now" (handleTriggerBuild),
// stored on the build so the UI can flag the automatic ones.
func (s *Server) triggerBuild(ctx context.Context, cb db.ConfiguredBuild, auto bool) (db.Build, error) {
	env, err := s.Queries.GetEnvironmentByRevisionAndPath(ctx, db.GetEnvironmentByRevisionAndPathParams{
		ContentRevisionID: cb.CurrentContentRevisionID, Path: cb.EnvironmentPath,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Build{}, errors.New("environment file not found in the tracked revision -- content may have moved or been renamed")
		}
		return db.Build{}, err
	}
	return s.Queries.CreateBuild(ctx, db.CreateBuildParams{
		ConfiguredBuildID: cb.ID, ContentRevisionID: cb.CurrentContentRevisionID, EnvironmentName: env.Name, AutoBuilt: auto,
	})
}

// errNothingPending and errEnvironmentNotFound are applyUpcoming's own
// sentinel errors -- handleApplyUpcoming maps them to real HTTP status
// codes; reconcile (webhook.go) just checks for errNothingPending to
// treat as a normal no-op ("nothing to apply automatically") and logs
// anything else non-fatally, the same way it already treats a failed
// triggerBuild call.
var (
	errNothingPending      = errors.New("nothing pending -- this build already reflects the configured build's currently tracked commit")
	errEnvironmentNotFound = errors.New("environment file not found in the tracked revision -- content may have moved or been renamed")
)

// applyUpcoming is the deliberate action GET /builds/{id}/upcoming's
// preview exists to inform: "a new commit is applied to the existing
// build, not a teardown and rebuild from scratch. Only what changed is
// redeployed". Reuses build's own row -- team/deployed_object rows never move, so
// orchestrator.Reconcile's existing fingerprint diff (unchanged)
// naturally destroys only what changed and deploys only what's new the
// next time it's called against this same buildID, once
// internal/orchestrator's poll loop notices the status/content change.
//
// A plain function, not just an HTTP handler body, so both
// handleApplyUpcoming (a human's deliberate click) and reconcile
// (webhook.go's real, repository-level auto-deploy toggle -- item 13)
// call the exact same mechanism, the same way triggerBuild already
// serves both handleTriggerBuild and reconcile's auto-build call.
func (s *Server) applyUpcoming(ctx context.Context, build db.Build, cb db.ConfiguredBuild) (db.Build, error) {
	if !cb.CurrentContentRevisionID.Valid || cb.CurrentContentRevisionID == build.ContentRevisionID {
		return db.Build{}, errNothingPending
	}
	env, err := s.Queries.GetEnvironmentByRevisionAndPath(ctx, db.GetEnvironmentByRevisionAndPathParams{
		ContentRevisionID: cb.CurrentContentRevisionID, Path: cb.EnvironmentPath,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Build{}, errEnvironmentNotFound
		}
		return db.Build{}, err
	}
	updated, err := s.Queries.ApplyContentRevisionToBuild(ctx, db.ApplyContentRevisionToBuildParams{
		ID: build.ID, ContentRevisionID: cb.CurrentContentRevisionID, EnvironmentName: env.Name,
	})
	if err != nil {
		return db.Build{}, fmt.Errorf("applying commit: %w", err)
	}
	if _, err := s.Queries.CreateEvent(ctx, db.CreateEventParams{
		BuildID: updated.ID, Kind: "build.commit_applied",
		Message: "a new commit was applied to this build in place; existing infrastructure is reconciled, not duplicated",
		Payload: []byte("{}"),
	}); err != nil {
		log.Printf("applyUpcoming: recording event: %v", err)
	}
	return updated, nil
}

// handleApplyUpcoming is applyUpcoming as the operator-facing HTTP
// action -- the manual counterpart to reconcile's automatic call when a
// repository's auto-deploy toggle (item 13) is off.
func (s *Server) handleApplyUpcoming(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireLevel(r.Context(), r, repo, levelBuild)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if !build.ConfiguredBuildID.Valid {
		writeError(w, http.StatusBadRequest, errors.New("this build has no configured build behind it -- nothing to apply a newer commit from"))
		return
	}
	cb, err := s.Queries.GetConfiguredBuild(r.Context(), build.ConfiguredBuildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	updated, err := s.applyUpcoming(r.Context(), build, cb)
	if err != nil {
		switch {
		case errors.Is(err, errNothingPending):
			writeError(w, http.StatusConflict, err)
		case errors.Is(err, errEnvironmentNotFound):
			writeError(w, http.StatusBadRequest, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	// applyUpcoming records the commit-applied event itself (shared with
	// reconcile's automatic path); add who did it when it's a manual action.
	s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: updated.ID, Kind: "build.commit_applied",
		Message: sess.GithubLogin + " applied a newer commit to this build",
		Payload: []byte("{}"),
	})
	writeJSON(w, http.StatusOK, updated)
}

// handleTriggerBuild is triggerBuild as the operator-facing HTTP action
// (webhook.go's reconcile is what advances current_content_revision_id
// automatically for a following build -- this only ever reads it, never
// chooses a commit itself, and works the same whether that revision got
// there automatically or was set some other way).
func (s *Server) handleTriggerBuild(w http.ResponseWriter, r *http.Request) {
	cb, repo, err := s.buildAndOwningRepo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireLevel(r.Context(), r, repo, levelBuild)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if !cb.CurrentContentRevisionID.Valid {
		writeError(w, http.StatusBadRequest, errors.New("no CI-passed commit tracked yet for this configured build"))
		return
	}
	build, err := s.triggerBuild(r.Context(), cb, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// A manual build is an operator action, not the content lifecycle advancing
	// on its own -- record who kicked it off in the build's own log, the same way
	// teardown/access actions record their actor. (The webhook's automatic
	// triggerBuild path has no operator and writes no such event; AutoBuilt on
	// the build row already marks those.)
	s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: build.ID, Kind: "build.triggered",
		Message: sess.GithubLogin + " started a manual build",
		Payload: []byte("{}"),
	})
	// Go straight to deploying rather than leaving a 'planned' build for a
	// second, manual "Deploy" click: now that a Build Now only runs off a
	// CI-passed tracked commit, the plan-then-deploy split is friction, not a
	// safety gate. The orchestrator's deploy loop watches for 'deploying' the
	// same way it would after the old Deploy button, so this is exactly that
	// transition, folded in. If the flip somehow fails the build simply stays
	// 'planned' and the Deploy action still works as a fallback.
	if deployed, err := s.Queries.SetBuildStatus(r.Context(), db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err == nil {
		build = deployed
	}
	writeJSON(w, http.StatusCreated, build)
}

// handleDeployBuild is "Deploy: apply that build ... deploying, then
// deployed" -- flipping a planned build's status is the whole action.
// Nothing here talks to a hoster directly: the status change is what
// internal/orchestrator's deployTasks gate is watching for, so the
// already-running poll loop picks it up on its next pass and starts
// creating real deploy tasks. Refuses anything not currently "planned" --
// deploying an already-deploying/deployed/failed/torn_down build isn't a
// meaningful action and its status word already says what happened to it.
func (s *Server) handleDeployBuild(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireLevel(r.Context(), r, repo, levelBuild)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if build.Status != "planned" {
		writeError(w, http.StatusConflict, errors.New("build is "+build.Status+", not planned -- only a planned build can be deployed"))
		return
	}
	updated, err := s.Queries.SetBuildStatus(r.Context(), db.SetBuildStatusParams{ID: build.ID, Status: "deploying"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Record who deployed it, same as Build Now / teardown record their actor.
	s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: build.ID, Kind: "build.deploy.requested",
		Message: sess.GithubLogin + " deployed this build",
		Payload: []byte("{}"),
	})
	writeJSON(w, http.StatusOK, updated)
}

// handleTeardownBuild is "teardown
// (infrastructure)... available from the UI and the API",
// found missing entirely by direct audit -- the only destroy machinery
// that existed was Reconcile's own per-object diff against desired
// content, never a whole build on a deliberate operator action. Gated at
// levelManage, not levelBuild -- like close/open access
// (handleSetTeamAccess), this acts on a live build's real infrastructure,
// not its existence as a configured/planned thing. Sets 'tearing_down'
// and returns immediately; internal/orchestrator.Teardown (driven by
// cmd/laforge-orchestrator's own poll loop, same as every other
// asynchronous build action in this system) does the actual destroying
// and moves the build to 'torn_down' once every object genuinely is.
func (s *Server) handleTeardownBuild(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireLevel(r.Context(), r, repo, levelManage)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	switch build.Status {
	case "tearing_down", "torn_down", "purged":
		writeError(w, http.StatusConflict, errors.New("build is already "+build.Status))
		return
	}
	updated, err := s.Queries.SetBuildStatus(r.Context(), db.SetBuildStatusParams{ID: build.ID, Status: "tearing_down"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: build.ID, Kind: "build.teardown.requested",
		Message: sess.GithubLogin + " requested teardown", Payload: []byte("{}"),
	})
	writeJSON(w, http.StatusAccepted, updated)
}
