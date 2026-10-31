package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// The power poll records where each listed object lives -- networks too -- and
// an object the listing left out keeps the location it had.
func TestPowerPollRecordsLocation(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "location")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 3})
	mk := func(kind, name, ref, loc string) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: kind, ObjectName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = 'running', external_ref = $2, deployed_at = now() - interval '1 hour', instance_location = $3 WHERE id = $1", o.ID, ref, loc); err != nil {
			t.Fatal(err)
		}
		return o
	}
	web := mk("host", "web", "lf-web", "")
	lan := mk("network", "lan", "lf-lan", "")
	db1 := mk("host", "db", "lf-db", "incus-01")

	states := map[string]string{"lf-web": "running", "lf-db": "running"}
	locations := map[string]string{"lf-web": "incus-03", "lf-lan": "incus-03"}
	if err := applyInstanceStates(ctx, pool, build.ID, states, locations, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		o    db.DeployedObject
		want string
	}{{web, "incus-03"}, {lan, "incus-03"}, {db1, "incus-01"}} {
		got, _ := q.GetDeployedObject(ctx, c.o.ID)
		if got.InstanceLocation != c.want {
			t.Errorf("%s: location %q, want %q", c.o.ObjectName, got.InstanceLocation, c.want)
		}
	}
}
