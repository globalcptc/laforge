// Instance admins: the GitHub logins with instance-wide admin (see
// Server.AdminLogins). The list lives in the instance_admin table
// (migration 00042) and is managed by the admins themselves through these
// endpoints; LAFORGE_ADMIN_LOGINS only seeds it on first startup.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/globalcptc/laforge/internal/db"
)

var errLastInstanceAdmin = errors.New("this is the only admin; add another before removing them")

// githubLoginPattern is GitHub's own username rule (alphanumerics and single
// hyphens, 39 characters at most), checked before the login goes into a
// GitHub API path.
var githubLoginPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

// SeedInstanceAdmins runs once at startup: if no admin has ever been
// recorded, the given logins (LAFORGE_ADMIN_LOGINS) become the initial list.
// Once the table has any row the logins are ignored, so editing the env var
// later neither adds an admin nor brings back one removed in the UI.
func (s *Server) SeedInstanceAdmins(ctx context.Context, logins []string) error {
	n, err := s.Queries.CountInstanceAdmins(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		for _, login := range logins {
			if _, err := s.Queries.AddInstanceAdmin(ctx, db.AddInstanceAdminParams{GithubLogin: login}); err != nil {
				return err
			}
		}
	}
	return s.reloadInstanceAdmins(ctx)
}

// reloadInstanceAdmins replaces AdminLogins with what the table holds now.
func (s *Server) reloadInstanceAdmins(ctx context.Context) error {
	rows, err := s.Queries.ListInstanceAdmins(ctx)
	if err != nil {
		return err
	}
	logins := make([]string, 0, len(rows))
	for _, row := range rows {
		logins = append(logins, row.GithubLogin)
	}
	s.adminMu.Lock()
	s.AdminLogins = logins
	s.adminMu.Unlock()
	return nil
}

func (s *Server) isAdminLogin(login string) bool {
	s.adminMu.RLock()
	defer s.adminMu.RUnlock()
	for _, admin := range s.AdminLogins {
		if strings.EqualFold(admin, login) {
			return true
		}
	}
	return false
}

func (s *Server) handleListInstanceAdmins(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListInstanceAdmins(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if rows == nil {
		rows = []db.ListInstanceAdminsRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// handleAddInstanceAdmin looks the login up on GitHub first, the same way
// granting repository access does: it catches a typo before it becomes an
// admin entry nobody can sign in as, and stores GitHub's own spelling.
func (s *Server) handleAddInstanceAdmin(w http.ResponseWriter, r *http.Request) {
	sess, err := s.requireInstanceAdmin(r.Context(), r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	login := strings.TrimPrefix(strings.TrimSpace(req.Login), "@")
	if !githubLoginPattern.MatchString(login) {
		writeError(w, http.StatusBadRequest, errors.New("login must be a GitHub username"))
		return
	}
	ghUser, err := s.GH.GetUserByLogin(r.Context(), sess.GithubToken, login)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("looking up %q on GitHub: %w", login, err))
		return
	}
	if _, err := s.Queries.AddInstanceAdmin(r.Context(), db.AddInstanceAdminParams{
		GithubLogin: ghUser.Login, AddedBy: db.StrPtr(sess.GithubLogin),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.reloadInstanceAdmins(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"github_login": ghUser.Login})
}

// handleDeleteInstanceAdmin takes effect on the removed admin's very next
// request -- their session stays signed in, it just stops being an admin's.
// An admin may remove themselves, as long as someone else is left.
func (s *Server) handleDeleteInstanceAdmin(w http.ResponseWriter, r *http.Request) {
	if _, err := s.requireInstanceAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	login := r.PathValue("login")
	removed, err := s.Queries.DeleteInstanceAdmin(r.Context(), login)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.reloadInstanceAdmins(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Nothing removed: either they weren't an admin (fine, same result) or
	// they're the last one, which DeleteInstanceAdmin refuses.
	if removed == 0 && s.isAdminLogin(login) {
		writeError(w, http.StatusConflict, errLastInstanceAdmin)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
