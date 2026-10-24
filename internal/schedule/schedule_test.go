package schedule

import (
	"strings"
	"testing"
	"time"
)

// TestParseSevenExamples is every worked example from
// the "Schedule" section, the real phrases this
// grammar was built against.
func TestParseSevenExamples(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Expr
	}{
		{
			"repeats every 30 minutes",
			"Repeats every 30 minutes",
			Expr{Kind: KindInterval, N: 30, Unit: time.Minute},
		},
		{
			"runs 45 minutes after competition start",
			"Runs 45 minutes after competition start",
			Expr{Kind: KindAnchored, N: 45, Unit: time.Minute, Direction: DirectionAfter, Anchor: AnchorCompetitionStart},
		},
		{
			"runs every day at 10:00am and 2:00pm",
			"Runs every day at 10:00am and 2:00pm",
			Expr{Kind: KindDailyAt, Times: []TimeOfDay{{10, 0}, {14, 0}}},
		},
		{
			"every 30 minutes after 2:00pm",
			"Every 30 minutes after 2:00pm",
			Expr{Kind: KindInterval, N: 30, Unit: time.Minute, BoundDirection: DirectionAfter, BoundTime: &TimeOfDay{14, 0}},
		},
		{
			"every hour",
			"Every hour",
			Expr{Kind: KindInterval, N: 1, Unit: time.Hour},
		},
		{
			"30 minutes before access closes",
			"30 minutes before access closes",
			Expr{Kind: KindAnchored, N: 30, Unit: time.Minute, Direction: DirectionBefore, Anchor: AnchorAccessClose},
		},
		{
			"1 hour after access opens",
			"1 hour after access opens",
			Expr{Kind: KindAnchored, N: 1, Unit: time.Hour, Direction: DirectionAfter, Anchor: AnchorAccessOpen},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.in, err)
			}
			w := c.want
			if got.Kind != w.Kind || got.N != w.N || got.Unit != w.Unit ||
				got.BoundDirection != w.BoundDirection || got.BoundAnchor != w.BoundAnchor ||
				got.Direction != w.Direction || got.Anchor != w.Anchor {
				t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, w)
			}
			if (got.BoundTime == nil) != (w.BoundTime == nil) {
				t.Errorf("Parse(%q).BoundTime = %v, want %v", c.in, got.BoundTime, w.BoundTime)
			} else if got.BoundTime != nil && *got.BoundTime != *w.BoundTime {
				t.Errorf("Parse(%q).BoundTime = %v, want %v", c.in, *got.BoundTime, *w.BoundTime)
			}
			if len(got.Times) != len(w.Times) {
				t.Errorf("Parse(%q).Times = %v, want %v", c.in, got.Times, w.Times)
			} else {
				for i := range w.Times {
					if got.Times[i] != w.Times[i] {
						t.Errorf("Parse(%q).Times[%d] = %v, want %v", c.in, i, got.Times[i], w.Times[i])
					}
				}
			}
		})
	}
}

func TestParseDailyAtTimesMatch(t *testing.T) {
	got, err := Parse("Runs every day at 10:00am and 2:00pm")
	if err != nil {
		t.Fatal(err)
	}
	want := []TimeOfDay{{10, 0}, {14, 0}}
	if len(got.Times) != len(want) {
		t.Fatalf("Times = %v, want %v", got.Times, want)
	}
	for i := range want {
		if got.Times[i] != want[i] {
			t.Errorf("Times[%d] = %v, want %v", i, got.Times[i], want[i])
		}
	}
}

func TestParseBoundTimeValue(t *testing.T) {
	got, err := Parse("Every 30 minutes after 2:00pm")
	if err != nil {
		t.Fatal(err)
	}
	if got.BoundTime == nil || *got.BoundTime != (TimeOfDay{14, 0}) {
		t.Errorf("BoundTime = %v, want 14:00", got.BoundTime)
	}
}

// TestParseRewordings checks real variation in phrasing beyond the
// literal examples -- the whole point of a composable clause grammar
// over four fixed templates.
func TestParseRewordings(t *testing.T) {
	cases := []struct {
		in           string
		wantKind     Kind
		wantAnchor   Anchor
		wantDir      Direction
		wantN        int
		wantUnit     time.Duration
		wantBoundAnc Anchor
	}{
		{in: "every 15 minutes after access opens", wantKind: KindInterval, wantN: 15, wantUnit: time.Minute, wantBoundAnc: AnchorAccessOpen},
		{in: "every hour before access closes", wantKind: KindInterval, wantN: 1, wantUnit: time.Hour, wantBoundAnc: AnchorAccessClose},
		{in: "every day at 14:00", wantKind: KindDailyAt},
		{in: "every day at 9am, 12pm, 3pm", wantKind: KindDailyAt},
		{in: "before access closes", wantKind: KindAnchored, wantN: 0, wantDir: DirectionBefore, wantAnchor: AnchorAccessClose},
		{in: "after the competition ends", wantKind: KindAnchored, wantN: 0, wantDir: DirectionAfter, wantAnchor: AnchorCompetitionEnd},
		{in: "2 hours before competition end", wantKind: KindAnchored, wantN: 2, wantUnit: time.Hour, wantDir: DirectionBefore, wantAnchor: AnchorCompetitionEnd},
		{in: "run every 3 days", wantKind: KindInterval, wantN: 3, wantUnit: 24 * time.Hour},
		{in: "repeat every day", wantKind: KindInterval, wantN: 1, wantUnit: 24 * time.Hour},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", c.in, err)
			}
			if got.Kind != c.wantKind {
				t.Errorf("Kind = %v, want %v", got.Kind, c.wantKind)
			}
			if c.wantKind == KindAnchored {
				if got.Anchor != c.wantAnchor || got.Direction != c.wantDir || got.N != c.wantN {
					t.Errorf("got N=%d dir=%v anchor=%v, want N=%d dir=%v anchor=%v", got.N, got.Direction, got.Anchor, c.wantN, c.wantDir, c.wantAnchor)
				}
			}
			if c.wantKind == KindInterval {
				if got.N != c.wantN || got.Unit != c.wantUnit {
					t.Errorf("got N=%d unit=%v, want N=%d unit=%v", got.N, got.Unit, c.wantN, c.wantUnit)
				}
				if c.wantBoundAnc != "" && got.BoundAnchor != c.wantBoundAnc {
					t.Errorf("BoundAnchor = %v, want %v", got.BoundAnchor, c.wantBoundAnc)
				}
			}
		})
	}
}

func TestFiresOnce(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"45 minutes after competition start", true},
		{"2 hours before competition end", true},
		{"30 minutes before access closes", false}, // once PER access window, not once overall
		{"1 hour after access opens", false},
		{"every hour", false},
		{"every day at 10:00am", false},
	}
	for _, c := range cases {
		e, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got := e.FiresOnce(); got != c.want {
			t.Errorf("Parse(%q).FiresOnce() = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRejectsInvalidInput(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"every purple minutes",
		"every day at teatime",
		"every 0 minutes",
		"every -5 hours",
		"45 minutes after lunch",
		"every 30 minutes after somewhere",
		"blah blah blah",
		"every day at",
		"before",
		"after",
	}
	for _, in := range cases {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) succeeded, want a rejection error", in)
		}
	}
}

func TestParseErrorNamesTheInput(t *testing.T) {
	_, err := Parse("every purple minutes")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "every purple minutes") {
		t.Errorf("error %q doesn't name the offending input", err.Error())
	}
}
