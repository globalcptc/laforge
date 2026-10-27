package orchestrator

import (
	"context"
	"sync"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

// TestPowerJobPreparesThenRunsInTheBackground: PreparePower matches the target
// without touching the hoster (hosts never deployed are failed right there),
// and Run reports each host as it finishes and records its new power state at
// once, so the UI follows a bulk action that runs after the request answered.
func TestPowerJobPreparesThenRunsInTheBackground(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "power-job")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	var ids []string
	mk := func(as string, deployed bool) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		if deployed {
			pool.Exec(ctx, "UPDATE deployed_object SET status = 'finished', external_ref = $2, power_state = 'running' WHERE id = $1", o.ID, "lf-"+as)
		}
		ids = append(ids, o.ID.String())
		return o
	}
	up1, up2, pending := mk("web-1", true), mk("web-2", true), mk("web-3", false)

	job, err := PreparePower(ctx, pool, build.ID, AdHocTarget{IDs: ids}, builder.PowerStop, false)
	if err != nil {
		t.Fatal(err)
	}
	if job.Pending() != 2 || len(job.Results) != 3 {
		t.Fatalf("prepared %d to run of %d matched, want 2 of 3", job.Pending(), len(job.Results))
	}
	for _, r := range job.Results {
		if (r.ObjectID == pending.ID.String()) != (r.Err != "") {
			t.Fatalf("%s: error %q before running; only the undeployed host should fail up front", r.Name, r.Err)
		}
	}

	var mu sync.Mutex
	reported := map[string]bool{}
	results := job.Run(ctx, func(r PowerResult) {
		mu.Lock()
		reported[r.ObjectID] = true
		mu.Unlock()
	})
	if len(results) != 3 || !reported[up1.ID.String()] || !reported[up2.ID.String()] || reported[pending.ID.String()] {
		t.Fatalf("reported %v; want each deployed host reported once as it finished", reported)
	}
	for _, o := range []db.DeployedObject{up1, up2} {
		got, _ := q.GetDeployedObject(ctx, o.ID)
		if got.PowerState != builder.PowerStateStopped {
			t.Errorf("%s: power state %q after a stop, want stopped right away", *o.AsName, got.PowerState)
		}
	}
}
