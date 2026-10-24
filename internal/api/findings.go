// Findings: "Findings instantiated in this build, grouped by severity,
// with the object each came from and whether it was direct or inherited
// from a script", plus "Findings export: an API pull,
// scoped to a build". Nothing in the repo read or
// resolved findings before this -- content rows
// (environment/network/host/container/script) each carry their own
// `findings jsonb` column (ingested from the start), but "a finding on
// a script comes along with the script to every host or container that
// runs it" is a real resolution step that had never been implemented.
//
// Resolution walks each build's real deployed_object rows (one per team
// per network/host/container -- so a finding genuinely appears once per
// team, matching "findings instantiated in this build", not once per
// content definition) and, for a host or container, also looks up every
// script its `steps` reference and includes that script's own findings,
// tagged as inherited. Environment-level findings appear once per build,
// not per team, since the environment itself isn't instantiated per team.
package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// findingInstance is one resolved finding, attached to the specific
// build/team/object it applies to -- the unit both the JSON and CSV
// export shapes are built from.
type findingInstance struct {
	TeamNumber  *int32 `json:"team_number,omitempty"` // nil for an environment-level finding
	ObjectKind  string `json:"object_kind"`           // environment | network | host | container
	ObjectName  string `json:"object_name"`
	AsName      string `json:"as_name,omitempty"`
	Source      string `json:"source"` // "direct" or "script:<name>"
	Severity    int    `json:"severity"`
	Difficulty  int    `json:"difficulty"`
	Description string `json:"description"`
}

func decodeFindings(raw []byte) []loader.Finding {
	var out []loader.Finding
	if len(raw) == 0 {
		return nil
	}
	json.Unmarshal(raw, &out) // malformed/empty jsonb -> no findings, not a 500; schema validation at ingest is what should have caught this
	return out
}

// scriptNamesFromSteps extracts every script a host/container's steps
// reference via a direct `script: name` step. Scripts referenced only by a
// top-level `schedule:` entry are NOT collected here: `schedule:` is a separate
// list from `steps:` (see internal/schedule and loader.Host.Schedule), the
// schedule engine materializes those entries into scheduled_task rows at deploy,
// and the persisted steps blob this reads never contains a schedule entry.
// (Consequence: a script referenced only by a schedule entry isn't attributed
// findings in the per-script breakdown -- a known gap, not something
// this function can close without the
// schedule list being available at this query point.)
func scriptNamesFromSteps(raw []byte) []string {
	var steps []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for _, step := range steps {
		if raw, ok := step["script"]; ok {
			var name string
			if json.Unmarshal(raw, &name) == nil {
				add(name)
			}
		}
	}
	return names
}

