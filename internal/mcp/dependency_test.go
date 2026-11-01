package mcp

import (
	"os/exec"
	"strings"
	"testing"
)

// The MCP server must have no way to change anything: it may not import
// anything that talks to a hypervisor, the agents, GitHub, or the API's
// handlers. If this fails, move what you need behind an interface the API
// implements (like Access) instead of importing it.
func TestMCPImportsNothingThatCanWrite(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	forbidden := []string{
		"github.com/globalcptc/laforge/internal/api",
		"github.com/globalcptc/laforge/internal/builder",
		"github.com/globalcptc/laforge/internal/runner",
		"github.com/globalcptc/laforge/internal/orchestrator",
		"github.com/globalcptc/laforge/internal/gateway",
		"github.com/globalcptc/laforge/internal/ghclient",
		"github.com/globalcptc/laforge/internal/checkout",
		"github.com/globalcptc/laforge/internal/ingest",
	}
	for _, dep := range strings.Fields(string(out)) {
		for _, f := range forbidden {
			if dep == f || strings.HasPrefix(dep, f+"/") {
				t.Errorf("internal/mcp depends on %s", dep)
			}
		}
	}
}
