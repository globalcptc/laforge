package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

// recordingNetAccessBuilder captures the NetworkAccess the runner hands to the
// builder, so a test can prove every intended guest's NIC policy was applied --
// and that nothing was applied while guests were still unready.
type recordingNetAccessBuilder struct {
	*fake.Builder
	calls    int
	lastRefs map[string]bool
}

func (b *recordingNetAccessBuilder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	b.calls++
	b.lastRefs = map[string]bool{}
	for _, n := range networks {
		for _, h := range n.Hosts {
			b.lastRefs[h.ExternalRef] = true
		}
	}
	return b.Builder.ConfigureNetworkAccess(ctx, team, networks)
}

// TestNetworkAccessDefersUntilGuestsReady is the issue regression: a private-NIC
// ACL task must WAIT (defer, no retry consumed) while a guest is still being
// created, past the normal three-attempt budget, and then apply every guest's
// policy once ready -- never report success over a skipped guest.
func TestNetworkAccessDefersUntilGuestsReady(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "netaccess-readiness")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	// One network (vpn) and one guest on it (wireguard), both real lm-test content.
	netObj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "network", ObjectName: "vpn", Fingerprint: "net"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='finished', external_ref='net-vpn' WHERE id=$1", netObj.ID); err != nil {
		t.Fatal(err)
	}
	host, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "wireguard", AsName: db.StrPtr("wireguard"), NetworkName: db.StrPtr("vpn"), Fingerprint: "h",
	})
	if err != nil {
		t.Fatal(err)
	}
	// The guest starts still-being-created: pending, no instance reference.
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='pending', external_ref=NULL WHERE id=$1", host.ID); err != nil {
		t.Fatal(err)
	}

	task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: build.ID, Kind: "configure_network_access", Payload: []byte(`{"team":"1","fingerprint":"fp"}`)})
	if err != nil {
		t.Fatal(err)
	}
	b := &recordingNetAccessBuilder{Builder: fake.New(pool)}
	r := &Runner{Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test", ID: "netaccess-runner", BuildID: &build.ID, LeaseDuration: time.Minute, HeartbeatInterval: time.Second, RetryBackoffBase: time.Minute}

	// Readiness waits must not consume the three-attempt budget, no matter how
	// many passes the guest takes to come up.
	for i := 0; i < int(MaxAttempts)+2; i++ {
		if _, err := pool.Exec(ctx, "UPDATE task SET lease_expires_at=now()-interval '1 second' WHERE id=$1", task.ID); err != nil {
			t.Fatal(err)
		}
		ran, err := r.LeaseAndExecuteOne(ctx)
		if err != nil || !ran {
			t.Fatalf("pass %d: ran=%v err=%v", i, ran, err)
		}
		held, err := q.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if held.Status == "failed" || held.Attempts != 0 {
			t.Fatalf("readiness wait consumed the retry budget: status=%s attempts=%d err=%v", held.Status, held.Attempts, held.LastError)
		}
		if b.calls != 0 {
			t.Fatalf("builder ConfigureNetworkAccess was called %d time(s) while the guest was unready", b.calls)
		}
	}

	// Guest is now up with a real reference; the ACL must now be applied to it.
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='running', external_ref='lf-wireguard-abc123' WHERE id=$1", host.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE task SET lease_expires_at=now()-interval '1 second' WHERE id=$1", task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LeaseAndExecuteOne(ctx); err != nil {
		t.Fatal(err)
	}
	done, err := q.GetTask(ctx, task.ID)
	if err != nil || done.Status != "done" {
		t.Fatalf("task did not complete once the guest was ready: %+v %v", done, err)
	}
	if b.calls != 1 {
		t.Fatalf("builder ConfigureNetworkAccess calls = %d, want 1 (once ready)", b.calls)
	}
	if !b.lastRefs["lf-wireguard-abc123"] {
		t.Errorf("the ready guest's NIC was not included in the applied policy: %v", b.lastRefs)
	}
}
