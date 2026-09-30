package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
)

// adminConnString is a full-privilege connection, used only to seed test
// data the way internal/orchestrator or a real deploy would -- never how
// the gateway itself connects.
func adminConnString(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("LAFORGE_TEST_DATABASE_URL"); v != "" {
		return v
	}
	if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err != nil {
		t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
	}
	return "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
}

// gatewayConnString connects as the REAL restricted laforge_gateway role
// (migrations/00004) -- the whole point of server_test.go is that the
// gateway is proven against its actual, limited database identity, not a
// full-privilege stand-in.
func gatewayConnString() string {
	return "host=/tmp port=5432 user=laforge_gateway password=laforge_gateway_dev_only_do_not_use_in_prod dbname=laforge_dev sslmode=disable"
}

func openAdminPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_, pool, err := db.Open(context.Background(), adminConnString(t))
	if err != nil {
		t.Fatalf("db.Open (admin): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func openGatewayPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	_, pool, err := db.Open(context.Background(), gatewayConnString())
	if err != nil {
		t.Fatalf("db.Open (laforge_gateway role): %v -- is migrations/00004_agent_tables.sql applied?", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestGatewayRoleCannotReadContent is the least-privilege proof the
// migration's own comment promises: connect AS laforge_gateway and try a
// query on something it was never granted -- real Postgres has to refuse
// it, not application code choosing not to ask.
func TestGatewayRoleCannotReadContent(t *testing.T) {
	pool := openGatewayPool(t)
	ctx := context.Background()

	// QueryRow (not Query) deliberately: pgx's Query() can return a nil
	// error immediately and only surface a permission-denied failure once
	// the returned rows are actually iterated -- a first real run of this
	// test hung the whole suite on exactly that: Query() "succeeded",
	// the rows were never drained or closed, and the pool's one
	// connection was never released back for the second query to use.
	// QueryRow forces real execution (and releases the connection) before
	// returning, so the permission error comes back where it belongs.
	var discard string
	err := pool.QueryRow(ctx, "SELECT github_owner FROM repository LIMIT 1").Scan(&discard)
	if err == nil {
		t.Fatal("laforge_gateway was able to query repository -- it must not have SELECT on content tables")
	}
	t.Logf("repository query correctly rejected: %v", err)

	err = pool.QueryRow(ctx, "SELECT branch FROM configured_build LIMIT 1").Scan(&discard)
	if err == nil {
		t.Fatal("laforge_gateway was able to query configured_build -- it must not have SELECT on this table")
	}
	t.Logf("configured_build query correctly rejected: %v", err)

	// The flip side: it MUST be able to do its actual job.
	if _, err := pool.Exec(ctx, "SELECT 1 FROM deployed_object LIMIT 0"); err != nil {
		t.Fatalf("laforge_gateway should be able to SELECT deployed_object, got: %v", err)
	}

	// agent_heartbeat (migrations/00007) is deliberately write-only for
	// this role -- it can log its own history but never read any of it
	// back, same reasoning as agent_session's own grant. Confirmed against
	// real Postgres, not just "the migration says INSERT only": a
	// compromised gateway connection must not be able to reconstruct any
	// object's check-in history, remote addresses included.
	err = pool.QueryRow(ctx, "SELECT cert_fingerprint FROM agent_heartbeat LIMIT 1").Scan(&discard)
	if err == nil {
		t.Fatal("laforge_gateway was able to query agent_heartbeat -- it must be INSERT-only, no SELECT")
	}
	t.Logf("agent_heartbeat read correctly rejected: %v", err)

	// event (migrations/00009) is the same shape: the gateway logs its
	// own step events into it but must never be able to read the
	// journal back -- a compromised gateway connection reconstructing
	// deploy/destroy/access history (everything else in `event`, not
	// just its own step rows) would be a real privilege escalation.
	err = pool.QueryRow(ctx, "SELECT kind FROM event LIMIT 1").Scan(&discard)
	if err == nil {
		t.Fatal("laforge_gateway was able to query event -- it must be INSERT-only, no SELECT")
	}
	t.Logf("event read correctly rejected: %v", err)

	// builder_credential (migration 00016) holds private keys LaForge
	// uses to drive real hosters -- the gateway, the one service facing
	// competition hosts, must never be able to read them.
	err = pool.QueryRow(ctx, "SELECT client_key_pem FROM builder_credential LIMIT 1").Scan(&discard)
	if err == nil {
		t.Fatal("laforge_gateway was able to query builder_credential -- it must have no access to hoster credentials")
	}
	t.Logf("builder_credential read correctly rejected: %v", err)
}

type seededHost struct {
	deployedObjectID pgtype.UUID
	buildID          pgtype.UUID
}

func seedDeployedHost(t *testing.T, adminPool *pgxpool.Pool) seededHost {
	t.Helper()
	ctx := context.Background()
	q := db.New(adminPool)

	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-gateway-test", GithubRepo: t.Name()})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { adminPool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })

	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "gateway-test-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "lm-test"})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "webserver", AsName: db.StrPtr("web01"), NetworkName: db.StrPtr("prod"),
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}

	return seededHost{deployedObjectID: obj.ID, buildID: build.ID}
}

// startTestGateway wires a real Server (connected as laforge_gateway) to
// a real mTLS listener on an ephemeral port, and returns a client TLS
// config already trusting the same CA, plus the CA itself for issuing
// more client certs.
func startTestGateway(t *testing.T) (addr string, ca *agentpki.CA, gatewayPool *pgxpool.Pool) {
	t.Helper()
	return startTestGatewayWithLease(t, 5*time.Second)
}

func startTestGatewayWithLease(t *testing.T, leaseDuration time.Duration) (addr string, ca *agentpki.CA, gatewayPool *pgxpool.Pool) {
	t.Helper()
	ca, err := agentpki.GenerateCA("gateway-test-ca")
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	serverCert, serverKey, err := ca.IssueLeaf("localhost", true, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf (server): %v", err)
	}
	tlsCfg, err := agentpki.ServerTLSConfig(ca.CertPEM, serverCert, serverKey)
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	gatewayPool = openGatewayPool(t)
	srv := &Server{
		Pool: gatewayPool, TLSConfig: tlsCfg,
		LeaseDuration: leaseDuration, BasePollMS: 1000, JitterMS: 500,
	}
	ln, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.Serve(ctx, ln)
	t.Cleanup(func() { ln.Close() })

	return ln.Addr().String(), ca, gatewayPool
}

// testAgentClient is a minimal, real speaker of the wire protocol,
// standing in for the Rust agent in this Go-side test -- the actual Rust
// implementation gets its own separate end-to-end proof; this
// proves the gateway's own protocol
// handling is correct independent of that.
type testAgentClient struct {
	conn net.Conn
}

func dialTestAgent(t *testing.T, addr string, ca *agentpki.CA, hostID string) *testAgentClient {
	t.Helper()
	clientCert, clientKey, err := ca.IssueLeaf(hostID, false, nil, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf (client %s): %v", hostID, err)
	}
	clientTLS, err := agentpki.ClientTLSConfig(ca.CertPEM, clientCert, clientKey, "localhost")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}
	conn, err := tls.Dial("tcp", addr, clientTLS)
	if err != nil {
		t.Fatalf("dialing gateway: %v", err)
	}
	return &testAgentClient{conn: conn}
}

func (c *testAgentClient) heartbeat(t *testing.T) agentproto.HeartbeatResponsePayload {
	t.Helper()
	if err := agentproto.WriteFrame(c.conn, agentproto.HeartbeatRequest, nil); err != nil {
		t.Fatalf("writing heartbeat: %v", err)
	}
	mt, body, err := agentproto.ReadFrame(c.conn)
	if err != nil {
		t.Fatalf("reading heartbeat response: %v", err)
	}
	if mt != agentproto.HeartbeatResponse {
		t.Fatalf("got message type %v, want HeartbeatResponse", mt)
	}
	var resp agentproto.HeartbeatResponsePayload
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decoding heartbeat response: %v", err)
	}
	return resp
}

func (c *testAgentClient) getTask(t *testing.T) *agentproto.Task {
	t.Helper()
	if err := agentproto.WriteFrame(c.conn, agentproto.GetTaskRequest, nil); err != nil {
		t.Fatalf("writing get-task: %v", err)
	}
	mt, body, err := agentproto.ReadFrame(c.conn)
	if err != nil {
		t.Fatalf("reading get-task response: %v", err)
	}
	if mt != agentproto.GetTaskResponse {
		t.Fatalf("got message type %v, want GetTaskResponse", mt)
	}
	var resp agentproto.GetTaskResponsePayload
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decoding get-task response: %v", err)
	}
	return resp.Task
}

func (c *testAgentClient) reportStatus(t *testing.T, taskID, status, output, errMsg string) agentproto.ReportStatusResponsePayload {
	t.Helper()
	body, _ := json.Marshal(agentproto.ReportStatusRequestPayload{TaskID: taskID, Status: status, Output: output, Error: errMsg})
	if err := agentproto.WriteFrame(c.conn, agentproto.ReportStatusRequest, body); err != nil {
		t.Fatalf("writing report-status: %v", err)
	}
	mt, respBody, err := agentproto.ReadFrame(c.conn)
	if err != nil {
		t.Fatalf("reading report-status response: %v", err)
	}
	if mt != agentproto.ReportStatusResponse {
		t.Fatalf("got message type %v, want ReportStatusResponse", mt)
	}
	var resp agentproto.ReportStatusResponsePayload
	json.Unmarshal(respBody, &resp)
	return resp
}
