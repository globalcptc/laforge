package builder

import (
	"context"
	"sync"
	"time"
)

// Phase is one timed stage of a builder call -- an Incus image copy, a
// config-drive import, an instance start -- so a slow deploy shows WHERE its
// time went, not just its total.
type Phase struct {
	Name     string
	Duration time.Duration
}

// PhaseTimer collects the phases a builder records during one call. The caller
// (internal/runner) attaches one to the call's context with WithPhaseTimer and
// reads it back afterwards; a builder records into it with RecordPhase or
// StartPhase. Optional on both sides: a builder that records nothing leaves it
// empty, and recording into a context without a timer is a no-op, so no
// Builder method signature changes.
type PhaseTimer struct {
	mu     sync.Mutex
	phases []Phase
}

type phaseTimerKey struct{}

// WithPhaseTimer returns ctx carrying a fresh PhaseTimer, and that timer.
func WithPhaseTimer(ctx context.Context) (context.Context, *PhaseTimer) {
	t := &PhaseTimer{}
	return context.WithValue(ctx, phaseTimerKey{}, t), t
}

// RecordPhase adds one completed phase to ctx's timer, if it has one. A phase
// recorded twice (a retried start, say) accumulates into the same entry, keeping
// first-seen order.
func RecordPhase(ctx context.Context, name string, d time.Duration) {
	t, _ := ctx.Value(phaseTimerKey{}).(*PhaseTimer)
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.phases {
		if t.phases[i].Name == name {
			t.phases[i].Duration += d
			return
		}
	}
	t.phases = append(t.phases, Phase{Name: name, Duration: d})
}

// StartPhase starts timing a phase and returns the func that records it:
//
//	done := builder.StartPhase(ctx, "create")
//	err := create(...)
//	done()
func StartPhase(ctx context.Context, name string) func() {
	start := time.Now()
	return func() { RecordPhase(ctx, name, time.Since(start)) }
}

// Phases returns the recorded phases in the order they were first seen.
func (t *PhaseTimer) Phases() []Phase {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Phase(nil), t.phases...)
}
