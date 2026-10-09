// Real dropdowns instead of blind text fields for "a build is always
// configured explicitly: repository, branch, environment file, builder
// config" -- direct product feedback: branch and
// environment file should be select inputs "as we should have all of
// that information," the same way builder config already can be
// (useBuilderConfigs already existed). These two endpoints are what that
// information actually comes from: GitHub's own branch list, and a real
// parse of whatever environment files exist on a given branch right now.
package api

import (
	"errors"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/loader"
)

var (
	errNoGitHubCredentials = errors.New("no GitHub App or service token is configured on this server -- real branches can't be listed")
	errBranchRequired      = errors.New("branch is required")
)

type branchInfo struct {
	Name string `json:"name"`
}

type listBranchesResponse struct {
	DefaultBranch string       `json:"default_branch"`
	Branches      []branchInfo `json:"branches"`
}

// handleListBranches lists a repository's real branches from GitHub --
// requires a real token (GitHub App installation or GITHUB_SERVICE_TOKEN)
// to be configured, same requirement handleRenderObject's own checkout
// cache already has; a deployment with neither gets a clear error here
// rather than a silently empty list, so the UI can fall back to a plain
// text input instead of pretending a dropdown with nothing in it is real.
func (s *Server) handleListBranches(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	if s.Checkouts == nil {
		writeError(w, http.StatusServiceUnavailable, errNoGitHubCredentials)
		return
	}
	token, err := s.Checkouts.ResolveToken(r.Context(), repo)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	ghRepo, err := s.GH.GetRepo(r.Context(), token, repo.GithubOwner, repo.GithubRepo)
	if err != nil {
		writeError(w, statusUpstreamFailed, err)
		return
	}
	branches, err := s.GH.ListBranches(r.Context(), token, repo.GithubOwner, repo.GithubRepo)
	if err != nil {
		writeError(w, statusUpstreamFailed, err)
		return
	}
	out := make([]branchInfo, 0, len(branches))
	for _, b := range branches {
		out = append(out, branchInfo{Name: b.Name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, listBranchesResponse{DefaultBranch: ghRepo.DefaultBranch, Branches: out})
}

type environmentFileInfo struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type listEnvironmentFilesResponse struct {
	Files     []environmentFileInfo `json:"files"`
	HasErrors bool                  `json:"has_errors"`
	// YAMLFiles is every .yaml/.yml file on the branch (paths relative to
	// the repository root), for browsing the branch in a file picker; the
	// ones in Files are the environment files among them.
	YAMLFiles []string `json:"yaml_files"`
}

// maxYAMLFiles caps the file picker's listing for very large repositories.
const maxYAMLFiles = 5000

func listYAMLFiles(root string) []string {
	out := []string{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		if rel, err := filepath.Rel(root, path); err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		if len(out) >= maxYAMLFiles {
			return filepath.SkipAll
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// handleListEnvironmentFiles parses a real checkout of the given branch
// (?branch=, required when s.Checkouts is configured) and returns every
// environment: file it found -- loader.Environment.SourceFile is exactly
// the path a configured build's own environment_path field expects.
// Falls back to the fixed s.RepoRoot (ignoring branch) when no
// GitHub credentials are configured, same degraded single-repo behavior
// handleRenderObject already has. A file that fails to parse doesn't
// hide the ones that did (loader.Load's own "partial results" doc
// comment) -- HasErrors just tells the UI to also mention that
// somewhere, not to hide the dropdown.
func (s *Server) handleListEnvironmentFiles(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}

	root := s.RepoRoot
	if s.Checkouts != nil {
		branch := r.URL.Query().Get("branch")
		if branch == "" {
			writeError(w, http.StatusBadRequest, errBranchRequired)
			return
		}
		dir, err := s.Checkouts.ForRef(r.Context(), repo.ID, branch)
		if err != nil {
			writeError(w, statusUpstreamFailed, err)
			return
		}
		root = dir
	}

	c, err := loader.Load(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	files := make([]environmentFileInfo, 0, len(c.Environments))
	for _, e := range c.Environments {
		files = append(files, environmentFileInfo{Path: e.SourceFile, Name: e.Name})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	writeJSON(w, http.StatusOK, listEnvironmentFilesResponse{Files: files, HasErrors: len(c.Errors) > 0, YAMLFiles: listYAMLFiles(root)})
}
