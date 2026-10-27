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
	if err := applyInstanceStates(ctx, pool, build.ID, map[string]string{"lf-web-listed": "running"}, snapshotAt, nil); err != nil {
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

// stubStater answers single-instance checks from a fixed table.
type stubStater map[string]absenceCheck

func (s stubStater) InstanceState(_ context.Context, ref string) (string, bool, error) {
	c, ok := s[ref]
	if !ok {
		return "", false, nil
	}
	return c.state, c.exists, c.err
}

// TestPowerStateConfirmsLeftOutInstances: a hoster listing that leaves out a
// running instance (a busy cluster member that didn't answer in time) doesn't
// mark it missing. Each left-out instance is checked directly: one that exists
// keeps its real state, one the hoster says is gone is missing, and one that
// can't be checked is left as it was.
func TestPowerStateConfirmsLeftOutInstances(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "power-confirm")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	mk := func(as string) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = 'finished', external_ref = $2, deployed_at = now() - interval '1 hour', power_state = 'running' WHERE id = $1", o.ID, "lf-"+as); err != nil {
			t.Fatal(err)
		}
		return o
	}
	omitted, stopped, gone, unsure := mk("web-omitted"), mk("web-stopped"), mk("web-gone"), mk("web-unsure")
	stater := stubStater{
		"lf-web-omitted": {state: "running", exists: true},
		"lf-web-stopped": {state: "stopped", exists: true},
		"lf-web-unsure":  {err: context.DeadlineExceeded},
	}
	if err := applyInstanceStates(ctx, pool, build.ID, map[string]string{}, time.Now(), stater); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		o    db.DeployedObject
		want string
	}{{omitted, "running"}, {stopped, "stopped"}, {gone, "missing"}, {unsure, "running"}} {
		got, _ := q.GetDeployedObject(ctx, c.o.ID)
		if got.PowerState != c.want {
			t.Errorf("%s: power state %q, want %q", *c.o.AsName, got.PowerState, c.want)
		}
	}
}
