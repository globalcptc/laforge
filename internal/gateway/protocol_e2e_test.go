package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// TestIgnoreErrorsStepIsToleratedAndDoesNotBlock proves the ignore_errors wiring
// end to end over the real protocol: a step whose task carries ignore_errors, when
// it fails terminally (MaxAttempts), becomes 'ignored' rather than 'failed' -- so
// it does NOT block the next step (get-task serves step 1) and, per
// SummarizeAgentTasksByObjectForBuild, does not count as a build failure. A normal
// task in its place would go 'failed' and wedge the whole host at step 0.
func TestIgnoreErrorsStepIsToleratedAndDoesNotBlock(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute",
		Payload: []byte(`{"command":"/bin/false"}`), IgnoreErrors: true,
	})
	if err != nil {
		t.Fatalf("CreateAgentTask step0: %v", err)
	}
	step1, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 1, Command: "write_file",
		Payload: []byte(`{"path":"/tmp/after","content":"ran"}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask step1: %v", err)
	}

	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()
	client.heartbeat(t)

	// Fail step 0 the full MaxAttempts so it goes terminal.
	for i := 0; i < MaxAttempts; i++ {
		task := client.getTask(t)
		if task == nil || task.ID != step0.ID.String() {
			t.Fatalf("attempt %d: got %+v, want step0 %s (an unfinished step must be re-offered)", i, task, step0.ID)
		}
		client.reportStatus(t, task.ID, "failed", "", "boom")
	}

	got, err := adminQ.GetAgentTask(ctx, step0.ID)
	if err != nil {
		t.Fatalf("GetAgentTask step0: %v", err)
	}
	if got.Status != "ignored" {
		t.Fatalf("step0 status = %q, want ignored (a tolerated terminal failure)", got.Status)
	}

	// The sequence must continue: get-task now serves step 1, not nil.
	next := client.getTask(t)
	if next == nil {
		t.Fatal("get-task returned nil -- an ignored step 0 must not block step 1")
	}
	if next.ID != step1.ID.String() {
		t.Fatalf("next task = %s, want step1 %s", next.ID, step1.ID)
	}
}

// TestGatewayEndToEnd is the real proof of the whole verb set, over a
// real mTLS connection, against the gateway's actual restricted database
// role: heartbeat records a session, get-task offers steps in order (not
// step 2 before step 1 is done), report-status advances the queue, and a
// host with nothing left to do gets task:null rather than an error.
func TestGatewayEndToEnd(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)

	adminQ := db.New(adminPool)
	ctx := context.Background()
	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "write_file", Payload: []byte(`{"path":"/tmp/a","content":"hi"}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask (step 0): %v", err)
	}
	_, err = adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 1, Command: "execute", Payload: []byte(`{"command":"/bin/bash","args":["/tmp/a"]}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask (step 1): %v", err)
	}

	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()

	hb := client.heartbeat(t)
	if hb.NextPollMS <= 0 {
		t.Fatalf("heartbeat response NextPollMS = %d, want > 0", hb.NextPollMS)
	}
	session, err := adminQ.GetAgentSessionByDeployedObject(ctx, host.deployedObjectID)
	if err != nil {
		t.Fatalf("GetAgentSessionByDeployedObject: %v (heartbeat should have created this row)", err)
	}
	if session.CertFingerprint == "" {
		t.Fatal("agent_session has no cert_fingerprint recorded")
	}

	// The append-only history alongside the "current status" row --
	// real, over the real mTLS connection this test already dialed
	// (client.conn), not a synthetic insert.
	heartbeats, err := adminQ.ListAgentHeartbeatsByObject(ctx, db.ListAgentHeartbeatsByObjectParams{
		DeployedObjectID: host.deployedObjectID, Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListAgentHeartbeatsByObject: %v", err)
	}
	if len(heartbeats) != 1 {
		t.Fatalf("agent_heartbeat has %d row(s) after one real heartbeat, want 1", len(heartbeats))
	}
	hbRow := heartbeats[0]
	if hbRow.CertFingerprint != session.CertFingerprint {
		t.Fatalf("heartbeat cert_fingerprint = %q, want %q (matching agent_session)", hbRow.CertFingerprint, session.CertFingerprint)
	}
	if hbRow.RemoteAddr == nil || *hbRow.RemoteAddr == "" {
		t.Fatal("heartbeat remote_addr is empty -- should be the real TCP peer address")
	}
	if hbRow.NextPollMs == nil || *hbRow.NextPollMs != int32(hb.NextPollMS) {
		t.Fatalf("heartbeat next_poll_ms = %v, want %d (matching the actual response sent to the agent)", hbRow.NextPollMs, hb.NextPollMS)
	}

	task := client.getTask(t)
	if task == nil {
		t.Fatal("get-task returned nil, want step 0")
	}
	if task.ID != step0.ID.String() || task.Command != "write_file" {
		t.Fatalf("get-task returned %+v, want step 0 (id=%s, command=write_file)", task, step0.ID.String())
	}

	// Asking again while step 0's lease is still fresh must NOT jump ahead
	// to step 1 -- it must return nil ("nothing available right now"), the
	// same recovery model as the task queue: a well-behaved
	// agent never re-asks while it already holds a task in hand, and a
	// genuinely crashed agent's abandoned lease is recovered the same way
	// a crashed runner's is -- by waiting for it to expire, not by the
	// server guessing who's allowed to reclaim it early.
	task2 := client.getTask(t)
	if task2 != nil {
		t.Fatalf("second get-task (before reporting status, lease still fresh) = %+v, want nil", task2)
	}

	resp := client.reportStatus(t, task.ID, "done", "wrote file ok", "")
	if !resp.OK {
		t.Fatal("report-status(done) for step 0 returned OK=false")
	}

	// The real fix for "the gateway doesn't log step-level events into
	// the journal" -- a real event, over
	// this same real restricted laforge_gateway connection, readable
	// back through the same per-object query the API's own
	// ListEventsByDeployedObject uses.
	events, err := adminQ.ListEventsByDeployedObject(ctx, host.deployedObjectID)
	if err != nil {
		t.Fatalf("ListEventsByDeployedObject: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events after step 0 done = %d, want 1", len(events))
	}
	if events[0].Kind != "step_done" {
		t.Fatalf("event kind = %q, want step_done", events[0].Kind)
	}
	if events[0].Message != "step 0 (write_file) done" {
		t.Fatalf("event message = %q, want %q", events[0].Message, "step 0 (write_file) done")
	}
	if events[0].BuildID != host.buildID {
		t.Fatalf("event build_id = %v, want %v", events[0].BuildID, host.buildID)
	}
	if !events[0].DeployedObjectID.Valid || events[0].DeployedObjectID != host.deployedObjectID {
		t.Fatalf("event deployed_object_id = %v, want %v", events[0].DeployedObjectID, host.deployedObjectID)
	}
	if events[0].TaskID.Valid {
		t.Fatalf("event task_id = %v, want NULL (no orchestrator task owns a step event -- see migrations/00009)", events[0].TaskID)
	}

	task3 := client.getTask(t)
	if task3 == nil {
		t.Fatal("get-task after completing step 0 returned nil, want step 1")
	}
	if task3.Command != "execute" {
		t.Fatalf("get-task after step 0 done returned command %q, want execute (step 1)", task3.Command)
	}

	resp = client.reportStatus(t, task3.ID, "done", "ran ok", "")
	if !resp.OK {
		t.Fatal("report-status(done) for step 1 returned OK=false")
	}

	events, err = adminQ.ListEventsByDeployedObject(ctx, host.deployedObjectID)
	if err != nil {
		t.Fatalf("ListEventsByDeployedObject (after step 1): %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events after both steps done = %d, want 2", len(events))
	}
	if events[1].Message != "step 1 (execute) done" {
		t.Fatalf("second event message = %q, want %q", events[1].Message, "step 1 (execute) done")
	}

	task4 := client.getTask(t)
	if task4 != nil {
		t.Fatalf("get-task after finishing every step returned %+v, want nil", task4)
	}
}

// TestGatewayRetryThenFailCap proves the retry-with-a-cap behavior:
// report-status(failed) under MaxAttempts puts a step back to pending
// (immediately re-offered), and reaching MaxAttempts makes it terminal.
func TestGatewayRetryThenFailCap(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	if _, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{"command":"/bin/false"}`),
	}); err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}

	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()
	client.heartbeat(t)

	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		task := client.getTask(t)
		if task == nil {
			t.Fatalf("attempt %d: get-task returned nil, want the still-retryable step", attempt)
		}
		resp := client.reportStatus(t, task.ID, "failed", "", "exit status 1")
		if !resp.OK {
			t.Fatalf("attempt %d: report-status(failed) returned OK=false", attempt)
		}

		taskID, err := parseUUID(task.ID)
		if err != nil {
			t.Fatalf("attempt %d: parsing task id %q: %v", attempt, task.ID, err)
		}
		dbTask, err := adminQ.GetAgentTask(ctx, taskID)
		if err != nil {
			t.Fatalf("attempt %d: GetAgentTask: %v", attempt, err)
		}
		events, err := adminQ.ListEventsByDeployedObject(ctx, host.deployedObjectID)
		if err != nil {
			t.Fatalf("attempt %d: ListEventsByDeployedObject: %v", attempt, err)
		}
		if attempt < MaxAttempts {
			if dbTask.Status != "pending" {
				t.Fatalf("attempt %d: status = %q, want pending (still under MaxAttempts=%d)", attempt, dbTask.Status, MaxAttempts)
			}
			// A transient retry under the cap is exactly the "internal
			// retry noise" logStepEvent's own doc comment says not to
			// log -- matching deploy/destroy lifecycle events, which
			// only fire on a real state transition.
			if len(events) != 0 {
				t.Fatalf("attempt %d: %d event(s) logged for a retry under MaxAttempts, want 0", attempt, len(events))
			}
		} else {
			if dbTask.Status != "failed" {
				t.Fatalf("attempt %d: status = %q, want failed (reached MaxAttempts=%d)", attempt, dbTask.Status, MaxAttempts)
			}
			if len(events) != 1 {
				t.Fatalf("attempt %d: %d event(s) logged after the terminal failure, want exactly 1", attempt, len(events))
			}
			if events[0].Kind != "step_failed" {
				t.Fatalf("terminal event kind = %q, want step_failed", events[0].Kind)
			}
		}
	}

	// A failed (terminal) task is never offered again.
	task := client.getTask(t)
	if task != nil {
		t.Fatalf("get-task after the task reached MaxAttempts and failed = %+v, want nil", task)
	}
}

