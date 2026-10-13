package microcloud

import (
	"context"
	"testing"
)

func TestTargetMember(t *testing.T) {
	ctx := context.Background()

	// No blocklist -> no targeting (let LXD auto-schedule).
	empty := &Builder{Config: Config{}}
	if tgt, err := empty.targetMember(ctx, "anything"); err != nil || tgt != "" {
		t.Fatalf("empty blocklist targetMember = %q, %v; want \"\"", tgt, err)
	}

	// With a blocklist, pick deterministically from the allowed members. Seed the
	// resolved-members cache so the test doesn't need a live cluster.
	b := &Builder{Config: Config{HostBlocklist: []string{"micro-07-gpu"}}}
	allowed := []string{"micro-01", "micro-02", "micro-03"}
	b.membersOnce.Do(func() { b.membersCached = allowed })

	seen := map[string]bool{}
	for _, name := range []string{"host-a", "host-b", "host-c", "host-d", "host-e", "host-f"} {
		t1, err := b.targetMember(ctx, name)
		if err != nil {
			t.Fatalf("targetMember(%q): %v", name, err)
		}
		// Stable: the same instance always lands on the same member.
		if t2, _ := b.targetMember(ctx, name); t2 != t1 {
			t.Errorf("targetMember(%q) not stable: %q vs %q", name, t1, t2)
		}
		found := false
		for _, a := range allowed {
			if a == t1 {
				found = true
			}
		}
		if !found {
			t.Errorf("targetMember(%q) = %q, not an allowed member", name, t1)
		}
		seen[t1] = true
	}
	// Over several instances it should spread across more than one member.
	if len(seen) < 2 {
		t.Errorf("targetMember mapped every instance to one member (%v); expected spread", seen)
	}

	// Blocklist set but no member survived -> a clear error, not a blank target.
	none := &Builder{Config: Config{HostBlocklist: []string{"micro-01"}}}
	none.membersOnce.Do(func() { none.membersCached = nil })
	if _, err := none.targetMember(ctx, "x"); err == nil {
		t.Error("targetMember with no allowed members should error")
	}
}
