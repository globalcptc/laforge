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
