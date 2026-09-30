package schedule

import (
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parsing %q: %v", s, err)
	}
	return v
}

func twoDayWindows(t *testing.T) []AccessWindow {
	return []AccessWindow{
		{Open: mustTime(t, "2026-10-01T09:00:00Z"), Close: mustTime(t, "2026-10-01T18:00:00Z")},
		{Open: mustTime(t, "2026-10-02T09:00:00Z"), Close: mustTime(t, "2026-10-02T17:00:00Z")},
	}
}

func TestScheduledOpen(t *testing.T) {
	w := twoDayWindows(t)
	cases := []struct {
		now  string
		want bool
	}{
		{"2026-09-29T12:00:00Z", false}, // before first window
		{"2026-10-01T08:59:59Z", false}, // one second before open
		{"2026-10-01T09:00:00Z", true},  // exactly at open (inclusive)
		{"2026-10-01T12:00:00Z", true},  // mid-window
		{"2026-10-01T18:00:00Z", false}, // exactly at close (exclusive)
		{"2026-10-01T20:00:00Z", false}, // overnight gap
		{"2026-10-02T10:00:00Z", true},  // second window
		{"2026-10-03T10:00:00Z", false}, // after last window
	}
	for _, c := range cases {
		if got := ScheduledOpen(w, mustTime(t, c.now)); got != c.want {
			t.Errorf("ScheduledOpen(%s) = %v, want %v", c.now, got, c.want)
		}
	}
	if ScheduledOpen(nil, mustTime(t, "2026-10-01T12:00:00Z")) {
		t.Error("no windows should never be open")
	}
}

func TestNextBoundary(t *testing.T) {
	w := twoDayWindows(t)
	cases := []struct {
		now      string
		wantEdge string // "" means ok=false
	}{
		{"2026-09-29T12:00:00Z", "2026-10-01T09:00:00Z"}, // next edge is first open
		{"2026-10-01T12:00:00Z", "2026-10-01T18:00:00Z"}, // mid-window -> this window's close
		{"2026-10-01T20:00:00Z", "2026-10-02T09:00:00Z"}, // overnight -> next open
		{"2026-10-02T17:00:00Z", ""},                     // exactly at last close -> nothing strictly after
		{"2026-10-05T00:00:00Z", ""},                     // after everything
	}
	for _, c := range cases {
		got, ok := NextBoundary(w, mustTime(t, c.now))
		if c.wantEdge == "" {
			if ok {
				t.Errorf("NextBoundary(%s) = %s, want no boundary", c.now, got)
			}
			continue
		}
		if !ok || !got.Equal(mustTime(t, c.wantEdge)) {
			t.Errorf("NextBoundary(%s) = (%s, %v), want %s", c.now, got, ok, c.wantEdge)
		}
	}
	if _, ok := NextBoundary(nil, mustTime(t, "2026-10-01T12:00:00Z")); ok {
		t.Error("no windows should have no boundary")
	}
}

func TestParseAccessWindows(t *testing.T) {
	raw := []byte(`[{"open":"2026-10-01T09:00:00Z","close":"2026-10-01T18:00:00Z"},{"open":"bad","close":"2026-10-02T17:00:00Z"}]`)
	got := ParseAccessWindows(raw)
	if len(got) != 1 {
		t.Fatalf("want 1 valid window (malformed one skipped), got %d", len(got))
	}
	if !got[0].Open.Equal(mustTime(t, "2026-10-01T09:00:00Z")) {
		t.Errorf("unexpected open %s", got[0].Open)
	}
	if ParseAccessWindows(nil) != nil {
		t.Error("nil raw should parse to nil")
	}
	if ParseAccessWindows([]byte(`not json`)) != nil {
		t.Error("garbage should parse to nil, not panic")
	}
}
