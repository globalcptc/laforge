package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

// TestReportStatusPersistsStructuredValidatorResults is the real fix for
// a gap found by direct audit: the agent always
// computed a real, structured per-check result (kind/args/passed/message)
// for a `validate:` task, but this gateway only ever persisted the
// flattened Output/Error text blob every other command already carries.
// RecordValidatorResult and the validator_result table it writes existed
// since the agent schema, with zero callers, until now.
func TestReportStatusPersistsStructuredValidatorResults(t *testing.T) {
	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	task, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "validate",
		Payload: []byte(`{"checks":[{"kind":"file_exists","args":{"path":"/etc/hosts"}},{"kind":"port_listening","args":{"port":22}}]}`),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}

	addr, ca, _ := startTestGateway(t)
	client := dialTestAgent(t, addr, ca, host.deployedObjectID.String())
	defer client.conn.Close()
	client.heartbeat(t)
	got := client.getTask(t)
	if got == nil || got.ID != task.ID.String() {
		t.Fatalf("getTask = %+v, want the real validate task", got)
	}

	// Exactly the shape the real Rust agent's own validators::run
	// produces: one entry per check, passed and failed both included,
	// regardless of the overall task status.
	report := agentproto.ReportStatusRequestPayload{
		TaskID: task.ID.String(),
		Status: "failed",
		Error:  "file_exists: PASS (/etc/hosts exists)\nport_listening: FAIL (port 22 is not listening)",
		ValidatorResults: []agentproto.ValidatorResult{
			{Kind: "file_exists", Args: json.RawMessage(`{"path":"/etc/hosts"}`), Passed: true, Message: "/etc/hosts exists"},
			{Kind: "port_listening", Args: json.RawMessage(`{"port":22}`), Passed: false, Message: "port 22 is not listening"},
		},
	}
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshaling report: %v", err)
	}
	if err := agentproto.WriteFrame(client.conn, agentproto.ReportStatusRequest, body); err != nil {
		t.Fatalf("writing report-status: %v", err)
	}
	mt, respBody, err := agentproto.ReadFrame(client.conn)
	if err != nil {
		t.Fatalf("reading report-status response: %v", err)
	}
	if mt != agentproto.ReportStatusResponse {
		t.Fatalf("got message type %v, want ReportStatusResponse", mt)
	}
	var resp agentproto.ReportStatusResponsePayload
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !resp.OK {
		t.Fatal("report-status response OK = false")
	}

	results, err := adminQ.ListValidatorResultsByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListValidatorResultsByTask: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2: %+v", len(results), results)
	}
	byKind := map[string]db.ValidatorResult{}
	for _, r := range results {
		byKind[r.Kind] = r
	}
	fe, ok := byKind["file_exists"]
	if !ok || !fe.Passed || fe.Message == nil || *fe.Message != "/etc/hosts exists" {
		t.Fatalf("file_exists result = %+v, ok=%v", fe, ok)
	}
	var feArgs map[string]string
	if err := json.Unmarshal(fe.Args, &feArgs); err != nil || feArgs["path"] != "/etc/hosts" {
		t.Fatalf("file_exists args = %s, err=%v", fe.Args, err)
	}
	pl, ok := byKind["port_listening"]
	if !ok || pl.Passed || pl.Message == nil || *pl.Message != "port 22 is not listening" {
		t.Fatalf("port_listening result = %+v, ok=%v", pl, ok)
	}
}
