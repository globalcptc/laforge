package gateway

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

// MaxAttempts caps agent_task retries, the same "a cap" half of "retries
// use backoff and a cap" internal/runner applies to builder tasks --
// backoff isn't implemented here either, for the same reason:
// a failed step under the cap is immediately
// re-offered by NextAgentTaskForHost's own lease-reclaim query, no
// separate delay logic yet.
const MaxAttempts = 3

// Server is the agent-gateway: "its own container, its own domain, scaled
// on its own. Speaks only the agent protocol." Pool must be a connection
// using the restricted laforge_gateway role (migrations/00004) -- Server
// itself enforces nothing about that; the database does, by simply
// refusing any query this role wasn't granted, which internal/gateway's
// own tests connect as that exact role to confirm.
type Server struct {
	Pool          *pgxpool.Pool
	TLSConfig     *tls.Config
	LeaseDuration time.Duration
	// BasePollMS/JitterMS: "jittered intervals, so agents deployed
	// together do not stay synchronised for the rest of the event" --
	// HeartbeatResponse hands back BasePollMS plus a random amount up to
	// JitterMS, a new random draw every heartbeat.
	BasePollMS int
	JitterMS   int
	// LogSink, when set, receives every container's captured console lines
	// (enriched with build/team/object identity) for forwarding to an external
	// ingester. Nil disables log forwarding -- agents still send, the gateway
	// just acks and discards, so delivery is an operator choice, not a build
	// requirement. See logsink.go.
	LogSink LogSink

	// logCtx caches each object's enrichment identity (build/team/name/kind),
	// resolved once per object rather than on every log batch.
	logCtxMu sync.Mutex
	logCtx   map[string]logContext

	// relaySessions holds in-flight interactive shell sessions keyed by session
	// id, each pairing a client (api) connection with the host agent's
	// connection for a live PTY. In-memory only -- the relay is a dumb byte
	// pipe; the api owns the durable shell_session row. See relay.go. Guarded
	// by relayMu.
	relayMu       sync.Mutex
	relaySessions map[string]*relaySession
}

// Serve accepts connections on ln (expected to be a *tls.Conn listener,
// wrapping TLSConfig -- see Listen) until ctx is done or Accept fails.
// Each connection is handled in its own goroutine and lives until the
// agent disconnects or sends something malformed; "agents tolerate it
// being down... they retry with backoff" is the agent's own
// responsibility, not this server's.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
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
		go s.handleConn(ctx, conn)
	}
}

// Listen wraps net.Listen with s.TLSConfig -- a small convenience so
// cmd/laforge-gateway doesn't need to import crypto/tls itself.
func (s *Server) Listen(addr string) (net.Listener, error) {
	return tls.Listen("tcp", addr, s.TLSConfig)
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	tconn, ok := conn.(*tls.Conn)
	if !ok {
		return
	}
	// Handshake explicitly, before touching the database at all: a peer
	// that fails mTLS never gets far enough to cause a query, and errors
	// here are the ordinary, expected shape of "not a real agent" (a port
	// scanner, a misconfigured client), not something to log loudly.
	if err := tconn.HandshakeContext(ctx); err != nil {
		return
	}
	state := tconn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return
	}
	peerCert := state.PeerCertificates[0]
	objID, err := parseUUID(peerCert.Subject.CommonName)
	if err != nil {
		// A cert the CA genuinely signed, but whose CommonName isn't a
		// deployed_object id -- not possible from the real agent factory,
		// but a gateway that crashed on it would be a much worse failure
		// mode than just closing the connection.
		return
	}
	fingerprint := certFingerprint(peerCert)

	for {
		mt, payload, err := agentproto.ReadFrame(tconn)
		if err != nil {
			return // disconnect, malformed frame, or idle timeout -- session just ends
		}
		switch mt {
		case agentproto.HeartbeatRequest:
			s.handleHeartbeat(ctx, tconn, objID, fingerprint, payload)
		case agentproto.GetTaskRequest:
			s.handleGetTask(ctx, tconn, objID)
		case agentproto.ReportStatusRequest:
			s.handleReportStatus(ctx, tconn, payload)
		case agentproto.LogBatchRequest:
			s.handleLogBatch(ctx, tconn, objID, payload)
		case agentproto.ShellAttach:
			// This connection is a dedicated interactive-shell channel, not the
			// poll loop: hand it to the relay (which blocks until the session
			// ends) and then let the connection close. See relay.go.
			s.handleShellAgent(tconn, objID, payload)
			return
		default:
			return
		}
	}
}

