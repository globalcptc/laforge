package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/db"
)

type createRepoRequest struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

// handleCreateRepo is "repo registration": "Admin registers a repository.
// Authorization is GitHub's, per repository: if you can push to the repo,
// you can build its environments." Nothing here reaches out to GitHub to
// double-check the repo actually exists beyond the permission check
// itself -- GetRepo already 404s (surfaced as 403, see auth.go) if it
// doesn't, or if the token can't see it either way.
func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Owner == "" || req.Repo == "" {
		writeError(w, http.StatusBadRequest, errors.New("owner and repo are both required"))
		return
	}
	if err := s.requirePush(r.Context(), r, req.Owner, req.Repo); err != nil {
		writeAuthError(w, err)
		return
	}
	repo, err := s.Queries.CreateRepository(r.Context(), db.CreateRepositoryParams{
		GithubOwner: req.Owner, GithubRepo: req.Repo,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, repo)
}

// handleListRepos lists repositories. A signed-in session (the web UI)
// gets only the repositories it can access -- level above none on each,
// the same rule every repository endpoint applies -- and an instance admin
// gets all of them. Without a session (the CLI's bearer-token calls) the
// list is unfiltered, as before: owner/name isn't sensitive, and every
// operation on a repository is checked on its own.
func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := s.Queries.ListRepositories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if repos == nil {
		repos = []db.Repository{}
	}
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusOK, repos)
		return
	}
	out := s.visibleRepositories(r.Context(), sess, repos)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

// repoByIDParam resolves the {id} path value to a repository row -- used
// by every handler that needs to know which owner/repo a request concerns
// before it can run requirePush against GitHub.
func (s *Server) repoByIDParam(r *http.Request) (db.Repository, error) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		return db.Repository{}, err
	}
	repo, err := s.Queries.GetRepository(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Repository{}, errors.New("no such repository")
		}
		return db.Repository{}, err
	}
	return repo, nil
}

func (s *Server) handleListRevisionEnvironments(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	envs, err := s.Queries.ListEnvironmentsByRevision(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, envs)
}

// visibleRepositories narrows repos to the ones sess can access (level
// above none); an instance admin sees them all.
func (s *Server) visibleRepositories(ctx context.Context, sess authSession, repos []db.Repository) []db.Repository {
	out := []db.Repository{}
	if s.isInstanceAdmin(sess) {
		return append(out, repos...)
	}
	visible := make([]bool, len(repos))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, repo := range repos {
		wg.Add(1)
		go func(i int, repo db.Repository) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			level, err := s.effectiveLevel(ctx, sess, repo)
			visible[i] = err == nil && level > levelNone
		}(i, repo)
	}
	wg.Wait()
	for i, repo := range repos {
		if visible[i] {
			out = append(out, repo)
		}
	}
	return out
}
