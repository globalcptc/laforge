package mcp

import (
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// limiter is a token bucket per key owner: rate calls a second, burst at once.
// Keyed by account rather than by key, so ten keys don't mean ten times the load.
type limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[pgtype.UUID]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(rate float64, burst int) *limiter {
	return &limiter{rate: rate, burst: float64(burst), buckets: map[pgtype.UUID]*bucket{}}
}

// allow takes a token for the account, or says how long until one is free.
func (l *limiter) allow(account pgtype.UUID, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[account]
	if !ok {
		if len(l.buckets) > 10_000 {
			l.buckets = map[pgtype.UUID]*bucket{} // bounded memory; a reset only forgives
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[account] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}
