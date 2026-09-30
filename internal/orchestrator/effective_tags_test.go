package orchestrator

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

// TestEffectiveTagsCascade proves the deployed_object tag cascade: environment,
// then network, then the scripts an object runs, then the object's own tags --
// most specific wins a clash, less-specific-only keys survive.
func TestEffectiveTagsCascade(t *testing.T) {
	c := &loader.Content{
		Environments: []loader.Environment{{Name: "e", Tags: map[string]string{
			"scope": "env", "envonly": "1",
		}}},
		Networks: []loader.Network{{Name: "n", Tags: map[string]string{
			"scope": "net", "netonly": "1",
		}}},
		Scripts: []loader.Script{{Name: "harden", Tags: map[string]string{
			"scope": "script", "scriptonly": "1",
		}}},
	}
	steps := []loader.Step{{"script": "harden"}}
	own := map[string]string{"scope": "host", "hostonly": "1"}

	got := effectiveTags(c, "e", "n", "web", steps, nil, own)

	// Most specific (the host's own) wins the shared key.
	if got["scope"] != "host" {
		t.Errorf("scope = %q, want host (object tags win the cascade)", got["scope"])
	}
	// Every level's unique keys survive.
	for k, want := range map[string]string{
		"envonly": "1", "netonly": "1", "scriptonly": "1", "hostonly": "1",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, want %q (level dropped from cascade)", k, got[k], want)
		}
	}
}

// TestEffectiveTagsScriptFromSchedule proves a script referenced only by a
// schedule entry (not a step) still contributes its tags.
func TestEffectiveTagsScriptFromSchedule(t *testing.T) {
	c := &loader.Content{
		Scripts: []loader.Script{{Name: "nightly", Tags: map[string]string{"job": "nightly"}}},
	}
	sched := []loader.Step{{"script": "nightly", "when": "every day at 2:00am"}}
	got := effectiveTags(c, "e", "n", "web", nil, sched, nil)
	if got["job"] != "nightly" {
		t.Errorf("schedule-referenced script tag not applied: %v", got)
	}
}
