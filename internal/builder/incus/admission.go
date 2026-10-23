package incus

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// Per-Incus-host deploy admission. Every team's ~20 instances landing on one
// Incus host at the same moment (one team per host, an unbounded runner pool)
// made each create -- image unpack/copy, config-drive import, first boot --
// crawl: a domain controller that deploys in 30 seconds alone took up to 28
// minutes in a 7-team build, and every Windows host in its team waited on it.
// Config.MaxConcurrentDeploys bounds how many instance deploys run on one host
// at once; the rest queue here and are admitted highest-priority first
// (HostSpec.Priority: the hosts the rest of the team depends_on), then in
// arrival order.
//
// The gate is process-wide and keyed by the host's API URL, not held on the
// Builder: the runner resolves a fresh Builder for every task, so a per-Builder
// semaphore would bound nothing. Two builder configs pointing at the same Incus
// host share its gate, which is what the hardware wants.

var (
	admissionMu    sync.Mutex
	admissionGates = map[string]*admissionGate{}
)

// gateFor returns the shared admission gate for one Incus host.
func gateFor(key string) *admissionGate {
	admissionMu.Lock()
	defer admissionMu.Unlock()
	g := admissionGates[key]
	if g == nil {
		g = &admissionGate{}
		admissionGates[key] = g
	}
	return g
}

// admissionKey identifies the Incus host a Builder deploys to.
func (b *Builder) admissionKey() string {
	if b.Client == nil {
		return ""
	}
	return strings.TrimRight(strings.ToLower(b.Client.BaseURL), "/")
}

// admit waits for a deploy slot on this builder's Incus host and returns the
// func that frees it. With no cap configured it returns immediately. The time
// spent queued is recorded as the "queued" deploy phase, so a slow deploy shows
// how much of it was waiting for the host versus working on it.
func (b *Builder) admit(ctx context.Context, priority int) (release func(), err error) {
	limit := b.Config.MaxConcurrentDeploys
	if limit <= 0 {
		return func() {}, nil
	}
	start := time.Now()
	release, err = gateFor(b.admissionKey()).acquire(ctx, limit, priority)
	builder.RecordPhase(ctx, "queued", time.Since(start))
	return release, err
}

// admissionGate is a counting semaphore whose waiters are served by priority
// (highest first), then arrival order -- unlike a channel semaphore, whose
// wake-up order is unspecified, so a domain controller queued behind twenty
// workstations would not reliably go next.
type admissionGate struct {
	mu      sync.Mutex
	inUse   int
	limit   int // the most recent caller's limit; a config edit takes effect on the next acquire/release
	seq     uint64
	waiters []*admissionWaiter
}

type admissionWaiter struct {
	priority int
	seq      uint64
	ready    chan struct{} // closed once the slot is granted
}

func (g *admissionGate) acquire(ctx context.Context, limit, priority int) (func(), error) {
	g.mu.Lock()
	g.limit = limit
	g.seq++
	w := &admissionWaiter{priority: priority, seq: g.seq, ready: make(chan struct{})}
	g.waiters = append(g.waiters, w)
	g.dispatchLocked()
	g.mu.Unlock()

	select {
	case <-w.ready:
		return g.releaseFunc(), nil
	case <-ctx.Done():
		g.mu.Lock()
		defer g.mu.Unlock()
		select {
		case <-w.ready:
			// Granted while we were cancelling: hand the slot straight back.
			g.inUse--
			g.dispatchLocked()
		default:
			g.removeLocked(w)
		}
		return nil, ctx.Err()
	}
}

// releaseFunc frees one slot, at most once however many times it is called.
func (g *admissionGate) releaseFunc() func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			g.inUse--
			g.dispatchLocked()
			g.mu.Unlock()
		})
	}
}

// dispatchLocked grants free slots to the best waiters.
func (g *admissionGate) dispatchLocked() {
	for g.inUse < g.limit && len(g.waiters) > 0 {
		best := 0
		for i, w := range g.waiters[1:] {
			b := g.waiters[best]
			if w.priority > b.priority || (w.priority == b.priority && w.seq < b.seq) {
				best = i + 1
			}
		}
		w := g.waiters[best]
		g.waiters = append(g.waiters[:best], g.waiters[best+1:]...)
		g.inUse++
		close(w.ready)
	}
}

func (g *admissionGate) removeLocked(w *admissionWaiter) {
	for i, x := range g.waiters {
		if x == w {
			g.waiters = append(g.waiters[:i], g.waiters[i+1:]...)
			return
		}
	}
}

// stats reports the gate's current occupancy (for tests and metrics).
func (g *admissionGate) stats() (inUse, waiting int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.inUse, len(g.waiters)
}
