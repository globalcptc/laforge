package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentproto"
)

// agentAttachTimeout bounds how long a client (api) relay connection waits for
// the host's agent to open its half. The agent learns of the session on its
// next heartbeat (which the gateway shortens while one is pending), so this only
// has to tolerate an agent that is slow, offline, or gone -- not the normal
// sub-second open.
const agentAttachTimeout = 30 * time.Second

// relaySession is one interactive shell's rendezvous inside the gateway: the
// client half (the api, relaying a UI/CLI user -- duplex) and the agent half
// (the host's agent, holding the PTY). Both dial IN to the gateway; it pairs
// them by session id and pipes raw frames between them, understanding nothing
// about the bytes. objectID is the deployed_object the (trusted) client named;
// the agent half is admitted only if its own cert CN matches it, so an agent can
// never attach to a shell aimed at a different host.
type relaySession struct {
	objectID  string
	client    net.Conn
	agent     net.Conn
	ready     chan struct{} // closed once BOTH halves are present
	done      chan struct{} // closed once the session tears down
	closeOnce sync.Once
}

func (rs *relaySession) teardown() {
	rs.closeOnce.Do(func() {
		if rs.client != nil {
			rs.client.Close()
		}
		if rs.agent != nil {
			rs.agent.Close()
		}
		close(rs.done)
	})
}

// ServeRelay accepts internal mTLS connections from the api on a SEPARATE
// listener from the agent protocol -- not funnel-exposed, internal network only.
// Each connection's first (and only setup) frame is a ShellAttach with
// role="client"; the gateway then bridges it to the matching agent half. Mirrors
// Serve's accept/close-on-ctx shape.
func (s *Server) ServeRelay(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return err
			}
		}
		go s.handleRelayConn(ctx, conn)
	}
}

// handleRelayConn does the mTLS handshake and reads the opening ShellAttach from
// an api relay connection, then hands off to the client half. Any CA-signed
// cert is accepted here (the listener is internal), so -- unlike the agent
// listener -- the CN is not required to be a deployed_object id.
func (s *Server) handleRelayConn(ctx context.Context, conn net.Conn) {
	tconn, ok := conn.(*tls.Conn)
	if !ok {
		conn.Close()
		return
	}
	if err := tconn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return
	}
	mt, payload, err := agentproto.ReadFrame(tconn)
	if err != nil || mt != agentproto.ShellAttach {
		conn.Close()
		return
	}
	s.handleRelayClient(tconn, payload)
}

// handleRelayClient registers the client half, waits for the agent half to
// attach, then pipes bytes between the two until either side ends. It owns the
// session's lifecycle: it creates the registry entry and removes it on return.
func (s *Server) handleRelayClient(conn net.Conn, payload []byte) {
	defer conn.Close()
	var att agentproto.ShellAttachPayload
	if err := json.Unmarshal(payload, &att); err != nil || att.SessionID == "" || att.ObjectID == "" || att.Role != agentproto.ShellRoleClient {
		return
	}
	rs := &relaySession{objectID: att.ObjectID, client: conn, ready: make(chan struct{}), done: make(chan struct{})}

	s.relayMu.Lock()
	if s.relaySessions == nil {
		s.relaySessions = make(map[string]*relaySession)
	}
	if _, exists := s.relaySessions[att.SessionID]; exists {
		s.relayMu.Unlock()
		return // a session with this id is already in flight
	}
	s.relaySessions[att.SessionID] = rs
	s.relayMu.Unlock()

	defer func() {
		s.relayMu.Lock()
		delete(s.relaySessions, att.SessionID)
		s.relayMu.Unlock()
		rs.teardown()
	}()

	select {
	case <-rs.ready:
		// both halves present -- relay until one side ends
		pipe(rs)
	case <-time.After(agentAttachTimeout):
		log.Printf("gateway: shell %s: agent never attached within %s", att.SessionID, agentAttachTimeout)
	}
}

// handleShellAgent is called from the agent listener (handleConn) when an agent
// opens a connection whose first frame is a ShellAttach with role="agent". It
// authorizes the agent against the session's object, pairs it with the waiting
// client, and blocks until the session ends (so handleConn does not close this
// connection out from under the pipe).
func (s *Server) handleShellAgent(conn net.Conn, objID pgtype.UUID, payload []byte) {
	var att agentproto.ShellAttachPayload
	if err := json.Unmarshal(payload, &att); err != nil || att.SessionID == "" {
		return
	}
	s.relayMu.Lock()
	rs := s.relaySessions[att.SessionID]
	s.relayMu.Unlock()
	if rs == nil {
		return // no client is waiting (timed out, or never existed)
	}
	// The agent may only serve a shell aimed at its OWN object.
	if uuidString(objID) != rs.objectID {
		log.Printf("gateway: shell %s: agent %s may not attach (session targets %s)", att.SessionID, uuidString(objID), rs.objectID)
		return
	}
	s.relayMu.Lock()
	if rs.agent != nil {
		s.relayMu.Unlock()
		return // already has an agent half
	}
	rs.agent = conn
	s.relayMu.Unlock()

	close(rs.ready)
	<-rs.done // hold the connection open until the pipe tears down
}

// pipe relays frames both ways until either side ends, then tears the session
// down. Go's crypto/tls allows one concurrent reader and one concurrent writer
// per connection, so each direction is its own goroutine reading one conn and
// writing the other.
func pipe(rs *relaySession) {
	errc := make(chan error, 2)
	go func() { errc <- copyFrames(rs.client, rs.agent) }() // agent -> client (stdout/stderr)
	go func() { errc <- copyFrames(rs.agent, rs.client) }() // client -> agent (stdin/resize/close)
	<-errc
	rs.teardown()  // closing both conns unblocks the other copyFrames
	<-errc         // drain it so neither goroutine leaks
}

// copyFrames forwards whole frames from src to dst until a read/write error or a
// ShellClose (which it forwards, then stops so the other side learns). It never
// inspects ShellData payloads -- raw PTY bytes pass through untouched.
func copyFrames(dst, src net.Conn) error {
	for {
		mt, payload, err := agentproto.ReadFrame(src)
		if err != nil {
			return err
		}
		if err := agentproto.WriteFrame(dst, mt, payload); err != nil {
			return err
		}
		if mt == agentproto.ShellClose {
			return nil
		}
	}
}

// pendingSessionsForObject returns the ids of shell sessions whose client half
// is waiting for THIS object's agent to attach -- what handleHeartbeat hands
// back so the agent opens its half. In-memory and tiny (one or two at a time).
func (s *Server) pendingSessionsForObject(objID pgtype.UUID) []string {
	key := uuidString(objID)
	s.relayMu.Lock()
	defer s.relayMu.Unlock()
	var out []string
	for id, rs := range s.relaySessions {
		if rs.objectID == key && rs.agent == nil {
			out = append(out, id)
		}
	}
	return out
}
