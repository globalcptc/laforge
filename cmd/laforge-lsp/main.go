// laforge-lsp is the language server: "the logic lives once
// in Go beside the validator, and VS Code, Neovim, and anything else
// speaking LSP get it for free". This binary is
// deliberately thin -- everything real is internal/lsp; this just wires
// it to stdio using the LSP's own Content-Length header framing, the
// standard transport every LSP client (VS Code, Neovim, ...) speaks by
// default.
package main

import (
	"context"
	"io"
	"log"
	"os"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/globalcptc/laforge/internal/lsp"
)

// stdio adapts os.Stdin/os.Stdout into the single io.ReadWriteCloser
// jsonrpc2.NewHeaderStream wants. Close on this is a no-op: the
// process's own stdio isn't ours to close, and Conn.Close (called on
// Shutdown/Exit below) manages the connection's own teardown separately.
type stdio struct {
	io.Reader
	io.Writer
}

func (stdio) Close() error { return nil }

func main() {
	// Every real diagnostic message goes to stderr, never stdout --
	// stdout is the LSP wire itself; writing anything else to it would
	// corrupt the Content-Length-framed stream a real client is parsing.
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	srv := lsp.NewServer()
	ctx := context.Background()
	stream := jsonrpc2.NewHeaderStream(stdio{Reader: os.Stdin, Writer: os.Stdout})

	_, conn, client := protocol.NewServer(ctx, srv, stream)
	srv.SetClient(client)

	<-conn.Done()
	if err := conn.Err(); err != nil {
		log.Printf("laforge-lsp: connection closed: %v", err)
	}
}
