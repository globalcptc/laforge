package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/globalcptc/laforge/internal/loader"
)

// templateErrLine picks the real source file and line back out of a Go
// template error's own formatting (`template: <name>:<line>:<col>: ...`)
// -- render.RenderString deliberately names every template after the real
// file it came from ("a failure's error message names the real source
// file and line... for free from Go's own template error formatting, not
// reimplemented here", internal/render/template.go's own doc comment),
// so this is recovering information Go's template package already
// carries, not inventing it.
var templateErrLine = regexp.MustCompile(`template: (.+?):(\d+):\d*:?`)

// Diagnostics is "Schema errors, unknown template references, unknown
// script/host/network names, collisions. The same checks as `laforge
// check`, live as you type", grouped by the real on-disk file each
// belongs to. Precision
// varies with what the underlying check actually knows: schema errors carry a real line and
// column; cross-file reference errors (depends_on, collisions, extends)
// are file-level only; render errors are file+line when the failing
// template names a real script file, file-level (attached to the
// environment) otherwise.
func Diagnostics(w *Workspace) map[uri.URI][]protocol.Diagnostic {
	out := map[uri.URI][]protocol.Diagnostic{}
	content, renderErrs := w.Snapshot()
	if content == nil {
		return out
	}

	add := func(absPath string, line, col int, message string) {
		u := uri.File(absPath)
		var start protocol.Position
		if line > 0 {
			start = protocol.Position{Line: uint32(line - 1), Character: uint32(maxInt(col-1, 0))}
		}
		end := protocol.Position{Line: start.Line, Character: start.Character + 1}
		out[u] = append(out[u], protocol.Diagnostic{
			Range:    protocol.Range{Start: start, End: end},
			Severity: protocol.DiagnosticSeverityError,
			Source:   protocol.NewOptional("laforge"),
			Message:  protocol.String(message),
		})
	}

	for _, fe := range content.Errors {
		absPath := filepath.Join(w.RepoRoot, fe.File)
		line, col := fe.Line, fe.Column
		if line == 0 {
			// Real gap found live: "step references script X, which does
			// not exist" (and every other cross-file check --
			// depends_on, collisions, extends, duplicate `as` names) is
			// file-level only, per this function's own doc comment --
			// the checks in internal/loader/checks.go run after
			// decoding into plain Go structs, which carry no YAML
			// position of their own to report. Line 0 renders as a
			// squiggle on the file's own header line, easy to miss when
			// the actual problem is deeper in -- e.g. the exact real
			// case that surfaced this: a typo'd script name seven lines
			// into hosts/scoreboard.yaml showed nothing anyone would
			// notice there. Best-effort recovery: these messages always
			// name the offending value in quotes (the loader always
			// formats it with %q), almost always as the LAST quoted
			// term -- the specific script/host/network/extends target
			// that's actually wrong, not the object reporting the
			// error. A literal text search for that value is not a real
			// YAML-position resolver, but for a real value that only
			// appears once or twice in a real file, it reliably lands
			// on the right line -- a substantial, honest improvement
			// over always pointing at line 1, even though it can't
			// promise precision the way a real schema error's line
			// number (from the YAML node tree directly) already has.
			if name := lastQuoted(fe.Message); name != "" {
				// The scratch mirror, not RepoRoot -- content.Errors was
				// computed from loader.Load(w.scratch), the live buffer
				// (unsaved edits included), and that's what needs
				// searching for a line number that's actually still
				// correct against what's on screen right now.
				if found := findLineContaining(filepath.Join(w.scratch, fe.File), name); found > 0 {
					line = found
				}
			}
		}
		add(absPath, line, col, fe.Message)
	}

	for _, re := range renderErrs {
		if m := templateErrLine.FindStringSubmatch(re.Message); m != nil {
			if rel, err := filepath.Rel(w.scratch, m[1]); err == nil {
				if line, err := strconv.Atoi(m[2]); err == nil {
					add(filepath.Join(w.RepoRoot, rel), line, 0, re.Message)
					continue
				}
			}
		}
		// Fallback: an inline step's template name is "<as> (step
		// <kind>)", not a real file (see RenderStepFields' own doc
		// comment) -- attach to the environment file instead, with no
		// precise line, naming the team and host in the message itself
		// so the information isn't lost, just not positioned.
		if envFile := environmentSourceFile(content, re.Environment); envFile != "" {
			add(filepath.Join(w.RepoRoot, envFile), 0, 0, re.String())
		}
	}

	return out
}

func environmentSourceFile(c *loader.Content, name string) string {
	for i := range c.Environments {
		if c.Environments[i].Name == name {
			return c.Environments[i].SourceFile
		}
	}
	return ""
}

var quotedRE = regexp.MustCompile(`"([^"]*)"`)

// lastQuoted returns the last double-quoted substring in message, or ""
// if there isn't one -- internal/loader/checks.go's own cross-check
// errors always format the offending value with %q, and it's
// consistently the last one when a message names more than one (e.g.
// "environment %q uses network %q, which does not exist" -- the
// environment's own name first, the actually-missing network second).
func lastQuoted(message string) string {
	matches := quotedRE.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1][1]
}

// findLineContaining is a real text search, not a YAML-aware one -- the
// same honest tradeoff this package's own templateErrLine/adjustPointerForMessage
// already make elsewhere for exactly this reason (the library/pass in
// question doesn't carry real position data, and a text search on a
// distinctive value is the pragmatic way to recover one anyway). Returns
// the first matching line, 1-indexed to match schema.FieldError.Line's
// own convention, or 0 if the file can't be read or nothing matches.
func findLineContaining(absPath, needle string) int {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return 0
	}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
