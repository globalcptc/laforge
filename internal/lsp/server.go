package lsp

import (
	"context"
	"encoding/json"
	"fmt"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/globalcptc/laforge/internal/updatecheck"
)

// Server is the go.lsp.dev/protocol.Server implementation -- the only
// file in this package that knows about wire-protocol types beyond the
// pieces diagnostics.go/completion.go/hover.go/definition.go/custom.go
// already return; everything it does is call one of those, then shape
// the result into whatever the LSP method's own response type is.
type Server struct {
	protocol.UnimplementedServer

	client  protocol.Client
	ws      *Workspace
	version string // the laforge-lsp build version, for the update nudge

	// lastDiagURIs is every file that had at least one diagnostic on the
	// previous publishAll pass. Diagnostics(w) only ever returns entries
	// for files that currently have errors -- a file that just became
	// clean simply isn't in that map any more -- so without this, fixing
	// the last error in a file would never tell the client to clear its
	// now-stale squiggles: PublishDiagnostics is a full replace per URI,
	// not a diff, and an editor keeps showing whatever it was last told
	// until told otherwise.
	lastDiagURIs map[uri.URI]bool
}

// NewServer builds the language server. version is the laforge-lsp binary's
// build version (stamped at release, "dev" for a local build), used only for
// the one-time "a newer release is available" nudge on Initialized.
func NewServer(version string) *Server {
	return &Server{version: version}
}

// SetClient wires the server-initiated notifications (PublishDiagnostics)
// use -- called once, right after protocol.NewServer hands back the
// Client dispatcher for this same connection.
func (s *Server) SetClient(c protocol.Client) {
	s.client = c
}

func (s *Server) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	root, err := rootFromInitialize(params)
	if err != nil {
		return nil, err
	}
	ws, err := NewWorkspace(root)
	if err != nil {
		return nil, fmt.Errorf("opening workspace at %s: %w", root, err)
	}
	s.ws = ws

	fullSync := protocol.TextDocumentSyncKindFull
	openClose := true
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: &openClose,
				Change:    &fullSync,
			},
			CompletionProvider: &protocol.CompletionOptions{TriggerCharacters: []string{".", "{", " ", "\""}},
			HoverProvider:      protocol.Boolean(true),
			DefinitionProvider: protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{Name: "laforge-lsp", Version: protocol.NewOptional(s.version)},
	}, nil
}

func (s *Server) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	s.publishAll(ctx)
	s.notifyIfOutdated()
	return nil
}

// notifyIfOutdated checks this language server's build version against the
// latest GitHub release once per session and, if a newer one exists, surfaces
// a warning in the editor (window/showMessage). It runs in the background so
// it never delays startup, and is best-effort throughout: a dev build,
// LAFORGE_NO_UPDATE_CHECK, no client, or any network error just means no
// nudge. People routinely forget to rebuild laforge-lsp after pulling, so an
// editor that silently speaks an old protocol to a moved-on schema is exactly
// what this catches.
func (s *Server) notifyIfOutdated() {
	if s.client == nil || updatecheck.Disabled() || !updatecheck.IsRelease(s.version) {
		return
	}
	go func() {
		res, err := updatecheck.Check(context.Background(), s.version)
		if err != nil || !res.Outdated {
			return
		}
		_ = s.client.ShowMessage(context.Background(), &protocol.ShowMessageParams{
			Type:    protocol.MessageTypeWarning,
			Message: res.Message(),
		})
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.ws != nil {
		return s.ws.Close()
	}
	return nil
}

func (s *Server) Exit(ctx context.Context) error { return nil }

func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil // a file outside the workspace root -- nothing to track
	}
	if err := s.ws.DidOpen(rel, params.TextDocument.Text); err != nil {
		return err
	}
	s.publishAll(ctx)
	return nil
}

func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil
	}
	text, ok := fullText(params.ContentChanges)
	if !ok {
		return nil
	}
	if err := s.ws.DidChange(rel, text); err != nil {
		return err
	}
	s.publishAll(ctx)
	return nil
}

func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	s.publishAll(ctx)
	return nil
}

func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil
	}
	if err := s.ws.DidClose(rel); err != nil {
		return err
	}
	s.publishAll(ctx)
	return nil
}

