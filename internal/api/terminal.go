// Interactive shell over a WebSocket: handleTerminal upgrades a browser/CLI
// connection, authenticates it (manage access, like `laforge run`), and bridges
// it to the host's agent through the gateway's internal relay. The api is a
// transparent pump: WebSocket binary frames carry raw PTY bytes both ways, a
// small JSON control message carries terminal resizes, and every session is
// recorded in shell_session for audit and the small global concurrency cap.
package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

// handleTerminal opens an interactive root/admin shell on a host/container. It
// is the GET /builds/{id}/objects/{objectId}/terminal WebSocket endpoint.
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
	// The shell feature must be configured (the gateway relay + api client cert).
	if s.GatewayRelayAddr == "" || s.RelayTLSConfig == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("interactive shell is not configured on this server"))
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
	if !shellReachable(obj.Status) {
		writeError(w, http.StatusConflict, fmt.Errorf("object is %q -- a shell needs a running host with a live agent", obj.Status))
		return
	}

	// Global concurrency cap -- "one or two at a time."
	limit := s.MaxShellSessions
	if limit <= 0 {
		limit = 2
	}
	if n, err := s.Queries.CountLiveShellSessions(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if int(n) >= limit {
		writeError(w, http.StatusTooManyRequests, fmt.Errorf("the shell session limit (%d) is in use; try again shortly", limit))
		return
	}

	// Record the session (audit + cap) BEFORE upgrading, so a refused/failed
	// relay still leaves a trail.
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
	s.auditShell(ctx, build.ID, "shell.opened", fmt.Sprintf("%s opened a shell on %s", sess.GithubLogin, hostName))
	// Whatever happens next, the session ends and is audited exactly once.
	defer func() {
		_ = s.Queries.MarkShellSessionClosed(context.Background(), row.ID)
		s.auditShell(context.Background(), build.ID, "shell.closed", fmt.Sprintf("shell on %s closed", hostName))
	}()

	// Upgrade. We did the Origin check ourselves, so skip the library's.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return // the handshake failed; nothing more to write
	}
	defer c.CloseNow()

	// Dial the gateway relay and attach as the client half for this session.
	raw, err := tls.Dial("tcp", s.GatewayRelayAddr, s.RelayTLSConfig)
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
func (s *Server) pumpTerminal(c *websocket.Conn, raw *tls.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Unblock the blocking ReadFrame below and the WS reader when either side
	// ends.
	go func() {
		<-ctx.Done()
		raw.Close()
		c.CloseNow()
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

// shellReachable reports whether an object's deploy status means its box is up
// with an agent that could serve a shell. A box still deploying, failed, or torn
// down cannot. (A dead-but-"running" agent is caught later by the relay's
// attach timeout.)
func shellReachable(status string) bool {
	switch status {
	case "running", "building", "finished":
		return true
	}
	return false
}

// auditShell writes one shell lifecycle event to the build journal. Best-effort:
// a journal write must never fail the session itself.
func (s *Server) auditShell(ctx context.Context, buildID pgtype.UUID, kind, message string) {
	s.Queries.CreateEvent(ctx, db.CreateEventParams{
		BuildID: buildID, Kind: kind, Message: message, Payload: []byte("{}"),
	})
}
