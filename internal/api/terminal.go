// Interactive shell over a WebSocket: handleTerminal upgrades a browser/CLI
// connection, authenticates it (manage access, like `laforge run`), and bridges
// it to the host's agent through the gateway's internal relay. The api is a
// transparent pump: WebSocket binary frames carry raw PTY bytes both ways, a
// small JSON control message carries terminal resizes, and every session is
// recorded in shell_session for audit and the small global concurrency cap.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

// gatewayRelayAddr is where the api reaches the gateway's shell relay: fixed
// internal plumbing over the docker network (plaintext, like the DB
// connection), not a configurable knob.
const gatewayRelayAddr = "gateway:8445"

// An interactive shell can sit idle indefinitely (someone reading output, or
// just leaving the window open), during which the PTY sends no bytes. A proxy
// in front of the api closes a WebSocket that has been silent too long, which
// is what was killing idle sessions. A periodic ping keeps the
// connection alive for as long as the window is open, and doubles as dead-peer
// detection: if the window is gone the ping (really its pong) fails and the
// session tears down promptly instead of lingering. The interval is well under
// the ~60s idle timeout common to such proxies.
const (
	keepaliveInterval = 20 * time.Second
	keepaliveTimeout  = 10 * time.Second
)

// handleTerminal opens an interactive root/admin shell on a host/container. It
// is the GET /builds/{id}/objects/{objectId}/terminal WebSocket endpoint. The
// feature is always on; access is gated by manage level, like `laforge run`.
func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	// Running a root shell is at least as privileged as the ad-hoc `run`
	// command, so it uses the same bar: manage access (cookie or bearer, via
	// requireLevel's own auth). sess identifies who to audit.
	sess, err := s.requireLevel(ctx, r, repo, levelManage)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	// CSRF defense for WebSockets: the CORS wrapper does not guard a WS upgrade,
	// and a browser cannot set an Authorization header (it rides the cookie), so
	// a cross-site page could otherwise open a shell as the signed-in user. A
	// browser always sends Origin; require it to match the UI. A non-browser
	// client (the CLI) sends no Origin and authenticates with a bearer token, so
	// an empty Origin is allowed.
	if origin := r.Header.Get("Origin"); origin != "" && origin != s.UIBaseURL {
		writeError(w, http.StatusForbidden, errors.New("cross-origin terminal connection refused"))
		return
	}

	objectID, err := s.objectInBuild(r, build.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	obj, err := s.Queries.GetDeployedObject(ctx, objectID)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such object"))
		return
	}
	if obj.Kind != "host" && obj.Kind != "container" {
		writeError(w, http.StatusBadRequest, errors.New("only hosts and containers have a shell"))
		return
	}
	// A shell is gated on a LIVE AGENT, not a successful build. The terminal's
	// main job is debugging the very scripts that fail, so a build_failed (or
	// invalid) host whose agent is still checking in must still get a shell --
	// that's exactly when an operator needs one. We look up the agent's last
	// heartbeat; a missing session or a stale one is the only thing that blocks.
	agentSess, sessErr := s.Queries.GetAgentSessionByDeployedObject(ctx, objectID)
	if reason := shellBlockedReason(obj.Status, sessErr == nil, agentSess.LastHeartbeatAt.Time, time.Now()); reason != "" {
		writeError(w, http.StatusConflict, errors.New(reason))
		return
	}

	// Record the session BEFORE upgrading, so even a failed relay leaves an
	// audit trail (who opened a shell on which host). Sessions are independent
	// and cheap, so there is no concurrency cap -- this row and its events are
	// purely the log.
	row, err := s.Queries.CreateShellSession(ctx, db.CreateShellSessionParams{
		DeployedObjectID: objectID, OpenedByAccountID: sess.AccountID, ClientAddr: db.StrPtr(r.RemoteAddr),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	hostName := obj.ObjectName
	if obj.AsName != nil && *obj.AsName != "" {
		hostName = *obj.AsName
	}
	s.auditShell(ctx, build.ID, objectID, "shell.opened", fmt.Sprintf("%s opened a shell on %s", sess.GithubLogin, hostName))
	// Whatever happens next, the session ends and is audited exactly once.
	defer func() {
		_ = s.Queries.MarkShellSessionClosed(context.Background(), row.ID)
		s.auditShell(context.Background(), build.ID, objectID, "shell.closed", fmt.Sprintf("%s's shell on %s closed", sess.GithubLogin, hostName))
	}()

	// Upgrade. We did the Origin check ourselves, so skip the library's.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return // the handshake failed; nothing more to write
	}
	defer c.CloseNow()

	// Dial the gateway relay (plaintext, internal network -- see the gateway's
	// ServeRelay) and attach as the client half for this session.
	raw, err := net.Dial("tcp", gatewayRelayAddr)
	if err != nil {
		c.Close(websocket.StatusInternalError, "cannot reach the shell relay")
		return
	}
	defer raw.Close()
	attach, _ := json.Marshal(agentproto.ShellAttachPayload{
		SessionID: row.ID.String(), Role: agentproto.ShellRoleClient, ObjectID: objectID.String(),
	})
	if err := agentproto.WriteFrame(raw, agentproto.ShellAttach, attach); err != nil {
		c.Close(websocket.StatusInternalError, "cannot start the shell relay")
		return
	}
	_ = s.Queries.MarkShellSessionActive(ctx, row.ID)

	s.pumpTerminal(c, raw)
}

