package lsp

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func hoverText(t *testing.T, h *protocol.Hover) string {
	t.Helper()
	if h == nil {
		t.Fatal("Hover returned nil, want a real result")
	}
	mc, ok := h.Contents.(*protocol.MarkupContent)
	if !ok {
		t.Fatalf("Contents = %T, want *protocol.MarkupContent", h.Contents)
	}
	return mc.Value
}

// TestHoverOnVarShowsValueAndSourceFile is the plan's own Hover row
// made real: "What a var holds and which file it came from in the
// environment/network/host cascade" -- against examples/lm-test's real
// "company" var (environment-level, set in lm-test.yaml).
func TestHoverOnVarShowsValueAndSourceFile(t *testing.T) {
	w := newTestWorkspace(t)
	line := "echo {{ .vars.company }}"
	col := uint32(strings.Index(line, "company") + 3) // land inside the word
	h := Hover(w, "scripts/base.sh", line+"\n", protocol.Position{Line: 0, Character: col})
	text := hoverText(t, h)
	if !strings.Contains(text, "Allports") {
		t.Errorf("hover text = %q, want it to contain the real value \"Allports\"", text)
	}
	if !strings.Contains(text, "environment") {
		t.Errorf("hover text = %q, want it to name the \"environment\" cascade level", text)
	}
	if !strings.Contains(text, "lm-test.yaml") {
		t.Errorf("hover text = %q, want it to name the real source file lm-test.yaml", text)
	}
}

// TestHoverOnPartiallyAvailableVarMentionsIt is the same real-content
// case completion's partial-availability test uses, checked on hover
// instead.
func TestHoverOnPartiallyAvailableVarMentionsIt(t *testing.T) {
	w := newTestWorkspace(t)
	line := "echo {{ .vars.db_admin_pw }}"
	col := uint32(strings.Index(line, "db_admin_pw") + 3)
	h := Hover(w, "scripts/base.sh", line+"\n", protocol.Position{Line: 0, Character: col})
	text := hoverText(t, h)
	if !strings.Contains(text, "Available on") {
		t.Errorf("hover text = %q, want it to mention partial availability (set only on the database host)", text)
	}
	if !strings.Contains(text, "database.yaml") {
		t.Errorf("hover text = %q, want it to name database.yaml as the source", text)
	}
}

// TestHoverOnConfigFieldShowsSchemaInfo covers the other real hover
// source: a config file's own field, from the schema.
func TestHoverOnConfigFieldShowsSchemaInfo(t *testing.T) {
	w := newTestWorkspace(t)
	text := "host:\n  name: webserver\n  disk: 40\n"
	h := Hover(w, "hosts/webserver.yaml", text, protocol.Position{Line: 2, Character: 4})
	got := hoverText(t, h)
	if !strings.Contains(got, "integer") {
		t.Errorf("hover text = %q, want it to mention disk's real type \"integer\"", got)
	}
}

// TestHoverInSecondDocumentOfMultiDocFile is Hover's own half of the
// multi-document fix (see completion_test.go's
// TestConfigCompletionSecondDocumentInMultiDocFile) -- hovering
// "cidr" in a network document that comes AFTER a host document in the
// same file must resolve against network's own schema, not host's.
func TestHoverInSecondDocumentOfMultiDocFile(t *testing.T) {
	w := newTestWorkspace(t)
	text := "host:\n  name: web01\n  os: ubuntu22\n---\nnetwork:\n  name: extra\n  cidr: 10.9.0.0/24\n"
	h := Hover(w, "networks/extra.yaml", text, protocol.Position{Line: 6, Character: 4})
	got := hoverText(t, h)
	if !strings.Contains(got, "cidr") {
		t.Errorf("hover text = %q, want it to name the real field \"cidr\"", got)
	}
}

func TestHoverOutsideAnyKnownTokenReturnsNil(t *testing.T) {
	w := newTestWorkspace(t)
	h := Hover(w, "scripts/base.sh", "echo hello\n", protocol.Position{Line: 0, Character: 3})
	if h != nil {
		t.Errorf("Hover on plain shell text = %+v, want nil", h)
	}
}
