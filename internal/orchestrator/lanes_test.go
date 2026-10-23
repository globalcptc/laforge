package orchestrator

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// laneStore is an in-memory stand-in for the build table plus a recording
// reconcile pass, so the lane scheduler -- the part that failed on prod -- is
// tested on its own, with no Postgres, checkout or builder involved.
type laneStore struct {
	mu          sync.Mutex
	builds      map[string]db.Build
	counts      map[string]int
	inflight    int32
	maxInflight int32
	running     map[string]bool
	// gate, when non-nil, blocks every pass until closed (or the lane's
	// context is cancelled), so slot occupancy can be observed.
	gate chan struct{}
	// onReconcile, when set, runs inside the pass after it is counted -- used
	// to flip a build's status mid-pass the way Teardown/AdvanceBuildStatus do.
	onReconcile func(b db.Build)
}

func newLaneStore() *laneStore {
	return &laneStore{builds: map[string]db.Build{}, counts: map[string]int{}, running: map[string]bool{}}
}

func laneID(n byte) pgtype.UUID {
	var b [16]byte
	b[15] = n
	b[0] = 0xAB
	return pgtype.UUID{Bytes: b, Valid: true}
}

func (s *laneStore) add(n byte, status string) db.Build {
	b := db.Build{ID: laneID(n), Status: status}
	s.mu.Lock()
	s.builds[b.ID.String()] = b
	s.mu.Unlock()
	return b
}

func (s *laneStore) setStatus(n byte, status string) {
	s.mu.Lock()
	b := s.builds[laneID(n).String()]
	b.Status = status
	s.builds[b.ID.String()] = b
	s.mu.Unlock()
}

func (s *laneStore) remove(n byte) {
	s.mu.Lock()
	delete(s.builds, laneID(n).String())
	s.mu.Unlock()
}

// listed mimics ListBuildsByStatus(planned/deploying/building/tearing_down).
func (s *laneStore) listed() []db.Build {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []db.Build
	for _, b := range s.builds {
		switch b.Status {
		case "planned", "deploying", "building", "tearing_down":
			out = append(out, b)
		}
	}
	return out
}

func (s *laneStore) get(_ context.Context, id pgtype.UUID) (db.Build, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.builds[id.String()]
	if !ok {
		return db.Build{}, pgx.ErrNoRows
	}
	return b, nil
}

func (s *laneStore) reconcile(ctx context.Context, b db.Build) {
	s.mu.Lock()
	s.counts[b.ID.String()]++
	s.running[b.ID.String()] = true
	s.inflight++
	if s.inflight > s.maxInflight {
		s.maxInflight = s.inflight
	}
	gate := s.gate
	hook := s.onReconcile
	s.mu.Unlock()
	if hook != nil {
		hook(b)
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
		}
	}
	s.mu.Lock()
	s.inflight--
	delete(s.running, b.ID.String())
	s.mu.Unlock()
}

func (s *laneStore) count(n byte) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[laneID(n).String()]
}

func (s *laneStore) inFlight() (n int, ids map[string]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids = map[string]bool{}
	for k := range s.running {
		ids[k] = true
	}
	return int(s.inflight), ids
}

// waitFor polls cond for up to 2s.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestLanesBoundConcurrencyAndReservePlannedShare is the prod failure in
// miniature: many planned builds must never occupy more than half the slots,
// and live builds arriving later must get the reserved half immediately, ahead
// of every queued planned lane.
func TestLanesBoundConcurrencyAndReservePlannedShare(t *testing.T) {
	store := newLaneStore()
	store.gate = make(chan struct{})
	for i := byte(1); i <= 10; i++ {
		store.add(i, "planned")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 4, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed())

	waitFor(t, "two planned passes in flight", func() bool { n, _ := store.inFlight(); return n == 2 })
	time.Sleep(30 * time.Millisecond)
	if n, _ := store.inFlight(); n != 2 {
		t.Fatalf("planned lanes hold %d slots, want at most half of 4 = 2", n)
	}

	// Two live builds show up: they take the two reserved slots at once, even
	// though eight planned lanes are already queued for a slot.
	store.add(11, "deploying")
	store.add(12, "tearing_down")
	lanes.Supervise(ctx, store.listed())
	waitFor(t, "live builds to get the reserved slots", func() bool {
		n, ids := store.inFlight()
		return n == 4 && ids[laneID(11).String()] && ids[laneID(12).String()]
	})
	store.mu.Lock()
	max := store.maxInflight
	store.mu.Unlock()
	if max > 4 {
		t.Fatalf("max in-flight passes = %d, want <= Concurrency 4", max)
	}

	// Open the gate: everything drains; every planned build got exactly one
	// pass (PlannedInterval is an hour), the live ones keep cycling.
	close(store.gate)
	store.mu.Lock()
	store.gate = nil
	store.mu.Unlock()
	waitFor(t, "every planned build to reconcile once", func() bool {
		for i := byte(1); i <= 10; i++ {
			if store.count(i) < 1 {
				return false
			}
		}
		return true
	})
	waitFor(t, "live builds to cycle", func() bool { return store.count(11) >= 3 && store.count(12) >= 3 })
	time.Sleep(30 * time.Millisecond)
	for i := byte(1); i <= 10; i++ {
		if c := store.count(i); c != 1 {
			t.Fatalf("planned build %d reconciled %d times, want exactly 1 on a 1h planned cadence", i, c)
		}
	}
	cancel()
	lanes.Wait()
}

