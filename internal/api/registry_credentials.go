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
	"sort"
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

// handleTestRegistryCredential checks that a stored credential can actually
// PULL from its registry -- not just that it authenticates. It's the answer an
// operator wants before a build depends on a private image, and it catches the
// common trap where a login succeeds but the account lacks pull permission (a
// Harbor robot never granted Pull on the project), which otherwise only surfaces
// as a 401 at deploy time. A failed test is a normal result (200 with ok:false
// and the reason), not an API error.
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
	repo, err := registryclient.New().VerifyPull(r.Context(), cred.RegistryHost, cred.Username, cred.Secret)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, registryTestResult{OK: true, Message: "authenticated and pull verified (tested " + repo + ")"})
	case errors.Is(err, registryclient.ErrPullUnverifiable):
		// The login works; we just can't auto-probe pull (no catalog). Not a
		// failure -- pulls by exact image name may still work (Docker Hub, GHCR).
		writeJSON(w, http.StatusOK, registryTestResult{OK: true, Message: err.Error()})
	case errors.Is(err, registryclient.ErrPullDenied):
		// Authenticated but not authorized to pull -- name the repo we tried so
		// the operator knows exactly what permission to grant.
		msg := err.Error()
		if repo != "" {
			msg = "authenticated, but not authorized to pull " + repo + " -- grant this account Pull permission on the registry/project"
		}
		writeJSON(w, http.StatusOK, registryTestResult{OK: false, Message: msg})
	default:
		writeJSON(w, http.StatusOK, registryTestResult{OK: false, Message: err.Error()})
	}
}

// handleListRegistryImages lists the repositories a registry holds and each
// one's tags, so a content author can confirm the image they pushed is really
// there (and spelled the way their `container:` names it). Registries that
// don't offer a catalog (Docker Hub, GHCR) come back supported:false with an
// explanation rather than an error.
type registryImage struct {
	Repository string   `json:"repository"`
	Tags       []string `json:"tags"`      // at most maxTagsShown, newest-looking first
	TagCount   int      `json:"tag_count"` // total tags the repo has (Tags may be fewer)
	Error      string   `json:"error,omitempty"`
}

// maxTagsShown bounds how many tags per repository the images view returns. A
// cache like docker-hub-cache/library/php can hold hundreds; shipping and
// rendering them all makes the list unusable, and an author only needs to
// confirm the handful of current ones. The response carries the full TagCount so
// the UI can say "+N more".
const maxTagsShown = 10

// topTags returns the newest-looking tags first (the `latest` tag, then a
// natural, numeric-aware descending order so "8.11" sorts above "8.9"), capped
// to maxTagsShown, along with the total count.
func topTags(tags []string) (shown []string, total int) {
	total = len(tags)
	s := append([]string(nil), tags...)
	sort.Slice(s, func(i, j int) bool {
		if (s[i] == "latest") != (s[j] == "latest") {
			return s[i] == "latest" // pin `latest` to the front
		}
		return natLess(s[j], s[i]) // descending natural order
	})
	if len(s) > maxTagsShown {
		s = s[:maxTagsShown]
	}
	return s, total
}

// natLess compares two tags in natural order: digit runs compare as numbers
// (so "8.11" > "8.9"), everything else byte-by-byte. Keeps "a few of the
// latest" meaningful without a full semver parser for arbitrary docker tags.
func natLess(a, b string) bool {
	ia, ib := 0, 0
	for ia < len(a) && ib < len(b) {
		if isDigit(a[ia]) && isDigit(b[ib]) {
			sa, sb := ia, ib
			for ia < len(a) && isDigit(a[ia]) {
				ia++
			}
			for ib < len(b) && isDigit(b[ib]) {
				ib++
			}
			na := strings.TrimLeft(a[sa:ia], "0")
			nb := strings.TrimLeft(b[sb:ib], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb) // fewer digits = smaller number
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if a[ia] != b[ib] {
			return a[ia] < b[ib]
		}
		ia++
		ib++
	}
	return len(a) < len(b)
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

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
			} else {
				img.Tags, img.TagCount = topTags(tags)
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
