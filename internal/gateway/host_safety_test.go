package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

func (c *testAgentClient) heartbeatWith(t *testing.T, p agentproto.HeartbeatRequestPayload) {
	t.Helper()
	body, _ := json.Marshal(p)
	if err := agentproto.WriteFrame(c.conn, agentproto.HeartbeatRequest, body); err != nil {
		t.Fatalf("writing heartbeat: %v", err)
	}
	if mt, _, err := agentproto.ReadFrame(c.conn); err != nil || mt != agentproto.HeartbeatResponse {
		t.Fatalf("heartbeat response: type %v, err %v", mt, err)
	}
}

// TestHeartbeatRenewsInflightLeaseAcrossReconnect is the duplicate-execution
// regression: an agent whose step outlives one lease, and whose connection
// drops mid-step, must not get that step handed out a second time. The agent
// names the step on each heartbeat and the gateway renews its lease.
func TestHeartbeatRenewsInflightLeaseAcrossReconnect(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{"command":"/bin/sleep"}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}
	addr, ca, _ := startTestGatewayWithLease(t, 1*time.Second)

	client1 := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	client1.heartbeat(t)
	if task := client1.getTask(t); task == nil || task.ID != step0.ID.String() {
		t.Fatalf("first get-task = %+v, want step 0", task)
	}
	client1.conn.Close() // the session drops; the step is still running on the box

	// The reconnected agent keeps heartbeating with the step in flight for
	// well past the 1s lease.
	client2 := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client2.conn.Close()
	for i := 0; i < 6; i++ {
		client2.heartbeatWith(t, agentproto.HeartbeatRequestPayload{InflightTaskID: step0.ID.String()})
		time.Sleep(400 * time.Millisecond)
	}
	if got := client2.getTask(t); got != nil {
		t.Fatalf("get-task while the step is still in flight returned %+v; the lease should have been renewed", got)
	}
	dbTask, err := adminQ.GetAgentTask(ctx, step0.ID)
	if err != nil {
		t.Fatalf("GetAgentTask: %v", err)
	}
	if dbTask.Status != "leased" || dbTask.Attempts != 1 {
		t.Fatalf("task = %s attempts %d, want leased attempts 1", dbTask.Status, dbTask.Attempts)
	}

	// The result reported on the new session completes the original lease.
	if resp := client2.reportStatus(t, step0.ID.String(), "done", "slept", ""); !resp.OK {
		t.Fatal("report-status after reconnect was not ok")
	}
	if dbTask, _ = adminQ.GetAgentTask(ctx, step0.ID); dbTask.Status != "done" {
		t.Fatalf("status = %s, want done", dbTask.Status)
	}
}

// TestHeartbeatLeaseRenewalIsScopedToOwnObject: an agent can't keep another
// host's step leased by naming it.
func TestHeartbeatLeaseRenewalIsScopedToOwnObject(t *testing.T) {
	adminPool := openAdminPool(t)
	hostA := seedDeployedHost(t, adminPool)
	hostB := seedSecondHost(t, adminPool, hostA)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	stepA, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: hostA.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}
	addr, ca, _ := startTestGatewayWithLease(t, 1*time.Second)
	clientA := dialTestAgent(t, addr, ca, hostA.deployedObjectID.String())
	defer clientA.conn.Close()
	clientA.heartbeat(t)
	clientA.getTask(t)
	before, _ := adminQ.GetAgentTask(ctx, stepA.ID)

	clientB := dialTestAgent(t, addr, ca, hostB.String())
	defer clientB.conn.Close()
	clientB.heartbeatWith(t, agentproto.HeartbeatRequestPayload{InflightTaskID: stepA.ID.String()})

	after, _ := adminQ.GetAgentTask(ctx, stepA.ID)
	if !after.LeaseExpiresAt.Time.Equal(before.LeaseExpiresAt.Time) {
		t.Fatalf("host B's heartbeat moved host A's lease from %v to %v", before.LeaseExpiresAt.Time, after.LeaseExpiresAt.Time)
	}
}

func seedSecondHost(t *testing.T, adminPool *pgxpool.Pool, first seededHost) pgtype.UUID {
	t.Helper()
	ctx := context.Background()
	q := db.New(adminPool)
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: first.buildID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "other", AsName: db.StrPtr("other01"), NetworkName: db.StrPtr("prod"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}
	return obj.ID
}

