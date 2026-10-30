package orchestrator

import (
	"fmt"
	"testing"
)

func TestFingerprintCacheGetPut(t *testing.T) {
	c := &fingerprintCache{m: make(map[string]string), max: 4}

	if _, ok := c.get("k"); ok {
		t.Fatal("empty cache should miss")
	}
	c.put("k", "fp1")
	if v, ok := c.get("k"); !ok || v != "fp1" {
		t.Fatalf("get after put = (%q, %v), want (fp1, true)", v, ok)
	}

	// Filling past the cap resets the map (bounded, never unbounded growth) --
	// the key just put before the reset is gone, and a later put repopulates.
	for i := 0; i < c.max; i++ {
		c.put(fmt.Sprintf("x%d", i), "v")
	}
	if len(c.m) > c.max {
		t.Fatalf("cache grew past cap: %d > %d", len(c.m), c.max)
	}
	c.put("fresh", "v2")
	if v, ok := c.get("fresh"); !ok || v != "v2" {
		t.Fatalf("cache unusable after reset: get fresh = (%q, %v)", v, ok)
	}
}

// TestFingerprintCacheKeySeparatesEnvironments: two environments on one content
// revision must not share fingerprint entries -- the same host and team render
// differently in each, and sharing them made a build recreate every host.
func TestFingerprintCacheKeySeparatesEnvironments(t *testing.T) {
	a := fingerprintCacheKey("rev", "cptc12-socal", "host", "bff-dc-01", 9)
	b := fingerprintCacheKey("rev", "cptc12-central", "host", "bff-dc-01", 9)
	if a == b {
		t.Fatalf("same key %q for two environments", a)
	}
	if a != fingerprintCacheKey("rev", "cptc12-socal", "host", "bff-dc-01", 9) {
		t.Fatal("key isn't stable for the same object")
	}
}
