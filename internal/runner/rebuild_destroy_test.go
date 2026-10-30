package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

// recordingDestroyBuilder captures the exact reference a destroy is asked to
// delete, so a test can prove the runner used the real recorded reference and
// not a guessed fallback name.
type recordingDestroyBuilder struct {
	*fake.Builder
	destroyedRef string
}

func (b *recordingDestroyBuilder) DestroyHost(ctx context.Context, team, externalRef string) error {
	b.destroyedRef = externalRef
	return b.Builder.DestroyHost(ctx, team, externalRef)
}

// TestRebuildDestroyUsesPayloadRef is the issue-#1 runner-side regression: when a
// rebuild has already cleared the object's external_ref (to hold the build in the
// deploy phase), the destroy must still target the real instance via the ref
// carried in the task payload -- not fall back to the logical external name,
// which on a real hoster is a different string and "succeeds" as a no-op while
// the original VM keeps running.
func TestRebuildDestroyUsesPayloadRef(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "rebuild-destroy-payload")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "vdi-kali-02", AsName: db.StrPtr("vdi-kali-02"), NetworkName: db.StrPtr("lan"), Fingerprint: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	// The real recorded instance, deliberately NOT equal to the logical external
	// name the fallback would compute.
	const realRef = "lf-cptc12-forrest-t1-vdi-kali-02-efc5c7"
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='running', external_ref=$2 WHERE id=$1", obj.ID, realRef); err != nil {
		t.Fatal(err)
	}
	// The old instance's agent had checked in.
	if _, err := pool.Exec(ctx, "INSERT INTO agent_session (deployed_object_id, cert_fingerprint) VALUES ($1, 'fp-old')", obj.ID); err != nil {
		t.Fatal(err)
	}

	// Simulate what Rebuild does: enqueue the destroy carrying the real ref, then
	// clear the row's external_ref.
	if _, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: build.ID, DeployedObjectID: obj.ID, Kind: "destroy_host",
		Payload: []byte(`{"remove":false,"external_ref":"` + realRef + `"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ResetDeployedObjectForRedeploy(ctx, obj.ID); err != nil { // clears external_ref
		t.Fatal(err)
	}

	b := &recordingDestroyBuilder{Builder: fake.New(pool)}
	r := &Runner{Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test", ID: "rebuild-runner", BuildID: &build.ID, LeaseDuration: time.Minute, HeartbeatInterval: time.Second}
	ran, err := r.LeaseAndExecuteOne(ctx)
	if err != nil || !ran {
		t.Fatalf("LeaseAndExecuteOne: ran=%v err=%v", ran, err)
	}
	if b.destroyedRef != realRef {
		t.Errorf("destroyed %q, want the real recorded ref %q (fell back to a guessed name)", b.destroyedRef, realRef)
	}

	// After a rebuild destroy the object is reset for redeploy, not deleted.
	reread, err := q.GetDeployedObject(ctx, obj.ID)
	if err != nil {
		t.Fatalf("GetDeployedObject: %v", err)
	}
	if reread.Status != "pending" {
		t.Errorf("object status = %q, want pending (reset for redeploy)", reread.Status)
	}

	// The destroyed instance's agent session is gone too, so the rebuilt host
	// waits for its new agent instead of inheriting the old heartbeat.
	var sessions int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM agent_session WHERE deployed_object_id = $1", obj.ID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 {
		t.Errorf("agent_session rows after rebuild destroy = %d, want 0", sessions)
	}
}
