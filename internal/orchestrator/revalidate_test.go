package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// validateHost makes a host in status with authored steps: an execute, then
// nValidate validate tasks, all done (its build finished them).
func (f *recoverFixture) validateHost(name, status string, nValidate int) db.DeployedObject {
	ctx := context.Background()
	o := f.host(name, status, nil, 0)
	if _, err := f.q.CreateAgentTask(ctx, db.CreateAgentTaskParams{DeployedObjectID: o.ID, StepIndex: 0, Command: "execute", Payload: []byte(`{}`), StepGroup: optInt32(0)}); err != nil {
		f.t.Fatal(err)
	}
	for i := 1; i <= nValidate; i++ {
		if _, err := f.q.CreateAgentTask(ctx, db.CreateAgentTaskParams{DeployedObjectID: o.ID, StepIndex: int32(i), Command: "validate", Payload: []byte(`{"checks":[{"kind":"port_open","args":{"port":22}}]}`), StepGroup: optInt32(0)}); err != nil {
			f.t.Fatal(err)
		}
	}
	if _, err := f.pool.Exec(ctx, "UPDATE agent_task SET status = 'done' WHERE deployed_object_id = $1", o.ID); err != nil {
		f.t.Fatal(err)
	}
	return f.status(o)
}

// report simulates the agent running a run's checks: each passes unless its
// position is in failIdx.
func (f *recoverFixture) report(run db.ValidationRun, failIdx ...int) {
	ctx := context.Background()
	tasks, err := f.q.ListValidationRunTasks(ctx, run.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	fail := map[int]bool{}
	for _, i := range failIdx {
		fail[i] = true
	}
	for i, t := range tasks {
		status := "done"
		if fail[i] {
			status = "failed"
		}
		if _, err := f.pool.Exec(ctx, "UPDATE agent_task SET status = $2, finished_at = now() WHERE id = $1", t.ID, status); err != nil {
			f.t.Fatal(err)
		}
		if err := f.q.RecordValidatorResult(ctx, db.RecordValidatorResultParams{AgentTaskID: t.ID, Kind: "port_open", Args: []byte(`{"port":22}`), Passed: !fail[i], Message: db.StrPtr("checked")}); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *recoverFixture) start(o db.DeployedObject) db.ValidationRun {
	run, err := StartValidationRun(context.Background(), f.pool, newReportID(), f.build.ID, o, pgtype.UUID{})
	if err != nil {
		f.t.Fatalf("StartValidationRun(%s): %v", *o.AsName, err)
	}
	return run
}

func newReportID() pgtype.UUID {
	var u pgtype.UUID
	u.Valid = true
	for i := range u.Bytes {
		u.Bytes[i] = byte(i*7 + 3)
	}
	return u
}

func (f *recoverFixture) run(run db.ValidationRun) db.ValidationRun {
	var got db.ValidationRun
	if err := f.pool.QueryRow(context.Background(), "SELECT status, status_after FROM validation_run WHERE id = $1", run.ID).Scan(&got.Status, &got.StatusAfter); err != nil {
		f.t.Fatal(err)
	}
	return got
}

// TestStartValidationRunCopiesOnlyValidators: a revalidation queues a copy of
// each validate task -- nothing else -- in the ad-hoc lane, one attempt each;
// a second one while it runs is refused, and a host with no validators can't
// be revalidated.
func TestStartValidationRunCopiesOnlyValidators(t *testing.T) {
	f := newRecoverFixture(t, "revalidate-start", "finished")
	ctx := context.Background()
	o := f.validateHost("web", "finished", 2)

	run := f.start(o)
	tasks, err := f.q.ListValidationRunTasks(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("copied %d tasks, want 2 (the validate steps only)", len(tasks))
	}
	var adHoc, oneAttempt int
	f.pool.QueryRow(ctx, "SELECT count(*) FILTER (WHERE ad_hoc), count(*) FILTER (WHERE max_attempts = 1) FROM agent_task WHERE validation_run_id = $1", run.ID).Scan(&adHoc, &oneAttempt)
	if adHoc != 2 || oneAttempt != 2 || tasks[0].Status != "pending" {
		t.Fatalf("copies: ad_hoc=%d max_attempts1=%d first status %s; want both ad-hoc, one attempt, pending", adHoc, oneAttempt, tasks[0].Status)
	}
	if run.StatusBefore != "finished" {
		t.Fatalf("status_before %q, want finished", run.StatusBefore)
	}
	if _, err := StartValidationRun(ctx, f.pool, newReportID(), f.build.ID, o, pgtype.UUID{}); !errors.Is(err, ErrValidationRunning) {
		t.Fatalf("second start: %v, want ErrValidationRunning", err)
	}
	none := f.validateHost("db", "finished", 0)
	if _, err := StartValidationRun(ctx, f.pool, newReportID(), f.build.ID, none, pgtype.UUID{}); !errors.Is(err, ErrNoValidators) {
		t.Fatalf("no validators: %v, want ErrNoValidators", err)
	}
}

// TestSettleValidationRuns: a finished run sets the host finished <-> invalid
// and nothing else; a run whose tasks vanished is canceled.
func TestSettleValidationRuns(t *testing.T) {
	f := newRecoverFixture(t, "revalidate-settle", "finished")
	ctx := context.Background()

	goesInvalid := f.validateHost("goes-invalid", "finished", 2)
	r1 := f.start(goesInvalid)
	f.report(r1, 1)

	goesFinished := f.validateHost("goes-finished", "invalid", 2)
	r2 := f.start(goesFinished)
	f.report(r2)

	stepFailed := f.validateHost("step-failed", "build_failed", 1)
	r3 := f.start(stepFailed)
	f.report(r3)

	rebuilt := f.validateHost("rebuilt", "finished", 1)
	r4 := f.start(rebuilt)
	if err := f.q.DeleteAgentTasksForObject(ctx, rebuilt.ID); err != nil {
		t.Fatal(err)
	}

	still := f.validateHost("still-running", "finished", 2)
	r5 := f.start(still)

	changed, err := SettleValidationRuns(ctx, f.q, f.build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false, want true (two hosts flipped)")
	}
	for _, c := range []struct {
		o          db.DeployedObject
		run        db.ValidationRun
		host, want string
	}{
		{goesInvalid, r1, "invalid", "failed"},
		{goesFinished, r2, "finished", "passed"},
		{stepFailed, r3, "build_failed", "passed"},
		{rebuilt, r4, "finished", "canceled"},
		{still, r5, "finished", "running"},
	} {
		if got := f.status(c.o).Status; got != c.host {
			t.Errorf("%s: host %s, want %s", *c.o.AsName, got, c.host)
		}
		if got := f.run(c.run).Status; got != c.want {
			t.Errorf("%s: run %s, want %s", *c.o.AsName, got, c.want)
		}
	}
	// The host's own build-time validate steps are untouched.
	var authoredDone int
	f.pool.QueryRow(ctx, "SELECT count(*) FROM agent_task WHERE deployed_object_id = $1 AND validation_run_id IS NULL AND status = 'done'", goesInvalid.ID).Scan(&authoredDone)
	if authoredDone != 3 {
		t.Errorf("authored done steps on %s = %d, want 3 (untouched)", *goesInvalid.AsName, authoredDone)
	}
}

// TestSettleValidationRunTimesOut: checks the agent never ran are dropped and
// the run times out, leaving the host as it was.
func TestSettleValidationRunTimesOut(t *testing.T) {
	f := newRecoverFixture(t, "revalidate-timeout", "finished")
	ctx := context.Background()
	o := f.validateHost("web", "finished", 2)
	run := f.start(o)
	if _, err := f.pool.Exec(ctx, "UPDATE validation_run SET created_at = now() - interval '1 hour' WHERE id = $1", run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := SettleValidationRuns(ctx, f.q, f.build.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.run(run).Status; got != "timed_out" {
		t.Fatalf("run %s, want timed_out", got)
	}
	var left int
	f.pool.QueryRow(ctx, "SELECT count(*) FROM agent_task WHERE validation_run_id = $1", run.ID).Scan(&left)
	if left != 0 {
		t.Fatalf("%d unrun checks left queued, want 0", left)
	}
	if got := f.status(o).Status; got != "finished" {
		t.Fatalf("host %s, want finished (unchanged)", got)
	}
}

// TestRevalidationDoesNotHoldOrFailTheBuild: a building host whose authored
// steps are all done finishes even while a revalidation is queued on it, and
// a failing revalidation check doesn't fail its build -- only the run's own
// settle decides anything.
func TestRevalidationDoesNotHoldOrFailTheBuild(t *testing.T) {
	f := newRecoverFixture(t, "revalidate-build", "building")
	ctx := context.Background()
	o := f.validateHost("web", "building", 1)
	f.heartbeat(o, 0)
	run := f.start(o)
	tasks, _ := f.q.ListValidationRunTasks(ctx, run.ID)
	if _, err := f.pool.Exec(ctx, "UPDATE agent_task SET status = 'failed' WHERE id = $1", tasks[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, "UPDATE validation_run SET status = 'passed' WHERE id = $1", run.ID); err != nil { // keep settle out of it
		t.Fatal(err)
	}
	if err := AdvanceObjectLifecycle(ctx, f.pool, f.build.ID); err != nil {
		t.Fatal(err)
	}
	if got := f.status(o).Status; got != "finished" {
		t.Fatalf("host %s, want finished (revalidation tasks aren't build steps)", got)
	}
}

// TestRevalidationFlipReopensSettledBuild: a finished build whose host a
// revalidation turns invalid re-enters building, so its status re-settles.
func TestRevalidationFlipReopensSettledBuild(t *testing.T) {
	f := newRecoverFixture(t, "revalidate-reopen", "finished")
	ctx := context.Background()
	o := f.validateHost("web", "finished", 1)
	run := f.start(o)
	f.report(run, 0)
	if err := AdvanceObjectLifecycle(ctx, f.pool, f.build.ID); err != nil {
		t.Fatal(err)
	}
	b, err := f.q.GetBuild(ctx, f.build.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != "building" {
		t.Fatalf("build %s, want building (to re-settle)", b.Status)
	}
	if settled, err := AdvanceBuildStatus(ctx, f.pool, f.build.ID); err != nil || !settled {
		t.Fatalf("AdvanceBuildStatus: settled=%v err=%v", settled, err)
	}
	if b, _ = f.q.GetBuild(ctx, f.build.ID); b.Status != "failed" {
		t.Fatalf("build re-settled as %s, want failed (a host is invalid)", b.Status)
	}
}
