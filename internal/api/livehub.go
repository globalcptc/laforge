package api

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// liveHub serves every open live stream from memory, so the number of people
// watching costs the database nothing.
//
//   - One LISTEN on 'laforge_event', on a dedicated connection outside the
//     pool, hears that a build has new events.
//   - Each build being watched has one feed. On a notification the feed reads
//     the new events once, renders each frame once, keeps the recent ones in
//     a ring, and hands the frames to every stream watching that build.
//   - A stream that connects (or reconnects) inside the ring's window is
//     caught up from the ring with no query at all; only one starting from
//     further back reads the gap from the database, once.
//   - A stream that can't keep up is disconnected rather than buffered without
//     bound; the browser reconnects and resumes from its Last-Event-ID.
//
// Database work therefore scales with how many builds have new events, not
// with how many tabs are open on them.
type liveHub struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	// changed is told when a build has new events (drops its shared reads).
	changed func(buildID string)

	mu        sync.Mutex
	feeds     map[string]*buildFeed
	connected bool
	started   bool
}

const (
	// liveRingSize is how many recent frames a feed keeps for catching up
	// streams that connect or reconnect.
	liveRingSize = 2000
	// liveSubBuffer is how many frames may wait for one slow stream before
	// it's disconnected (it reconnects and resumes from the ring).
	liveSubBuffer = 1024
	// liveOverlap re-reads this far behind the newest event seen: an event's
	// created_at is when its transaction started, so one committed a moment
	// late can carry a time just before events already sent. Ids already
	// sent are skipped.
	liveOverlap = 5 * time.Second
	// liveFeedIdle is how long a feed with no streams is kept (so a page
	// reload resumes from its ring) before it's dropped.
	liveFeedIdle = 2 * time.Minute
)

type liveFrame struct {
	id   pgtype.UUID
	at   time.Time
	data []byte // the complete SSE frame, rendered once for every stream
}

type liveSub struct {
	frames chan liveFrame
	gone   chan struct{} // closed when the feed drops a stream that fell behind
	once   sync.Once
}

func (s *liveSub) drop() { s.once.Do(func() { close(s.gone) }) }

type buildFeed struct {
	hub     *liveHub
	buildID pgtype.UUID
	key     string
	wake    chan struct{}

	mu          sync.Mutex
	subs        map[*liveSub]struct{}
	ring        []liveFrame // ascending by at
	coveredFrom time.Time   // the ring holds every event after this
	cursor      time.Time   // newest event read
	sent        map[pgtype.UUID]time.Time
	idleSince   time.Time
}

func newLiveHub(pool *pgxpool.Pool, q *db.Queries) *liveHub {
	return &liveHub{pool: pool, queries: q, feeds: map[string]*buildFeed{}}
}

// listening reports whether notifications are flowing; while they aren't,
// feeds poll every livePollInterval instead.
func (h *liveHub) listening() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.connected
}

func (h *liveHub) setConnected(v bool) {
	h.mu.Lock()
	h.connected = v
	h.mu.Unlock()
}

// subscribe joins a build's feed. It returns the frames after `since` that
// the feed already holds; when `since` is older than the ring reaches,
// needDB is true and the caller reads events after `since` from the database
// itself (skipping ids it then sees again in the returned or live frames).
func (h *liveHub) subscribe(buildID pgtype.UUID, since time.Time) (sub *liveSub, catchUp []liveFrame, needDB bool, cancel func()) {
	key := buildID.String()
	h.mu.Lock()
	if !h.started && h.pool != nil {
		h.started = true
		go h.listen()
	}
	f := h.feeds[key]
	if f == nil {
		now := time.Now()
		f = &buildFeed{hub: h, buildID: buildID, key: key, wake: make(chan struct{}, 1),
			subs: map[*liveSub]struct{}{}, coveredFrom: now, cursor: now.Add(-liveOverlap), sent: map[pgtype.UUID]time.Time{}}
		h.feeds[key] = f
		go f.run()
	}
	sub = &liveSub{frames: make(chan liveFrame, liveSubBuffer), gone: make(chan struct{})}
	f.mu.Lock()
	f.subs[sub] = struct{}{}
	needDB = since.Before(f.coveredFrom)
	for _, fr := range f.ring {
		if fr.at.After(since) {
			catchUp = append(catchUp, fr)
		}
	}
	f.mu.Unlock()
	h.mu.Unlock()
	return sub, catchUp, needDB, func() {
		f.mu.Lock()
		delete(f.subs, sub)
		if len(f.subs) == 0 {
			f.idleSince = time.Now()
		}
		f.mu.Unlock()
	}
}

func (h *liveHub) notify(buildID string) {
	h.mu.Lock()
	f := h.feeds[buildID]
	h.mu.Unlock()
	if f != nil {
		select {
		case f.wake <- struct{}{}:
		default: // a read is already due; it picks up everything new
		}
	}
}