// TestGatewayReclaimsExpiredLease is the agent_task side of "a dead
// runner's lease expires and the task is re-leased" -- same recovery
// model as the task queue (see internal/runner), proven here for a
// crashed-and-restarted agent:
// step 0 is leased, never completed (standing in for the agent dying
// mid-step), and once the lease genuinely expires a fresh connection from
// the same host gets step 0 again -- not step 1, not nothing.
func TestGatewayReclaimsExpiredLease(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{"command":"/bin/true"}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}

	addr, ca, _ := startTestGatewayWithLease(t, 2*time.Second)

	client1 := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	client1.heartbeat(t)
	task := client1.getTask(t)
	if task == nil || task.ID != step0.ID.String() {
		t.Fatalf("first get-task = %+v, want step 0", task)
	}
	client1.conn.Close() // simulate the agent dying: no report-status ever sent

	time.Sleep(3 * time.Second) // past the 2s lease

	client2 := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client2.conn.Close()
	client2.heartbeat(t)
	reclaimed := client2.getTask(t)
	if reclaimed == nil {
		t.Fatal("get-task after lease expiry returned nil, want step 0 reclaimed")
	}
	if reclaimed.ID != step0.ID.String() {
		t.Fatalf("get-task after lease expiry returned %+v, want step 0 (%s) again", reclaimed, step0.ID.String())
	}

	dbTask, err := adminQ.GetAgentTask(ctx, step0.ID)
	if err != nil {
		t.Fatalf("GetAgentTask: %v", err)
	}
	if dbTask.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2 (leased once by the dead agent, once more on reclaim)", dbTask.Attempts)
	}
}