func (s *Server) handleHeartbeat(ctx context.Context, conn net.Conn, objID pgtype.UUID, fingerprint string, payload []byte) {
	q := db.New(s.Pool)
	if _, err := q.UpsertAgentSession(ctx, db.UpsertAgentSessionParams{
		DeployedObjectID: objID, CertFingerprint: fingerprint,
	}); err != nil {
		log.Printf("gateway: heartbeat: recording session: %v", err)
	}
	// Host metrics, when the agent sent them -- a malformed or empty payload
	// just leaves them nil (older agents send none), never fails the heartbeat.
	var metrics agentproto.HeartbeatRequestPayload
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &metrics)
	}
	next := s.BasePollMS + rand.Intn(max(s.JitterMS, 1))
	// If a client is waiting for an interactive shell on this object, tell the
	// agent to open its half now and poll back quickly so the shell comes up
	// sub-second instead of waiting out the normal (jittered, multi-second)
	// interval.
	pending := s.pendingSessionsForObject(objID)
	if len(pending) > 0 {
		next = 250
		// Diagnostic: proves this (fixed) gateway is telling the agent to open
		// the shell. If you see this but the agent never logs "opening shell",
		// the agent on that host is stale (predates the shell feature) and the
		// host needs redeploying.
		log.Printf("gateway: heartbeat %s: signaling agent of pending shell session(s) %v", uuidString(objID), pending)
	}
	// The append-only log, alongside (never instead of) the upsert above
	// -- "log as much data as we can to help with live troubleshooting
	// and rules checking" (see migrations/00007's own doc comment for
	// the full reasoning). remote_addr comes straight off the real
	// connection, not a header a client could claim to be -- exactly
	// like cert_fingerprint's own trust model. A logging failure here is
	// deliberately non-fatal to the heartbeat itself: an agent's actual
	// liveness (the upsert above, and the response below) must not
	// depend on this table being writable.
	nextI32 := int32(next)
	if err := q.CreateAgentHeartbeat(ctx, db.CreateAgentHeartbeatParams{
		DeployedObjectID: objID, CertFingerprint: fingerprint,
		RemoteAddr: remoteAddrString(conn), NextPollMs: &nextI32,
		CpuPct: metrics.CPUPct, MemPct: metrics.MemPct, DiskPct: metrics.DiskPct,
		NetRxBps: metrics.NetRxBps, NetTxBps: metrics.NetTxBps,
	}); err != nil {
		log.Printf("gateway: heartbeat: logging history: %v", err)
	}
	// PendingSessions is what actually tells the agent to open its shell half --
	// without it the agent only polls faster and never learns which session to
	// join, so every shell times out with "agent never attached".
	body, _ := json.Marshal(agentproto.HeartbeatResponsePayload{NextPollMS: next, PendingSessions: pending})
	if err := agentproto.WriteFrame(conn, agentproto.HeartbeatResponse, body); err != nil {
		log.Printf("gateway: heartbeat: writing response: %v", err)
	}
}

// remoteAddrString is nil-safe: conn.RemoteAddr() can itself return nil
// on some connection types, and a nil interface holding a typed nil
// (net.Addr(nil)) would otherwise stringify to a useless "<nil>" instead
// of leaving the column genuinely NULL.
func remoteAddrString(conn net.Conn) *string {
	addr := conn.RemoteAddr()
	if addr == nil {
		return nil
	}
	s := addr.String()
	return &s
}

func (s *Server) handleGetTask(ctx context.Context, conn net.Conn, objID pgtype.UUID) {
	q := db.New(s.Pool)
	task, err := q.NextAgentTaskForHost(ctx, db.NextAgentTaskForHostParams{
		DeployedObjectID: objID, Column2: db.Interval(s.LeaseDuration),
	})
	var resp agentproto.GetTaskResponsePayload
	switch {
	case err == nil:
		resp.Task = &agentproto.Task{ID: task.ID.String(), Command: task.Command, Payload: task.Payload}
	case errors.Is(err, pgx.ErrNoRows):
		// resp.Task stays nil -- "nothing to do right now," not an error.
	default:
		log.Printf("gateway: get-task: %v", err)
	}
	body, _ := json.Marshal(resp)
	if err := agentproto.WriteFrame(conn, agentproto.GetTaskResponse, body); err != nil {
		log.Printf("gateway: get-task: writing response: %v", err)
	}
}

