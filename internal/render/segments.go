package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/globalcptc/laforge/internal/loader"
)

// Segment is one run of rendered output, flagged with whether it came from a
// template substitution. Concatenating every Segment.Text reproduces the plain
// rendered output exactly; Templated marks the spans the UI colorizes so an
// operator can see at a glance what the templates replaced.
type Segment struct {
	Text      string `json:"text"`
	Templated bool   `json:"templated"`
}

// Private-use code points used to bracket substituted output during the marked
// render. They never occur in real scripts, so splitting on them is unambiguous.
const (
	markStart = ""
	markEnd   = ""
)

// actionRE matches a single template action `{{ … }}` (with optional `-`
// trim markers), shortest-match and across newlines. Good enough for LaForge's
// scripts; the one case it can mis-split -- a literal `}}` inside a quoted
// string in an action -- just makes the marked template fail to parse, and the
// caller falls back to the plain render.
var actionRE = regexp.MustCompile(`(?s)\{\{-?.*?-?\}\}`)

// controlActions are the template keywords whose actions produce no output of
// their own (they drive structure), so they must not be wrapped.
var controlActions = map[string]bool{
	"if": true, "else": true, "end": true, "range": true, "with": true,
	"define": true, "template": true, "block": true, "break": true, "continue": true,
}

// markValueActions wraps every VALUE-producing action in text with sentinel
// actions, so after execution each substituted value is bracketed by
// markStart/markEnd. Control actions and comments are left untouched. The
// original action (including any `-` trim markers) is preserved between the
// sentinels, so trimming behaves as before.
func markValueActions(text string) string {
	return actionRE.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "{{"), "}}")
		inner = strings.TrimSpace(strings.Trim(strings.TrimSpace(inner), "-"))
		if inner == "" || strings.HasPrefix(inner, "/*") {
			return m // empty or a comment
		}
		first := inner
		if i := strings.IndexAny(inner, " \t\n"); i >= 0 {
			first = inner[:i]
		}
		if controlActions[first] {
			return m
		}
		return `{{"` + markStart + `"}}` + m + `{{"` + markEnd + `"}}`
	})
}

// splitSegments turns a marked render back into plain/templated runs.
func splitSegments(s string) []Segment {
	var segs []Segment
	for len(s) > 0 {
		start := strings.Index(s, markStart)
		if start < 0 {
			segs = append(segs, Segment{Text: s})
			break
		}
		if start > 0 {
			segs = append(segs, Segment{Text: s[:start]})
		}
		rest := s[start+len(markStart):]
		end := strings.Index(rest, markEnd)
		if end < 0 {
			segs = append(segs, Segment{Text: rest}) // malformed -- keep as plain
			break
		}
		if end > 0 {
			segs = append(segs, Segment{Text: rest[:end], Templated: true})
		}
		s = rest[end+len(markEnd):]
	}
	return segs
}

// RenderStringSegments renders text like RenderString but returns the output
// split into plain/templated Segments (the substituted spans flagged). Used
// only for the Render tab's display; delivery/validation use the plain render.
func RenderStringSegments(name, text string, ctx *Context, content *loader.Content) ([]Segment, error) {
	marked, err := RenderString(name, markValueActions(text), ctx, content)
	if err != nil {
		return nil, err
	}
	return splitSegments(marked), nil
}

// RenderScriptSegments is RenderScript's segment-returning twin -- same file,
// same per-script people fold -- for the Render tab's colorized view.
func RenderScriptSegments(repoRoot string, script *loader.Script, ctx *Context, content *loader.Content) ([]Segment, error) {
	path := filepath.Join(repoRoot, filepath.Dir(script.SourceFile), script.Source)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rctx := ctx
	if len(script.People) > 0 {
		cp := *ctx
		cp.ObjectPeople = mergePeopleRows(ctx.ObjectPeople, resolveObjectPeople(content, script.People))
		rctx = &cp
	}
	return RenderStringSegments(path, string(data), rctx, content)
}
