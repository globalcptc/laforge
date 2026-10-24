package schedule

import (
	"encoding/json"
	"time"
)

// This file holds the small, pure helpers the access reconciler
// (internal/orchestrator) and the manual access endpoint (internal/api) both
// use to turn an environment's authored `access:` windows into a live
// open/closed decision. Keeping them here -- next to AccessWindow and with no
// dependency on loader or db -- means one definition of "is a team supposed to
// be open right now" and "when does the schedule next change," shared by the
// automatic enforcer and the hand-override path so the two can never disagree.

// ScheduledOpen reports whether now falls inside any authored access window
// (open inclusive, close exclusive). With no windows it is false: an
// environment that authored no schedule opens nothing on its own.
func ScheduledOpen(windows []AccessWindow, now time.Time) bool {
	for _, w := range windows {
		if !now.Before(w.Open) && now.Before(w.Close) {
			return true
		}
	}
	return false
}

// NextBoundary returns the earliest window edge (an open or a close) strictly
// after now -- the next moment the schedule's own open/closed answer changes.
// ok is false when no edge lies in the future (every window is in the past, or
// there are none): a hand override taken then has no schedule to hand back to,
// so it holds until changed. This is what makes a manual open/close "sticky
// until the next boundary."
func NextBoundary(windows []AccessWindow, now time.Time) (t time.Time, ok bool) {
	for _, w := range windows {
		for _, edge := range [...]time.Time{w.Open, w.Close} {
			if edge.After(now) && (!ok || edge.Before(t)) {
				t, ok = edge, true
			}
		}
	}
	return t, ok
}

// CurrentWindowClose returns the close time of the access window now falls
// inside (open inclusive, close exclusive), if any. It's the team's scheduled
// access end right now -- what "extend/reduce by N" move up or down.
func CurrentWindowClose(windows []AccessWindow, now time.Time) (time.Time, bool) {
	for _, w := range windows {
		if !now.Before(w.Open) && now.Before(w.Close) {
			return w.Close, true
		}
	}
	return time.Time{}, false
}

// OpenInWindowStartingAfter reports whether now falls inside a window whose OPEN
// edge is at or after `after`. Once a manually-set access end (`after`, from an
// extend/reduce override) has passed, this is how the reconciler tells a
// genuinely NEW window that has since opened (reopen) from still being inside
// the window that end cut short (stay closed). Without it, shortening a window's
// end wouldn't stick: the schedule would re-open the very window just trimmed.
//
// The edge is at-or-after, not strictly-after, on purpose. A plain manual OPEN
// override (not an extend/reduce) records its expiry as the schedule's NEXT
// BOUNDARY -- and when a team is opened early, that next boundary IS the coming
// window's open edge. So the override lapses at the exact instant the window
// opens, with `after` == that window's Open. A strict `.After` would fail to
// see the window as new (its open equals `after`, not after it) and force the
// team CLOSED at the very moment access was supposed to begin. Equality means
// that window counts as the new one and the team follows the schedule open. A
// reduce's `after` is a point strictly INSIDE its window (open < after < close),
// so the trimmed window's open is still strictly before `after` and remains
// excluded -- a reduced end still sticks.
func OpenInWindowStartingAfter(windows []AccessWindow, now, after time.Time) bool {
	for _, w := range windows {
		if !w.Open.Before(after) && !now.Before(w.Open) && now.Before(w.Close) {
			return true
		}
	}
	return false
}

// ParseAccessWindows reads an environment row's stored `access` JSON (the same
// [{open,close}] RFC3339 shape loader.AccessWindow marshals to, persisted on
// the environment table at ingest) into real AccessWindow timestamps. A window
// whose timestamps don't parse is skipped rather than failing the whole set --
// the loader/schema already rejects malformed windows at commit time, so
// reaching here with one means content that never should have validated.
func ParseAccessWindows(raw []byte) []AccessWindow {
	if len(raw) == 0 {
		return nil
	}
	var rs []struct {
		Open  string `json:"open"`
		Close string `json:"close"`
	}
	if json.Unmarshal(raw, &rs) != nil {
		return nil
	}
	out := make([]AccessWindow, 0, len(rs))
	for _, r := range rs {
		open, errO := time.Parse(time.RFC3339, r.Open)
		close, errC := time.Parse(time.RFC3339, r.Close)
		if errO != nil || errC != nil {
			continue
		}
		out = append(out, AccessWindow{Open: open, Close: close})
	}
	return out
}