// pumpTerminal copies bytes between the WebSocket and the gateway relay until
// either side ends. One goroutine drives WS->relay (stdin/resize), this
// goroutine drives relay->WS (stdout); cancelling closes both so neither blocks.
func (s *Server) pumpTerminal(c *websocket.Conn, raw net.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Unblock the blocking ReadFrame below and the WS reader when either side
	// ends.
	go func() {
		<-ctx.Done()
		raw.Close()
		c.CloseNow()
	}()

	// Keepalive pings so an idle session survives for as long as the window is
	// open (see keepaliveInterval). Safe alongside the Read loop below -- the
	// pong is read there -- and alongside the relay->WS writes.
	go func() {
		t := time.NewTicker(keepaliveInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				pctx, pcancel := context.WithTimeout(ctx, keepaliveTimeout)
				err := c.Ping(pctx)
				pcancel()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	// WebSocket -> relay: binary is stdin, a text control message is a resize.
	go func() {
		defer cancel()
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			switch typ {
			case websocket.MessageBinary:
				if err := agentproto.WriteFrame(raw, agentproto.ShellData, data); err != nil {
					return
				}
			case websocket.MessageText:
				var ctl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &ctl) == nil && ctl.Type == "resize" {
					body, _ := json.Marshal(agentproto.ShellResizePayload{Cols: ctl.Cols, Rows: ctl.Rows})
					if err := agentproto.WriteFrame(raw, agentproto.ShellResize, body); err != nil {
						return
					}
				}
			}
		}
	}()

	// relay -> WebSocket: PTY output as binary; a close ends it.
	for {
		mt, payload, err := agentproto.ReadFrame(raw)
		if err != nil {
			break
		}
		switch mt {
		case agentproto.ShellData:
			if err := c.Write(ctx, websocket.MessageBinary, payload); err != nil {
				return
			}
		case agentproto.ShellClose:
			c.Close(websocket.StatusNormalClosure, "shell closed")
			return
		}
	}
}

// shellWindow is how recently the agent must have checked in for a shell to be
// opened. It's the "late" band from health.go (not the tighter "healthy" one):
// within it the agent is still polling and will pick up the pending session on
// its next heartbeat, so a slightly-behind agent should not be denied a shell;
// past it the agent is written off as missing and the relay would just time out.
const shellWindow = lateWindow

// shellBlockedReason returns "" if an interactive shell can be opened on an
// object in this state, or a human-readable reason it cannot.
//
// A shell is gated on a LIVE AGENT, not a successful build. An operator mostly
// opens a terminal to debug a script -- including the script that just failed --
// so a build_failed or invalid host whose agent is still checking in MUST get a
// shell. The only real requirements are therefore: the instance is actually up
// at the hoster (infraUp, which already counts build_failed/invalid as up), and
// the agent has heartbeat within shellWindow. A box still deploying, failed to
// deploy, or torn down has no live agent to relay through.
func shellBlockedReason(status string, hasSession bool, lastHeartbeat time.Time, now time.Time) string {
	if !infraUp(status) {
		return fmt.Sprintf("the host is %q -- a shell needs a deployed host, not one still deploying or torn down", status)
	}
	if !hasSession {
		return "the agent on this host hasn't checked in yet -- a shell needs a live agent"
	}
	if age := now.Sub(lastHeartbeat); age > shellWindow {
		return fmt.Sprintf("the agent on this host last checked in %s ago -- a shell needs a live agent", age.Round(time.Second))
	}
	return ""
}

// auditShell writes one shell lifecycle event attributed to the host, so it
// shows both in the build log and on that host's own timeline (who opened a
// prompt, where, when). Best-effort: a journal write must never fail the
// session itself.
func (s *Server) auditShell(ctx context.Context, buildID, objectID pgtype.UUID, kind, message string) {
	s.Queries.CreateObjectEvent(ctx, db.CreateObjectEventParams{
		BuildID: buildID, DeployedObjectID: objectID, Kind: kind, Message: message, Payload: []byte("{}"),
	})
}
