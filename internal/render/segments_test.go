package render

import (
	"strings"
	"testing"
)

func TestMarkValueActions(t *testing.T) {
	out := markValueActions("a {{ .host.hostname }} b {{- .host.ip -}} c {{ if .x }}{{ .user }}{{ end }} {{/* note */}}")

	// Value actions (incl. one with trim markers, and one inside a block) wrapped.
	for _, want := range []string{
		`{{"` + markStart + `"}}{{ .host.hostname }}{{"` + markEnd + `"}}`,
		`{{"` + markStart + `"}}{{- .host.ip -}}{{"` + markEnd + `"}}`,
		`{{"` + markStart + `"}}{{ .user }}{{"` + markEnd + `"}}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing wrapped value action %q in:\n%s", want, out)
		}
	}
	// Control actions and comments are NOT wrapped.
	if strings.Contains(out, markStart+`"}}{{ if`) || strings.Contains(out, markStart+`"}}{{ end`) {
		t.Errorf("control action was wrapped:\n%s", out)
	}
	if strings.Contains(out, markStart+`"}}{{/* note */}}`) {
		t.Errorf("comment was wrapped:\n%s", out)
	}
}

func TestSplitSegments(t *testing.T) {
	segs := splitSegments("pre " + markStart + "VALUE" + markEnd + " mid " + markStart + "X" + markEnd)
	want := []Segment{
		{Text: "pre ", Templated: false},
		{Text: "VALUE", Templated: true},
		{Text: " mid ", Templated: false},
		{Text: "X", Templated: true},
	}
	if len(segs) != len(want) {
		t.Fatalf("segments = %+v, want %+v", segs, want)
	}
	var round strings.Builder
	for i, s := range segs {
		if s != want[i] {
			t.Errorf("segment %d = %+v, want %+v", i, s, want[i])
		}
		round.WriteString(s.Text)
	}
	// Concatenation reproduces the plain (marker-free) text.
	if round.String() != "pre VALUE mid X" {
		t.Errorf("round-trip = %q", round.String())
	}
}
