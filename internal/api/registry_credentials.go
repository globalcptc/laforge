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
	"sync"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/registryclient"
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

// handleTestRegistryCredential checks that a stored credential actually
// authenticates against its registry -- the answer an operator wants before a
// build depends on a private image. A failed test is a normal result (200 with
// ok:false and the reason), not an API error.
type registryTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *Server) handleTestRegistryCredential(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	host := r.PathValue("host")
	cred, err := s.Queries.GetRegistryCredentialByHost(r.Context(), host)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no stored credential for that registry host"))
		return
	}
	if err := registryclient.New().Ping(r.Context(), cred.RegistryHost, cred.Username, cred.Secret); err != nil {
		writeJSON(w, http.StatusOK, registryTestResult{OK: false, Message: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, registryTestResult{OK: true, Message: "authenticated successfully"})
}

// handleListRegistryImages lists the repositories a registry holds and each
// one's tags, so a content author can confirm the image they pushed is really
// there (and spelled the way their `container:` names it). Registries that
// don't offer a catalog (Docker Hub, GHCR) come back supported:false with an
// explanation rather than an error.
type registryImage struct {
	Repository string   `json:"repository"`
	Tags       []string `json:"tags"`
	Error      string   `json:"error,omitempty"`
}

type registryImagesView struct {
	Supported bool            `json:"supported"`
	Message   string          `json:"message,omitempty"`
	Truncated bool            `json:"truncated"`
	Images    []registryImage `json:"images"`
}

func (s *Server) handleListRegistryImages(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	host := r.PathValue("host")
	cred, err := s.Queries.GetRegistryCredentialByHost(r.Context(), host)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no stored credential for that registry host"))
		return
	}
	reg := registryclient.New()
	repos, err := reg.Catalog(r.Context(), cred.RegistryHost, cred.Username, cred.Secret)
	if errors.Is(err, registryclient.ErrCatalogUnsupported) {
		writeJSON(w, http.StatusOK, registryImagesView{Supported: false, Message: err.Error(), Images: []registryImage{}})
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	// Bounded concurrency: a registry can hold many repositories, and each
	// tags lookup is its own request.
	const maxRepos = 500
	truncated := false
	if len(repos) > maxRepos {
		repos, truncated = repos[:maxRepos], true
	}
	images := make([]registryImage, len(repos))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, repo string) {
			defer wg.Done()
			defer func() { <-sem }()
			img := registryImage{Repository: repo, Tags: []string{}}
			if tags, terr := reg.Tags(r.Context(), cred.RegistryHost, cred.Username, cred.Secret, repo); terr != nil {
				img.Error = terr.Error()
			} else if tags != nil {
				img.Tags = tags
			}
			images[i] = img
		}(i, repo)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, registryImagesView{Supported: true, Truncated: truncated, Images: images})
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