func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (protocol.CompletionResult, error) {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil, nil
	}
	text, ok := s.currentText(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	items := Completion(s.ws, rel, text, params.Position)
	if items == nil {
		return protocol.CompletionItemSlice{}, nil
	}
	return protocol.CompletionItemSlice(items), nil
}

func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil, nil
	}
	text, ok := s.currentText(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	return Hover(s.ws, rel, text, params.Position), nil
}

func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	rel, err := s.rel(params.TextDocument.URI)
	if err != nil {
		return nil, nil
	}
	text, ok := s.currentText(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	locs := Definition(s.ws, rel, text, params.Position)
	if locs == nil {
		return protocol.LocationSlice{}, nil
	}
	return protocol.LocationSlice(locs), nil
}

// Request is the escape hatch go.lsp.dev/protocol.Server's own interface
// provides for methods it doesn't otherwise generate a typed handler
// for -- exactly where "laforge/context," "laforge/renderPreview," and
// "laforge/pickerOptions" (custom.go) live, since none of them are
// standard LSP methods.
func (s *Server) Request(ctx context.Context, method string, params any) (any, error) {
	raw, _ := json.Marshal(params)
	switch method {
	case "laforge/context":
		var p struct {
			Environment string `json:"environment"`
			Host        string `json:"host"`
			Team        int    `json:"team"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		return LaForgeContext(s.ws, p.Environment, p.Host, p.Team)
	case "laforge/renderPreview":
		var p struct {
			URI         string `json:"uri"`
			Environment string `json:"environment"`
			Host        string `json:"host"`
			Team        int    `json:"team"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		rel, err := s.rel(uri.URI(p.URI))
		if err != nil {
			return nil, err
		}
		rendered, err := LaForgeRenderPreview(s.ws, rel, p.Environment, p.Host, p.Team)
		if err != nil {
			return nil, err
		}
		return map[string]string{"rendered": rendered}, nil
	case "laforge/pickerOptions":
		return LaForgePickerOptions(s.ws), nil
	case "laforge/typeReference":
		var p struct {
			URI      string            `json:"uri"`
			Position protocol.Position `json:"position"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		rel, err := s.rel(uri.URI(p.URI))
		if err != nil {
			return nil, err
		}
		text, ok := s.currentText(uri.URI(p.URI))
		if !ok {
			return nil, fmt.Errorf("document not open: %s", p.URI)
		}
		return LaForgeTypeReference(rel, text, p.Position)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (s *Server) rel(u uri.URI) (string, error) {
	if s.ws == nil {
		return "", fmt.Errorf("workspace not initialized")
	}
	return s.ws.RelPath(u.FsPath())
}

// currentText re-derives a document's current text from the workspace's
// own scratch mirror (kept live by DidOpen/DidChange) rather than
// tracking a second, parallel copy of every open buffer in the server
// itself -- one source of truth.
func (s *Server) currentText(u uri.URI) (string, bool) {
	rel, err := s.rel(u)
	if err != nil {
		return "", false
	}
	return s.ws.ReadScratchFile(rel)
}

func (s *Server) publishAll(ctx context.Context) {
	if s.client == nil || s.ws == nil {
		return
	}
	diags := Diagnostics(s.ws)
	next := make(map[uri.URI]bool, len(diags))
	for u, ds := range diags {
		next[u] = true
		s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{URI: u, Diagnostics: ds})
	}
	for u := range s.lastDiagURIs {
		if !next[u] {
			s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{URI: u, Diagnostics: []protocol.Diagnostic{}})
		}
	}
	s.lastDiagURIs = next
}

func fullText(changes []protocol.TextDocumentContentChangeEvent) (string, bool) {
	if len(changes) == 0 {
		return "", false
	}
	// Full-document sync (TextDocumentSyncKindFull, as advertised in
	// Initialize) always sends exactly one change carrying the whole new
	// text.
	if w, ok := changes[len(changes)-1].(*protocol.TextDocumentContentChangeWholeDocument); ok {
		return w.Text, true
	}
	return "", false
}

func rootFromInitialize(params *protocol.InitializeParams) (string, error) {
	if params.RootURI != nil && *params.RootURI != "" {
		return params.RootURI.FsPath(), nil
	}
	if folders, ok := params.WorkspaceFolders.Get(); ok {
		for _, f := range folders {
			return f.URI.FsPath(), nil
		}
	}
	return "", fmt.Errorf("no rootUri or workspaceFolders in initialize params")
}
