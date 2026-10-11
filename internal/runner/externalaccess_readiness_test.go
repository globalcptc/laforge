package runner

import (
	"context"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

type startingPublicBuilder struct {
	*fake.Builder
	running bool
	calls   int
}

func (b *startingPublicBuilder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	b.calls++
	if !b.running {
		return nil, builder.ErrExternalAccessNotReady
	}
	return b.Builder.ConfigureExternalAccess(ctx, team, hosts)
}

func TestForwardingWaitDoesNotExhaustRetries(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "forwarding-readiness")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "wireguard", AsName: db.StrPtr("wireguard"), NetworkName: db.StrPtr("vpn"), Fingerprint: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='running', external_ref='public-vm' WHERE id=$1", obj.ID); err != nil {
		t.Fatal(err)
	}
	task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: build.ID, Kind: "configure_external_access", Payload: []byte(`{"team":"1"}`)})
	if err != nil {
		t.Fatal(err)
	}
	b := &startingPublicBuilder{Builder: fake.New(pool)}
	r := &Runner{Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test", ID: "readiness-runner", BuildID: &build.ID, LeaseDuration: time.Minute, HeartbeatInterval: time.Second, RetryBackoffBase: time.Minute}
	for i := 0; i < int(MaxAttempts)+2; i++ {
		if _, err := pool.Exec(ctx, "UPDATE task SET lease_expires_at=now()-interval '1 second' WHERE id=$1", task.ID); err != nil {
			t.Fatal(err)
		}
		ran, err := r.LeaseAndExecuteOne(ctx)
		if err != nil || !ran {
			t.Fatalf("attempt %d: ran=%v error=%v", i, ran, err)
		}
		held, err := q.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if held.Status != "leased" || held.Attempts != 0 {
			t.Fatalf("startup wait consumed retry: status=%s attempts=%d error=%v", held.Status, held.Attempts, held.LastError)
		}
		endpoints, err := q.ListExternalAccessByObject(ctx, obj.ID)
		if err != nil || len(endpoints) != 0 {
			t.Fatalf("recorded endpoints before running: %v error=%v", endpoints, err)
		}
	}
	b.running = true
	if _, err := pool.Exec(ctx, "UPDATE task SET lease_expires_at=now()-interval '1 second' WHERE id=$1", task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LeaseAndExecuteOne(ctx); err != nil {
		t.Fatal(err)
	}
	done, err := q.GetTask(ctx, task.ID)
	if err != nil || done.Status != "done" {
		t.Fatalf("task did not complete: %+v %v", done, err)
	}
	endpoints, err := q.ListExternalAccessByObject(ctx, obj.ID)
	if err != nil || len(endpoints) != 2 {
		t.Fatalf("running VM endpoints: %+v %v", endpoints, err)
	}
}