// TestLanesPlannedCadenceIsSlow: a planned build reconciles once, then not
// again until PlannedInterval, while a live build on the same scheduler
// reconciles every Interval.
func TestLanesPlannedCadenceIsSlow(t *testing.T) {
	store := newLaneStore()
	store.add(1, "planned")
	store.add(2, "deploying")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 8, Interval: 5 * time.Millisecond, PlannedInterval: 150 * time.Millisecond}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed())

	waitFor(t, "first passes", func() bool { return store.count(1) == 1 && store.count(2) >= 1 })
	time.Sleep(60 * time.Millisecond)
	if c := store.count(1); c != 1 {
		t.Fatalf("planned build reconciled %d times within the planned interval, want 1", c)
	}
	if c := store.count(2); c < 5 {
		t.Fatalf("live build reconciled only %d times in ~60ms at a 5ms cadence", c)
	}
	waitFor(t, "planned build's second pass after PlannedInterval", func() bool { return store.count(1) == 2 })
	cancel()
	lanes.Wait()
}

// TestLaneStopsOnTerminalStatus: a lane exits by itself once its build is
// terminal -- here the pass itself moves the build to torn_down, as Teardown
// does -- without waiting for the supervisor to reap it.
func TestLaneStopsOnTerminalStatus(t *testing.T) {
	store := newLaneStore()
	store.add(1, "tearing_down")
	store.onReconcile = func(b db.Build) { store.setStatus(1, "torn_down") }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 2, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed()) // the only Supervise call: no reaping from here on

	waitFor(t, "lane to exit on its own", func() bool { return lanes.Running() == 0 })
	time.Sleep(30 * time.Millisecond)
	if c := store.count(1); c != 1 {
		t.Fatalf("torn-down build reconciled %d times, want exactly 1 (the pass that tore it down)", c)
	}
	lanes.Wait()

	// A discarded planned build (the new terminal state) is never reconciled.
	store2 := newLaneStore()
	store2.add(1, "planned")
	lanes2 := NewLanes(LaneConfig{Concurrency: 2, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store2.get, store2.reconcile)
	store2.setStatus(1, "discarded") // discarded between the listing and the lane's first load
	lanes2.Supervise(ctx, []db.Build{{ID: laneID(1), Status: "planned"}})
	waitFor(t, "discarded lane to exit", func() bool { return lanes2.Running() == 0 })
	if c := store2.count(1); c != 0 {
		t.Fatalf("discarded build reconciled %d times, want 0", c)
	}
}

// TestLaneExitsWhenBuildDeleted: a force-deleted build (GetBuild -> no rows)
// ends its lane.
func TestLaneExitsWhenBuildDeleted(t *testing.T) {
	store := newLaneStore()
	store.add(1, "deploying")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 2, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed())
	waitFor(t, "first pass", func() bool { return store.count(1) >= 1 })
	store.remove(1)
	waitFor(t, "lane to exit after delete", func() bool { return lanes.Running() == 0 })
}