// listen keeps the LISTEN connection up for the life of the process,
// reconnecting with a backoff when it drops.
func (h *liveHub) listen() {
	backoff := time.Second
	for {
		err := h.listenOnce()
		h.setConnected(false)
		log.Printf("api: live updates: listener connection lost (%v); feeds poll every %s until it's back, reconnecting in %s", err, livePollInterval, backoff)
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (h *liveHub) listenOnce() error {
	ctx := context.Background()
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	conn, err := pgx.ConnectConfig(connectCtx, h.pool.Config().ConnConfig.Copy())
	cancel()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(ctx, "LISTEN laforge_event"); err != nil {
		return err
	}
	h.setConnected(true)
	log.Printf("api: live updates: listening for new events on one shared connection")
	// Catch up anything that landed while the listener was down.
	h.mu.Lock()
	for key := range h.feeds {
		go h.notify(key)
	}
	h.mu.Unlock()
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		h.notify(n.Payload)
	}
}

// run is a feed's reader: it wakes on a notification (or, as a safety net,
// every liveKeepAlive -- every livePollInterval while the listener is down),
// reads what's new once, and fans it out. It exits once the feed has had no
// streams for liveFeedIdle.
func (f *buildFeed) run() {
	ticker := time.NewTicker(livePollInterval)
	defer ticker.Stop()
	lastRead := time.Time{}
	failures := 0
	for {
		select {
		case <-f.wake:
		case <-ticker.C:
			if f.retire() {
				return
			}
			if f.hub.listening() && time.Since(lastRead) < liveKeepAlive {
				continue
			}
		}
		if err := f.read(); err != nil {
			failures++
			if failures == 1 || failures%60 == 0 {
				log.Printf("api: live updates: reading new events for build %s failed (%d in a row): %v", f.key, failures, err)
			}
			continue
		}
		failures = 0
		lastRead = time.Now()
	}
}

// retire drops the feed once it's been without streams long enough.
func (f *buildFeed) retire() bool {
	f.hub.mu.Lock()
	defer f.hub.mu.Unlock()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.subs) > 0 || f.idleSince.IsZero() || time.Since(f.idleSince) < liveFeedIdle {
		return false
	}
	delete(f.hub.feeds, f.key)
	return true
}

func (f *buildFeed) read() error {
	f.mu.Lock()
	from := f.cursor.Add(-liveOverlap)
	f.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rows, err := f.hub.queries.ListEventsByBuildSinceWithObject(ctx, db.ListEventsByBuildSinceWithObjectParams{
		BuildID: f.buildID, CreatedAt: pgtype.Timestamptz{Time: from, Valid: true},
	})
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var fresh []liveFrame
	for _, row := range rows {
		if _, done := f.sent[row.Event.ID]; done {
			continue
		}
		fr := renderLiveFrame(row)
		f.sent[fr.id] = fr.at
		fresh = append(fresh, fr)
		if fr.at.After(f.cursor) {
			f.cursor = fr.at
		}
	}
	for id, at := range f.sent {
		if at.Before(f.cursor.Add(-2 * liveOverlap)) {
			delete(f.sent, id)
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	if f.hub.changed != nil {
		f.hub.changed(f.key)
	}
	f.ring = append(f.ring, fresh...)
	sort.SliceStable(f.ring, func(i, j int) bool { return f.ring[i].at.Before(f.ring[j].at) })
	if over := len(f.ring) - liveRingSize; over > 0 {
		f.coveredFrom = f.ring[over-1].at
		f.ring = append([]liveFrame(nil), f.ring[over:]...)
	}
	for sub := range f.subs {
		for _, fr := range fresh {
			select {
			case sub.frames <- fr:
			default:
				// Too far behind: disconnect it rather than hold frames for it
				// without bound. The browser reconnects with its Last-Event-ID
				// and is caught up from the ring.
				delete(f.subs, sub)
				sub.drop()
			}
			if _, still := f.subs[sub]; !still {
				break
			}
		}
	}
	if len(f.subs) == 0 && f.idleSince.IsZero() {
		f.idleSince = time.Now()
	}
	return nil
}

// renderLiveFrame renders one event as a complete SSE frame. The frame's id
// is the event's time, which the browser sends back as Last-Event-ID when it
// reconnects, so a dropped stream resumes where it left off.
func renderLiveFrame(row db.ListEventsByBuildSinceWithObjectRow) liveFrame {
	ev := row.Event
	payload, _ := json.Marshal(makeEventView(ev, row.ObjAsName, row.ObjObjectName, row.TeamNumber))
	// One fixed SSE event name ("laforge-event") for every row, with the
	// distinguishing kind inside the JSON payload's own "kind" field --
	// EventSource has no wildcard listener, so a single frame name is what
	// lets a client "get everything."
	data := make([]byte, 0, len(payload)+96)
	data = append(data, "id: "...)
	data = append(data, ev.CreatedAt.Time.UTC().Format(time.RFC3339Nano)...)
	data = append(data, "\nevent: laforge-event\ndata: "...)
	data = append(data, payload...)
	data = append(data, "\n\n"...)
	return liveFrame{id: ev.ID, at: ev.CreatedAt.Time, data: data}
}
