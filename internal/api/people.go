// People: "the user database from this repo's CSVs. Searchable by name,
// username, department, title. Shows passwords, because during an event
// that is the point." internal/loader already parses
// every people/*.csv column into a free-form attributes bag per person
// (internal/loader/people.go), and internal/ingest already persists it
// (person.attributes jsonb) -- nothing read it back before;
// ListPersonByRevision (internal/db/queries/content.sql) is
// the query this was missing.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type personView struct {
	ID               string            `json:"id"`
	PeopleSourceName string            `json:"people_source_name"`
	Username         string            `json:"username"`
	Attributes       map[string]string `json:"attributes"`
}

// handleListPeople is scoped by repository (not build): the plan's own
// model is "the user database from this repo's CSVs" -- people are
// content, not a build's runtime state, so the natural home is the
// repository's latest ingested revision, not any one build. `?revision=`
// picks a specific content_revision_id instead, for looking at an older
// commit's cast; `?q=` searches username and every attribute value
// case-insensitively (department, title, domain_admin, ... -- whatever
// columns that repo's CSVs actually have, since attributes is free-form).
func (s *Server) handleListPeople(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	revisionID, err := s.resolveRevisionParam(r, repo.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rows, err := s.Queries.ListPersonByRevision(r.Context(), revisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	q := strings.ToLower(r.URL.Query().Get("q"))
	people := make([]personView, 0, len(rows))
	for _, row := range rows {
		var attrs map[string]string
		json.Unmarshal(row.Attributes, &attrs)
		if q != "" && !personMatches(row.Username, attrs, q) {
			continue
		}
		people = append(people, personView{
			ID: row.ID.String(), PeopleSourceName: row.PeopleSourceName, Username: row.Username, Attributes: attrs,
		})
	}
	writeJSON(w, http.StatusOK, people)
}

func personMatches(username string, attrs map[string]string, needle string) bool {
	if strings.Contains(strings.ToLower(username), needle) {
		return true
	}
	for _, v := range attrs {
		if strings.Contains(strings.ToLower(v), needle) {
			return true
		}
	}
	return false
}

// resolveRevisionParam is shared by handleListPeople and (via the same
// ?revision= convention) anything else that's scoped to a repository's
// content rather than a specific build -- defaults to the repository's
// most recently ingested revision when not given explicitly.
func (s *Server) resolveRevisionParam(r *http.Request, repoID pgtype.UUID) (pgtype.UUID, error) {
	if v := r.URL.Query().Get("revision"); v != "" {
		return parseUUID(v)
	}
	rev, err := s.Queries.GetLatestContentRevisionByRepository(r.Context(), repoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, errors.New("this repository has no ingested content yet")
		}
		return pgtype.UUID{}, err
	}
	return rev.ID, nil
}
