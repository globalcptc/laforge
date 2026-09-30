// handleDetectDrift is a leftover wired to a real caller: an
// admin-triggered "detect drift for this
// build" action (internal/orchestrator.DetectDrift), not a background
// sweep -- see that function's own doc comment for why. Gated at
// levelManage, same floor as teardown/close-open-access: this makes a
// real, live call to the hoster's own API (Builder.Inspect), not just a
// database read.
package api

import (
	"net/http"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

type driftReportResponse struct {
	Orphaned []builder.Resource  `json:"orphaned"`
	Missing  []db.DeployedObject `json:"missing"`
	Tracked  int                 `json:"tracked"`
}

func (s *Server) handleDetectDrift(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	report, err := orchestrator.DetectDrift(r.Context(), s.Pool, build.ID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, driftReportResponse{
		Orphaned: nonNilResources(report.Orphaned),
		Missing:  nonNilObjects(report.Missing),
		Tracked:  report.Tracked,
	})
}

func nonNilResources(r []builder.Resource) []builder.Resource {
	if r == nil {
		return []builder.Resource{}
	}
	return r
}

func nonNilObjects(o []db.DeployedObject) []db.DeployedObject {
	if o == nil {
		return []db.DeployedObject{}
	}
	return o
}
