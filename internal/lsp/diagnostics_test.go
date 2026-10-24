package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func newTestWorkspace(t *testing.T) *Workspace {
	t.Helper()
	w, err := NewWorkspace("../../examples/lm-test")
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

// TestDiagnosticsSchemaErrorHasRealLineAndColumn is the precise case:
// schema.FieldError carries real position info (via internal/schema's
// own yaml.Node-pointer walk), so a type mismatch must show up at the
// actual line/column of the offending value, not just "somewhere in this
// file."
func TestDiagnosticsSchemaErrorHasRealLineAndColumn(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "hosts/webserver.yaml"
	orig, err := os.ReadFile(filepath.Join(w.RepoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	// disk expects an integer -- a string is a real schema type error.
	broken := strings.Replace(string(orig), "disk: 40", `disk: "forty"`, 1)
	if broken == string(orig) {
		t.Fatal("test fixture assumption wrong: \"disk: 40\" not found in hosts/webserver.yaml")
	}
	if err := w.DidOpen(rel, broken); err != nil {
		t.Fatal(err)
	}

	diags := Diagnostics(w)
	u := uri.File(filepath.Join(w.RepoRoot, rel))
	found := diags[u]
	if len(found) == 0 {
		t.Fatalf("no diagnostics for %s, want a schema type error", rel)
	}
	d := found[0]
	if d.Range.Start.Line == 0 && d.Range.Start.Character == 0 {
		t.Fatalf("diagnostic %+v has no real position -- schema errors should carry a real line/column", d)
	}
	if d.Severity != protocol.DiagnosticSeverityError {
		t.Fatalf("severity = %v, want Error", d.Severity)
	}
}

// TestDiagnosticsCrossFileErrorIsFileLevel is the honest degradation:
// checks.go's own cross-file errors (depends_on, collisions, extends)
// never carry a line number, so these
// diagnostics land at the start of the file rather than claiming a
// precision that doesn't exist.
// TestDiagnosticsCrossFileErrorResolvesToRealDependsOnLine used to assert
// the opposite of what it does now: checks.go itself still carries no
// YAML position for a cross-file error, but Diagnostics now recovers a
// real one with a best-effort text search (see
// TestDiagnosticsCrossFileErrorResolvesRealLine's own doc comment for
// why) -- a distinctive depends_on target should resolve to the exact
// line naming it, not sit at (0,0).
func TestDiagnosticsCrossFileErrorResolvesToRealDependsOnLine(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "hosts/webserver.yaml"
	orig, err := os.ReadFile(filepath.Join(w.RepoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(orig), "depends_on: [database]", "depends_on: [does-not-exist-anywhere]", 1)
	if broken == string(orig) {
		t.Fatal("test fixture assumption wrong: \"depends_on: [database]\" not found")
	}
	brokenLine := 0
	for i, line := range strings.Split(broken, "\n") {
		if strings.Contains(line, "does-not-exist-anywhere") {
			brokenLine = i + 1
			break
		}
	}
	if brokenLine == 0 {
		t.Fatal("test fixture assumption wrong: couldn't find the broken line in the fixture itself")
	}
	if err := w.DidOpen(rel, broken); err != nil {
		t.Fatal(err)
	}

	diags := Diagnostics(w)
	u := uri.File(filepath.Join(w.RepoRoot, rel))
	found := diags[u]
	var d *protocol.Diagnostic
	for i := range found {
		if strings.Contains(string(found[i].Message.(protocol.String)), "does-not-exist-anywhere") {
			d = &found[i]
		}
	}
	if d == nil {
		t.Fatalf("no diagnostic mentioning the broken depends_on target; got %+v", found)
	}
	if int(d.Range.Start.Line) != brokenLine-1 {
		t.Fatalf("diagnostic landed on line %d, want line %d (the real depends_on line, 0-indexed)", d.Range.Start.Line, brokenLine-1)
	}
}

// TestDiagnosticsScriptRenderErrorMapsToRealScriptLine is the render-
// error case that DOES resolve to a precise location: a script's own
// source file, at the real line Go's template engine reports -- recovered
// from the template error's own text (render.RenderString names every
// template after its real file), not guessed.
func TestDiagnosticsScriptRenderErrorMapsToRealScriptLine(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "scripts/base.sh"
	orig, err := os.ReadFile(filepath.Join(w.RepoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	broken := string(orig) + "echo {{ .totally_bogus_key }}\n"
	if err := w.DidOpen(rel, broken); err != nil {
		t.Fatal(err)
	}

	diags := Diagnostics(w)
	u := uri.File(filepath.Join(w.RepoRoot, rel))
	found := diags[u]
	if len(found) == 0 {
		t.Fatalf("no diagnostics for %s, want a real template render error", rel)
	}
	wantLine := uint32(strings.Count(string(orig), "\n")) // the appended line, 0-indexed
	var sawRightLine bool
	for _, d := range found {
		if d.Range.Start.Line == wantLine {
			sawRightLine = true
		}
	}
	if !sawRightLine {
		t.Fatalf("diagnostics %+v don't include one at line %d (0-indexed) -- the appended broken line", found, wantLine)
	}
}

// TestDiagnosticsInlineStepErrorFallsBackToEnvironmentFile covers the
// other real render-error path: an inline step (no real "template name"
// to recover a file from -- see RenderStepFields' own doc comment)
// degrades to the environment file, at least keeping the failure
// visible rather than silently dropped.
// TestDiagnosticsCrossFileErrorResolvesRealLine is the real fix for a
// real report: a step referencing a script that doesn't exist showed no
// visible squiggle anywhere near the actual typo -- content.Errors
// carries no position for this whole class of check
// (internal/loader/checks.go decodes into plain Go structs, which carry
// no YAML line of their own by the time the cross-file pass runs), so
// every one of these errors used to land on line 0 (the file's own
// header line) regardless of how deep the real problem was. This proves
// the best-effort text-search recovery actually lands on the real
// broken line, not just "somewhere in the file" -- the exact live
// scenario that surfaced the gap: a typo'd script name seven lines into
// a real container file.
func TestDiagnosticsCrossFileErrorResolvesRealLine(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "hosts/scoreboard.yaml"
	orig, err := os.ReadFile(filepath.Join(w.RepoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(orig), "scoreboard-seed", "scoreboard-sed", 1)
	if broken == string(orig) {
		t.Fatal("test fixture assumption wrong: \"scoreboard-seed\" not found in hosts/scoreboard.yaml")
	}
	brokenLine := 0
	for i, line := range strings.Split(broken, "\n") {
		if strings.Contains(line, "scoreboard-sed") {
			brokenLine = i + 1
			break
		}
	}
	if brokenLine == 0 {
		t.Fatal("test fixture assumption wrong: couldn't find the broken line in the fixture itself")
	}
	if brokenLine == 1 {
		t.Fatal("test fixture assumption wrong: the broken line needs to be past line 1 to actually prove this fix (line 1 is what the old, unfixed behavior always pointed at)")
	}
	if err := w.DidOpen(rel, broken); err != nil {
		t.Fatal(err)
	}

	diags := Diagnostics(w)
	u := uri.File(filepath.Join(w.RepoRoot, rel))
	found := diags[u]
	var d *protocol.Diagnostic
	for i := range found {
		if strings.Contains(string(found[i].Message.(protocol.String)), "scoreboard-sed") {
			d = &found[i]
		}
	}
	if d == nil {
		t.Fatalf("no diagnostic mentioning the broken script name; got %+v", found)
	}
	if int(d.Range.Start.Line) != brokenLine-1 {
		t.Fatalf("diagnostic landed on line %d, want line %d (the real broken line, 0-indexed) -- not line 0", d.Range.Start.Line, brokenLine-1)
	}
}

func TestDiagnosticsInlineStepErrorFallsBackToEnvironmentFile(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "hosts/webserver.yaml"
	orig, err := os.ReadFile(filepath.Join(w.RepoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	// Inserted right after the last real steps: entry, not appended to
	// the file's end -- webserver.yaml's own last line is now inside its
	// schedule: block (a sibling field, not part of steps:), so blindly
	// appending here would land the new line inside schedule: instead of
	// steps:, breaking this test's own premise.
	const anchor = "    - script: vuln-sqli\n"
	if !strings.Contains(string(orig), anchor) {
		t.Fatal("test fixture assumption wrong: \"- script: vuln-sqli\" not found in hosts/webserver.yaml")
	}
	broken := strings.Replace(string(orig), anchor, anchor+"    - run: \"echo {{ .totally_not_a_field }}\"\n", 1)
	if err := w.DidOpen(rel, broken); err != nil {
		t.Fatal(err)
	}

	diags := Diagnostics(w)
	u := uri.File(filepath.Join(w.RepoRoot, "lm-test.yaml"))
	found := diags[u]
	var sawIt bool
	for _, d := range found {
		if strings.Contains(string(d.Message.(protocol.String)), "totally_not_a_field") {
			sawIt = true
		}
	}
	if !sawIt {
		t.Fatalf("no diagnostic on the environment file mentioning the broken inline step; got %+v", found)
	}
}
