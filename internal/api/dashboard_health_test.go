package api

import (
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestClassifyHealthFoldsPowerState pins the point of folding builder power
// state into the health buckets: a down instance is an infra failure, not a
// "failed check-in", and a check-in failure is reserved for a host that's
// actually up.
func TestClassifyHealthFoldsPowerState(t *testing.T) {
	none := map[string]bool{}
	host := func(status, power, agent string) objectView {
		o := objectView{DeployedObject: db.DeployedObject{Kind: "host", Status: status, PowerState: power}}
		if agent != "" {
			o.Agent = &agentHealth{Status: agent}
		}
		return o
	}

	cases := []struct {
		name string
		o    objectView
		want string
	}{
		{"finished + agent missing is a real check-in failure", host("finished", "running", "missing"), healthFailedCheckin},
		{"stopped instance is infra, even with a 'healthy' agent reading", host("finished", "stopped", "healthy"), healthFailedInfra},
		{"missing instance is infra", host("finished", "missing", "missing"), healthFailedInfra},
		{"unpolled power falls back to the agent signal", host("finished", "", "missing"), healthFailedCheckin},
		{"finished + healthy is completed", host("finished", "running", "healthy"), healthCompleted},
		{"a failed deploy is infra regardless of power", host("deploy_failed", "running", "healthy"), healthFailedInfra},
		{"a failed build step is a failed-steps outcome", host("build_failed", "running", "healthy"), healthFailedSteps},
		{"a failed validator is a failed-steps outcome", host("invalid", "running", "healthy"), healthFailedSteps},
		{"an object still building is in progress", host("building", "running", "booting"), healthInProgress},
	}
	for _, c := range cases {
		if got := classifyHealth(c.o, none); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
