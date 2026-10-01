package lsp

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// fakeClient is the minimal peer a real LSP client would be, just
// enough to receive and record the server's own PublishDiagnostics
// notifications -- proving the server actually pushes real diagnostics
// over the real wire protocol, not just that the pure Go functions
// behind them work in isolation (already covered by every other test in
// this package).
type fakeClient struct {
	protocol.UnimplementedClient
	mu    sync.Mutex
	diags map[uri.URI][]protocol.Diagnostic
}

func (f *fakeClient) PublishDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.diags == nil {
		f.diags = map[uri.URI][]protocol.Diagnostic{}
	}
	f.diags[params.URI] = params.Diagnostics
	return nil
}

func (f *fakeClient) get(u uri.URI) []protocol.Diagnostic {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.diags[u]
}

// waitFor polls fn (guarded by fakeClient's own mutex via the passed
// closure) until it returns true or the deadline passes -- diagnostics
// arrive asynchronously over the real connection, so a test reading them
// right after a notify would race.
func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition never became true within the deadline")
}

// TestServerIntegrationRealJSONRPCRoundTrip drives the actual Server
// over a real, in-memory JSON-RPC 2.0 connection (go.lsp.dev/jsonrpc2's
// own channel stream pair, real message framing and encoding, not a
// direct Go function call) through initialize, didOpen (with content
// broken on purpose), completion, hover, definition, and the custom
// laforge/context request -- proving the wire protocol wiring itself
// works, not just the pure logic underneath it.
func TestServerIntegrationRealJSONRPCRoundTrip(t *testing.T) {
	left, right := jsonrpc2.NewChannelStreamPair(16)
	ctx := context.Background()

	srv := NewServer("dev") // "dev" skips the update-check network call in tests
	_, serverConn, client := protocol.NewServer(ctx, srv, left)
	srv.SetClient(client)
	t.Cleanup(func() { serverConn.Close() })

	fc := &fakeClient{}
	_, clientConn, dispatcher := protocol.NewClient(ctx, fc, right)
	t.Cleanup(func() { clientConn.Close() })

	root := uri.File(mustAbs(t, "../../examples/lm-test"))
	initResult, err := dispatcher.Initialize(ctx, &protocol.InitializeParams{RootURI: &root})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if initResult.Capabilities.HoverProvider == nil {
		t.Fatal("InitializeResult advertises no hover support")
	}
	if err := dispatcher.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized: %v", err)
	}

	hostURI := uri.File(mustAbs(t, "../../examples/lm-test/hosts/webserver.yaml"))
	broken := "host:\n  name: webserver\n  os: ubuntu22\n  size: small\n  disk: \"not-a-number\"\n  ports: { tcp: [\"80\"] }\n  steps:\n    - script: base\n  \n"
	if err := dispatcher.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: hostURI, LanguageID: "yaml", Version: 1, Text: broken},
	}); err != nil {
		t.Fatalf("DidOpen: %v", err)
	}

	waitFor(t, func() bool { return len(fc.get(hostURI)) > 0 })
	diags := fc.get(hostURI)
	if len(diags) == 0 {
		t.Fatal("no diagnostics published for the broken host file")
	}

	// Completion: cursor on the real trailing line at the type's own
	// nesting depth (index 8 -- the text ends "...script: base\n  \n", so
	// line 7 is "    - script: base" and line 8 is the 2-space-indented
	// blank line after it) should offer real schema fields.
	compResult, err := dispatcher.Completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: hostURI},
			Position:     protocol.Position{Line: 8, Character: 2},
		},
	})
	if err != nil {
		t.Fatalf("Completion: %v", err)
	}
	items, ok := compResult.(protocol.CompletionItemSlice)
	if !ok || len(items) == 0 {
		t.Fatalf("Completion result = %#v (type %T), want a non-empty CompletionItemSlice", compResult, compResult)
	}

	// Hover on the "script: base" step (line 7: "    - script: base")
	// should resolve the real script.
	hover, err := dispatcher.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: hostURI},
			Position:     protocol.Position{Line: 7, Character: 16}, // inside "base"
		},
	})
	_ = hover // "base" has no vars.* token to hover on the host file itself; this just proves the call round-trips without error
	if err != nil {
		t.Fatalf("Hover: %v", err)
	}

	// Definition: "base" on the script step must jump to scripts/base.yaml.
	defResult, err := dispatcher.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: hostURI},
			Position:     protocol.Position{Line: 7, Character: 16},
		},
	})
	if err != nil {
		t.Fatalf("Definition: %v", err)
	}
	locs, ok := defResult.(protocol.LocationSlice)
	if !ok || len(locs) != 1 {
		t.Fatalf("Definition result = %#v, want exactly one location", defResult)
	}

	// Revert the broken buffer -- DidClose falls back to the real,
	// valid on-disk file, which laforge/context below needs (with the
	// "disk" field still broken, `webserver` drops out of the typed
	// content entirely and every "web01" resolution fails, exactly as
	// it should for genuinely invalid content).
	if err := dispatcher.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: hostURI},
	}); err != nil {
		t.Fatalf("DidClose: %v", err)
	}
	waitFor(t, func() bool { return len(fc.get(hostURI)) == 0 })

	// The custom laforge/context request.
	var ctxView struct {
		Environment string `json:"environment"`
		Host        struct {
			As string `json:"as"`
		} `json:"host"`
	}
	if _, err := dispatcher.Request(ctx, "laforge/context", map[string]any{
		"environment": "lm-test", "host": "web01", "team": 1,
	}); err != nil {
		t.Fatalf("laforge/context: %v", err)
	}
	// Request's return type is `any`; re-decode through a typed Call to
	// check the real payload made it across the wire correctly.
	if err := protocol.Call(ctx, clientConn, "laforge/context", map[string]any{
		"environment": "lm-test", "host": "web01", "team": 1,
	}, &ctxView); err != nil {
		t.Fatalf("laforge/context (typed): %v", err)
	}
	if ctxView.Environment != "lm-test" || ctxView.Host.As != "web01" {
		t.Fatalf("laforge/context result = %+v, want environment=lm-test host.as=web01", ctxView)
	}

	// The sidebar's own custom request, laforge/typeReference -- real
	// wire round trip, not just the pure-Go LaForgeTypeReference tests
	// in custom_test.go. hostURI is back to its real on-disk content
	// after DidClose above (a real host file), so Line 1 ("  name:
	// webserver") is a real, valid position inside the host: block.
	var typeRef struct {
		Kind   string `json:"kind"`
		Fields []struct {
			Name string `json:"name"`
		} `json:"fields"`
	}
	if err := protocol.Call(ctx, clientConn, "laforge/typeReference", map[string]any{
		"uri":      string(hostURI),
		"position": protocol.Position{Line: 1, Character: 2},
	}, &typeRef); err != nil {
		t.Fatalf("laforge/typeReference: %v", err)
	}
	if typeRef.Kind != "host" {
		t.Fatalf("laforge/typeReference kind = %q, want host", typeRef.Kind)
	}
	var sawOS bool
	for _, f := range typeRef.Fields {
		if f.Name == "os" {
			sawOS = true
		}
	}
	if !sawOS {
		t.Fatalf("laforge/typeReference fields missing \"os\"; got %+v", typeRef.Fields)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
