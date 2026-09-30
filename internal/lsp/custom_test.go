package lsp

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// TestLaForgeTypeReferenceListsRealHostFields is the sidebar's own real
// data source: "show all of the configuration options that are
// available" for whatever type the cursor's document actually is.
func TestLaForgeTypeReferenceListsRealHostFields(t *testing.T) {
	text := "host:\n  name: webserver\n  os: ubuntu22\n"
	ref, err := LaForgeTypeReference("hosts/webserver.yaml", text, protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("LaForgeTypeReference: %v", err)
	}
	if ref.Kind != "host" {
		t.Fatalf("Kind = %q, want host", ref.Kind)
	}
	names := make(map[string]bool, len(ref.Fields))
	for _, f := range ref.Fields {
		names[f.Name] = true
	}
	for _, want := range []string{"name", "os", "size", "disk", "ports", "steps", "depends_on", "vars", "tags", "findings", "people", "extends"} {
		if !names[want] {
			t.Errorf("type reference for host missing field %q; got %+v", want, ref.Fields)
		}
	}
}

// TestLaForgeTypeReferenceUsesDocumentUnderCursor is the sidebar's own
// half of the multi-document fix -- the cursor being in the SECOND
// document of a `---`-separated file must resolve that document's own
// type, not the file's first one (see documentAt's own doc comment).
func TestLaForgeTypeReferenceUsesDocumentUnderCursor(t *testing.T) {
	text := "host:\n  name: web01\n---\nnetwork:\n  name: extra\n"
	ref, err := LaForgeTypeReference("mixed.yaml", text, protocol.Position{Line: 4, Character: 2})
	if err != nil {
		t.Fatalf("LaForgeTypeReference: %v", err)
	}
	if ref.Kind != "network" {
		t.Fatalf("Kind = %q, want network (the document actually under the cursor)", ref.Kind)
	}
}

func TestLaForgeTypeReferenceRejectsNonYAMLFile(t *testing.T) {
	if _, err := LaForgeTypeReference("scripts/base.sh", "#!/bin/bash\n", protocol.Position{}); err == nil {
		t.Fatal("expected an error for a non-YAML file, got nil")
	}
}

func TestLaForgeContextMatchesResolvedContent(t *testing.T) {
	w := newTestWorkspace(t)
	v, err := LaForgeContext(w, "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("LaForgeContext: %v", err)
	}
	if v.Environment != "lm-test" || v.Team != 1 || v.Host.As != "web01" {
		t.Fatalf("view = %+v, want environment=lm-test team=1 host.as=web01", v)
	}
}

func TestLaForgeRenderPreviewMatchesRealRender(t *testing.T) {
	w := newTestWorkspace(t)
	out, err := LaForgeRenderPreview(w, "scripts/base.sh", "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("LaForgeRenderPreview: %v", err)
	}
	if !strings.Contains(out, "web01") || !strings.Contains(out, "team 1") {
		t.Fatalf("rendered output = %q, want it to mention web01 and team 1", out)
	}
}

func TestLaForgeRenderPreviewReflectsLiveEdit(t *testing.T) {
	w := newTestWorkspace(t)
	const rel = "scripts/base.sh"
	if err := w.DidOpen(rel, "#!/usr/bin/env bash\necho \"edited live: {{ .host.hostname }}\"\n"); err != nil {
		t.Fatal(err)
	}
	out, err := LaForgeRenderPreview(w, rel, "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("LaForgeRenderPreview: %v", err)
	}
	if !strings.Contains(out, "edited live: web01") {
		t.Fatalf("rendered output = %q, want it to reflect the unsaved buffer, not the file on disk", out)
	}
}

func TestLaForgePickerOptionsListsRealTopology(t *testing.T) {
	w := newTestWorkspace(t)
	opts := LaForgePickerOptions(w)
	if len(opts.Environments) == 0 {
		t.Fatal("no environments in picker options")
	}
	var lmTest *EnvironmentPicker
	for i := range opts.Environments {
		if opts.Environments[i].Name == "lm-test" {
			lmTest = &opts.Environments[i]
		}
	}
	if lmTest == nil {
		t.Fatalf("lm-test not found in picker options; got %+v", opts.Environments)
	}
	if lmTest.Teams != 5 {
		t.Fatalf("lm-test.Teams = %d, want 5", lmTest.Teams)
	}
	var sawWeb01 bool
	for _, h := range lmTest.Hosts {
		if h == "web01" {
			sawWeb01 = true
		}
	}
	if !sawWeb01 {
		t.Fatalf("lm-test.Hosts = %v, want to include web01", lmTest.Hosts)
	}
}