// mergeScriptNames unions two script-name lists, preserving order and dropping
// duplicates -- used to combine the scripts a host runs from its steps with
// those it runs from its schedule entries.
func mergeScriptNames(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, name := range append(append([]string{}, a...), b...) {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// resolveFindings does the real work described in this file's package
// doc. scriptFindingsCache and {host,container,network}Cache are
// per-request memoization -- every team shares the same content
// definitions, so without this a build with N teams would look up the
// same host/script/network row N times over.
func resolveFindings(ctx context.Context, q *db.Queries, build db.Build, revisionID pgtype.UUID) ([]findingInstance, error) {
	var out []findingInstance

	env, err := q.GetEnvironmentByRevisionAndName(ctx, db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: revisionID, Name: build.EnvironmentName,
	})
	if err == nil {
		for _, f := range decodeFindings(env.Findings) {
			out = append(out, findingInstance{ObjectKind: "environment", ObjectName: env.Name, Source: "direct", Severity: f.Severity, Difficulty: f.Difficulty, Description: f.Description})
		}
	}

	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		return nil, err
	}
	teamNumberByID := make(map[string]int32, len(teams))
	for _, tm := range teams {
		teamNumberByID[tm.ID.String()] = tm.TeamNumber
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		return nil, err
	}

	type scriptResult struct {
		findings []loader.Finding
		names    []string // the scripts THIS host/container's steps reference (only set for host/container cache entries)
	}
	networkCache := map[string][]loader.Finding{}
	hostCache := map[string]scriptResult{}
	containerCache := map[string]scriptResult{}
	scriptCache := map[string][]loader.Finding{}

	scriptFindings := func(name string) []loader.Finding {
		if f, ok := scriptCache[name]; ok {
			return f
		}
		s, err := q.GetScriptByRevisionAndName(ctx, db.GetScriptByRevisionAndNameParams{ContentRevisionID: revisionID, Name: name})
		var f []loader.Finding
		if err == nil {
			f = decodeFindings(s.Findings)
		}
		scriptCache[name] = f
		return f
	}

	for _, o := range objs {
		teamNumber := teamNumberByID[o.TeamID.String()]
		asName := db.StrOrEmpty(o.AsName)

		switch o.Kind {
		case "network":
			findings, ok := networkCache[o.ObjectName]
			if !ok {
				n, err := q.GetNetworkByRevisionAndName(ctx, db.GetNetworkByRevisionAndNameParams{ContentRevisionID: revisionID, Name: o.ObjectName})
				if err == nil {
					findings = decodeFindings(n.Findings)
				}
				networkCache[o.ObjectName] = findings
			}
			for _, f := range findings {
				out = append(out, findingInstance{TeamNumber: &teamNumber, ObjectKind: "network", ObjectName: o.ObjectName, Source: "direct", Severity: f.Severity, Difficulty: f.Difficulty, Description: f.Description})
			}

		case "host", "container":
			var res scriptResult
			var ok bool
			if o.Kind == "host" {
				res, ok = hostCache[o.ObjectName]
			} else {
				res, ok = containerCache[o.ObjectName]
			}
			if !ok {
				if o.Kind == "host" {
					h, err := q.GetHostByRevisionAndName(ctx, db.GetHostByRevisionAndNameParams{ContentRevisionID: revisionID, Name: o.ObjectName})
					if err == nil {
						// Scripts a host runs from either its steps OR its schedule
						// entries expose it to those scripts' findings.
						res = scriptResult{findings: decodeFindings(h.Findings), names: mergeScriptNames(scriptNamesFromSteps(h.Steps), scriptNamesFromSteps(h.Schedule))}
					}
					hostCache[o.ObjectName] = res
				} else {
					c, err := q.GetContainerByRevisionAndName(ctx, db.GetContainerByRevisionAndNameParams{ContentRevisionID: revisionID, Name: o.ObjectName})
					if err == nil {
						res = scriptResult{findings: decodeFindings(c.Findings), names: mergeScriptNames(scriptNamesFromSteps(c.Steps), scriptNamesFromSteps(c.Schedule))}
					}
					containerCache[o.ObjectName] = res
				}
			}
			for _, f := range res.findings {
				out = append(out, findingInstance{TeamNumber: &teamNumber, ObjectKind: o.Kind, ObjectName: o.ObjectName, AsName: asName, Source: "direct", Severity: f.Severity, Difficulty: f.Difficulty, Description: f.Description})
			}
			for _, scriptName := range res.names {
				for _, f := range scriptFindings(scriptName) {
					out = append(out, findingInstance{TeamNumber: &teamNumber, ObjectKind: o.Kind, ObjectName: o.ObjectName, AsName: asName, Source: "script:" + scriptName, Severity: f.Severity, Difficulty: f.Difficulty, Description: f.Description})
				}
			}
		}
	}

	return out, nil
}

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	findings, err := resolveFindings(r.Context(), s.Queries, build, build.ContentRevisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if r.URL.Query().Get("format") == "csv" {
		writeFindingsCSV(w, findings)
		return
	}
	if findings == nil {
		findings = []findingInstance{}
	}
	writeJSON(w, http.StatusOK, findings)
}

func writeFindingsCSV(w http.ResponseWriter, findings []findingInstance) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="findings.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"team", "object_kind", "object_name", "as_name", "source", "severity", "difficulty", "description"})
	for _, f := range findings {
		team := ""
		if f.TeamNumber != nil {
			team = strconv.Itoa(int(*f.TeamNumber))
		}
		cw.Write([]string{
			team, f.ObjectKind, f.ObjectName, f.AsName, f.Source,
			strconv.Itoa(f.Severity), strconv.Itoa(f.Difficulty), f.Description,
		})
	}
	cw.Flush()
}
