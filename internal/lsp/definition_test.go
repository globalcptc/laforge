package lsp

import (
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// TestDefinitionJumpsFromScriptStepToScriptFile is "a script name jumps
// to the script" against a real steps: reference in
// examples/lm-test/hosts/webserver.yaml ("- script: vuln-sqli").
func TestDefinitionJumpsFromScriptStepToScriptFile(t *testing.T) {
	w := newTestWorkspace(t)
	line := "  - script: vuln-sqli"
	col := uint32(strings.Index(line, "vuln-sqli") + 2)
	locs := Definition(w, "hosts/webserver.yaml", line+"\n", protocol.Position{Line: 0, Character: col})
	if len(locs) != 1 {
		t.Fatalf("locations = %+v, want exactly one", locs)
	}
	if !strings.HasSuffix(locs[0].URI.FsPath(), filepath.FromSlash("scripts/vuln-sqli.yaml")) {
		t.Fatalf("jumped to %q, want scripts/vuln-sqli.yaml", locs[0].URI.FsPath())
	}
}

// TestDefinitionJumpsFromDependsOnToHostFile is "a host name to the host
// file" against webserver's real depends_on: [database].
func TestDefinitionJumpsFromDependsOnToHostFile(t *testing.T) {
	w := newTestWorkspace(t)
	line := "depends_on: [database]"
	col := uint32(strings.Index(line, "database") + 2)
	locs := Definition(w, "hosts/webserver.yaml", line+"\n", protocol.Position{Line: 0, Character: col})
	if len(locs) != 1 {
		t.Fatalf("locations = %+v, want exactly one", locs)
	}
	if !strings.HasSuffix(locs[0].URI.FsPath(), filepath.FromSlash("hosts/database.yaml")) {
		t.Fatalf("jumped to %q, want hosts/database.yaml", locs[0].URI.FsPath())
	}
}

// TestDefinitionJumpsFromPeopleReferenceToCSV is "a people source to the
// CSV" -- a script file's real `people "employees"` reference.
func TestDefinitionJumpsFromPeopleReferenceToCSV(t *testing.T) {
	w := newTestWorkspace(t)
	line := `{{ range people "employees" }}`
	col := uint32(strings.Index(line, "employees") + 2)
	locs := Definition(w, "scripts/some-script.sh", line+"\n", protocol.Position{Line: 0, Character: col})
	if len(locs) != 1 {
		t.Fatalf("locations = %+v, want exactly one", locs)
	}
	if !strings.HasSuffix(locs[0].URI.FsPath(), filepath.FromSlash("people/employees.csv")) {
		t.Fatalf("jumped to %q, want people/employees.csv", locs[0].URI.FsPath())
	}
}

func TestDefinitionOnUnknownNameReturnsNothing(t *testing.T) {
	w := newTestWorkspace(t)
	locs := Definition(w, "hosts/webserver.yaml", "depends_on: [totally-unknown-thing]\n", protocol.Position{Line: 0, Character: 20})
	if len(locs) != 0 {
		t.Fatalf("locations = %+v, want none", locs)
	}
}
