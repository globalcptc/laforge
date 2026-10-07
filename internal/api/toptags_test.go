package api

import "testing"

func TestTopTagsOrdersNewestFirstAndCaps(t *testing.T) {
	tags := []string{"8.2", "8.11", "8.9", "latest", "7.4", "8.3", "8.10", "8.1", "8.0", "8.4", "8.5", "8.6"}
	shown, total := topTags(tags)
	if total != len(tags) {
		t.Fatalf("total = %d, want %d", total, len(tags))
	}
	if len(shown) != maxTagsShown {
		t.Fatalf("shown = %d, want %d", len(shown), maxTagsShown)
	}
	if shown[0] != "latest" {
		t.Fatalf("first = %q, want latest pinned first", shown[0])
	}
	// 8.11 must outrank 8.9 (natural, not lexical) and appear before it.
	i11, i9 := indexOf(shown, "8.11"), indexOf(shown, "8.9")
	if i11 < 0 || i9 < 0 || i11 > i9 {
		t.Fatalf("natural order wrong: 8.11 at %d, 8.9 at %d (shown=%v)", i11, i9, shown)
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}
