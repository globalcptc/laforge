package incus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// waitForWaiters blocks until g has n queued waiters, so a test's acquire order
// is its arrival order.
func waitForWaiters(t *testing.T, g *admissionGate, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, waiting := g.stats(); waiting == n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	_, waiting := g.stats()
	t.Fatalf("waiting = %d, want %d", waiting, n)
}

func TestAdmissionGateServesHighestPriorityThenArrival(t *testing.T) {
	g := &admissionGate{}
	ctx := context.Background()
	hold, err := g.acquire(ctx, 1, 0)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var order []string
	var wg sync.WaitGroup
	queue := func(name string, prio, ahead int) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := g.acquire(ctx, 1, prio)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			release()
		}()
		waitForWaiters(t, g, ahead+1)
	}
	// Twenty workstations queue first; the domain controller arrives last.
	queue("ws-1", 0, 0)
	queue("ws-2", 0, 1)
	queue("dc", 12, 2)
	queue("ws-3", 0, 3)

	hold()
	wg.Wait()
	want := []string{"dc", "ws-1", "ws-2", "ws-3"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestAdmissionGateNeverExceedsLimit(t *testing.T) {
	g := &admissionGate{}
	const limit = 3
	var running, peak int64
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(prio int) {
			defer wg.Done()
			release, err := g.acquire(context.Background(), limit, prio)
			if err != nil {
				t.Error(err)
				return
			}
			n := atomic.AddInt64(&running, 1)
			for {
				p := atomic.LoadInt64(&peak)
				if n <= p || atomic.CompareAndSwapInt64(&peak, p, n) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			atomic.AddInt64(&running, -1)
			release()
			release() // a second release is a no-op, not a stolen slot
		}(i % 4)
	}
	wg.Wait()
	if peak > limit {
		t.Fatalf("peak concurrency %d exceeds limit %d", peak, limit)
	}
	if inUse, waiting := g.stats(); inUse != 0 || waiting != 0 {
		t.Fatalf("after all releases: inUse=%d waiting=%d, want 0/0", inUse, waiting)
	}
}

func TestAdmissionGateCancelledWaiterLeavesQueue(t *testing.T) {
	g := &admissionGate{}
	hold, _ := g.acquire(context.Background(), 1, 0)
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := g.acquire(ctx, 1, 5)
		errc <- err
	}()
	waitForWaiters(t, g, 1)
	cancel()
	if err := <-errc; err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, waiting := g.stats(); waiting != 0 {
		t.Fatalf("cancelled waiter still queued (%d)", waiting)
	}
	hold()
	// The slot is free again: a new caller gets it immediately.
	release, err := g.acquire(context.Background(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestAdmitUnboundedAndQueuedPhase(t *testing.T) {
	// No cap: admitted immediately, nothing recorded.
	b := &Builder{Client: &Client{BaseURL: "https://unbounded.test:8443"}}
	ctx, timer := builder.WithPhaseTimer(context.Background())
	release, err := b.admit(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if p := timer.Phases(); len(p) != 0 {
		t.Fatalf("unbounded admit recorded phases %v", p)
	}

	// Capped: two Builders for the SAME host (the runner builds one per task)
	// share one gate, keyed case- and slash-insensitively.
	a := &Builder{Client: &Client{BaseURL: "https://capped.test:8443/"}, Config: Config{MaxConcurrentDeploys: 1}}
	c := &Builder{Client: &Client{BaseURL: "HTTPS://CAPPED.test:8443"}, Config: Config{MaxConcurrentDeploys: 1}}
	holdA, err := a.admit(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, timer = builder.WithPhaseTimer(context.Background())
	got := make(chan struct{})
	go func() {
		release, err := c.admit(ctx, 0)
		if err == nil {
			release()
		}
		close(got)
	}()
	waitForWaiters(t, gateFor(a.admissionKey()), 1)
	time.Sleep(20 * time.Millisecond)
	holdA()
	<-got
	p := timer.Phases()
	if len(p) != 1 || p[0].Name != "queued" || p[0].Duration < 20*time.Millisecond {
		t.Fatalf("phases = %v, want one queued phase >= 20ms", p)
	}
}
