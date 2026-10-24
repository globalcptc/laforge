// The one endpoint a booting host hits before it has any identity of its
// own: it downloads its per-host agent binary (stored by the runner at
// deploy time) using the one-time token its cloud-init user-data carries.
// No session, no mTLS -- the token is the capability, and the bytes it
// returns are that host's own agent, already carrying its own credentials.
// A wrong or spent token is a plain 404.
package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) handleGetAgentBinary(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, errors.New("token is required"))
		return
	}
	row, err := s.Queries.GetAgentArtifactByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// The {id} in the path is cosmetic (it makes the URL self-describing
	// and logs readable); the token is what authorizes. A mismatch is
	// still a 404 rather than leaking that the token exists for a
	// different object.
	if id := r.PathValue("id"); id != "" && id != row.DeployedObjectID.String() {
		http.NotFound(w, r)
		return
	}
	log.Printf("agent-binary: serving %s (%s, %d bytes) to %s", row.DeployedObjectID.String(), row.Platform, len(row.AgentBinary), r.RemoteAddr)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=laforge-agent")
	w.WriteHeader(http.StatusOK)
	w.Write(row.AgentBinary)
}
