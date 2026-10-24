// Force-rebuild: tear down and recreate a host/container and, optionally,
// everything in its team that depends on it. Unlike a power action (power.go),
// which just restarts the existing instance at the hoster, a rebuild destroys
// the instance and deploys it fresh, re-running its steps/validators -- the
// "blow it away and start clean" action, cascaded down the depends_on graph so
// nothing is left running against infrastructure that was recreated under it.
// Manage-level, like power actions and ad-hoc dispatch.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

type rebuildRequest struct {
	Target            adHocTarget `json:"target"`
	IncludeDependents bool        `json:"include_dependents"`
	// DryRun resolves and returns the affected set without touching anything --
	// what the confirm dialog calls first to show the blast radius.
	DryRun bool `json:"dry_run"`
}

func (s *Server) handleRebuild(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	var req rebuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// A rebuild only means something once infrastructure exists. A planned build
	// has nothing deployed yet (deploy it); a torn-down one is gone.
	switch build.Status {
	case "deploying", "building", "finished", "failed":
	default:
		writeError(w, http.StatusConflict, errors.New("build is "+build.Status+" -- only a deployed build can be rebuilt"))
		return
	}

	// The depends_on graph comes from the build's own content revision; only
	// needed when cascading to dependents.
	var content *loader.Content
	if req.IncludeDependents {
		content, err = s.loadBuildContent(r, build)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	affected, err := orchestrator.Rebuild(r.Context(), s.Pool, build, content, req.Target, req.IncludeDependents, req.DryRun)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(affected) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("no hosts match this target"))
		return
	}

	if !req.DryRun {
		s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
			BuildID: build.ID, Kind: "rebuild.requested",
			Message: rebuildMessage(len(affected)), Payload: []byte("{}"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"affected": affected, "count": len(affected), "dry_run": req.DryRun})
}

// loadBuildContent resolves a build's own content revision (its git checkout via
// s.Checkouts, or the fixed s.RepoRoot) and loads it -- the same resolution the
// render and step-group handlers use. Soft content errors are tolerated: a
// deployed build's content parsed fine when it deployed, and the depends_on
// edges that load are what the rebuild cascade needs.
func (s *Server) loadBuildContent(r *http.Request, build db.Build) (*loader.Content, error) {
	repoRoot := s.RepoRoot
	if s.Checkouts != nil {
		dir, err := s.Checkouts.ForBuild(r.Context(), build.ID)
		if err != nil {
			return nil, fmt.Errorf("resolving content: %w", err)
		}
		repoRoot = dir
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}
	return c, nil
}

func rebuildMessage(total int) string {
	noun := "host"
	if total != 1 {
		noun = "hosts"
	}
	return strconv.Itoa(total) + " " + noun + ": rebuild"
}
