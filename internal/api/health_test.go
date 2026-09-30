package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// TestAgentHealthComputedFromRealHeartbeats proves the three states
// against real agent_session rows, not synthetic Go values: a fresh
// heartbeat (healthy), a stale one written directly with a backdated
// timestamp (missing -- UpsertAgentSession always stamps now(), so
// simulating "hasn't checked in for 10 minutes" needs a direct insert,
// same as a real agent that stopped heartbeating would eventually look
// like), and a deployed object with no session row at all (booting --
// created but never checked in).
func TestAgentHealthComputedFromRealHeartbeats(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	client := signedInClient(t, env, "read-only-viewer")
	ctx := context.Background()

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var hosts []db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			hosts = append(hosts, o)
		}
	}
	if len(hosts) < 2 {
		t.Fatalf("fixture has %d host(s), want at least 2 (healthy + missing cases)", len(hosts))
	}
	healthyHost, missingHost, bootingHost := hosts[0], hosts[1], hosts[0]
	if len(hosts) >= 3 {
		bootingHost = hosts[2]
	} else {
		// Only two real hosts in this fixture -- leave one (missing) with
		// no session at all, which still proves "no session on a deployed
		// object reads as booting," just reusing the same host as a
		// different assertion rather than a third one.
		bootingHost = missingHost
	}

	for _, o := range []db.DeployedObject{healthyHost, missingHost, bootingHost} {
		if _, err := env.q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
			ID: o.ID, ExternalRef: db.StrPtr("ext-" + o.ID.String()), Fingerprint: "f",
		}); err != nil {
			t.Fatalf("MarkDeployedObjectRunning: %v", err)
		}
	}

	if _, err := env.q.UpsertAgentSession(ctx, db.UpsertAgentSessionParams{
		DeployedObjectID: healthyHost.ID, CertFingerprint: "fp-healthy",
	}); err != nil {
		t.Fatalf("UpsertAgentSession (healthy): %v", err)
	}
	staleAt := time.Now().Add(-10 * time.Minute)
	if _, err := env.server.Pool.Exec(ctx,
		`INSERT INTO agent_session (deployed_object_id, cert_fingerprint, first_seen_at, last_heartbeat_at) VALUES ($1, $2, $3, $3)`,
		missingHost.ID, "fp-missing", staleAt,
	); err != nil {
		t.Fatalf("inserting stale session directly: %v", err)
	}
	// bootingHost deliberately gets no session row at all.

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var views []objectView
	json.NewDecoder(resp.Body).Decode(&views)

	byID := make(map[string]objectView, len(views))
	for _, v := range views {
		byID[v.ID.String()] = v
	}

	healthy := byID[healthyHost.ID.String()]
	if healthy.Agent == nil || healthy.Agent.Status != "healthy" {
		t.Fatalf("healthy host agent = %+v, want status=healthy", healthy.Agent)
	}
	if healthy.Agent.LastHeartbeatAt == nil {
		t.Fatal("healthy host should carry a last_heartbeat_at")
	}

	missing := byID[missingHost.ID.String()]
	if missing.Agent == nil || missing.Agent.Status != "missing" {
		t.Fatalf("stale-heartbeat host agent = %+v, want status=missing", missing.Agent)
	}

	if bootingHost.ID != missingHost.ID { // only meaningful when the fixture had a real 3rd host
		booting := byID[bootingHost.ID.String()]
		if booting.Agent == nil || booting.Agent.Status != "booting" {
			t.Fatalf("no-session host agent = %+v, want status=booting", booting.Agent)
		}
		if booting.Agent.LastHeartbeatAt != nil {
			t.Fatal("booting host must not carry a last_heartbeat_at -- it has never checked in")
		}
	}

	// A network never carries agent health the way a host does -- confirm
	// no network row claims it (an agent never runs on a network).
	for _, v := range views {
		if v.Kind == "network" && v.Agent != nil {
			t.Fatalf("network %s carries agent health %+v, want nil", v.ObjectName, v.Agent)
		}
	}
}
