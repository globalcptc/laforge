// handleLiveStatus is "Build → Logs: Live over SSE while anything is running"
// and the plan's "Performance: Stream, do not poll, for live
// state." It LISTENs on the Postgres 'laforge_event' channel (fired by the
// AFTER INSERT trigger on `event`, migration 00034) and forwards new events as
// SSE frames the instant they land -- no polling loop. It queries the event
// table only when a notification for THIS build arrives (plus a rare catch-up on
// the keep-alive tick, in case a notification is ever missed). If a dedicated
// LISTEN connection can't be acquired, it falls back to the old interval poll so
// live updates still work.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

const (
	// livePollInterval is the fallback poll cadence, used only when a LISTEN
	// connection couldn't be acquired.
	livePollInterval = 1 * time.Second
	// liveKeepAlive is how long a LISTEN wait blocks before sending an SSE
	// keep-alive comment (to hold proxies/load balancers open) and doing a
	// safety catch-up query. The browser's EventSource reconnects on a real
	// connection drop; nothing client-side watches for silence, so this only
	// has to be short enough to keep intermediaries from idle-closing.
	liveKeepAlive = 20 * time.Second
)

func (s *Server) handleLiveStatus(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	buildID := build.ID.String()
	var since time.Time

	// sendNew forwards every event since the last one seen as an SSE frame.
	// Returns the count sent and whether the stream should keep going (false on
	// a client-gone / fatal error).
	sendNew := func() (int, bool) {
		events, err := s.Queries.ListEventsByBuildSinceWithObject(ctx, db.ListEventsByBuildSinceWithObjectParams{
			BuildID: build.ID, CreatedAt: pgtype.Timestamptz{Time: since, Valid: true},
		})
		if err != nil {
			if ctx.Err() != nil {
				return 0, false
			}
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return 0, true
		}
		for _, row := range events {
			ev := row.Event
			payload, _ := json.Marshal(makeEventView(ev, row.ObjAsName, row.ObjObjectName, row.TeamNumber))
			// One fixed SSE event name ("laforge-event") for every row, with
			// the real distinguishing kind carried inside the JSON payload's
			// own "kind" field -- EventSource has no wildcard listener, so a
			// single frame name is what lets a client "get everything."
			fmt.Fprintf(w, "event: laforge-event\ndata: %s\n\n", payload)
			since = ev.CreatedAt.Time
		}
		if len(events) > 0 {
			flusher.Flush()
		}
		return len(events), true
	}

	keepAlive := func() {
		// A comment frame keeps intermediate proxies from idle-closing the
		// connection.
		fmt.Fprint(w, ": keep-alive\n\n")
		flusher.Flush()
	}

	// Acquire a dedicated connection to LISTEN on. On any failure, fall back to
	// the interval poll (below) so live updates still work.
	conn, err := s.Pool.Acquire(ctx)
	if err == nil {
		if _, lerr := conn.Exec(ctx, "LISTEN laforge_event"); lerr != nil {
			conn.Release()
		} else {
			defer conn.Release()
			if _, cont := sendNew(); !cont { // initial catch-up
				return
			}
			for {
				wctx, cancel := context.WithTimeout(ctx, liveKeepAlive)
				n, werr := conn.Conn().WaitForNotification(wctx)
				cancel()
				if ctx.Err() != nil {
					return
				}
				if werr != nil {
					if errors.Is(werr, context.DeadlineExceeded) {
						keepAlive()
						if _, cont := sendNew(); !cont { // safety catch-up
							return
						}
						continue
					}
					return // connection error -- let the client reconnect
				}
				if n.Payload == buildID {
					if _, cont := sendNew(); !cont {
						return
					}
				}
			}
		}
	}

	// Fallback: no LISTEN connection available -- poll the event table on the
	// interval and push what's new, exactly as before LISTEN/NOTIFY existed.
	ticker := time.NewTicker(livePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, cont := sendNew()
			if !cont {
				return
			}
			if n == 0 {
				keepAlive()
			}
		}
	}
}
