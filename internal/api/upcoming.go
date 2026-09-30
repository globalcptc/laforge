// handleGetUpcoming is the environment dashboard's "upcoming changes":
// "if a newer commit has built, what deploying it would alter, at
// environment, team, network, and host level". Needs the API's
// git-checkout capability, provided by RepoRoot. Reuses
// internal/orchestrator.DiffUpcoming, the same resolution and
// fingerprinting Reconcile itself uses, read-only.
package api

import (
	"net/http"

	"github.com/globalcptc/laforge/internal/orchestrator"
)

type upcomingResponse struct {
	// Pending is false whenever there's nothing meaningful to compare:
	// this build has no configured_build behind it (an ad-hoc/manually
	// triggered build), or its configured_build's currently tracked
	// commit is the same one this build was created from. Changes is
	// always empty when Pending is false.
	Pending bool                          `json:"pending"`
	Changes []orchestrator.UpcomingChange `json:"changes"`
}

func (s *Server) handleGetUpcoming(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}

	if !build.ConfiguredBuildID.Valid {
		writeJSON(w, http.StatusOK, upcomingResponse{Changes: []orchestrator.UpcomingChange{}})
		return
	}
	cb, err := s.Queries.GetConfiguredBuild(r.Context(), build.ConfiguredBuildID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !cb.CurrentContentRevisionID.Valid || cb.CurrentContentRevisionID == build.ContentRevisionID {
		writeJSON(w, http.StatusOK, upcomingResponse{Changes: []orchestrator.UpcomingChange{}})
		return
	}

	changes, err := orchestrator.DiffUpcoming(r.Context(), s.Pool, s.RepoRoot, build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if changes == nil {
		changes = []orchestrator.UpcomingChange{}
	}
	writeJSON(w, http.StatusOK, upcomingResponse{Pending: true, Changes: changes})
}
