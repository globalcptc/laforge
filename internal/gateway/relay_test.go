package gateway

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentproto"
)

// TestRelayBridgesClientAndAgent drives the in-memory relay end to end: a client
// half and both agent directions attach for the same session, a data frame flows
// each way (client input over the IN connection, PTY output over the OUT one), a
// mismatched-object agent is rejected, and a ShellClose tears the whole thing
// down and de-registers it. No DB and no TLS -- the relay is a pure in-memory
// byte pipe.
func TestRelayBridgesClientAndAgent(t *testing.T) {
	s := &Server{}

	objID := pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true}
	objStr := uuidString(objID)

	clientGW, clientUser := net.Pipe()
	agentOutGW, agentOutHost := net.Pipe()
	agentInGW, agentInHost := net.Pipe()

	clientAttach, _ := json.Marshal(agentproto.ShellAttachPayload{SessionID: "s1", Role: agentproto.ShellRoleClient, ObjectID: objStr})
	go s.handleRelayClient(clientGW, clientAttach)

	// Wait until the client half has registered the session.
	waitFor(t, func() bool {
		s.relayMu.Lock()
		defer s.relayMu.Unlock()
		return s.relaySessions["s1"] != nil
	})

	// An agent for the WRONG object must be refused (returns immediately, no
	// pairing). Use a different uuid.
	wrong := pgtype.UUID{Bytes: [16]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}, Valid: true}
	badGW, _ := net.Pipe()
	outAttach, _ := json.Marshal(agentproto.ShellAttachPayload{SessionID: "s1", Role: agentproto.ShellRoleAgent, Dir: agentproto.ShellDirOut})
	inAttach, _ := json.Marshal(agentproto.ShellAttachPayload{SessionID: "s1", Role: agentproto.ShellRoleAgent, Dir: agentproto.ShellDirIn})
	done := make(chan struct{})
	go func() { s.handleShellAgent(badGW, wrong, outAttach); close(done) }()
	select {
	case <-done: // good: rejected and returned
	case <-time.After(2 * time.Second):
		t.Fatal("handleShellAgent did not reject a mismatched-object agent")
	}

	// The real agent attaches both directions; the relay starts only once both
	// are present.
	go s.handleShellAgent(agentOutGW, objID, outAttach)
	go s.handleShellAgent(agentInGW, objID, inAttach)

	// client -> agent, over the IN connection
	writeFrame(t, clientUser, agentproto.ShellData, []byte("hello"))
	if mt, body := readFrame(t, agentInHost); mt != agentproto.ShellData || string(body) != "hello" {
		t.Fatalf("agent IN got (%v, %q), want (ShellData, hello)", mt, body)
	}
	// agent -> client, over the OUT connection
	writeFrame(t, agentOutHost, agentproto.ShellData, []byte("world"))
	if mt, body := readFrame(t, clientUser); mt != agentproto.ShellData || string(body) != "world" {
		t.Fatalf("client got (%v, %q), want (ShellData, world)", mt, body)
	}

	// Closing from the client is forwarded to the agent IN side (which, being a
	// real agent, reads it -- net.Pipe is unbuffered, so the test must consume
	// it), then the session tears down and de-registers.
	writeFrame(t, clientUser, agentproto.ShellClose, []byte(`{"reason":"bye"}`))
	if mt, _ := readFrame(t, agentInHost); mt != agentproto.ShellClose {
		t.Fatalf("agent IN got %v on close, want ShellClose", mt)
	}
	waitFor(t, func() bool {
		s.relayMu.Lock()
		defer s.relayMu.Unlock()
		return s.relaySessions["s1"] == nil
	})
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func writeFrame(t *testing.T, conn net.Conn, mt agentproto.MessageType, payload []byte) {
	t.Helper()
	if err := agentproto.WriteFrame(conn, mt, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
}

func readFrame(t *testing.T, conn net.Conn) (agentproto.MessageType, []byte) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	mt, body, err := agentproto.ReadFrame(conn)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	return mt, body
}
