package gateway

import (
	"context"
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

// relaySession is one interactive shell's rendezvous inside the gateway. The
// client half (the api, relaying a UI/CLI user) is a single DUPLEX connection
// -- Go's crypto/tls allows one concurrent reader and one writer. The agent
// half (the host's agent, holding the PTY) is TWO strictly one-directional
// connections: agentOut carries PTY output (agent->client), agentIn carries
// client input (client->agent). The agent splits itself this way because
// rustls cannot be read and written from two threads at once (see
// agentproto.ShellDir). All three dial IN to the gateway; it pairs them by
// session id and pipes raw frames, understanding nothing about the bytes.
// objectID is the deployed_object the (trusted) client named; an agent
// connection is admitted only if its own cert CN matches it, so an agent can
// never attach to a shell aimed at a different host.
type relaySession struct {
	objectID  string
	client    net.Conn
	agentOut  net.Conn // agent -> client (PTY stdout/stderr)
	agentIn   net.Conn // client -> agent (stdin/resize/close)
	ready     chan struct{} // closed once ALL THREE halves are present
	readyOnce sync.Once
	done      chan struct{} // closed once the session tears down
	closeOnce sync.Once
}

func (rs *relaySession) teardown() {
	rs.closeOnce.Do(func() {
		if rs.client != nil {
			rs.client.Close()
		}
		if rs.agentOut != nil {
			rs.agentOut.Close()
		}
		if rs.agentIn != nil {
			rs.agentIn.Close()
		}
		close(rs.done)
	})
}

// ServeRelay accepts internal connections from the api on a SEPARATE listener
// from the agent protocol. This listener is plaintext on purpose: it is never
// funnel-exposed and only reachable on the internal (docker) network, exactly
// like every laforge service's Postgres connection (sslmode=disable) -- the
// agent<->gateway link stays mTLS because that one crosses the untrusted
// network. Each connection's first (and only setup) frame is a ShellAttach with
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
		go s.handleRelayConn(conn)
	}
}

// handleRelayConn reads the opening ShellAttach from an api relay connection and
// hands off to the client half. No TLS/auth here: the listener is internal-only
// (see ServeRelay), so reachability is the trust boundary, as it is for the DB.
func (s *Server) handleRelayConn(conn net.Conn) {
	mt, payload, err := agentproto.ReadFrame(conn)
	if err != nil || mt != agentproto.ShellAttach {
		conn.Close()
		return
	}
	s.handleRelayClient(conn, payload)
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
// opens a connection whose first frame is a ShellAttach with role="agent". The
// agent opens ONE such connection per direction (ShellDirOut, ShellDirIn); this
// authorizes each against the session's object, parks it in the right slot,
// closes ready once all three halves (client + both agent directions) are
// present, and blocks until the session ends (so handleConn does not close this
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
	switch att.Dir {
	case agentproto.ShellDirOut:
		if rs.agentOut != nil {
			s.relayMu.Unlock()
			return // already have this direction
		}
		rs.agentOut = conn
	case agentproto.ShellDirIn:
		if rs.agentIn != nil {
			s.relayMu.Unlock()
			return
		}
		rs.agentIn = conn
	default:
		s.relayMu.Unlock()
		log.Printf("gateway: shell %s: agent %s attached with unknown dir %q", att.SessionID, uuidString(objID), att.Dir)
		return
	}
	bothPresent := rs.agentOut != nil && rs.agentIn != nil
	s.relayMu.Unlock()

	log.Printf("gateway: shell %s: agent %s attached (dir=%s)", att.SessionID, uuidString(objID), att.Dir)
	if bothPresent {
		log.Printf("gateway: shell %s: both agent directions attached, relaying", att.SessionID)
		rs.readyOnce.Do(func() { close(rs.ready) })
	}
	<-rs.done // hold the connection open until the pipe tears down
}

// pipe relays frames both ways until either side ends, then tears the session
// down. Each direction has its own dedicated agent connection, so this is two
// independent one-way copies: the client connection is read by one goroutine
// and written by the other (Go's crypto/tls permits one concurrent reader and
// one writer), while each agent connection is used in a single direction only.
func pipe(rs *relaySession) {
	errc := make(chan error, 2)
	go func() { errc <- copyFrames(rs.client, rs.agentOut) }() // agent -> client (stdout/stderr)
	go func() { errc <- copyFrames(rs.agentIn, rs.client) }()  // client -> agent (stdin/resize/close)
	<-errc
	rs.teardown()  // closing the conns unblocks the other copyFrames
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
		// Advertise the session until BOTH agent directions are attached; the
		// agent dedupes by id (one spawned worker opens both connections), so
		// re-advertising while only one has landed is harmless.
		if rs.objectID == key && (rs.agentOut == nil || rs.agentIn == nil) {
			out = append(out, id)
		}
	}
	return out
}
