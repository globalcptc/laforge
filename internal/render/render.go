package render

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/globalcptc/laforge/internal/loader"
)

// RenderScript renders a script's real source file (the .sh/.ps1/.bat next
// to its .yaml definition) against a resolved Context. repoRoot + the
// script's SourceFile + its Source field together locate the file on disk
// -- see internal/loader/types.go's Script.
func RenderScript(repoRoot string, script *loader.Script, ctx *Context, content *loader.Content) (string, error) {
	path := filepath.Join(repoRoot, filepath.Dir(script.SourceFile), script.Source)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading script source %s: %w", path, err)
	}
	// Fold the script's own `people:` into the context so `{{ .people }}` inside
	// the script sees the host's assigned people plus any the script itself
	// pulls in (deduped by username). Most scripts have none and render against
	// the host's context unchanged.
	rctx := ctx
	if len(script.People) > 0 {
		cp := *ctx
		cp.ObjectPeople = mergePeopleRows(ctx.ObjectPeople, resolveObjectPeople(content, script.People))
		rctx = &cp
	}
	return RenderString(path, string(data), rctx, content)
}

func findScript(c *loader.Content, name string) *loader.Script {
	for i := range c.Scripts {
		if c.Scripts[i].Name == name {
			return &c.Scripts[i]
		}
	}
	return nil
}

// RenderStepFields renders every string value found anywhere in a step's
// action object that contains "{{" -- the doc's own step examples template
// field values directly (`password: "{{ vars.db_pw }}"`), not only script
// source files, so `laforge check` validates those too. file/line are for
// error messages only; Step itself carries no position information (it
// came from a typed decode, not the yaml.Node tree -- see loader.go).
func RenderStepFields(file string, step loader.Step, ctx *Context, content *loader.Content) []error {
	var errs []error
	name := fmt.Sprintf("%s (step %s)", file, step.ActionKey())
	walkStrings(step, func(s string) {
		if !containsTemplate(s) {
			return
		}
		if _, err := RenderString(name, s, ctx, content); err != nil {
			errs = append(errs, err)
		}
	})
	return errs
}

func containsTemplate(s string) bool {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '{' && s[i+1] == '{' {
			return true
		}
	}
	return false
}

// walkStrings visits every string found anywhere inside v, recursing
// through the map/slice shapes step and validator fields are actually
// built from (see internal/schema's step and validator schemas).
func walkStrings(v interface{}, visit func(string)) {
	switch x := v.(type) {
	case string:
		visit(x)
	case loader.Step:
		for _, val := range x {
			walkStrings(val, visit)
		}
	case map[string]interface{}:
		for _, val := range x {
			walkStrings(val, visit)
		}
	case []interface{}:
		for _, val := range x {
			walkStrings(val, visit)
		}
	case []string:
		for _, val := range x {
			visit(val)
		}
	}
}