func (s *Server) handleReportStatus(ctx context.Context, conn net.Conn, payload []byte) {
	q := db.New(s.Pool)
	var req agentproto.ReportStatusRequestPayload
	ok := true
	if err := json.Unmarshal(payload, &req); err != nil {
		ok = false
	} else if taskID, err := parseUUID(req.TaskID); err != nil {
		ok = false
	} else if req.Status == "done" {
		task, err := q.CompleteAgentTask(ctx, db.CompleteAgentTaskParams{ID: taskID, Output: db.StrPtr(req.Output)})
		if err != nil {
			log.Printf("gateway: report-status: completing %s: %v", req.TaskID, err)
			ok = false
		} else {
			s.logStepEvent(ctx, q, task, "step_done", req.Output)
			s.recordValidatorResults(ctx, q, taskID, req.ValidatorResults)
		}
	} else {
		current, ignoreErrors, err := currentAttempts(ctx, q, taskID)
		if err != nil {
			log.Printf("gateway: report-status: loading task %s: %v", req.TaskID, err)
			ok = false
		} else {
			// A retryable transient stays 'pending' until MaxAttempts. Once
			// terminal, a task the author marked ignore_errors becomes 'ignored'
			// (does not block later steps or fail the build -- see the query and
			// AdvanceObjectLifecycle) rather than 'failed'.
			status := "pending"
			if current >= MaxAttempts {
				status = "failed"
				if ignoreErrors {
					status = "ignored"
				}
			}
			task, err := q.FailAgentTask(ctx, db.FailAgentTaskParams{ID: taskID, Status: status, LastError: db.StrPtr(req.Error)})
			if err != nil {
				log.Printf("gateway: report-status: failing %s: %v", req.TaskID, err)
				ok = false
			} else {
				// Every attempt's own real validator results, not just
				// the terminal one -- unlike a generic error message
				// (internal retry noise, only worth logging once
				// terminal), a check that genuinely failed (e.g. "port
				// 22 is not listening") is real signal the first time
				// it's reported, and an operator debugging "why did my
				// validation fail" benefits from the same real history
				// agent_heartbeat's own append-only log already gives
				// every other check-in.
				s.recordValidatorResults(ctx, q, taskID, req.ValidatorResults)
				if status == "failed" {
					// Only the terminal failure gets a step_failed
					// event, not every transient retry attempt under
					// MaxAttempts -- matching the granularity deploy/
					// destroy lifecycle events already use.
					s.logStepEvent(ctx, q, task, "step_failed", req.Error)
				} else if status == "ignored" {
					// A tolerated terminal failure still gets its own event so
					// an operator can see the step failed and was ignored --
					// distinct from step_failed, which fails the build.
					s.logStepEvent(ctx, q, task, "step_ignored", req.Error)
				}
			}
		}
	}
	body, _ := json.Marshal(agentproto.ReportStatusResponsePayload{OK: ok})
	if err := agentproto.WriteFrame(conn, agentproto.ReportStatusResponse, body); err != nil {
		log.Printf("gateway: report-status: writing response: %v", err)
	}
}

// recordValidatorResults is the real fix for a gap found by direct audit:
// the agent already computed a real, structured
// per-check result (kind/args/passed/message) for every `validate:`
// check, but this gateway only ever persisted the flattened text blob
// every other command's Output/Error already carries -- RecordValidatorResult
// and the validator_result table it writes existed with real typed
// columns since the agent schema, with zero callers. A
// no-op when results is empty, which is every non-validate report.
func (s *Server) recordValidatorResults(ctx context.Context, q *db.Queries, taskID pgtype.UUID, results []agentproto.ValidatorResult) {
	for _, r := range results {
		args := r.Args
		if args == nil {
			args = json.RawMessage("{}")
		}
		if err := q.RecordValidatorResult(ctx, db.RecordValidatorResultParams{
			AgentTaskID: taskID, Kind: r.Kind, Args: []byte(args), Passed: r.Passed, Message: db.StrPtr(r.Message),
		}); err != nil {
			log.Printf("gateway: report-status: recording validator result %q for task %s: %v", r.Kind, taskID, err)
		}
	}
}

