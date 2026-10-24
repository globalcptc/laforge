package api

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestDeleteAgentHeartbeatsOlderThan is the real answer to migrations/00007's
// own "deliberately unaddressed: retention" note (agent_heartbeat grows
// without bound). cmd/laforge-orchestrator
// calls this query on its own periodic tick; this proves the query itself
// against a real deployed_object and real, backdated rows: an old
// heartbeat is removed, a recent one survives, and the reported row count
// matches exactly what was removed.
func TestDeleteAgentHeartbeatsOlderThan(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	build := realBuildFixture(t, env) // m7-two-team: 2 teams x (network + host + container)

	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) == 0 {
		t.Fatal("fixture produced no deployed objects")
	}
	objID := objs[0].ID

	old := time.Now().Add(-96 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	insert := func(at time.Time, fp string) {
		if _, err := env.server.Pool.Exec(ctx,
			`INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr, created_at) VALUES ($1,$2,$3,$4)`,
			objID, fp, "10.0.0.9:4433", at,
		); err != nil {
			t.Fatalf("inserting heartbeat at %s: %v", at, err)
		}
	}
	insert(old, "fp-old-1")
	insert(old, "fp-old-2")
	insert(recent, "fp-recent")

	cutoff := pgtype.Timestamptz{Time: time.Now().Add(-72 * time.Hour), Valid: true}
	removed, err := env.q.DeleteAgentHeartbeatsOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteAgentHeartbeatsOlderThan: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2 (the two backdated rows, not the recent one -- and not rows from other tests' builds)", removed)
	}

	remaining, err := env.q.ListAgentHeartbeatsByObject(ctx, db.ListAgentHeartbeatsByObjectParams{DeployedObjectID: objID, Limit: 100})
	if err != nil {
		t.Fatalf("ListAgentHeartbeatsByObject: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("remaining heartbeats for this object = %d, want 1", len(remaining))
	}
	if remaining[0].CertFingerprint != "fp-recent" {
		t.Fatalf("surviving heartbeat = %q, want the recent one (fp-recent)", remaining[0].CertFingerprint)
	}

	// Idempotent: nothing left to delete a second time.
	removedAgain, err := env.q.DeleteAgentHeartbeatsOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteAgentHeartbeatsOlderThan (second call): %v", err)
	}
	if removedAgain != 0 {
		t.Fatalf("second call removed %d, want 0", removedAgain)
	}
}
