// Package lsp is the language server: "the logic lives once in
// Go beside the validator, and VS Code, Neovim, and anything else speaking
// LSP get it for free". Every capability here --
// diagnostics, completion, hover, go to definition -- is a plain Go
// function over a *Workspace, independent of the wire protocol; server.go
// is the only file that knows about go.lsp.dev/protocol types, so the
// actual logic is unit-testable without a JSON-RPC connection at all.
package lsp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// Workspace is one open content repository. It mirrors RepoRoot into a
// scratch directory once, then keeps that mirror in sync with the
// editor's own open buffers (which may be unsaved, i.e. different from
// what's on disk) as they change -- loader.Load and render.CheckAll then
// run against the scratch mirror completely unmodified, the same code
// path `laforge check` uses, so validation is real rather than a
// reimplementation that could drift. "Live as you type"
// means the mirror reflects the
// live buffer for whatever's open, and the real on-disk file for
// everything else.
type Workspace struct {
	RepoRoot string
	scratch  string

	mu      sync.Mutex
	content *loader.Content
	errs    []render.RenderError
}

// NewWorkspace copies repoRoot into a fresh scratch directory (skipping
// .git -- irrelevant to content validation and often large) and runs the
// first real load.
func NewWorkspace(repoRoot string) (*Workspace, error) {
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	scratch, err := os.MkdirTemp("", "laforge-lsp-*")
	if err != nil {
		return nil, err
	}
	w := &Workspace{RepoRoot: repoRoot, scratch: scratch}
	if err := copyTree(repoRoot, scratch); err != nil {
		os.RemoveAll(scratch)
		return nil, fmt.Errorf("mirroring %s: %w", repoRoot, err)
	}
	w.reload()
	return w, nil
}

// Close removes the scratch mirror. Safe to call once, at server
// shutdown.
func (w *Workspace) Close() error {
	return os.RemoveAll(w.scratch)
}

// RelPath turns a real filesystem path (as decoded from a
// go.lsp.dev/uri.URI by the caller) into the path relative to RepoRoot
// that every loader.Host/Environment/etc.'s own SourceFile uses.
func (w *Workspace) RelPath(fsPath string) (string, error) {
	return filepath.Rel(w.RepoRoot, fsPath)
}

// DidOpen/DidChange overlay text onto the scratch mirror at rel (the
// live, possibly-unsaved buffer content) and reload. DidClose reverts
// that one file back to whatever's actually on disk -- an editor closing
// a buffer with unsaved changes means "go back to showing what's really
// there," not "keep validating against the discarded edit."
func (w *Workspace) DidOpen(rel, text string) error   { return w.overlay(rel, text) }
func (w *Workspace) DidChange(rel, text string) error { return w.overlay(rel, text) }

func (w *Workspace) DidClose(rel string) error {
	real := filepath.Join(w.RepoRoot, rel)
	scratchPath := filepath.Join(w.scratch, rel)
	b, err := os.ReadFile(real)
	if os.IsNotExist(err) {
		// The buffer was a file that doesn't exist on disk (a new,
		// never-saved file) -- remove it from the mirror entirely.
		os.Remove(scratchPath)
		w.reload()
		return nil
	}
	if err != nil {
		return err
	}
	if err := writeFile(scratchPath, b); err != nil {
		return err
	}
	w.reload()
	return nil
}

func (w *Workspace) overlay(rel, text string) error {
	if err := writeFile(filepath.Join(w.scratch, rel), []byte(text)); err != nil {
		return err
	}
	w.reload()
	return nil
}

// reload re-runs the real validation pipeline against the scratch
// mirror's current state -- loader.Load (schema + cross-file checks),
// then render.CheckAll (every script rendered for every host/team),
// exactly like cmd/laforge's runCheck, except render errors are still
// collected even when loader errors exist (an LSP wants maximal
// diagnostics on every keystroke, not "fix the first error to see the
// second"). Recovers from a panic in either pass -- content mid-edit can
// be arbitrarily malformed, and a language server crashing on invalid
// input the CLI would just exit non-zero on is a real reliability
// problem the CLI doesn't have.
func (w *Workspace) reload() {
	var content *loader.Content
	var renderErrs []render.RenderError
	func() {
		// content is assigned before CheckAll runs, so a panic
		// recovered here still leaves content set and only renderErrs
		// empty -- one recover covers both calls without losing
		// whichever result already succeeded.
		defer func() { recover() }()
		c, err := loader.Load(w.scratch)
		if err != nil {
			return
		}
		content = c
		renderErrs = render.CheckAll(w.scratch, c)
	}()

	w.mu.Lock()
	w.content = content
	w.errs = renderErrs
	w.mu.Unlock()
}

// ReadScratchFile returns rel's current content from the live scratch
// mirror -- the server's own single source of truth for "what does this
// open document currently say," instead of the server tracking a second,
// parallel copy of every open buffer itself.
func (w *Workspace) ReadScratchFile(rel string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(w.scratch, rel))
	if err != nil {
		return "", false
	}
	return string(b), true
}

// Snapshot returns the current content and render errors together,
// consistent with each other (both from the same reload pass) -- every
// caller (diagnostics, completion, hover, definition) reads through this
// rather than the two fields directly, so a reload landing mid-request
// can't hand back a loader.Content from one pass paired with render
// errors from a different one.
func (w *Workspace) Snapshot() (*loader.Content, []render.RenderError) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.content, w.errs
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		return copyFile(path, filepath.Join(dst, rel))
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
