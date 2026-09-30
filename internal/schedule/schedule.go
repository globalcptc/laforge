// Package schedule parses a `schedule:` entry's `when:` natural-language
// time expression into a
// structured Expr. This is deliberately a real tokenizer + parser with
// broad phrase coverage -- not four fixed whole-sentence templates, and
// not fuzzy/LLM interpretation (that would need network access during
// `laforge check`/CI and breaks this project's "same binary, same
// answers" determinism everywhere else). A phrasing outside the grammar
// is a real error naming what was expected, matching every other
// validation in this codebase.
//
// The grammar is three composable clause kinds:
//
//	interval: every [N] <minute|hour|day>(s) [before|after <bound>]
//	daily-at: every day at <time>[, <time>]*
//	anchored: [N <unit>(s)] before|after <anchor>
//
// where <bound> is a clock time or an anchor, and <anchor> is one of
// competition start, competition end, access opens, access closes.
package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	KindInterval Kind = "interval"
	KindDailyAt  Kind = "daily_at"
	KindAnchored Kind = "anchored"
)

// Anchor is a named point in a build's timeline a schedule can be
// relative to. competition start/end happen exactly once (they resolve
// against environment.start/.stop); access opens/closes happen once per
// access window, which is why a real multi-day event can say "every
// night, 30 minutes after access closes" and mean it literally.
type Anchor string

const (
	AnchorNone             Anchor = ""
	AnchorCompetitionStart Anchor = "competition_start"
	AnchorCompetitionEnd   Anchor = "competition_end"
	AnchorAccessOpen       Anchor = "access_open"
	AnchorAccessClose      Anchor = "access_close"
)

type Direction string

const (
	DirectionNone   Direction = ""
	DirectionBefore Direction = "before"
	DirectionAfter  Direction = "after"
)

