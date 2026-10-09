// The LaForge-side half of "install on GitHub, approve in LaForge":
// webhook.go's
// installation/installation_repositories handlers record what's
// technically reachable; these two endpoints are what an admin uses to
// turn "reachable" into "actually tracked" -- the second trust boundary,
// deliberately separate from the first.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// requireInstanceAdmin is deliberately not requireLevel: approving a
// repository is the one action with no repository row yet to check
// repository_access against (that's the whole point -- there isn't one
// until this succeeds), so it needs a floor that isn't scoped to any one
// repo. The admin list is intentionally narrow and explicit (a managed
// list of GitHub logins -- see admins.go -- not a GitHub org-ownership
// lookup) rather than growing a second parallel authorization model.
func (s *Server) requireInstanceAdmin(ctx context.Context, r *http.Request) (authSession, error) {
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		return authSession{}, err
	}
	if s.isInstanceAdmin(sess) {
		return sess, nil
	}
	return authSession{}, errForbidden
}

func (s *Server) isInstanceAdmin(sess authSession) bool {
	return s.isAdminLogin(sess.GithubLogin)
}

// installedRepoView is one repository an installation covers, with its
// real approval state -- the other half of installationView's own fix
// (see its doc comment).
type installedRepoView struct {
	GithubOwner  string `json:"github_owner"`
	GithubRepo   string `json:"github_repo"`
	GithubRepoID int64  `json:"github_repo_id"`
	Approved     bool   `json:"approved"`
	// RepositoryID is LaForge's own id once approved -- what the
	// repository's access page is keyed on. Empty while pending.
	RepositoryID string `json:"repository_id,omitempty"`
}

// installationView is one GitHub App installation, with every repository
// it currently covers -- approved or not. Real product feedback, not a
// speculative redesign: "Installations should list everywhere we have
// the app installed that we know about... even if it's just us linking
// to GitHub pages" -- the old screen only ever showed unapproved repos,
// so an installation with everything already approved (the common case
// once set up) simply disappeared from the one place meant to make every
// installation manageable.
type installationView struct {
	db.GithubInstallation
	Repos []installedRepoView `json:"repos"`
}

// handleListInstallations is the real fix: every installation this
// instance knows about (ListGithubInstallations), each with every repo
// it covers and that repo's real approved/pending state
// (ListAllInstallationRepositories) -- assembled here rather than one
// nested SQL query, the same "two plain queries, joined in Go" shape
// attachAgentHealth (health.go) already uses for a similar per-parent
// grouping.
func (s *Server) handleListInstallations(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	installations, err := s.Queries.ListGithubInstallations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	repos, err := s.Queries.ListAllInstallationRepositories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	byInstallation := make(map[pgtype.UUID][]installedRepoView, len(installations))
	for _, repo := range repos {
		view := installedRepoView{
			GithubOwner: repo.GithubOwner, GithubRepo: repo.GithubRepo, GithubRepoID: repo.GithubRepoID, Approved: repo.Approved,
		}
		if repo.RepositoryID.Valid {
			view.RepositoryID = repo.RepositoryID.String()
		}
		byInstallation[repo.InstallationID] = append(byInstallation[repo.InstallationID], view)
	}
	views := make([]installationView, len(installations))
	for i, inst := range installations {
		views[i] = installationView{GithubInstallation: inst, Repos: byInstallation[inst.ID]}
	}
	writeJSON(w, http.StatusOK, views)
}

// handleListUnapprovedInstalledRepositories is "install on GitHub,
// approve in LaForge"'s own list: every repository some installation
// currently covers that has no matching `repository` row yet.
func (s *Server) handleListUnapprovedInstalledRepositories(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListUnapprovedInstallationRepositories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []db.ListUnapprovedInstallationRepositoriesRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

type approveInstalledRepositoryRequest struct {
	Owner          string `json:"owner"`
	Repo           string `json:"repo"`
	InstallationID string `json:"installation_id"` // github_installation.id (uuid), not GitHub's own numeric installation id
}

// handleApproveInstalledRepository creates the real `repository` row --
// the same effect as the existing POST /repos, but sourced from an
// installed repository rather than typed in by hand, and tagged with
// which installation it came from so its content can be fetched through
// that installation's own scoped token (webhook.go's resolveCloneURL)
// rather than a broader one.
func (s *Server) handleApproveInstalledRepository(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	var req approveInstalledRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Owner == "" || req.Repo == "" || req.InstallationID == "" {
		writeError(w, http.StatusBadRequest, errors.New("owner, repo, and installation_id are all required"))
		return
	}
	installationID, err := parseUUID(req.InstallationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	repo, err := s.Queries.ApproveInstalledRepository(r.Context(), db.ApproveInstalledRepositoryParams{
		GithubOwner: req.Owner, GithubRepo: req.Repo, InstallationID: installationID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, repo)
}

// handleDeleteInstallation removes a GitHub connection. With a GitHub App
// configured it first uninstalls the App from that account on GitHub (the
// same as Uninstall on GitHub's own page), so the connection can't quietly
// come back; then it forgets the installation here, whether or not GitHub's
// own "deleted" webhook ever arrives -- which is the case this exists for: an
// install recorded before webhooks could reach LaForge, or one uninstalled
// while they couldn't. Approved repositories stay tracked (their builds are
// LaForge's own) but lose the installation; content fetches fall back to
// GITHUB_SERVICE_TOKEN.
func (s *Server) handleDeleteInstallation(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid installation id"))
		return
	}
	inst, err := s.Queries.GetGithubInstallation(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent) // already gone
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	uninstalled := false
	if s.AppID != "" && s.AppPrivateKey != nil {
		jwt, err := ghclient.GenerateAppJWT(s.AppID, s.AppPrivateKey, time.Now())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		err = s.GH.DeleteInstallation(r.Context(), jwt, inst.InstallationID)
		var apiErr *ghclient.APIError
		switch {
		case err == nil:
			uninstalled = true
		case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
			// Already uninstalled on GitHub -- only LaForge still had it.
		default:
			writeError(w, statusUpstreamFailed, fmt.Errorf("uninstalling the GitHub App from %s: %w -- nothing was removed; try again, or uninstall it on GitHub", inst.AccountLogin, err))
			return
		}
	}
	if err := s.Queries.DeleteGithubInstallation(r.Context(), inst.InstallationID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"uninstalled_on_github": uninstalled})
}
