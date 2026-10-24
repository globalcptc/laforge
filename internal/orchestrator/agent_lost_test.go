package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// TestLifecycleFailsHostsWhoseAgentIsLost: a building host whose agent has
// gone silent with steps still to run, and a running host whose agent never
// connected, are failed with the reason -- while a host whose agent is still
// heartbeating, and one that only recently stopped, are left alone.
func TestLifecycleFailsHostsWhoseAgentIsLost(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "agent-lost")
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "building"}); err != nil {
		t.Fatal(err)
	}
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	mk := func(as, status string, deployedAgo time.Duration, heartbeatAgo *time.Duration, openSteps int) db.DeployedObject {
		o, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "web", AsName: db.StrPtr(as)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = $2, deployed_at = now() - $3::interval, steps_materialized_at = now() WHERE id = $1", o.ID, status, deployedAgo.String()); err != nil {
			t.Fatal(err)
		}
		if heartbeatAgo != nil {
			if _, err := pool.Exec(ctx, "INSERT INTO agent_session (deployed_object_id, cert_fingerprint, first_seen_at, last_heartbeat_at) VALUES ($1, $3, now() - $2::interval, now() - $2::interval)", o.ID, heartbeatAgo.String(), "fp-"+as); err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < openSteps; i++ {
			if _, err := q.CreateAgentTask(ctx, db.CreateAgentTaskParams{DeployedObjectID: o.ID, StepIndex: int32(i), Command: "execute", Payload: []byte("{}"), StepGroup: optInt32(i)}); err != nil {
				t.Fatal(err)
			}
		}
		return o
	}
	twoHours, fiveMinutes, fourHours := 2*time.Hour, 5*time.Minute, 4*time.Hour
	lost := mk("web-lost", "building", fourHours, &twoHours, 3)
	alive := mk("web-alive", "building", fourHours, &fiveMinutes, 3)
	never := mk("web-never", "running", fourHours, nil, 0)
	fresh := mk("web-fresh", "running", 10*time.Minute, nil, 0)

	if err := AdvanceObjectLifecycle(ctx, pool, build.ID); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"web-lost": "build_failed", "web-alive": "building", "web-never": "build_failed", "web-fresh": "running"}
	for _, o := range []db.DeployedObject{lost, alive, never, fresh} {
		got, _ := q.GetDeployedObject(ctx, o.ID)
		if got.Status != want[*o.AsName] {
			t.Fatalf("%s: status %s, want %s (error %v)", *o.AsName, got.Status, want[*o.AsName], got.LastError)
		}
		if got.Status == "build_failed" && (got.LastError == nil || *got.LastError == "") {
			t.Fatalf("%s failed without a reason", *o.AsName)
		}
	}
	var events int
	pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE build_id = $1 AND kind = 'lifecycle.build_failed'", build.ID).Scan(&events)
	if events != 2 {
		t.Fatalf("lifecycle.build_failed events = %d, want 2", events)
	}
}
