package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// recoverFixture is one team in a build, with helpers to make hosts in a given
// failure state.
type recoverFixture struct {
	t     *testing.T
	pool  *pgxpool.Pool
	q     *db.Queries
	build db.Build
	team  db.Team
}

func newRecoverFixture(t *testing.T, name, buildStatus string) *recoverFixture {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, name)
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: buildStatus}); err != nil {
		t.Fatal(err)
	}
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	return &recoverFixture{t: t, pool: pool, q: q, build: build, team: team}
}

// host makes a host in status with failure kind (nil = none), failed 20
// minutes ago, deployed an hour ago, steps materialized, and the given number
// of open steps.
func (f *recoverFixture) host(name, status string, kind *string, openSteps int) db.DeployedObject {
	ctx := context.Background()
	o, err := f.q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: f.team.ID, Kind: "host", ObjectName: name, AsName: db.StrPtr(name)})
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE deployed_object SET status = $2, failure_kind = $3, failed_at = now() - interval '20 minutes',
		last_error = CASE WHEN $2 = 'build_failed' THEN 'failed' END, deployed_at = now() - interval '1 hour', steps_materialized_at = now() WHERE id = $1`,
		o.ID, status, kind); err != nil {
		f.t.Fatal(err)
	}
	for i := 0; i < openSteps; i++ {
		if _, err := f.q.CreateAgentTask(ctx, db.CreateAgentTaskParams{DeployedObjectID: o.ID, StepIndex: int32(i), Command: "execute", Payload: []byte("{}"), StepGroup: optInt32(i)}); err != nil {
			f.t.Fatal(err)
		}
	}
	got, err := f.q.GetDeployedObject(ctx, o.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	return got
}

func (f *recoverFixture) heartbeat(o db.DeployedObject, ago time.Duration) {
	if _, err := f.pool.Exec(context.Background(), "INSERT INTO agent_session (deployed_object_id, cert_fingerprint, last_heartbeat_at) VALUES ($1, $2, now() - $3::interval)", o.ID, "fp-"+o.ID.String(), ago.String()); err != nil {
		f.t.Fatal(err)
	}
}

func (f *recoverFixture) status(o db.DeployedObject) db.DeployedObject {
	got, err := f.q.GetDeployedObject(context.Background(), o.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	return got
}

// TestLifecycleRecoversHostsWhoseAgentCameBack: a host failed for a lost or
// never-seen agent goes back to building once that agent heartbeats after the
// failure, and its failed build re-enters building. A host failed by a step,
// or whose only heartbeat predates its failure, stays failed.
func TestLifecycleRecoversHostsWhoseAgentCameBack(t *testing.T) {
	f := newRecoverFixture(t, "agent-recover", "failed")
	ctx := context.Background()

	lost := f.host("web-lost", "build_failed", db.StrPtr(FailureAgentLost), 3)
	f.heartbeat(lost, 5*time.Second) // back
	never := f.host("web-never", "build_failed", db.StrPtr(FailureAgentNeverCheckedIn), 2)
	f.heartbeat(never, 5*time.Second) // finally connected
	stillGone := f.host("web-gone", "build_failed", db.StrPtr(FailureAgentLost), 3)
	f.heartbeat(stillGone, 30*time.Minute) // last seen before it was failed
	stepFailed := f.host("web-step", "build_failed", nil, 3)
	f.heartbeat(stepFailed, 5*time.Second) // agent fine, a step failed: operator's call

	if err := AdvanceObjectLifecycle(ctx, f.pool, f.build.ID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		o    db.DeployedObject
		want string
	}{{lost, "building"}, {never, "building"}, {stillGone, "build_failed"}, {stepFailed, "build_failed"}} {
		o, want := c.o, c.want
		got := f.status(o)
		if got.Status != want {
			t.Errorf("%s: status %s, want %s (error %v)", *o.AsName, got.Status, want, got.LastError)
		}
		if want == "building" && (got.FailureKind != nil || got.LastError != nil || got.FailedAt.Valid) {
			t.Errorf("%s: recovered but kept failure_kind=%v last_error=%v", *o.AsName, got.FailureKind, got.LastError)
		}
	}
	b, err := f.q.GetBuild(ctx, f.build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != "building" {
		t.Errorf("build status %s, want building (a host recovered)", b.Status)
	}
	var events int
	f.pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE build_id = $1 AND kind = 'lifecycle.recovered'", f.build.ID).Scan(&events)
	if events != 2 {
		t.Errorf("lifecycle.recovered events = %d, want 2", events)
	}
}

// TestLifecycleRecoveredHostWithNoStepsLeftFinishes: the common false alarm --
// the agent came back and ran every step while the host showed failed. It
// lands on finished in the same pass.
func TestLifecycleRecoveredHostWithNoStepsLeftFinishes(t *testing.T) {
	f := newRecoverFixture(t, "agent-recover-done", "building")
	o := f.host("web-done", "build_failed", db.StrPtr(FailureAgentLost), 0)
	f.heartbeat(o, 5*time.Second)
	if err := AdvanceObjectLifecycle(context.Background(), f.pool, f.build.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.status(o); got.Status != "finished" {
		t.Fatalf("status %s, want finished", got.Status)
	}
}

// TestRecoverCascade: a host failed because its dependency or wait_for target
// failed recovers once that target is no longer failed, and not before.
func TestRecoverCascade(t *testing.T) {
	f := newRecoverFixture(t, "cascade-recover", "building")
	ctx := context.Background()
	placed := map[string]bool{"kali": true}

	dep := f.host("dep", "build_failed", db.StrPtr(FailureDependency), 0)
	if _, err := f.pool.Exec(ctx, "UPDATE deployed_object SET steps_materialized_at = NULL WHERE id = $1", dep.ID); err != nil {
		t.Fatal(err)
	}
	dep = f.status(dep)
	if st, err := recoverCascade(ctx, f.q, f.build.ID, dep, []string{"kali"}, placed, map[string]bool{"kali": true}, nil); err != nil || st != "" {
		t.Fatalf("dependency still failed: recovered to %q (err %v), want no change", st, err)
	}
	if st, err := recoverCascade(ctx, f.q, f.build.ID, dep, []string{"kali"}, placed, map[string]bool{}, nil); err != nil || st != "running" {
		t.Fatalf("dependency recovered: got %q (err %v), want running (steps not yet queued)", st, err)
	}

	gated := f.host("gated", "build_failed", db.StrPtr(FailureGate), 0)
	if _, err := f.q.CreateAgentTask(ctx, db.CreateAgentTaskParams{DeployedObjectID: gated.ID, StepIndex: 0, Command: "execute", Payload: []byte("{}"), Status: "blocked", WaitFor: []byte(`[{"host":"kali"}]`), StepGroup: optInt32(0)}); err != nil {
		t.Fatal(err)
	}
	gatesWith := func(failed bool) *teamGates {
		return &teamGates{placed: placed, finished: map[string]bool{}, failed: map[string]bool{"kali": failed}}
	}
	if st, err := recoverCascade(ctx, f.q, f.build.ID, gated, nil, placed, nil, gatesWith(true)); err != nil || st != "" {
		t.Fatalf("gate target still failed: recovered to %q (err %v), want no change", st, err)
	}
	if st, err := recoverCascade(ctx, f.q, f.build.ID, gated, nil, placed, nil, gatesWith(false)); err != nil || st != "building" {
		t.Fatalf("gate target recovered: got %q (err %v), want building", st, err)
	}
	if got := f.status(gated); got.Status != "building" || got.FailureKind != nil {
		t.Fatalf("gated host: status %s kind %v, want building with no failure kind", got.Status, got.FailureKind)
	}
}
