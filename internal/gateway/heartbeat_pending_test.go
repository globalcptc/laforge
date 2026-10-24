package gateway

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/agentproto"
)

// TestHeartbeatCarriesPendingSessions guards the wiring that actually opens an
// interactive shell: when a client (api) half is waiting for an object's agent,
// that object's heartbeat response MUST carry the session id in PendingSessions
// (and shorten the poll). A regression here is silent -- the agent keeps
// heartbeating but is never told to attach, and every shell times out with
// "agent never attached" -- so it's worth a direct test.
func TestHeartbeatCarriesPendingSessions(t *testing.T) {
	admin := openAdminPool(t)
	host := seedDeployedHost(t, admin)
	s := &Server{Pool: openGatewayPool(t), BasePollMS: 15000, JitterMS: 5000}

	// A client half is waiting for this object's agent (agent not yet attached).
	s.relaySessions = map[string]*relaySession{
		"sess-1": {objectID: uuidString(host.deployedObjectID), ready: make(chan struct{}), done: make(chan struct{})},
	}

	srvConn, agentConn := net.Pipe()
	go s.handleHeartbeat(context.Background(), srvConn, host.deployedObjectID, "fp-test", nil)

	agentConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	mt, body, err := agentproto.ReadFrame(agentConn)
	if err != nil || mt != agentproto.HeartbeatResponse {
		t.Fatalf("read heartbeat response: mt=%v err=%v", mt, err)
	}
	var resp agentproto.HeartbeatResponsePayload
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.PendingSessions) != 1 || resp.PendingSessions[0] != "sess-1" {
		t.Fatalf("PendingSessions = %v, want [sess-1] -- the agent is never told to attach otherwise", resp.PendingSessions)
	}
	if resp.NextPollMS != 250 {
		t.Errorf("NextPollMS = %d, want 250 (poll fast while a shell is pending)", resp.NextPollMS)
	}
	agentConn.Close()
}
