package lsp

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// TestConfigCompletionGenerateObject: inside set_password's nested
// `generate: { }` the generator's own keys are offered, and `type:` offers
// the three generator names.
func TestConfigCompletionGenerateObject(t *testing.T) {
	line := "    - set_password: { name: root, generate: { "
	text := "host:\n  name: w\n  steps:\n" + line + "\n"
	got := labels(configCompletion(text, protocol.Position{Line: 3, Character: uint32(len(line))}))
	for _, want := range []string{"type", "size", "label"} {
		if _, ok := got[want]; !ok {
			t.Errorf("generate: { } completion missing %q; got %v", want, got)
		}
	}

	line = "    - set_password: { name: root, generate: { size: 3, type: pa"
	text = "host:\n  name: w\n  steps:\n" + line + "\n"
	got = labels(configCompletion(text, protocol.Position{Line: 3, Character: uint32(len(line))}))
	if _, ok := got["passphrase"]; !ok || len(got) != 1 {
		t.Errorf("type: pa completion = %v, want just passphrase", got)
	}

	line = "    - create_user: { name: ops, generate: { type: "
	text = "host:\n  name: w\n  steps:\n" + line + "\n"
	got = labels(configCompletion(text, protocol.Position{Line: 3, Character: uint32(len(line))}))
	for _, want := range []string{"passphrase", "wordchain", "random"} {
		if _, ok := got[want]; !ok {
			t.Errorf("type: completion missing %q; got %v", want, got)
		}
	}
}

// TestGeneratorCompletionInScriptsAndYAML: the generator functions complete
// inside `{{ }}` in a script and in a templated YAML value, but not outside one.
func TestGeneratorCompletionInScriptsAndYAML(t *testing.T) {
	w := newTestWorkspace(t)
	script := "#!/usr/bin/env bash\necho {{ pa\n"
	got := labels(Completion(w, "scripts/base.sh", script, protocol.Position{Line: 1, Character: uint32(len("echo {{ pa"))}))
	if _, ok := got["passphrase 3"]; !ok {
		t.Errorf("script completion missing passphrase; got %v", got)
	}
	line := `    - set_password: { name: root, password: "{{ `
	yaml := "host:\n  name: w\n  steps:\n" + line + "\n"
	got = labels(Completion(w, "hosts/w.yaml", yaml, protocol.Position{Line: 3, Character: uint32(len(line))}))
	for _, want := range []string{"passphrase 3", "wordchain 5", "randompassword 16"} {
		if _, ok := got[want]; !ok {
			t.Errorf("YAML {{ }} completion missing %q; got %v", want, got)
		}
	}
}

// TestHoverDocumentsGenerators: hovering a generator name inside {{ }} shows
// its doc; hovering the braces before it doesn't.
func TestHoverDocumentsGenerators(t *testing.T) {
	w := newTestWorkspace(t)
	line := `    - set_password: { name: root, password: "{{ wordchain 5 }}" }`
	yaml := "host:\n  name: w\n  steps:\n" + line + "\n"
	at := strings.Index(line, "wordchain") + 3
	h := Hover(w, "hosts/w.yaml", yaml, protocol.Position{Line: 3, Character: uint32(at)})
	if h == nil || !strings.Contains(h.Contents.(*protocol.MarkupContent).Value, "fork-Brick-Spoon-Snail-test") {
		t.Fatalf("hover on wordchain = %+v, want its doc", h)
	}
	if _, ok := templateFuncAt(line, uint32(strings.Index(line, "{{"))); ok {
		t.Error("hover on the {{ itself resolved a generator; want only the name")
	}
}
