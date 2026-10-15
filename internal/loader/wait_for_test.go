package loader_test

import (
	"reflect"
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

// waitRepo is a DC with a milestone and a workstation whose steps are given.
func waitRepo(t *testing.T, wsSteps string, extra map[string]string) *loader.Content {
	t.Helper()
	files := map[string]string{
		"scripts/winbase.yaml":    scriptDoc("winbase"),
		"scripts/winbase.sh":      "true\n",
		"scripts/populate.yaml":   scriptDoc("populate"),
		"scripts/populate.sh":     "true\n",
		"scripts/join.yaml":       scriptDoc("join"),
		"scripts/join.sh":         "true\n",
		"scripts/reboot-rec.yaml": scriptDoc("reboot-rec"),
		"scripts/reboot-rec.sh":   "true\n",
		"hosts/dc.yaml": simpleHost("dc") + `  steps:
    - script: winbase
    - script: reboot-rec
    - script: populate
      milestone: domain-ready
    - script: reboot-rec
`,
		"hosts/ws.yaml": simpleHost("ws") + "  steps:\n" + wsSteps,
	}
	for k, v := range extra {
		files[k] = v
	}
	c, err := loader.Load(writeRepo(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

func TestWaitForValidReferencesLoadClean(t *testing.T) {
	c := waitRepo(t, `    - script: winbase
    - script: join
      wait_for:
        host: dc
        milestone: domain-ready
    - script: winbase
      wait_for:
        - host: dc
          step: populate
          rebuild: false
        - host: dc
`, nil)
	if len(c.Errors) != 0 {
		t.Fatalf("expected clean load, got: %v", errorMessages(c))
	}
	ws := findHost(t, c, "ws")
	if got := ws.Steps[1].WaitFor(); len(got) != 1 || got[0].Host != "dc" || got[0].Milestone != "domain-ready" || !got[0].RebuildsWithTarget() {
		t.Fatalf("single-map wait_for decoded as %+v", got)
	}
	got := ws.Steps[2].WaitFor()
	if len(got) != 2 || got[0].Step != "populate" || got[0].RebuildsWithTarget() || got[1].Milestone != "" || !got[1].RebuildsWithTarget() {
		t.Fatalf("list wait_for decoded as %+v", got)
	}
	// wait_for/milestone ride beside the action; they are never the action.
	if ws.Steps[1].ActionKey() != "script" {
		t.Fatalf("ActionKey = %q, want script", ws.Steps[1].ActionKey())
	}
	if dc := findHost(t, c, "dc"); dc.Steps[2].ActionKey() != "script" || dc.Steps[2].Milestone() != "domain-ready" {
		t.Fatalf("milestone step: action %q milestone %q", dc.Steps[2].ActionKey(), dc.Steps[2].Milestone())
	}
}

func TestWaitForBadReferencesAreCaught(t *testing.T) {
	cases := []struct {
		name, steps, want string
	}{
		{"unknown host", "    - script: join\n      wait_for: { host: nope }\n", `references "nope"`},
		{"unknown milestone", "    - script: join\n      wait_for: { host: dc, milestone: dns-ready }\n", `declares no milestone "dns-ready"`},
		{"unknown step", "    - script: join\n      wait_for: { host: dc, step: join }\n", `has no script step "join"`},
		{"ambiguous step", "    - script: join\n      wait_for: { host: dc, step: reboot-rec }\n", `more than once`},
		{"self wait", "    - script: join\n      wait_for: { host: ws }\n", `waits on itself`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := waitRepo(t, tc.steps, nil)
			if !containsSubstring(errorMessages(c), tc.want) {
				t.Fatalf("expected an error containing %q, got: %v", tc.want, errorMessages(c))
			}
		})
	}
}

func TestWaitForSchemaRejectsMilestoneAndStepTogether(t *testing.T) {
	c := waitRepo(t, "    - script: join\n      wait_for: { host: dc, milestone: domain-ready, step: populate }\n", nil)
	if len(c.Errors) == 0 {
		t.Fatal("expected milestone+step together to be rejected")
	}
}

func TestWaitForRejectedOnScheduleEntry(t *testing.T) {
	c := waitRepo(t, "    - script: join\n", map[string]string{
		"hosts/sched.yaml": simpleHost("sched") + `  schedule:
    - when: every hour
      script: join
      wait_for: { host: dc }
`,
	})
	if !containsSubstring(errorMessages(c), "wait_for") {
		t.Fatalf("expected schedule wait_for to be rejected, got: %v", errorMessages(c))
	}
}

func TestDuplicateMilestoneIsCaught(t *testing.T) {
	c := waitRepo(t, `    - script: winbase
      milestone: ready
    - script: join
      milestone: ready
`, nil)
	if !containsSubstring(errorMessages(c), `milestone "ready" on more than one step`) {
		t.Fatalf("expected duplicate milestone error, got: %v", errorMessages(c))
	}
}

func TestWaitForCycleIsCaught(t *testing.T) {
	// dc waits (host-level) on ws while ws waits (step-level) on dc.
	c := waitRepo(t, "    - script: join\n      wait_for: { host: dc, milestone: domain-ready }\n", nil)
	if len(c.Errors) != 0 {
		t.Fatalf("baseline should be clean: %v", errorMessages(c))
	}
	files := map[string]string{
		"hosts/dc.yaml": simpleHost("dc") + "  depends_on: [ws]\n  steps:\n    - script: populate\n      milestone: domain-ready\n",
	}
	c = waitRepo(t, "    - script: join\n      wait_for: { host: dc, milestone: domain-ready }\n", files)
	if !containsSubstring(errorMessages(c), "depends_on/wait_for cycle") {
		t.Fatalf("expected a cycle error, got: %v", errorMessages(c))
	}
}

func TestWaitForInheritedThroughExtends(t *testing.T) {
	c := waitRepo(t, "    - script: join\n      wait_for: { host: dc, milestone: domain-ready }\n", map[string]string{
		"hosts/ws2.yaml": "host:\n  name: ws2\n  extends: ws\n",
	})
	if len(c.Errors) != 0 {
		t.Fatalf("expected clean load, got: %v", errorMessages(c))
	}
	if got := findHost(t, c, "ws2").Steps[0].WaitFor(); len(got) != 1 || got[0].Host != "dc" {
		t.Fatalf("extending host's wait_for = %+v", got)
	}
}

func TestDependentsFollowWaitForUnlessRebuildFalse(t *testing.T) {
	c := waitRepo(t, "    - script: join\n      wait_for: { host: dc, milestone: domain-ready }\n", map[string]string{
		"hosts/vdi.yaml": simpleHost("vdi") + "  steps:\n    - script: join\n      wait_for: { host: dc, rebuild: false }\n",
	})
	if len(c.Errors) != 0 {
		t.Fatalf("expected clean load, got: %v", errorMessages(c))
	}
	if got := c.Dependents([]string{"dc"}); !reflect.DeepEqual(got, []string{"dc", "ws"}) {
		t.Fatalf("Dependents(dc) = %v, want [dc ws] (vdi opted out with rebuild: false)", got)
	}
	depth := c.DependencyDepth()
	if depth["dc"] != 0 || depth["ws"] != 1 || depth["vdi"] != 1 {
		t.Fatalf("DependencyDepth = %v; every wait_for target deploys first, whatever its rebuild", depth)
	}
	// Deploy priority counts every waiter: vdi is on dc's critical path even
	// though a dc rebuild doesn't cascade to it.
	if counts := c.DependentCounts(); counts["dc"] != 2 || counts["ws"] != 0 {
		t.Fatalf("DependentCounts = %v, want dc=2 (ws and vdi both wait on it)", counts)
	}
}
