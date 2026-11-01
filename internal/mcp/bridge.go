package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// Most tools are the API's own read pages, fetched through the API's read
// bridge (api.MCPAccess.Get) as the key's owner: the same JSON the UI shows,
// passwords and rendered scripts included. Before each one, the tool checks
// the build or repository against the key's readable set, which is narrower
// than the owner's when the key is limited to some repositories.

const (
	defaultFieldChars = 4000
	maxFieldChars     = 200_000
)

// fieldChars is the per-string cap a tool applies to the page it returns.
type fieldChars struct {
	MaxFieldChars int `json:"max_field_chars,omitempty" jsonschema:"longest string (script, output, log line) to return in full; longer ones are cut with a note. Default 4000, at most 200000"`
}

func (f fieldChars) limit() int { return clamp(f.MaxFieldChars, defaultFieldChars, maxFieldChars) }

// get fetches one API read page and returns its JSON, long strings clipped.
func (c *call) get(ctx context.Context, path string, q url.Values, maxField int) (any, error) {
	status, body, err := c.s.cfg.Access.Get(ctx, c.key.AccountID, c.key.Login, path, q)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &e)
		if e.Error == "" {
			e.Error = strings.TrimSpace(string(body))
		}
		switch status {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, fmt.Errorf("%w (%s)", errDenied, e.Error)
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found: %s", e.Error)
		}
		return nil, fmt.Errorf("LaForge answered %d: %s", status, e.Error)
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	v = clipStrings(v, maxField)
	if list, ok := v.([]any); ok {
		// Tool results are objects; a page that returns a list comes back
		// as {"items": [...], "count": n}.
		return map[string]any{"items": list, "count": len(list)}, nil
	}
	return v, nil
}

func clipStrings(v any, max int) any {
	switch t := v.(type) {
	case string:
		if len(t) <= max {
			return t
		}
		cut := max
		for cut > 0 && !utf8.RuneStart(t[cut]) {
			cut--
		}
		return t[:cut] + fmt.Sprintf("… [cut: %d more bytes; ask again with a larger max_field_chars]", len(t)-cut)
	case []any:
		for i := range t {
			t[i] = clipStrings(t[i], max)
		}
	case map[string]any:
		for k := range t {
			t[k] = clipStrings(t[k], max)
		}
	}
	return v
}

// object resolves an object named by id, id prefix or name -- within a team
// when one is given -- to one object in the build.
func (c *call) object(ctx context.Context, build pgtype.UUID, ref string, team int) (pgtype.UUID, error) {
	row, err := c.findObject(ctx, build, ref, team)
	return row.ID, err
}

// findObject is object, returning the matched row (its kind and name too).
func (c *call) findObject(ctx context.Context, build pgtype.UUID, ref string, team int) (db.McpFindObjectInBuildRow, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return db.McpFindObjectInBuildRow{}, errors.New("object is required: an object id, or its name (with team when several teams have one)")
	}
	rows, err := c.q().McpFindObjectInBuild(ctx, db.McpFindObjectInBuildParams{BuildID: build, Ref: ref, Team: int32(team)})
	if err != nil {
		return db.McpFindObjectInBuildRow{}, err
	}
	if len(rows) > 1 {
		// An exact deployed name (as_name) beats an authored name that
		// several deployed copies share.
		var exact []db.McpFindObjectInBuildRow
		for _, r := range rows {
			if r.AsName != nil && strings.EqualFold(*r.AsName, ref) {
				exact = append(exact, r)
			}
		}
		if len(exact) > 0 {
			rows = exact
		}
	}
	switch len(rows) {
	case 0:
		return db.McpFindObjectInBuildRow{}, fmt.Errorf("no object %q in this build", ref)
	case 1:
		return rows[0], nil
	}
	matches := make([]string, 0, len(rows))
	for _, r := range rows {
		name := r.ObjectName
		if r.AsName != nil && *r.AsName != "" {
			name = *r.AsName
		}
		matches = append(matches, fmt.Sprintf("%s (team %d, id %s)", name, r.TeamNumber, r.ID.String()[:8]))
	}
	return db.McpFindObjectInBuildRow{}, fmt.Errorf("%q matches %d objects: %s; pass team, a more exact name, or the id", ref, len(rows), strings.Join(matches, ", "))
}

// repoID is a readable repository's id, given as owner/name or id.
func (c *call) repoID(ctx context.Context, ref string) (string, error) {
	r, err := c.repo(ctx, ref)
	return r.ID, err
}

// checkRepo confirms a repository id found some other way is readable.
func (c *call) checkRepo(id pgtype.UUID, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errDenied
	}
	if err != nil {
		return err
	}
	if !c.canRead(id) {
		return errDenied
	}
	return nil
}

// addPage registers a tool that returns one API read page. route builds the
// page's path and query after doing the tool's own access checks.
func addPage[In any](s *Server, name, title, description string, route func(context.Context, *call, In) (string, url.Values, int, error)) {
	addTool(s, name, title, description, func(ctx context.Context, c *call, in In) (any, error) {
		path, q, maxField, err := route(ctx, c, in)
		if err != nil {
			return nil, err
		}
		return c.get(ctx, path, q, maxField)
	})
}

// seg escapes one path segment.
func seg(s string) string { return url.PathEscape(s) }
