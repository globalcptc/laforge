package api

import (
	"reflect"
	"testing"
)

// TestScheduleScriptsAttributed proves a script referenced only by a schedule
// entry (same {script, when} shape as a step) is picked up for findings
// attribution, and that steps + schedule script names merge without duplicates.
func TestScheduleScriptsAttributed(t *testing.T) {
	steps := []byte(`[{"script":"base"},{"script":"harden"}]`)
	schedule := []byte(`[{"script":"nightly-scan","when":"every day at 2:00am"},{"script":"harden","when":"every hour"}]`)

	// A schedule entry's own script is extracted just like a step's.
	if got := scriptNamesFromSteps(schedule); !reflect.DeepEqual(got, []string{"nightly-scan", "harden"}) {
		t.Fatalf("scriptNamesFromSteps(schedule) = %v, want [nightly-scan harden]", got)
	}

	// Merged: steps first, then schedule-only additions, no dup of "harden".
	got := mergeScriptNames(scriptNamesFromSteps(steps), scriptNamesFromSteps(schedule))
	want := []string{"base", "harden", "nightly-scan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged script names = %v, want %v (nightly-scan is schedule-only; harden not duplicated)", got, want)
	}
}
