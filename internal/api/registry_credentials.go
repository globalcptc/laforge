// Private Docker registry credentials (migration 00025). A LaForge
// `container:` whose image ref names a private registry gets a `docker login`
// before the pull (internal/runner/dockerrun.go); these instance-admin
// endpoints manage the stored credentials. The secret is write-only over the
// API: it's accepted on upsert but never returned by list/get.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/globalcptc/laforge/internal/db"
)

func (s *Server) handleListRegistryCredentials(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListRegistryCredentials(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []db.ListRegistryCredentialsRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

type registryCredentialRequest struct {
	RegistryHost string `json:"registry_host"`
	Username     string `json:"username"`
	Secret       string `json:"secret"`
}

func (s *Server) handleUpsertRegistryCredential(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	var req registryCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.RegistryHost = strings.TrimSpace(req.RegistryHost)
	if req.RegistryHost == "" || req.Username == "" || req.Secret == "" {
		writeError(w, http.StatusBadRequest, errors.New("registry_host, username, and secret are all required"))
		return
	}
	row, err := s.Queries.UpsertRegistryCredential(r.Context(), db.UpsertRegistryCredentialParams{
		RegistryHost: req.RegistryHost, Username: req.Username, Secret: req.Secret,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, row) // no secret in the returned row
}

func (s *Server) handleDeleteRegistryCredential(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	host := r.PathValue("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, errors.New("host is required"))
		return
	}
	if err := s.Queries.DeleteRegistryCredential(r.Context(), host); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
