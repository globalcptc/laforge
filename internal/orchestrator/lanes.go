package orchestrator

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// Per-build reconcile lanes, bounded.
//
// cmd/laforge-orchestrator runs one lane per non-terminal build so a slow build
// only ever delays its OWN next pass. Unbounded, that model fell over on a
// 2-core box: 153 stale planned builds plus a few teardowns each got a lane
// doing a checkout + content load + per-object DB work every 2 seconds, the
// process sat above 300% CPU, and the one live 7-team build's pass could no
// longer finish inside its own RECONCILE_BUILD_TIMEOUT -- it restarted from
// team 1 every pass and never got past team two. Lanes keep their independence
// here, but every pass now takes a slot from one shared, priority-aware
// semaphore, and a planned build (nothing at a hoster, nothing to drive) only
// takes a pass on a much slower cadence than a build that is deploying,
// building or tearing down.
//
// This file is deliberately free of the per-build pass itself (checkout, hoster
// calls) and of flag parsing: it takes the pass as a callback and the build
// list from the supervisor, so the scheduling -- the part that actually
// failed -- is unit-testable without Postgres or a builder.

// LaneConfig tunes the lane scheduler.
type LaneConfig struct {
	// Concurrency caps how many reconcile passes may run at once across every
	// lane. Planned builds may use at most half of these slots (see
	// prioritySem's reserve), so a live build always finds a slot promptly.
	// Values < 1 mean 1.
	Concurrency int
	// Interval is the cadence of a live build's passes (deploying / building /
	// tearing_down), and how often a lane re-checks its build's status.
	Interval time.Duration
	// PlannedInterval is the cadence of a planned build's passes. A planned
	// build's content is immutable and it touches no hoster, so its pass only
	// re-derives teams/fingerprints and the builder-compatibility verdict; once
	// a minute is plenty for that to track a builder config change. A planned
	// build that is deployed is noticed within one Interval regardless (the
	// supervisor's next listing flips the lane to live priority at once).
	PlannedInterval time.Duration
}

// GetBuildFunc loads a build's current row; it must return pgx.ErrNoRows (or
// any error) for a build that no longer exists.
type GetBuildFunc func(ctx context.Context, id pgtype.UUID) (db.Build, error)

// ReconcileFunc is one full reconcile pass for one build (the orchestrator's
// reconcileOne). It is handed the freshly loaded row and a context that is
// cancelled when the lane is reaped; it should derive its own per-pass
// deadline from that context.
type ReconcileFunc func(ctx context.Context, b db.Build)

// Lanes owns one reconcile lane per non-terminal build and the shared slot
// budget they draw from. Supervise is the only entry point: call it with the
// current list of non-terminal builds on every tick.
type Lanes struct {
	cfg       LaneConfig
	get       GetBuildFunc
	reconcile ReconcileFunc
	sem       *prioritySem

	mu    sync.Mutex
	lanes map[string]*lane
	wg    sync.WaitGroup
}

type lane struct {
	cancel context.CancelFunc
	mu     sync.Mutex
	latest db.Build
	// kick wakes the lane out of its between-pass sleep when the supervisor
	// sees its status change (a planned build that was just deployed must not
	// wait out the rest of a planned-cadence sleep). Buffered so a kick never
	// blocks the supervisor.
	kick chan struct{}
}

func (ln *lane) snapshot() db.Build {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	return ln.latest
}

func (ln *lane) update(b db.Build) {
	ln.mu.Lock()
	changed := ln.latest.Status != b.Status
	ln.latest = b
	ln.mu.Unlock()
	if changed {
		select {
		case ln.kick <- struct{}{}:
		default:
		}
	}
}

// NewLanes builds a scheduler. get loads a build; reconcile is the pass.
func NewLanes(cfg LaneConfig, get GetBuildFunc, reconcile ReconcileFunc) *Lanes {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.PlannedInterval < cfg.Interval {
		cfg.PlannedInterval = cfg.Interval
	}
	return &Lanes{
		cfg:       cfg,
		get:       get,
		reconcile: reconcile,
		sem:       newPrioritySem(cfg.Concurrency, cfg.Concurrency/2),
		lanes:     map[string]*lane{},
	}
}

// IsTerminalBuildStatus reports whether a build in this status will never be
// reconciled again: finished/failed (the deploy settled), torn_down/purged (the
// infrastructure is gone) and discarded (a planned build nobody will deploy).
// A lane exits as soon as it loads its build in one of these, instead of
// running passes until the supervisor's next listing happens to omit it.
func IsTerminalBuildStatus(status string) bool {
	switch status {
	case "finished", "failed", "torn_down", "purged", "discarded":
		return true
	}
	return false
}

// isLiveStatus: a build that is driving real infrastructure right now and so
// reconciles at the fast cadence with priority. Everything else the supervisor
// lists is 'planned'.
func isLiveStatus(status string) bool {
	return status != "planned"
}