// TestTimeoutFailureIsTerminal: an agent-enforced timeout fails the step on its
// first attempt instead of re-running a deterministic hour-long wait loop. A
// script's own "timed out" output is still an ordinary, retryable failure.
func TestTimeoutFailureIsTerminal(t *testing.T) {
	cases := []struct {
		name, status, errMsg, want string
	}{
		{"timeout status", agentproto.StatusTimeout, "/bin/sh timed out after 3600s and was killed along with its child processes", "failed"},
		{"failed with the agent's timeout error", agentproto.StatusFailed, "/bin/sh timed out after 3600s and was killed along with its child processes; output before the kill: waiting", "failed"},
		{"script output mentioning a timeout", agentproto.StatusFailed, "/bin/sh exited 7: curl: (28) Connection timed out after 30001 milliseconds", "pending"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adminPool := openAdminPool(t)
			host := seedDeployedHost(t, adminPool)
			adminQ := db.New(adminPool)
			ctx := context.Background()
			step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
				DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{}`),
			})
			if err != nil {
				t.Fatalf("CreateAgentTask: %v", err)
			}
			addr, ca, _ := startTestGateway(t)
			client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
			defer client.conn.Close()
			client.heartbeat(t)
			client.getTask(t)
			client.reportStatus(t, step0.ID.String(), tc.status, "", tc.errMsg)
			got, _ := adminQ.GetAgentTask(ctx, step0.ID)
			if got.Status != tc.want {
				t.Fatalf("status after attempt 1 = %s, want %s", got.Status, tc.want)
			}
		})
	}
}

// TestScriptRetriesOverride: agent_task.max_attempts (a script's `retries:`)
// replaces MaxAttempts -- retries: 0 fails on the first failure.
func TestScriptRetriesOverride(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()
	one := int32(1)
	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "execute", Payload: []byte(`{}`), MaxAttempts: &one,
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}
	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()
	client.heartbeat(t)
	client.getTask(t)
	client.reportStatus(t, step0.ID.String(), "failed", "", "exit 1")
	if got, _ := adminQ.GetAgentTask(ctx, step0.ID); got.Status != "failed" {
		t.Fatalf("status = %s, want failed after one attempt with retries: 0", got.Status)
	}
}

// TestFirstCheckinRecordsBootTiming: the first heartbeat after a deploy writes
// one agent_checked_in event with the boot/agent-start phases, and the boot
// report lands on the session and the heartbeat log.
func TestFirstCheckinRecordsBootTiming(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()
	if _, err := adminQ.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: host.deployedObjectID, ExternalRef: db.StrPtr("vm-1")}); err != nil {
		t.Fatalf("MarkDeployedObjectRunning: %v", err)
	}
	// Pretend the deploy finished 10 minutes ago, the OS booted 4 minutes in
	// and the agent started a minute after that.
	deployed := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
	if _, err := adminPool.Exec(ctx, "UPDATE deployed_object SET deployed_at = $2 WHERE id = $1", host.deployedObjectID, deployed); err != nil {
		t.Fatal(err)
	}
	boot, started := deployed.Add(4*time.Minute).Unix(), deployed.Add(5*time.Minute).Unix()

	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()
	client.heartbeatWith(t, agentproto.HeartbeatRequestPayload{BootTime: &boot, AgentStartedAt: &started})
	client.heartbeatWith(t, agentproto.HeartbeatRequestPayload{BootTime: &boot, AgentStartedAt: &started})

	var n int
	var msg string
	if err := adminPool.QueryRow(ctx, "SELECT count(*), max(message) FROM event WHERE deployed_object_id = $1 AND kind = 'agent_checked_in'", host.deployedObjectID).Scan(&n, &msg); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("agent_checked_in events = %d, want exactly 1", n)
	}
	for _, want := range []string{"after deploy", "OS booted +4m0s", "agent started +5m0s"} {
		if !strings.Contains(msg, want) {
			t.Errorf("event message %q missing %q", msg, want)
		}
	}
	sess, err := adminQ.GetAgentSessionByDeployedObject(ctx, host.deployedObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if !sess.BootAt.Valid || sess.BootAt.Time.Unix() != boot || sess.AgentStartedAt.Time.Unix() != started {
		t.Fatalf("session boot_at=%v agent_started_at=%v, want %d/%d", sess.BootAt, sess.AgentStartedAt, boot, started)
	}
	var withBoot int
	adminPool.QueryRow(ctx, "SELECT count(*) FROM agent_heartbeat WHERE deployed_object_id = $1 AND boot_at IS NOT NULL", host.deployedObjectID).Scan(&withBoot)
	if withBoot != 2 {
		t.Fatalf("heartbeat rows with boot_at = %d, want 2", withBoot)
	}

	// A redeploy restarts the clock: the next heartbeat is a first check-in again.
	if _, err := adminQ.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: host.deployedObjectID, ExternalRef: db.StrPtr("vm-2")}); err != nil {
		t.Fatal(err)
	}
	client.heartbeat(t)
	adminPool.QueryRow(ctx, "SELECT count(*) FROM event WHERE deployed_object_id = $1 AND kind = 'agent_checked_in'", host.deployedObjectID).Scan(&n)
	if n != 2 {
		t.Fatalf("agent_checked_in events after redeploy = %d, want 2", n)
	}
}

func TestFailureIsTerminal(t *testing.T) {
	two := int32(2)
	cases := []struct {
		name     string
		task     db.AgentTask
		req      agentproto.ReportStatusRequestPayload
		terminal bool
	}{
		{"under default cap", db.AgentTask{Attempts: 1}, agentproto.ReportStatusRequestPayload{Status: "failed"}, false},
		{"at default cap", db.AgentTask{Attempts: MaxAttempts}, agentproto.ReportStatusRequestPayload{Status: "failed"}, true},
		{"timeout on first attempt", db.AgentTask{Attempts: 1}, agentproto.ReportStatusRequestPayload{Status: "timeout"}, true},
		{"override reached", db.AgentTask{Attempts: 2, MaxAttempts: &two}, agentproto.ReportStatusRequestPayload{Status: "failed"}, true},
		{"override not reached", db.AgentTask{Attempts: 1, MaxAttempts: &two}, agentproto.ReportStatusRequestPayload{Status: "failed"}, false},
	}
	for _, tc := range cases {
		if got := failureIsTerminal(tc.task, tc.req); got != tc.terminal {
			t.Errorf("%s: failureIsTerminal = %v, want %v", tc.name, got, tc.terminal)
		}
	}
}

func TestCheckinSummaryWithoutBootReport(t *testing.T) {
	now := time.Now()
	c := db.GetAgentCheckinContextRow{DeployedAt: pgtype.Timestamptz{Time: now.Add(-90 * time.Second), Valid: true}}
	msg, payload := checkinSummary(now, c, pgtype.Timestamptz{}, pgtype.Timestamptz{})
	if msg != "agent checked in 1m30s after deploy" {
		t.Fatalf("message = %q", msg)
	}
	if _, ok := payload["deploy_to_boot_sec"]; ok {
		t.Fatal("an older agent's missing boot report must not be guessed")
	}
}
