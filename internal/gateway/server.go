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
	"strings"
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
	// Container-log forwarding for the FALLBACK builders only -- Incus
	// (native-OCI) and OpenStack Zun, which have no native log driver, so their
	// agents ship captured console lines here and the gateway forwards them to
	// the environment's container_logs sink. Native builders (MicroCloud,
	// Fargate) ship straight from the platform and never reach this, which keeps
	// the gateway out of the high-volume data path. Sinks are built lazily from
	// each environment's container_logs and shared by identical configs; objSink
	// caches the resolved sink per object (a nil value = that object's
	// environment has no/unsupported container_logs, so no forwarding).
	sinkMu  sync.Mutex
	sinks   map[string]*HTTPSink
	objSink map[string]*HTTPSink

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
	// Host metrics, when the agent sent them -- a malformed or empty payload
	// just leaves them nil (older agents send none), never fails the heartbeat.
	var metrics agentproto.HeartbeatRequestPayload
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &metrics)
	}
	bootAt, agentStartedAt := unixTimestamptz(metrics.BootTime), unixTimestamptz(metrics.AgentStartedAt)
	// Read the check-in context BEFORE the upsert below moves last_heartbeat_at,
	// so this heartbeat can tell whether it's the first since the object was
	// deployed. Best-effort: a failure only skips the check-in event.
	checkin, checkinErr := q.GetAgentCheckinContext(ctx, objID)
	if _, err := q.UpsertAgentSession(ctx, db.UpsertAgentSessionParams{
		DeployedObjectID: objID, CertFingerprint: fingerprint, BootAt: bootAt, AgentStartedAt: agentStartedAt,
	}); err != nil {
		log.Printf("gateway: heartbeat: recording session: %v", err)
	}
	if checkinErr == nil && firstCheckinSinceDeploy(checkin) {
		s.logCheckinEvent(ctx, q, objID, checkin, bootAt, agentStartedAt)
	}
	// Keep the lease on the step the agent is still running. Without this a step
	// longer than one lease expired mid-run, and the next get-task (after a
	// reconnect) leased it again -- the same script running twice at once.
	if metrics.InflightTaskID != "" {
		if taskID, err := parseUUID(metrics.InflightTaskID); err == nil {
			if _, err := q.RenewAgentTaskLease(ctx, db.RenewAgentTaskLeaseParams{
				Lease: db.Interval(s.LeaseDuration), ID: taskID, DeployedObjectID: objID,
			}); err != nil {
				log.Printf("gateway: heartbeat: renewing lease on %s: %v", metrics.InflightTaskID, err)
			}
		}
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
		BootAt: bootAt, AgentStartedAt: agentStartedAt,
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

// unixTimestamptz turns an agent-reported Unix-seconds time into a nullable
// timestamptz. Absent (an older agent) or non-positive stays NULL.
func unixTimestamptz(sec *int64) pgtype.Timestamptz {
	if sec == nil || *sec <= 0 {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: time.Unix(*sec, 0).UTC(), Valid: true}
}

// firstCheckinSinceDeploy: the agent has not heartbeated since the object was
// last deployed. A rebuilt host keeps its old session row, so a heartbeat from
// before deployed_at doesn't count.
func firstCheckinSinceDeploy(c db.GetAgentCheckinContextRow) bool {
	if !c.LastHeartbeatAt.Valid {
		return true
	}
	return c.DeployedAt.Valid && c.LastHeartbeatAt.Time.Before(c.DeployedAt.Time)
}

// logCheckinEvent records an agent's first check-in after a deploy, split into
// the phases the agent's boot report makes visible: deploy -> OS boot -> agent
// start -> first heartbeat. Windows took 6-24 minutes to check in with no way
// to tell where the time went; this puts each phase on the host's timeline.
// Best-effort, like every other journal write here.
func (s *Server) logCheckinEvent(ctx context.Context, q *db.Queries, objID pgtype.UUID, c db.GetAgentCheckinContextRow, bootAt, agentStartedAt pgtype.Timestamptz) {
	buildID, err := q.GetBuildIDByDeployedObject(ctx, objID)
	if err != nil {
		log.Printf("gateway: check-in event: resolving build for %s: %v", objID, err)
		return
	}
	message, payload := checkinSummary(time.Now(), c, bootAt, agentStartedAt)
	body, _ := json.Marshal(payload)
	if err := q.CreateAgentStepEvent(ctx, db.CreateAgentStepEventParams{
		BuildID: buildID, DeployedObjectID: objID, Kind: "agent_checked_in", Message: message, Payload: body,
	}); err != nil {
		log.Printf("gateway: check-in event for %s: %v", objID, err)
	}
}

// checkinSummary builds the agent_checked_in message and payload. Offsets are
// measured from the deploy (when known); a phase the agent didn't report is
// left out rather than guessed.
func checkinSummary(now time.Time, c db.GetAgentCheckinContextRow, bootAt, agentStartedAt pgtype.Timestamptz) (string, map[string]any) {
	payload := map[string]any{"checked_in_at": now.UTC()}
	if c.DeployedAt.Valid {
		payload["deployed_at"] = c.DeployedAt.Time.UTC()
	}
	if bootAt.Valid {
		payload["boot_at"] = bootAt.Time.UTC()
	}
	if agentStartedAt.Valid {
		payload["agent_started_at"] = agentStartedAt.Time.UTC()
	}
	if c.AgentCheckinOverdueAt.Valid {
		payload["overdue_flagged_at"] = c.AgentCheckinOverdueAt.Time.UTC()
	}
	if !c.DeployedAt.Valid {
		return "agent checked in", payload
	}
	since := func(t time.Time) string { return t.Sub(c.DeployedAt.Time).Round(time.Second).String() }
	payload["deploy_to_checkin_sec"] = int64(now.Sub(c.DeployedAt.Time).Seconds())
	msg := "agent checked in " + since(now) + " after deploy"
	var phases []string
	if bootAt.Valid {
		phases = append(phases, "OS booted +"+since(bootAt.Time))
		payload["deploy_to_boot_sec"] = int64(bootAt.Time.Sub(c.DeployedAt.Time).Seconds())
	}
	if agentStartedAt.Valid {
		phases = append(phases, "agent started +"+since(agentStartedAt.Time))
		payload["deploy_to_agent_start_sec"] = int64(agentStartedAt.Time.Sub(c.DeployedAt.Time).Seconds())
	}
	if len(phases) > 0 {
		msg += " (" + strings.Join(phases, ", ") + ")"
	}
	if c.AgentCheckinOverdueAt.Valid {
		msg += "; it had been flagged as never checked in"
	}
	return msg, payload
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
	// Reboot-aware completion: the agent only asks for a task when it has nothing
	// in flight, so a get-task arriving while a `reboots: true` step is still
	// 'leased' means that step rebooted the box and the agent is now back. Treat
	// the reconnect as the step completing -- otherwise the lease would expire and
	// the script would re-run on an already-changed box ("already domain joined").
	if done, err := q.CompleteLeasedRebootsTaskForHost(ctx, objID); err == nil {
		s.logStepEvent(ctx, q, done, "step_done", "reboot step completed; host reconnected after reboot")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("gateway: get-task: completing reboots step: %v", err)
	}
	task, err := q.NextAgentTaskForHost(ctx, db.NextAgentTaskForHostParams{
		DeployedObjectID: objID, Column2: db.Interval(s.LeaseDuration),
	})
	var resp agentproto.GetTaskResponsePayload
	switch {
	case err == nil:
		resp.Task = &agentproto.Task{ID: task.ID.String(), Command: task.Command, Payload: task.Payload}
		// Log the START of the step, so the host's event timeline shows each step
		// beginning, not just its step_done/step_failed outcome. Fires on first
		// lease and on any re-lease after a lease expiry (a retry/reconnect), not
		// on idle polls (those return no task).
		s.logStepEvent(ctx, q, task, "step_started", "")
	case errors.Is(err, pgx.ErrNoRows):
		// resp.Task stays nil -- "nothing to do right now," not an error.
	default:
		log.Printf("gateway: get-task: %v", err)
	}
	// Now that any next step has been leased (its started_at = this reconnect),
	// fold the reboot downtime into the preceding reboot step so its duration
	// reflects the whole reboot, not just the pre-reboot ack. Idempotent; a
	// failure is non-fatal to handing out the task.
	if err := q.StampRebootDurationsForHost(ctx, objID); err != nil {
		log.Printf("gateway: get-task: stamping reboot durations for %s: %v", objID, err)
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
		current, err := q.GetAgentTask(ctx, taskID)
		if err != nil {
			log.Printf("gateway: report-status: loading task %s: %v", req.TaskID, err)
			ok = false
		} else {
			// A retryable transient stays 'pending' until its attempt cap. Once
			// terminal, a task the author marked ignore_errors becomes 'ignored'
			// (does not block later steps or fail the build -- see the query and
			// AdvanceObjectLifecycle) rather than 'failed'.
			status := "pending"
			if failureIsTerminal(current, req) {
				status = "failed"
				if current.IgnoreErrors {
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
				if status == "pending" {
					// A retryable failure under MaxAttempts gets its own
					// step_retry event carrying the attempt and the error.
					// Without it a retry was invisible: the step just
					// re-ran, and a later success wiped last_error, so a
					// script that failed its first attempt on every host
					// looked identical to one that never failed.
					s.logStepEventWith(ctx, q, task, "step_retry", req.Error, map[string]any{
						"attempt": task.Attempts, "max_attempts": attemptLimit(task),
					})
				} else if status == "failed" {
					// The terminal failure gets step_failed -- the event
					// that fails the build -- distinct from the
					// step_retry events for the attempts before it.
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
	s.logStepEventWith(ctx, q, task, kind, detail, nil)
}

// logStepEventWith is logStepEvent with extra structured payload fields
// alongside detail (step_retry's attempt / max_attempts).
func (s *Server) logStepEventWith(ctx context.Context, q *db.Queries, task db.AgentTask, kind, detail string, extra map[string]any) {
	buildID, err := q.GetBuildIDByDeployedObject(ctx, task.DeployedObjectID)
	if err != nil {
		log.Printf("gateway: logStepEvent: resolving build for %s: %v", task.DeployedObjectID, err)
		return
	}
	outcome := "failed"
	switch kind {
	case "step_started":
		outcome = "started"
	case "step_done":
		outcome = "done"
		// A step that only succeeded after retrying says so, so a flaky step
		// stands out even with the step_retry events filtered away.
		if task.Attempts > 1 {
			outcome = fmt.Sprintf("done on attempt %d", task.Attempts)
		}
	case "step_retry":
		outcome = fmt.Sprintf("failed on attempt %d/%d, retrying", task.Attempts, attemptLimit(task))
	case "step_ignored":
		outcome = "failed but ignored"
	}
	message := fmt.Sprintf("step %d (%s) %s", task.StepIndex, task.Command, outcome)
	// Name WHAT the step runs, inline on the message, so a reader learns something
	// from "step 19 (execute) started: win-domain-controller.ps1" without having to
	// expand (step start/done carry no payload detail to expand into anyway).
	if desc := stepDescriptor(task); desc != "" {
		message += ": " + desc
	}
	payload := map[string]any{"detail": detail}
	for k, v := range extra {
		payload[k] = v
	}
	payloadBody, _ := json.Marshal(payload)
	if err := q.CreateAgentStepEvent(ctx, db.CreateAgentStepEventParams{
		BuildID: buildID, DeployedObjectID: task.DeployedObjectID, Kind: kind, Message: message, Payload: payloadBody,
	}); err != nil {
		log.Printf("gateway: logStepEvent: writing event for %s: %v", task.DeployedObjectID, err)
	}
}

// stepDescriptor turns a step's command payload into a short, recognizable hint
// about what the step actually does. "step 19 (execute) started" says nothing;
// "...: win-domain-controller.ps1" names the script, and the script name is also
// how a reader maps a log line back to the authored step (they're generated as
// laforge-step-<n>-<slug>). Best-effort and display-only: an unparseable or
// featureless payload yields "", leaving the bare "step N (command) outcome".
func stepDescriptor(task db.AgentTask) string {
	var p struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Path    string   `json:"path"`
		From    string   `json:"from"`
	}
	if len(task.Payload) > 0 {
		_ = json.Unmarshal(task.Payload, &p)
	}
	switch task.Command {
	case "execute":
		// The script the command runs is the thing a reader recognizes (the long
		// temp path collapses to its basename); with no script arg it's a bare
		// command (dcpromo, /bin/false) -- show that whole, un-basenamed.
		if f := firstScriptArg(p.Args); f != "" {
			return baseName(f)
		}
		return p.Command
	case "write_file":
		if p.Path != "" {
			return baseName(p.Path)
		}
	case "download":
		// agentproto's download payload names its source "from" (and its
		// destination "to"); the URL is what a reader recognizes.
		if p.From != "" {
			return p.From
		}
	}
	return ""
}

// firstScriptArg returns the first non-flag argument -- for "powershell.exe -File
// C:\...\x.ps1" that's the script, for "/bin/bash /tmp/x.sh" likewise. Flags
// (leading '-') are skipped so "-File" doesn't win over the path it introduces.
func firstScriptArg(args []string) string {
	for _, a := range args {
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

// baseName is the last path segment, handling Windows '\\' and POSIX '/' alike
// (the agent runs both, so a single filepath call wouldn't cover the other OS).
func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// failureIsTerminal decides retry vs. give up for a failed report. A step the
// agent killed at its own timeout_sec is terminal on the first attempt: a step
// that ran out its whole deadline (a 60-minute wait loop whose peer never came
// up) would just spend that deadline again. Otherwise the step's own cap -- a
// script's `retries:` (agent_task.max_attempts) or MaxAttempts -- applies.
func failureIsTerminal(task db.AgentTask, req agentproto.ReportStatusRequestPayload) bool {
	if req.IsTimeout() {
		return true
	}
	return task.Attempts >= attemptLimit(task)
}

// attemptLimit is the task's own attempt cap (a script's `retries:`), falling
// back to MaxAttempts.
func attemptLimit(task db.AgentTask) int32 {
	if task.MaxAttempts != nil {
		return *task.MaxAttempts
	}
	return MaxAttempts
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
	} else if len(req.Records) > 0 {
		if sink := s.sinkForObject(ctx, objID); sink != nil {
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
				sink.Enqueue(rec)
			}
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

// sinkForObject returns the log sink for a fallback-builder object's
// environment, or nil when that environment set no (or an unsupported)
// container_logs. Resolved once per object (cached), so the DB is hit at most
// once per object and identical environment configs share one shipper.
func (s *Server) sinkForObject(ctx context.Context, objID pgtype.UUID) *HTTPSink {
	key := uuidString(objID)
	s.sinkMu.Lock()
	if s.objSink == nil {
		s.objSink = make(map[string]*HTTPSink)
	}
	if sink, ok := s.objSink[key]; ok {
		s.sinkMu.Unlock()
		return sink
	}
	s.sinkMu.Unlock()

	var sink *HTTPSink
	if raw, err := db.New(s.Pool).GetEnvironmentContainerLogsForObject(ctx, objID); err == nil && len(raw) > 0 {
		sink = s.sinkForConfig(raw)
	}
	s.sinkMu.Lock()
	s.objSink[key] = sink
	s.sinkMu.Unlock()
	return sink
}

// sinkForConfig maps an environment's container_logs ({driver, options}) to a
// shared HTTPSink, building one per distinct config. The gateway fallback only
// knows how to speak the "splunk" driver (its HEC raw endpoint); any other
// driver on a fallback builder is a documented gap -- native-driver builders
// never reach here, so only Incus/Zun are affected.
func (s *Server) sinkForConfig(raw []byte) *HTTPSink {
	var cl struct {
		Driver  string            `json:"driver"`
		Options map[string]string `json:"options"`
	}
	if err := json.Unmarshal(raw, &cl); err != nil || cl.Driver == "" {
		return nil
	}
	url, headers, ok := fallbackTarget(cl.Driver, cl.Options)
	if !ok {
		log.Printf("gateway: container_logs driver %q has no gateway-fallback mapping (only \"splunk\" is supported for Incus/Zun); not forwarding", cl.Driver)
		return nil
	}
	ckey := url + "\x00" + fmt.Sprint(headers)
	s.sinkMu.Lock()
	defer s.sinkMu.Unlock()
	if s.sinks == nil {
		s.sinks = make(map[string]*HTTPSink)
	}
	if sink, ok := s.sinks[ckey]; ok {
		return sink
	}
	sink := NewHTTPSink(url, headers, nil)
	s.sinks[ckey] = sink
	return sink
}

// fallbackTarget turns a log driver + options into the HTTP endpoint and headers
// the gateway POSTs NDJSON to. Only "splunk" is mapped (its HEC raw endpoint,
// which ingests newline JSON with the token as an Authorization header).
func fallbackTarget(driver string, opts map[string]string) (url string, headers map[string]string, ok bool) {
	switch driver {
	case "splunk":
		base := strings.TrimRight(opts["splunk-url"], "/")
		if base == "" {
			return "", nil, false
		}
		headers = map[string]string{}
		if tok := opts["splunk-token"]; tok != "" {
			headers["Authorization"] = "Splunk " + tok
		}
		return base + "/services/collector/raw", headers, true
	default:
		return "", nil, false
	}
}

// CloseSinks flushes and stops every forwarding sink; the gateway calls it on
// shutdown.
func (s *Server) CloseSinks() {
	s.sinkMu.Lock()
	defer s.sinkMu.Unlock()
	for _, sink := range s.sinks {
		if sink != nil {
			sink.Close()
		}
	}
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
