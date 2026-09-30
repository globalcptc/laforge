package schedule

import "time"

// AccessWindow is one real open/close pair, resolved from an
// environment's own `access:` list -- what an access_open/access_close
// anchor actually walks through. Real timestamps, not schedule.TimeOfDay:
// an access window is a specific day's occurrence, not a daily-repeating
// clock time.
type AccessWindow struct {
	Open, Close time.Time
}

// Anchors holds the real timestamps this package's anchors resolve
// against, so NextFireAfter never needs to know how to read an
// environment file -- just do the arithmetic. CompetitionStart/End come
// from environment.start/.stop (already exposed as render.Context's own
// Start/Stop). AccessWindows come from environment.access[], in order;
// only future windows relative to "now" matter for computing a next
// fire time, but passing the full list is simplest for the caller.
type Anchors struct {
	CompetitionStart time.Time
	CompetitionEnd   time.Time
	AccessWindows    []AccessWindow
}

// NextFireAfter computes e's next real fire time strictly after now, in
// now's own timezone/location. ok is false when there is genuinely no
// next occurrence -- a fires-once anchor already past, or an anchored
// access-window expression with no more windows left.
func (e Expr) NextFireAfter(now time.Time, a Anchors) (t time.Time, ok bool) {
	switch e.Kind {
	case KindDailyAt:
		return e.nextDailyAt(now)
	case KindInterval:
		return e.nextInterval(now, a)
	case KindAnchored:
		return e.nextAnchored(now, a)
	default:
		return time.Time{}, false
	}
}

func (e Expr) nextDailyAt(now time.Time) (time.Time, bool) {
	if len(e.Times) == 0 {
		return time.Time{}, false
	}
	var best time.Time
	for _, tod := range e.Times {
		c := time.Date(now.Year(), now.Month(), now.Day(), tod.Hour, tod.Minute, 0, 0, now.Location())
		if !c.After(now) {
			c = c.AddDate(0, 0, 1)
		}
		if best.IsZero() || c.Before(best) {
			best = c
		}
	}
	return best, true
}

// boundCrossingOnOrAfter resolves an interval's optional bound to a real
// timestamp for the day containing `day` -- either a fixed clock time,
// or (for an anchor bound) the nearest access window edge on or after
// day. A time-of-day bound is simple: same clock time, every day. An
// anchor bound only really makes sense against access windows (a
// competition only starts/ends once, so "every 30 minutes after
// competition start" is unusual phrasing but treated the same way: the
// single competition start/end timestamp, not a daily-repeating one).
func (e Expr) boundCrossingOnOrAfter(day time.Time, a Anchors) (time.Time, bool) {
	if e.BoundTime != nil {
		c := time.Date(day.Year(), day.Month(), day.Day(), e.BoundTime.Hour, e.BoundTime.Minute, 0, 0, day.Location())
		return c, true
	}
	switch e.BoundAnchor {
	case AnchorCompetitionStart:
		return a.CompetitionStart, !a.CompetitionStart.IsZero()
	case AnchorCompetitionEnd:
		return a.CompetitionEnd, !a.CompetitionEnd.IsZero()
	case AnchorAccessOpen:
		for _, w := range a.AccessWindows {
			if !w.Open.Before(day) {
				return w.Open, true
			}
		}
	case AnchorAccessClose:
		for _, w := range a.AccessWindows {
			if !w.Close.Before(day) {
				return w.Close, true
			}
		}
	}
	return time.Time{}, false
}

// nextInterval fires every N*Unit, optionally only from a bound onward
// (direction after) or only until a bound (direction before), resetting
// at midnight either way -- "every 30 minutes after 2:00pm" starts
// again at 2:00pm the next day, not just once.
func (e Expr) nextInterval(now time.Time, a Anchors) (time.Time, bool) {
	next := now.Add(e.Offset())
	if e.BoundDirection == DirectionNone {
		return next, true
	}
	bound, ok := e.boundCrossingOnOrAfter(now, a)
	if !ok {
		return time.Time{}, false
	}
	switch e.BoundDirection {
	case DirectionAfter:
		if next.Before(bound) {
			return bound, true
		}
		return next, true
	case DirectionBefore:
		if !next.Before(bound) {
			// Past today's cutoff -- next window reopens at midnight,
			// first fire is one offset past that.
			tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			return tomorrow.Add(e.Offset()), true
		}
		return next, true
	}
	return next, true
}

// nextAnchored fires once (competition start/end) or once per
// occurrence (access open/close -- see FiresOnce's own doc comment for
// why). A "before" direction on a recurring anchor targets the SAME
// occurrence's edge (e.g. "30 minutes before access closes" fires once
// per close, 30 minutes ahead of each one), not the next edge after now
// plus the offset.
func (e Expr) nextAnchored(now time.Time, a Anchors) (time.Time, bool) {
	if e.FiresOnce() {
		var anchor time.Time
		switch e.Anchor {
		case AnchorCompetitionStart:
			anchor = a.CompetitionStart
		case AnchorCompetitionEnd:
			anchor = a.CompetitionEnd
		}
		if anchor.IsZero() {
			return time.Time{}, false
		}
		var t time.Time
		if e.Direction == DirectionBefore {
			t = anchor.Add(-e.Offset())
		} else {
			t = anchor.Add(e.Offset())
		}
		if !t.After(now) {
			return time.Time{}, false // already past -- a real fires-once expression never re-fires
		}
		return t, true
	}
	for _, w := range a.AccessWindows {
		var edge time.Time
		if e.Anchor == AnchorAccessOpen {
			edge = w.Open
		} else {
			edge = w.Close
		}
		var t time.Time
		if e.Direction == DirectionBefore {
			t = edge.Add(-e.Offset())
		} else {
			t = edge.Add(e.Offset())
		}
		if t.After(now) {
			return t, true
		}
	}
	return time.Time{}, false // no more access windows left
}
