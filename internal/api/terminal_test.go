package api

import (
	"testing"
	"time"
)

// TestShellBlockedReason pins the rule: a shell is gated on a LIVE AGENT, not a
// successful build. Any host that is up at the hoster (including build_failed
// and invalid) with a recent heartbeat may open a shell; a host that never came
// up, or whose agent is missing/stale, may not.
func TestShellBlockedReason(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-10 * time.Second) // well within shellWindow
	stale := now.Add(-shellWindow - time.Minute)

	// Up at the hoster + a live agent -> allowed. build_failed and invalid are
	// the whole point: the agent is healthy, so debugging must work.
	for _, s := range []string{"running", "building", "finished", "build_failed", "invalid"} {
		if reason := shellBlockedReason(s, true, fresh, now); reason != "" {
			t.Errorf("shellBlockedReason(%q, live agent) = %q, want allowed", s, reason)
		}
	}

	// Not up at the hoster -> blocked no matter how fresh a (stale/leftover)
	// session looks: there is no running instance to shell into.
	for _, s := range []string{"pending", "deploying", "deploy_failed", "destroying", "destroyed", ""} {
		if reason := shellBlockedReason(s, true, fresh, now); reason == "" {
			t.Errorf("shellBlockedReason(%q) = allowed, want blocked (instance not up)", s)
		}
	}

	// Up, but the agent isn't live -> blocked.
	if reason := shellBlockedReason("running", false, time.Time{}, now); reason == "" {
		t.Error("shellBlockedReason(running, no session) = allowed, want blocked (no agent)")
	}
	if reason := shellBlockedReason("build_failed", true, stale, now); reason == "" {
		t.Error("shellBlockedReason(build_failed, stale heartbeat) = allowed, want blocked (agent missing)")
	}
}