// TimeOfDay is a clock time, minute precision, with no timezone of its
// own -- it's always interpreted in whatever timezone the environment's
// own start/stop/access times are already authored in.
type TimeOfDay struct {
	Hour   int
	Minute int
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// SinceMidnight is t as an offset from the start of its day -- what a
// real scheduler actually needs to compare against.
func (t TimeOfDay) SinceMidnight() time.Duration {
	return time.Duration(t.Hour)*time.Hour + time.Duration(t.Minute)*time.Minute
}

// Expr is one parsed schedule expression. Which fields are set depends
// on Kind: interval uses N/Unit/Bound*, daily_at uses Times, anchored
// uses N/Unit/Direction/Anchor.
type Expr struct {
	Kind Kind
	Raw  string

	N    int           // interval count (defaults 1) or anchored offset count (defaults 0: "right at the anchor")
	Unit time.Duration // minute/hour/day granularity, for interval and anchored

	BoundDirection Direction  // interval's optional "before|after <bound>" clause
	BoundTime      *TimeOfDay // set when the bound is a clock time
	BoundAnchor    Anchor     // set when the bound is an anchor instead

	Times []TimeOfDay // daily_at

	Direction Direction // anchored
	Anchor    Anchor    // anchored
}

// Offset is N*Unit -- the anchored/interval offset as a single duration.
func (e Expr) Offset() time.Duration {
	return time.Duration(e.N) * e.Unit
}

// FiresOnce is true only for an anchored expression whose anchor itself
// happens exactly once (competition start/end). Every other shape --
// interval, daily_at, and anchored-on-access-open/close (recurs once
// per access window) -- fires more than once over a build's lifetime.
func (e Expr) FiresOnce() bool {
	return e.Kind == KindAnchored && (e.Anchor == AnchorCompetitionStart || e.Anchor == AnchorCompetitionEnd)
}

// AnchorName is e's own anchor, if it has one -- an anchored expression's
// real Anchor, or an interval's optional BoundAnchor -- as a plain
// string, for storing in scheduled_task.anchor (internal/db). Empty for
// daily_at and an unbounded/time-bounded interval, neither of which has
// a real anchor at all.
func (e Expr) AnchorName() string {
	switch {
	case e.Kind == KindAnchored:
		return string(e.Anchor)
	case e.Kind == KindInterval && e.BoundAnchor != "":
		return string(e.BoundAnchor)
	default:
		return ""
	}
}

var unitWords = map[string]time.Duration{
	"minute": time.Minute, "minutes": time.Minute, "min": time.Minute, "mins": time.Minute,
	"hour": time.Hour, "hours": time.Hour,
	"day": 24 * time.Hour, "days": 24 * time.Hour,
}

var fillerLead = map[string]bool{
	"repeats": true, "repeat": true, "runs": true, "run": true,
}

var anchorPhrases = map[string]Anchor{
	"competition start":  AnchorCompetitionStart,
	"competition starts": AnchorCompetitionStart,
	"competition end":    AnchorCompetitionEnd,
	"competition ends":   AnchorCompetitionEnd,
	"access open":        AnchorAccessOpen,
	"access opens":       AnchorAccessOpen,
	"access close":       AnchorAccessClose,
	"access closes":      AnchorAccessClose,
}

// timeTokenRE matches one time-of-day token: "10:00am", "2pm", "14:00".
// A bare "2" (no colon, no am/pm) never matches -- that's genuinely
// ambiguous with an interval count, so this grammar simply never
// accepts it as a time.
var timeTokenRE = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?(am|pm)?$`)

func parseTimeOfDay(tok string) (TimeOfDay, bool) {
	m := timeTokenRE.FindStringSubmatch(strings.ToLower(tok))
	if m == nil {
		return TimeOfDay{}, false
	}
	hour, _ := strconv.Atoi(m[1])
	minute := 0
	if m[2] != "" {
		minute, _ = strconv.Atoi(m[2])
	}
	ampm := m[3]
	if ampm == "" && m[2] == "" {
		return TimeOfDay{}, false // bare number: not a time, could be a count
	}
	if ampm != "" {
		if hour < 1 || hour > 12 {
			return TimeOfDay{}, false
		}
		switch {
		case ampm == "pm" && hour != 12:
			hour += 12
		case ampm == "am" && hour == 12:
			hour = 0
		}
	} else if hour > 23 {
		return TimeOfDay{}, false
	}
	if minute > 59 {
		return TimeOfDay{}, false
	}
	return TimeOfDay{Hour: hour, Minute: minute}, true
}

// matchAnchorPhrase matches the FULL remaining word list against one of
// anchorPhrases, tolerating one leading "the" ("the competition starts").
func matchAnchorPhrase(words []string) (Anchor, bool) {
	if len(words) > 0 && words[0] == "the" {
		return matchAnchorPhrase(words[1:])
	}
	a, ok := anchorPhrases[strings.Join(words, " ")]
	return a, ok
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ",", " , ")
	return strings.Fields(s)
}

// Parse interprets a schedule expression per this package's grammar.
// Unrecognized input is a real error naming what was expected.
func Parse(s string) (Expr, error) {
	raw := s
	words := tokenize(s)
	if len(words) > 0 && fillerLead[words[0]] {
		words = words[1:]
	}
	if len(words) == 0 {
		return Expr{}, fmt.Errorf("empty schedule expression")
	}

	if e, ok, err := parseDailyAt(words, raw); ok || err != nil {
		return e, err
	}
	if e, ok, err := parseInterval(words, raw); ok || err != nil {
		return e, err
	}
	if e, ok, err := parseAnchored(words, raw); ok || err != nil {
		return e, err
	}
	return Expr{}, fmt.Errorf(
		"unrecognized schedule expression %q -- expected one of: "+
			`"every [N] <minute|hour|day>(s) [before|after <time or anchor>]", `+
			`"every day at <time>[, <time>...]", or `+
			`"[N <unit>] before|after <anchor>" `+
			"(anchor is one of: competition start, competition end, access opens, access closes)",
		raw,
	)
}

// parseDailyAt matches "every day at <time>[, <time>...]" exactly --
// checked first so "every day at ..." is never mistaken for a bare
// "every day" interval (see parseInterval's own note on this).
func parseDailyAt(words []string, raw string) (Expr, bool, error) {
	if len(words) < 4 || words[0] != "every" || words[1] != "day" || words[2] != "at" {
		return Expr{}, false, nil
	}
	var times []TimeOfDay
	for _, w := range words[3:] {
		if w == "," || w == "and" {
			continue
		}
		t, ok := parseTimeOfDay(w)
		if !ok {
			return Expr{}, true, fmt.Errorf("%q isn't a recognized time of day in %q (expected e.g. \"10:00am\" or \"14:00\")", w, raw)
		}
		times = append(times, t)
	}
	if len(times) == 0 {
		return Expr{}, true, fmt.Errorf("%q needs at least one time after \"at\"", raw)
	}
	return Expr{Kind: KindDailyAt, Raw: raw, Times: times}, true, nil
}

// parseInterval matches "every [N] <unit>(s) [before|after <bound>]".
// Falls through (ok=false) rather than erroring when the second token
// isn't a recognized unit -- that lets a malformed "every day at ..."
// (caught first by parseDailyAt when well-formed) or genuinely
// unrelated input reach Parse's final, single "unrecognized" error
// instead of two different parsers arguing about which one it almost was.
func parseInterval(words []string, raw string) (Expr, bool, error) {
	if len(words) < 2 || words[0] != "every" {
		return Expr{}, false, nil
	}
	i := 1
	n := 1
	if v, err := strconv.Atoi(words[i]); err == nil {
		n = v
		i++
	}
	if i >= len(words) {
		return Expr{}, false, nil
	}
	unit, ok := unitWords[words[i]]
	if !ok {
		return Expr{}, false, nil
	}
	i++
	if n <= 0 {
		return Expr{}, true, fmt.Errorf("interval count must be positive in %q", raw)
	}
	expr := Expr{Kind: KindInterval, Raw: raw, N: n, Unit: unit}
	if i >= len(words) {
		return expr, true, nil
	}
	dir := words[i]
	if dir != "before" && dir != "after" {
		return Expr{}, true, fmt.Errorf("unexpected %q after the interval in %q (expected \"before\" or \"after\")", dir, raw)
	}
	i++
	boundWords := words[i:]
	if len(boundWords) == 0 {
		return Expr{}, true, fmt.Errorf("expected a time or an anchor after %q in %q", dir, raw)
	}
	if a, ok := matchAnchorPhrase(boundWords); ok {
		expr.BoundDirection = Direction(dir)
		expr.BoundAnchor = a
		return expr, true, nil
	}
	if len(boundWords) == 1 {
		if t, ok := parseTimeOfDay(boundWords[0]); ok {
			expr.BoundDirection = Direction(dir)
			expr.BoundTime = &t
			return expr, true, nil
		}
	}
	return Expr{}, true, fmt.Errorf("%q after %q in %q isn't a recognized time or anchor", strings.Join(boundWords, " "), dir, raw)
}

// parseAnchored matches "[N <unit>(s)] before|after <anchor>". Falls
// through (ok=false) whenever the shape genuinely doesn't look like
// this clause kind, so Parse's final error covers it once instead of
// three parsers each reporting a different guess.
func parseAnchored(words []string, raw string) (Expr, bool, error) {
	i := 0
	n := 0
	var unit time.Duration
	if v, err := strconv.Atoi(words[0]); err == nil {
		if len(words) < 2 {
			return Expr{}, false, nil
		}
		u, ok := unitWords[words[1]]
		if !ok {
			return Expr{}, false, nil
		}
		n, unit, i = v, u, 2
	}
	if i >= len(words) {
		return Expr{}, false, nil
	}
	dir := words[i]
	if dir != "before" && dir != "after" {
		return Expr{}, false, nil
	}
	i++
	anchorWords := words[i:]
	a, ok := matchAnchorPhrase(anchorWords)
	if !ok {
		if i == 1 {
			// Bare "before"/"after" with no leading N/unit and no
			// recognized anchor -- genuinely not this clause kind;
			// something else entirely (or garbage), so let the
			// top-level error fire instead of guessing.
			return Expr{}, false, nil
		}
		return Expr{}, true, fmt.Errorf("expected an anchor after %q in %q (one of: competition start, competition end, access opens, access closes)", dir, raw)
	}
	if n < 0 {
		return Expr{}, true, fmt.Errorf("offset must not be negative in %q", raw)
	}
	return Expr{Kind: KindAnchored, Raw: raw, N: n, Unit: unit, Direction: Direction(dir), Anchor: a}, true, nil
}