// TestLaneDeployedPlannedBuildReconcilesAtOnce: clicking Deploy on a planned
// build must not wait out the rest of a planned-cadence sleep -- the
// supervisor's next listing flips the lane to live and kicks it.
func TestLaneDeployedPlannedBuildReconcilesAtOnce(t *testing.T) {
	store := newLaneStore()
	store.add(1, "planned")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 2, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed())
	waitFor(t, "planned first pass", func() bool { return store.count(1) == 1 })
	time.Sleep(20 * time.Millisecond)
	if c := store.count(1); c != 1 {
		t.Fatalf("planned build reconciled %d times before deploy, want 1", c)
	}
	store.setStatus(1, "deploying")
	lanes.Supervise(ctx, store.listed())
	waitFor(t, "deployed build to reconcile on the live cadence", func() bool { return store.count(1) >= 4 })
	cancel()
	lanes.Wait()
}

// TestLanesSupervisorReapsUnlistedLane: a build that leaves the listing has
// its lane cancelled, even mid-pass.
func TestLanesSupervisorReapsUnlistedLane(t *testing.T) {
	store := newLaneStore()
	store.gate = make(chan struct{})
	store.add(1, "deploying")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lanes := NewLanes(LaneConfig{Concurrency: 2, Interval: 5 * time.Millisecond, PlannedInterval: time.Hour}, store.get, store.reconcile)
	lanes.Supervise(ctx, store.listed())
	waitFor(t, "pass in flight", func() bool { n, _ := store.inFlight(); return n == 1 })
	store.setStatus(1, "finished")
	lanes.Supervise(ctx, store.listed()) // empty listing: reap
	waitFor(t, "lane to be reaped", func() bool { return lanes.Running() == 0 })
	if c := store.count(1); c != 1 {
		t.Fatalf("reaped build reconciled %d times, want 1", c)
	}
}

// TestPrioritySem pins the semaphore's contract directly: low-priority
// acquirers never take the reserved slots, a release goes to a waiting
// high-priority acquirer first, and a cancelled waiter leaves the count intact.
func TestPrioritySem(t *testing.T) {
	s := newPrioritySem(3, 1)
	ctx := context.Background()
	if err := s.Acquire(ctx, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Acquire(ctx, false); err != nil {
		t.Fatal(err)
	}
	// One slot left: the reserve. A low acquirer must wait; a high one gets it.
	lowGot := make(chan struct{})
	go func() { _ = s.Acquire(ctx, false); close(lowGot) }()
	select {
	case <-lowGot:
		t.Fatal("low-priority acquirer took the reserved slot")
	case <-time.After(20 * time.Millisecond):
	}
	if err := s.Acquire(ctx, true); err != nil {
		t.Fatal(err)
	}
	// Full. A high waiter queued now is served before the queued low waiter.
	highGot := make(chan struct{})
	go func() { _ = s.Acquire(ctx, true); close(highGot) }()
	time.Sleep(10 * time.Millisecond)
	s.Release()
	select {
	case <-highGot:
	case <-time.After(time.Second):
		t.Fatal("release did not go to the high-priority waiter")
	}
	select {
	case <-lowGot:
		t.Fatal("low-priority waiter was served while the reserve was not free")
	case <-time.After(20 * time.Millisecond):
	}
	// Two releases: the low waiter needs free > reserve, so the first release
	// (free=1) is not enough and the second (free=2) is.
	s.Release()
	select {
	case <-lowGot:
		t.Fatal("low-priority waiter served with only the reserve free")
	case <-time.After(20 * time.Millisecond):
	}
	s.Release()
	select {
	case <-lowGot:
	case <-time.After(time.Second):
		t.Fatal("low-priority waiter never served")
	}

	// Cancellation: a waiter that gives up does not leak or steal a slot.
	full := newPrioritySem(1, 0)
	if err := full.Acquire(ctx, true); err != nil {
		t.Fatal(err)
	}
	cctx, ccancel := context.WithCancel(ctx)
	var done int32
	go func() {
		if err := full.Acquire(cctx, true); err == nil {
			t.Error("cancelled acquire returned nil")
		}
		atomic.StoreInt32(&done, 1)
	}()
	time.Sleep(10 * time.Millisecond)
	ccancel()
	waitFor(t, "cancelled waiter to return", func() bool { return atomic.LoadInt32(&done) == 1 })
	full.Release()
	if full.free != 1 || len(full.high) != 0 {
		t.Fatalf("after cancel + release: free=%d highWaiters=%d, want 1 and 0", full.free, len(full.high))
	}
}
