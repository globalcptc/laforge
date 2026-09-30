// The read side of "A build keeps its resolved topology and every
// rendered script ... byte-identical to what agents received"
// handleRenderObject is the API's answer to "the
// exact script delivered to a given host in a given team can be
// retrieved from the UI" -- reusing internal/render.Resolve and
// RenderScript exactly as `laforge render` and internal/orchestrator's
// own Fingerprint already do, so this is guaranteed to match what the
// agent actually ran, not a second reimplementation that could drift.
package api

import (
	"errors"
	"net/http"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

type renderedStep struct {
	Index      int    `json:"index"`
	Action     string `json:"action"`
	ScriptName string `json:"script_name,omitempty"`
	// Rendered holds the real output of internal/render.RenderScript for
	// a "script" step -- byte-identical to what the agent would download
	// and run. Only "script" steps get this treatment; every other kind
	// (run, write_file, download, ...) is shown as its own raw step JSON
	// instead, same scope as internal/orchestrator.Fingerprint's own doc
	// comment on what it does and doesn't render.
	Rendered string      `json:"rendered,omitempty"`
	Raw      loader.Step `json:"raw"`
}

// handleRenderObject resolves this deployed object's real Context
// (internal/render.Resolve, keyed on its build's environment, its own
// `as` name, and its team number) against a real git checkout -- this
// build's own repository/commit via s.Checkouts when configured,
// or the fixed s.RepoRoot otherwise
// (see server.go's own doc comment on both fields) -- and renders every
// script-kind step in order.
func (s *Server) handleRenderObject(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	objectID, err := s.objectInBuild(r, build.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	obj, err := s.Queries.GetDeployedObject(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if obj.Kind == "network" || obj.AsName == nil {
		writeError(w, http.StatusBadRequest, errors.New("a network has no steps to render"))
		return
	}
	team, err := s.Queries.GetTeam(r.Context(), obj.TeamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	repoRoot := s.RepoRoot
	if s.Checkouts != nil {
		dir, err := s.Checkouts.ForBuild(r.Context(), build.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		repoRoot = dir
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(c.Errors) > 0 {
		writeError(w, http.StatusInternalServerError, errors.New("content is not currently valid -- cannot render"))
		return
	}
	ctx, err := render.Resolve(c, build.EnvironmentName, *obj.AsName, int(team.TeamNumber))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	steps := make([]renderedStep, 0, len(ctx.Steps))
	for i, step := range ctx.Steps {
		rs := renderedStep{Index: i, Action: step.ActionKey(), Raw: step}
		if rs.Action == "script" {
			scriptName, _ := step["script"].(string)
			rs.ScriptName = scriptName
			if script := findScriptByName(c, scriptName); script != nil {
				rendered, err := render.RenderScript(repoRoot, script, ctx, c)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err)
					return
				}
				rs.Rendered = rendered
			}
		}
		steps = append(steps, rs)
	}
	writeJSON(w, http.StatusOK, steps)
}

func findScriptByName(c *loader.Content, name string) *loader.Script {
	for i := range c.Scripts {
		if c.Scripts[i].Name == name {
			return &c.Scripts[i]
		}
	}
	return nil
}
