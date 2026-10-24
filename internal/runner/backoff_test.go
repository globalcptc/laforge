package runner

import (
	"testing"
	"time"
)

// TestRetryBackoff covers the fix for the teardown gap where a network's
// destroy_network task burned all MaxAttempts inside one second (the network
// was still draining its used_by), leaving it stuck in 'destroying'. Backoff
// must grow with attempts, honour a per-runner base, and stay capped.
func TestRetryBackoff(t *testing.T) {
	// Default base (field zero) uses retryBackoffBase, scaled by attempts.
	def := &Runner{}
	if got := def.RetryBackoff(1); got != retryBackoffBase {
		t.Fatalf("attempt 1 default = %s, want %s", got, retryBackoffBase)
	}
	if got := def.RetryBackoff(2); got != 2*retryBackoffBase {
		t.Fatalf("attempt 2 default = %s, want %s", got, 2*retryBackoffBase)
	}
	// A zero/negative attempt count is floored to 1, never 0 (which would
	// reinstate the hot-loop this fix removes).
	if got := def.RetryBackoff(0); got != retryBackoffBase {
		t.Fatalf("attempt 0 default = %s, want %s (floored to one base)", got, retryBackoffBase)
	}
	// Cap holds for large attempt counts.
	if got := def.RetryBackoff(1000); got != retryBackoffCap {
		t.Fatalf("attempt 1000 default = %s, want cap %s", got, retryBackoffCap)
	}
	// A small per-runner base (what tests set) keeps retries fast and is
	// never overridden by the production cap.
	fast := &Runner{RetryBackoffBase: 10 * time.Millisecond}
	if got := fast.RetryBackoff(3); got != 30*time.Millisecond {
		t.Fatalf("fast base attempt 3 = %s, want 30ms", got)
	}
}