// logStepEvent is the real fix for "the gateway doesn't log step-level
// events into the journal" -- one real entry per step outcome, in the
// same `event` table
// deploy/destroy/access lifecycle events already use, so "every object
// has its own log... a step" is actually true now
// (ListEventsByDeployedObject reads both). detail is req.Output or
// req.Error, kept in the payload rather than the message so the message
// itself stays a short, consistent one-liner (matching every other real
// event's kind/message split in this codebase) while the full text is
// still there for whoever clicks in. Best-effort: a failure to resolve
// build_id or write the event never fails the agent's own report-status
// call, which the response the caller already sent back doesn't depend
// on -- an agent's step genuinely completing must not hinge on the
// journal being writable, same principle CreateAgentHeartbeat's own
// doc comment already states for heartbeats.
func (s *Server) logStepEvent(ctx context.Context, q *db.Queries, task db.AgentTask, kind, detail string) {
	buildID, err := q.GetBuildIDByDeployedObject(ctx, task.DeployedObjectID)
	if err != nil {
		log.Printf("gateway: logStepEvent: resolving build for %s: %v", task.DeployedObjectID, err)
		return
	}
	outcome := "failed"
	switch kind {
	case "step_done":
		outcome = "done"
	case "step_ignored":
		outcome = "failed but ignored"
	}
	message := fmt.Sprintf("step %d (%s) %s", task.StepIndex, task.Command, outcome)
	payloadBody, _ := json.Marshal(map[string]string{"detail": detail})
	if err := q.CreateAgentStepEvent(ctx, db.CreateAgentStepEventParams{
		BuildID: buildID, DeployedObjectID: task.DeployedObjectID, Kind: kind, Message: message, Payload: payloadBody,
	}); err != nil {
		log.Printf("gateway: logStepEvent: writing event for %s: %v", task.DeployedObjectID, err)
	}
}

// currentAttempts returns how many times a task has been attempted and whether
// its failure should be tolerated (ignore_errors) -- both read from the one row
// report-status needs to decide retry vs. terminal-failed vs. terminal-ignored.
func currentAttempts(ctx context.Context, q *db.Queries, taskID pgtype.UUID) (attempts int32, ignoreErrors bool, err error) {
	task, err := q.GetAgentTask(ctx, taskID)
	if err != nil {
		return 0, false, err
	}
	return task.Attempts, task.IgnoreErrors, nil
}

// logContext is one object's enrichment identity, cached in Server.logCtx.
type logContext struct {
	buildID string
	team    int32
	object  string
	kind    string
}

// handleLogBatch enriches a batch of captured console lines with the object's
// build/team/name/kind (keyed off the agent's cert, like every other verb) and
// hands each to the log sink. Enqueue never blocks, so a slow backend can't
// stall the agent; the ack goes back regardless. A nil sink (forwarding
// disabled) still acks and discards -- an agent must not fail because the
// operator chose not to ship logs.
func (s *Server) handleLogBatch(ctx context.Context, conn net.Conn, objID pgtype.UUID, payload []byte) {
	var req agentproto.LogBatchRequestPayload
	ok := true
	if err := json.Unmarshal(payload, &req); err != nil {
		ok = false
	} else if s.LogSink != nil && len(req.Records) > 0 {
		lc, found := s.logContextFor(ctx, objID)
		oid := uuidString(objID)
		for _, r := range req.Records {
			rec := LogRecord{
				TS:       time.UnixMilli(r.TSMs).UTC().Format("2006-01-02T15:04:05.000Z07:00"),
				ObjectID: oid,
				Stream:   r.Stream,
				Line:     r.Line,
				Dropped:  r.Dropped,
			}
			if found {
				rec.BuildID, rec.Team, rec.Object, rec.Kind = lc.buildID, lc.team, lc.object, lc.kind
			}
			s.LogSink.Enqueue(rec)
		}
	}
	body, _ := json.Marshal(agentproto.LogBatchResponsePayload{OK: ok})
	if err := agentproto.WriteFrame(conn, agentproto.LogBatchResponse, body); err != nil {
		log.Printf("gateway: log-batch: writing response: %v", err)
	}
}

// logContextFor resolves and caches an object's enrichment identity. Cached
// because it never changes for the life of a build, so chatty container logs
// read it once, not once per batch. A resolution failure returns ok=false and
// the records still ship with their object_id (self-identifying), just without
// the human-readable build/team/name -- better than dropping the line.
func (s *Server) logContextFor(ctx context.Context, objID pgtype.UUID) (logContext, bool) {
	key := uuidString(objID)
	s.logCtxMu.Lock()
	if s.logCtx == nil {
		s.logCtx = make(map[string]logContext)
	}
	if lc, ok := s.logCtx[key]; ok {
		s.logCtxMu.Unlock()
		return lc, true
	}
	s.logCtxMu.Unlock()

	row, err := db.New(s.Pool).GetLogContextByDeployedObject(ctx, objID)
	if err != nil {
		return logContext{}, false
	}
	object := row.ObjectName
	if row.AsName != nil && *row.AsName != "" {
		object = *row.AsName
	}
	lc := logContext{buildID: uuidString(row.BuildID), team: row.TeamNumber, object: object, kind: row.Kind}
	s.logCtxMu.Lock()
	s.logCtx[key] = lc
	s.logCtxMu.Unlock()
	return lc, true
}

// uuidString renders a pgtype.UUID in canonical 8-4-4-4-12 form, for a map key
// and for the self-identifying object_id/build_id a log record carries.
func uuidString(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func certFingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
