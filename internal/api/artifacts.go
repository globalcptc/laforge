// Build -> Artifacts: "Storage by class -- agent binaries, rendered scripts,
// uploaded files, step output -- with sizes and a total. Actions: purge
// artifacts, delete build, pin against retention policies. Each states
// exactly what it removes and what survives."
//
// This reports the classes LaForge actually stores today: per-host agent
// binaries (regenerable), captured step output, and the event journal.
// Rendered scripts are produced on demand, not stored, so they're reported
// as a zero-byte, on-demand class rather than pretended to occupy space.
// "Delete build" is the existing teardown/purge lifecycle (the Tear Down
// action), not duplicated here; "purge artifacts" is the one destructive
// action this screen owns, and it removes only the regenerable agent
// binaries.
package api

import (
	"fmt"
	"net/http"

	"github.com/globalcptc/laforge/internal/db"
)

type artifactClass struct {
	Class       string `json:"class"`
	Count       int64  `json:"count"`
	Bytes       int64  `json:"bytes"`
	Purgeable   bool   `json:"purgeable"`   // removed by "Purge artifacts"
	OnDemand    bool   `json:"on_demand"`   // not stored; produced when asked
	Description string `json:"description"` // what it is / what survives
}

type artifactsResponse struct {
	Classes    []artifactClass `json:"classes"`
	TotalBytes int64           `json:"total_bytes"`
}

func (s *Server) handleGetArtifacts(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}

	bins, err := s.Queries.SumAgentBinaryStorageByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out, err := s.Queries.SumStepOutputStorageByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	evt, err := s.Queries.SumEventStorageByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := artifactsResponse{Classes: []artifactClass{
		{
			Class: "Agent binaries", Count: bins.Count, Bytes: bins.Bytes, Purgeable: true,
			Description: "Per-host agent binaries. Regenerated on the next deploy -- safe to purge.",
		},
		{
			Class: "Step output", Count: out.Count, Bytes: out.Bytes,
			Description: "Captured stdout/stderr of every build step. The record of what ran; kept.",
		},
		{
			Class: "Event journal", Count: evt.Count, Bytes: evt.Bytes,
			Description: "Lifecycle and step events. The audit trail; kept.",
		},
		{
			Class: "Rendered scripts", OnDemand: true,
			Description: "Produced on demand from content, not stored -- so they occupy no space here.",
		},
	}}
	resp.TotalBytes = bins.Bytes + out.Bytes + evt.Bytes
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePurgeArtifacts(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	// Destructive: manage, not read.
	sess, err := s.requireLevel(r.Context(), r, repo, levelManage)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	n, err := s.Queries.DeleteAgentArtifactsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: build.ID,
		Kind:    "artifacts.purged",
		Message: fmt.Sprintf("%s purged %d agent binary artifact(s)", sess.GithubLogin, n),
		Payload: []byte("{}"),
	}); err != nil {
		// The purge itself succeeded; a failed audit-log write shouldn't fail the request.
		writeJSON(w, http.StatusOK, map[string]any{"purged": n})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"purged": n})
}
