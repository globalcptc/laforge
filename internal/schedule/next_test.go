package schedule

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, s string) Expr {
	t.Helper()
	e, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return e
}

func at(y, m, d, hh, mm int) time.Time {
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, time.UTC)
}

func TestNextFireAfterPlainInterval(t *testing.T) {
	e := mustParse(t, "Every 30 minutes")
	now := at(2026, 10, 1, 9, 0)
	next, ok := e.NextFireAfter(now, Anchors{})
	if !ok {
		t.Fatal("expected ok")
	}
	want := at(2026, 10, 1, 9, 30)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextFireAfterDailyAtPicksSoonestTimeToday(t *testing.T) {
	e := mustParse(t, "Every day at 10:00am and 2:00pm")
	now := at(2026, 10, 1, 9, 0) // before both
	next, ok := e.NextFireAfter(now, Anchors{})
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 1, 10, 0); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextFireAfterDailyAtRollsToTomorrow(t *testing.T) {
	e := mustParse(t, "Every day at 10:00am and 2:00pm")
	now := at(2026, 10, 1, 15, 0) // after both today
	next, ok := e.NextFireAfter(now, Anchors{})
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 2, 10, 0); !next.Equal(want) {
		t.Errorf("next = %v, want %v (tomorrow's first time)", next, want)
	}
}

func TestNextFireAfterIntervalAfterBoundClampsForward(t *testing.T) {
	e := mustParse(t, "Every 30 minutes after 2:00pm")
	// now is well before the bound -- naive now+30m is also before the
	// bound, so the real next fire is the bound itself, not 9:30am.
	now := at(2026, 10, 1, 9, 0)
	next, ok := e.NextFireAfter(now, Anchors{})
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 1, 14, 0); !next.Equal(want) {
		t.Errorf("next = %v, want %v (clamped to the 2pm bound)", next, want)
	}
}

func TestNextFireAfterIntervalAfterBoundOnceInsideWindow(t *testing.T) {
	e := mustParse(t, "Every 30 minutes after 2:00pm")
	now := at(2026, 10, 1, 14, 10) // already past the bound
	next, ok := e.NextFireAfter(now, Anchors{})
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 1, 14, 40); !next.Equal(want) {
		t.Errorf("next = %v, want %v (plain now+30m, already inside the window)", next, want)
	}
}

func TestNextFireAfterIntervalBeforeBoundRollsToTomorrow(t *testing.T) {
	e := mustParse(t, "Every hour before access closes")
	windows := []AccessWindow{{Open: at(2026, 10, 1, 9, 0), Close: at(2026, 10, 1, 18, 0)}}
	now := at(2026, 10, 1, 17, 30) // now+1h = 18:30, past the 18:00 close
	next, ok := e.NextFireAfter(now, Anchors{AccessWindows: windows})
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 2, 1, 0); !next.Equal(want) {
		t.Errorf("next = %v, want %v (tomorrow, midnight + offset)", next, want)
	}
}

func TestNextFireAfterAnchoredCompetitionStartFiresOnce(t *testing.T) {
	e := mustParse(t, "45 minutes after competition start")
	anchors := Anchors{CompetitionStart: at(2026, 10, 1, 9, 0)}

	next, ok := e.NextFireAfter(at(2026, 9, 30, 0, 0), anchors)
	if !ok {
		t.Fatal("expected ok before the anchor")
	}
	if want := at(2026, 10, 1, 9, 45); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	// Asking again from AFTER that fire time: no more occurrences.
	_, ok = e.NextFireAfter(at(2026, 10, 1, 10, 0), anchors)
	if ok {
		t.Error("expected no more occurrences once the fires-once time has passed")
	}
}

func TestNextFireAfterAnchoredAccessCloseFiresOncePerWindow(t *testing.T) {
	e := mustParse(t, "30 minutes before access closes")
	windows := []AccessWindow{
		{Open: at(2026, 10, 1, 9, 0), Close: at(2026, 10, 1, 18, 0)},
		{Open: at(2026, 10, 2, 9, 0), Close: at(2026, 10, 2, 17, 0)},
	}
	anchors := Anchors{AccessWindows: windows}

	next, ok := e.NextFireAfter(at(2026, 10, 1, 8, 0), anchors)
	if !ok {
		t.Fatal("expected ok")
	}
	if want := at(2026, 10, 1, 17, 30); !next.Equal(want) {
		t.Errorf("night 1: next = %v, want %v", next, want)
	}

	// After night 1's fire, the SAME expression finds night 2's occurrence --
	// this is the real "once per access window, every night" proof.
	next2, ok := e.NextFireAfter(next, anchors)
	if !ok {
		t.Fatal("expected ok for the second access window")
	}
	if want := at(2026, 10, 2, 16, 30); !next2.Equal(want) {
		t.Errorf("night 2: next = %v, want %v", next2, want)
	}

	// After the last window's fire, nothing left.
	_, ok = e.NextFireAfter(next2, anchors)
	if ok {
		t.Error("expected no more occurrences after the last access window")
	}
}
