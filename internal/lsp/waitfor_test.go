package lsp

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/globalcptc/laforge/internal/loader"
)

func waitForContent() *loader.Content {
	return &loader.Content{
		Hosts: []loader.Host{
			{Name: "bff-dc-01", Steps: []loader.Step{
				{"script": "winbase"},
				{"script": "win-domain-populate", "milestone": "domain-ready"},
				{"script": "win-sysmon", "milestone": "logging-ready"},
			}},
			{Name: "bff-ws-2019"},
		},
		Containers: []loader.Container{{Name: "scoreboard"}},
	}
}

func completeAt(t *testing.T, text string, line, char uint32) ([]protocol.CompletionItem, bool) {
	t.Helper()
	return waitForValueCompletion(waitForContent(), text, protocol.Position{Line: line, Character: char})
}

func TestWaitForCompletesHostNames(t *testing.T) {
	text := "host:\n  name: ws\n  steps:\n    - script: join\n      wait_for:\n        host: bff\n"
	items, handled := completeAt(t, text, 5, uint32(len("        host: bff")))
	if !handled {
		t.Fatal("cursor in wait_for host value not handled")
	}
	got := labels(items)
	if _, ok := got["bff-dc-01"]; !ok || len(got) != 2 {
		t.Fatalf("host completion = %v, want the two bff hosts", got)
	}
}

func TestWaitForCompletesMilestonesOfTheNamedHost(t *testing.T) {
	text := "host:\n  name: ws\n  steps:\n    - script: join\n      wait_for:\n        host: bff-dc-01\n        milestone: \n"
	items, handled := completeAt(t, text, 6, uint32(len("        milestone: ")))
	if !handled {
		t.Fatal("milestone value not handled")
	}
	got := labels(items)
	if _, ok := got["domain-ready"]; !ok || len(got) != 2 {
		t.Fatalf("milestone completion = %v, want domain-ready + logging-ready", got)
	}
}

func TestWaitForCompletesInFlowAndListStyle(t *testing.T) {
	flow := "host:\n  steps:\n    - script: join\n      wait_for: { host: bff-dc-01, step: win-d"
	items, handled := completeAt(t, flow, 3, uint32(len("      wait_for: { host: bff-dc-01, step: win-d")))
	if got := labels(items); !handled || len(got) != 1 {
		t.Fatalf("flow step completion = %v (handled %v), want win-domain-populate", got, handled)
	}
	list := "host:\n  steps:\n    - script: join\n      wait_for:\n        - host: bff-dc-01\n          milestone: \n"
	items, handled = completeAt(t, list, 5, uint32(len("          milestone: ")))
	if got := labels(items); !handled || len(got) != 2 {
		t.Fatalf("list-entry milestone completion = %v (handled %v)", got, handled)
	}
}

func TestWaitForCompletionIgnoresOtherHostKeys(t *testing.T) {
	// The document's own `host:` header is not a wait_for value.
	if _, handled := completeAt(t, "host:\n  name: x\n", 0, 5); handled {
		t.Fatal("the host: header line was treated as a wait_for value")
	}
}
