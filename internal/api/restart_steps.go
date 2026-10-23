// Restart from a step (docs/plans/restart-from-step.md): run a host's build
// again from a chosen authored step, on the box it already has -- the
// alternative to a rebuild when step 7 of 12 failed and the box is fine.
// Picked once for a host definition and applied to its instances across
// teams. Manage-level, like rebuild, power actions and ad-hoc dispatch.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

type restartStepsRequest struct {
	Target    adHocTarget `json:"target"`
	FromGroup int         `json:"from_group"`
	// ResumeDependents defaults to true: hosts failed for depending on a
	// restarted one resume once it finishes.
	ResumeDependents *bool `json:"resume_dependents,omitempty"`
	Force            bool  `json:"force,omitempty"`
	// DryRun describes every instance of the targeted host definition across
	// the build -- what the confirm dialog shows -- and changes nothing.
	DryRun bool `json:"dry_run,omitempty"`
}

func (s *Server) handleRestartSteps(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	var req restartStepsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch build.Status {
	case "deploying", "building", "finished", "failed":
	default:
		writeError(w, http.StatusConflict, errors.New("build is "+build.Status+" -- only a deployed build's hosts can be restarted"))
		return
	}
	repoRoot, content, err := s.buildContentRoot(r, build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	target := req.Target
	if req.DryRun {
		// The dialog opens from one host and lists every team's copy of it:
		// widen the target to the whole definition.
		widened, err := s.wholeDefinition(r, build, target)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		target = widened
	}
	opts := orchestrator.RestartOptions{FromGroup: req.FromGroup, ResumeDependents: true, Force: req.Force, DryRun: req.DryRun}
	if req.ResumeDependents != nil {
		opts.ResumeDependents = *req.ResumeDependents
	}
	results, err := orchestrator.RestartFromStep(r.Context(), s.Pool, build, content, repoRoot, target, opts)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, orchestrator.ErrRestartRefused) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	restarted := 0
	for _, res := range results {
		if res.Restarted {
			restarted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": results, "restarted": restarted, "dry_run": req.DryRun, "from_group": req.FromGroup})
}

// wholeDefinition turns a target of specific ids into every instance of the
// same host definition in the build (all teams), for the dry run's listing.
// Any other kind of target is left as it is.
func (s *Server) wholeDefinition(r *http.Request, build db.Build, target adHocTarget) (adHocTarget, error) {
	if len(target.IDs) == 0 {
		return target, nil
	}
	matched, err := orchestrator.MatchAdHocTargets(r.Context(), s.Queries, build.ID, target)
	if err != nil {
		return target, err
	}
	if len(matched) == 0 {
		return target, errors.New("no hosts match this target")
	}
	name := matched[0].ObjectName
	for _, o := range matched[1:] {
		if o.ObjectName != name {
			return target, errors.New("a restart is for one host definition at a time: " + name + " and " + o.ObjectName + " were both selected")
		}
	}
	all, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		return target, err
	}
	ids := make([]string, 0, len(all))
	for _, o := range all {
		if o.ObjectName == name && o.Kind != "network" {
			ids = append(ids, o.ID.String())
		}
	}
	return adHocTarget{IDs: ids}, nil
}

// buildContentRoot is loadBuildContent plus the directory the content was
// loaded from, for callers that expand steps (ExpandSteps reads script files
// relative to it).
func (s *Server) buildContentRoot(r *http.Request, build db.Build) (string, *loader.Content, error) {
	repoRoot := s.RepoRoot
	if s.Checkouts != nil {
		dir, err := s.Checkouts.ForBuild(r.Context(), build.ID)
		if err != nil {
			return "", nil, err
		}
		repoRoot = dir
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return "", nil, err
	}
	return repoRoot, c, nil
}

// handleSetConfiguredBuildHoldOnStepChange is POST /configured-builds/{id}/hold-on-step-change:
// on, a content change that touches only a host's steps or scripts marks the
// host instead of recreating it (docs/plans/restart-from-step.md §6).
func (s *Server) handleSetConfiguredBuildHoldOnStepChange(w http.ResponseWriter, r *http.Request) {
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
	updated, err := s.Queries.SetConfiguredBuildHoldOnStepChange(r.Context(), db.SetConfiguredBuildHoldOnStepChangeParams{ID: cb.ID, HoldOnStepChange: req.Enabled})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
