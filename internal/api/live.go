// handleLiveStatus is "Build → Logs: Live over SSE while anything is running"
// and the plan's "Performance: Stream, do not poll, for live state." Each
// stream is fed from its build's shared feed (livehub.go): the API reads a
// build's new events once, when Postgres's 'laforge_event' notification
// (migration 00034's AFTER INSERT trigger) says there are some, and hands the
// same rendered frames to every stream on that build. A stream holds no
// database connection and runs no query of its own -- so the number of open
// streams is limited only by what the server can hold open, not by the
// database.
package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

const (
	// livePollInterval is the fallback poll cadence, used only while the
	// shared LISTEN connection is down.
	livePollInterval = 1 * time.Second
	// liveKeepAlive is how long a quiet stream waits before sending an SSE
	// keep-alive comment (to hold proxies/load balancers open), and how often
	// a feed re-reads as a safety net in case a notification was missed. The browser's EventSource reconnects on a real
	// connection drop; nothing client-side watches for silence, so this only
	// has to be short enough to keep intermediaries from idle-closing.
	liveKeepAlive = 20 * time.Second
)

func (s *Server) handleLiveStatus(w http.ResponseWriter, r *http.Request) {
	buildID, repo, err := s.buildRepository(r)
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
	// Where the stream starts: `?since=<RFC 3339>` -- the newest event the
	// client already has from GET /events -- or now. It used to start at
	// time zero, which replayed the whole journal through the stream in 500-
	// event batches; the page appended those as if they were new, so old
	// rows kept arriving at the "newest" end for minutes.
	since := time.Now().Add(-5 * time.Second)
	v := r.URL.Query().Get("since")
	// A reconnecting EventSource sends back the id of the last frame it got
	// (each frame's id is its event's time): resume there, not from the
	// original ?since, which would replay everything in between.
	if last := r.Header.Get("Last-Event-ID"); last != "" {
		v = last
	}
	if v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("since must be an RFC 3339 time"))
			return
		}
		since = t
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	// The stream is fed from the build's shared feed (livehub.go): it holds
	// no database connection and runs no query of its own, except reading
	// the gap once when it starts from further back than the feed holds.
	sub, catchUp, needDB, unsubscribe := s.live.subscribe(buildID, since)
	defer unsubscribe()

	seen := map[pgtype.UUID]struct{}{} // ids sent during catch-up, which live frames may repeat
	if needDB {
		rows, err := s.Queries.ListEventsByBuildSinceWithObject(ctx, db.ListEventsByBuildSinceWithObjectParams{
			BuildID: buildID, CreatedAt: pgtype.Timestamptz{Time: since, Valid: true},
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		}
		for _, row := range rows {
			fr := renderLiveFrame(row)
			seen[fr.id] = struct{}{}
			w.Write(fr.data)
		}
	}
	for _, fr := range catchUp {
		if _, dup := seen[fr.id]; dup {
			continue
		}
		seen[fr.id] = struct{}{}
		w.Write(fr.data)
	}
	flusher.Flush()

	keepAlive := time.NewTicker(liveKeepAlive)
	defer keepAlive.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.gone:
			return // fell too far behind; the browser reconnects and resumes
		case fr := <-sub.frames:
			if len(seen) > 0 {
				if _, dup := seen[fr.id]; dup {
					continue
				}
			}
			if _, err := w.Write(fr.data); err != nil {
				return
			}
			// Send whatever else is already waiting in the same flush.
			for more := true; more; {
				select {
				case next := <-sub.frames:
					if _, dup := seen[next.id]; !dup {
						w.Write(next.data)
					}
				default:
					more = false
				}
			}
			flusher.Flush()
		case <-keepAlive.C:
			// A comment frame keeps intermediate proxies from idle-closing
			// the connection.
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
			seen = nil // catch-up duplicates can't arrive this late
		}
	}
}
