package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// TestPowerStateIgnoresHostsDeployedDuringTheListing: the hoster listing takes
// a while on a busy cluster, and hosts finish deploying meanwhile. A host
// deployed after the listing began can't be in it, so it must not be marked
// "missing" -- it keeps its state until the next pass. A host deployed before
// the listing and absent from it is still missing.
func TestPowerStateIgnoresHostsDeployedDuringTheListing(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "power-race")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	mk := func(as string, deployedAgo time.Duration) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = 'running', external_ref = $2, deployed_at = now() - $3::interval, power_state = 'running' WHERE id = $1",
			o.ID, "lf-"+as, deployedAgo.String()); err != nil {
			t.Fatal(err)
		}
		return o
	}
	old := mk("web-old", time.Hour)
	fresh := mk("web-fresh", 0)
	listed := mk("web-listed", time.Hour)

	// The listing began 30 seconds ago and sees only web-listed.
	snapshotAt := time.Now().Add(-30 * time.Second)
	if err := applyInstanceStates(ctx, pool, build.ID, map[string]string{"lf-web-listed": "running"}, snapshotAt); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		o    db.DeployedObject
		want string
	}{{old, "missing"}, {fresh, "running"}, {listed, "running"}} {
		got, _ := q.GetDeployedObject(ctx, c.o.ID)
		if got.PowerState != c.want {
			t.Errorf("%s: power state %q, want %q", *c.o.AsName, got.PowerState, c.want)
		}
	}
}
