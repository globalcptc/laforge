package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorkspaceLoadsRealRepoCleanly proves the scratch mirror is a real,
// working copy of examples/lm-test: NewWorkspace's first load must be
// exactly as clean as `laforge check` reports for the same real
// content, matching what internal/orchestrator and internal/api's own
// tests already prove of examples/lm-test elsewhere in this repo.
func TestWorkspaceLoadsRealRepoCleanly(t *testing.T) {
	w, err := NewWorkspace("../../examples/lm-test")
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	content, renderErrs := w.Snapshot()
	if content == nil {
		t.Fatal("Snapshot() content is nil after NewWorkspace")
	}
	if len(content.Errors) != 0 {
		t.Fatalf("content.Errors = %+v, want none (examples/lm-test is real, clean content)", content.Errors)
	}
	if len(renderErrs) != 0 {
		t.Fatalf("renderErrs = %+v, want none", renderErrs)
	}
	if len(content.Hosts) == 0 || len(content.Environments) == 0 {
		t.Fatalf("content looks empty (hosts=%d, environments=%d) -- the mirror likely didn't copy real files", len(content.Hosts), len(content.Environments))
	}
}

// TestWorkspaceDidChangeReflectsUnsavedBufferNotDisk is the core "live
// as you type" property: breaking a host's YAML through DidChange must
// produce a real validation error on the NEXT Snapshot, without ever
// touching the real file on disk -- and DidClose must revert to the
// real (still-valid) on-disk content, proving the overlay is genuinely
// buffer-scoped, not a mutation of the workspace's source repo.
func TestWorkspaceDidChangeReflectsUnsavedBufferNotDisk(t *testing.T) {
	w, err := NewWorkspace("../../examples/lm-test")
	if err != nil {
		t.Fatalf("NewWorkspace: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	const rel = "hosts/webserver.yaml"
	realPath := filepath.Join(w.RepoRoot, rel)
	realBefore, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("reading the real file: %v", err)
	}

	// Break it: reference a var that doesn't exist, inside an inline
	// run step -- a real schema-valid-but-broken edit an author could
	// actually make while typing.
	broken := string(realBefore) + "\n  - run: \"echo {{ .totally_not_a_field }}\"\n"
	if err := w.DidOpen(rel, string(realBefore)); err != nil {
		t.Fatalf("DidOpen: %v", err)
	}
	if err := w.DidChange(rel, broken); err != nil {
		t.Fatalf("DidChange: %v", err)
	}

	content, renderErrs := w.Snapshot()
	if content == nil {
		t.Fatal("Snapshot() content is nil after DidChange")
	}
	if len(content.Errors) == 0 && len(renderErrs) == 0 {
		t.Fatal("broken content produced no errors at all -- the overlay isn't reaching loader.Load/render.CheckAll")
	}

	// The real file on disk must be completely untouched.
	realAfter, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("reading the real file after DidChange: %v", err)
	}
	if string(realAfter) != string(realBefore) {
		t.Fatal("the real on-disk file changed -- DidChange must only ever touch the scratch mirror")
	}

	// Closing the buffer reverts to the real (valid) content.
	if err := w.DidClose(rel); err != nil {
		t.Fatalf("DidClose: %v", err)
	}
	contentAfterClose, _ := w.Snapshot()
	if contentAfterClose == nil {
		t.Fatal("Snapshot() content is nil after DidClose")
	}
	if len(contentAfterClose.Errors) != 0 {
		t.Fatalf("after DidClose, content.Errors = %+v, want none (reverted to the real, valid file)", contentAfterClose.Errors)
	}
}