// Supervise reconciles the set of running lanes with builds -- the current
// non-terminal builds (planned/deploying/building/tearing_down): a build with
// no lane gets one and reconciles immediately; a build whose lane exists has
// the lane's view of it refreshed (its status may have flipped); a lane whose
// build is no longer listed (terminal, or deleted) is cancelled. ctx is the
// parent of every lane's context.
func (l *Lanes) Supervise(ctx context.Context, builds []db.Build) {
	active := make(map[string]bool, len(builds))
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, b := range builds {
		key := b.ID.String()
		active[key] = true
		if ln, running := l.lanes[key]; running {
			ln.update(b)
			continue
		}
		lctx, cancel := context.WithCancel(ctx)
		ln := &lane{cancel: cancel, latest: b, kick: make(chan struct{}, 1)}
		l.lanes[key] = ln
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.run(lctx, ln, b.ID)
			cancel()
			l.mu.Lock()
			if l.lanes[key] == ln { // never delete a newer lane for the same build
				delete(l.lanes, key)
			}
			l.mu.Unlock()
		}()
	}
	for key, ln := range l.lanes {
		if !active[key] {
			ln.cancel()
		}
	}
}

// Running reports how many lanes currently exist (for tests and diagnostics).
func (l *Lanes) Running() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.lanes)
}

// Wait blocks until every lane goroutine has exited -- after the parent
// context is cancelled, or after Supervise with an empty list.
func (l *Lanes) Wait() { l.wg.Wait() }

// run is one lane. It loops until its context is cancelled, its build vanishes,
// or its build is loaded in a terminal status. Each iteration decides whether
// a pass is DUE -- always for a live build; for a planned one only when
// PlannedInterval has elapsed since its last pass (its first pass runs at
// once, so a fresh build is inspectable immediately) -- then takes a slot,
// re-reads the row (the status may have moved while it waited), runs the pass
// and releases the slot. Between iterations it sleeps one Interval or until
// the supervisor kicks it.
func (l *Lanes) run(ctx context.Context, ln *lane, id pgtype.UUID) {
	var last time.Time
	for {
		b := ln.snapshot()
		live := isLiveStatus(b.Status)
		if live || last.IsZero() || time.Since(last) >= l.cfg.PlannedInterval {
			if err := l.sem.Acquire(ctx, live); err != nil {
				return
			}
			fresh, err := l.get(ctx, id)
			if err != nil {
				l.sem.Release()
				if errors.Is(err, pgx.ErrNoRows) || ctx.Err() != nil {
					return // deleted (force-removed), or reaped while waiting
				}
				// A transient DB error: keep the lane, try again next tick.
				goto sleep
			}
			if IsTerminalBuildStatus(fresh.Status) {
				l.sem.Release()
				return
			}
			ln.update(fresh)
			l.reconcile(ctx, fresh)
			l.sem.Release()
			last = time.Now()
		}
	sleep:
		select {
		case <-ctx.Done():
			return
		case <-ln.kick:
		case <-time.After(l.cfg.Interval):
		}
	}
}

// prioritySem is a counting semaphore with two priority classes. A release
// always hands the slot to a waiting high-priority acquirer first, and
// low-priority acquirers may never take the last `reserve` slots -- so with
// 8 slots and a reserve of 4, planned builds share at most 4 slots among
// themselves and a live build never queues behind more than one in-flight
// pass for a slot. Waiters within a class are served FIFO. Acquire honours
// context cancellation, including the race where the slot is granted just as
// the waiter gives up (the slot is handed back).
type prioritySem struct {
	mu      sync.Mutex
	free    int
	reserve int
	high    []chan struct{}
	low     []chan struct{}
}

func newPrioritySem(n, reserve int) *prioritySem {
	if n < 1 {
		n = 1
	}
	if reserve < 0 {
		reserve = 0
	}
	if reserve >= n {
		reserve = n - 1
	}
	return &prioritySem{free: n, reserve: reserve}
}

func (s *prioritySem) Acquire(ctx context.Context, high bool) error {
	s.mu.Lock()
	if high && s.free > 0 {
		// free > 0 implies no high waiter is queued (dispatch would have served
		// it), so taking the slot here keeps FIFO within the class.
		s.free--
		s.mu.Unlock()
		return nil
	}
	if !high && s.free > s.reserve && len(s.low) == 0 {
		s.free--
		s.mu.Unlock()
		return nil
	}
	ch := make(chan struct{}, 1)
	if high {
		s.high = append(s.high, ch)
	} else {
		s.low = append(s.low, ch)
	}
	s.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		s.mu.Lock()
		if s.remove(ch) {
			s.mu.Unlock()
			return ctx.Err()
		}
		// Granted concurrently: the slot is ours; give it back.
		s.mu.Unlock()
		s.Release()
		return ctx.Err()
	}
}

// remove drops ch from whichever queue holds it; false when it was already
// granted (dispatched and removed). Caller holds s.mu.
func (s *prioritySem) remove(ch chan struct{}) bool {
	for i, w := range s.high {
		if w == ch {
			s.high = append(s.high[:i], s.high[i+1:]...)
			return true
		}
	}
	for i, w := range s.low {
		if w == ch {
			s.low = append(s.low[:i], s.low[i+1:]...)
			return true
		}
	}
	return false
}

func (s *prioritySem) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.free++
	for s.free > 0 && len(s.high) > 0 {
		ch := s.high[0]
		s.high = s.high[1:]
		s.free--
		ch <- struct{}{}
	}
	for s.free > s.reserve && len(s.low) > 0 {
		ch := s.low[0]
		s.low = s.low[1:]
		s.free--
		ch <- struct{}{}
	}
}
