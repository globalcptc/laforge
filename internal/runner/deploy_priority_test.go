package runner

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
)

// phasedBuilder is the fake builder plus what the Incus builder adds on a
// deploy: it records phases into the call's context and sees each spec's
// critical-path priority, in the order the runner deployed them.
type phasedBuilder struct {
	*fake.Builder
	mu    sync.Mutex
	hosts []builder.HostSpec
}

func (b *phasedBuilder) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	b.mu.Lock()
	b.hosts = append(b.hosts, spec)
	b.mu.Unlock()
	builder.RecordPhase(ctx, "queued", 1500*time.Millisecond)
	done := builder.StartPhase(ctx, "create")
	ref, err := b.Builder.DeployHost(ctx, spec)
	done()
	return ref, err
}

// TestDeployPriorityAndPhaseTimings: in lm-test, webserver depends_on database
// and workstation depends_on domain-controller. Every team's dependency roots
// must deploy before any team's leaf hosts -- leasing by critical-path
// priority, not team-by-team creation order -- with the priority handed to the
// builder, and the phases a builder records must land on the deploy task's
// task.completed event.
func TestDeployPriorityAndPhaseTimings(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "runner-deploy-priority")

	b := &phasedBuilder{Builder: fake.New(pool)}
	r := &Runner{
		Pool: pool, Builder: b, RepoRoot: "../../examples/lm-test",
		ID: "test-runner-priority", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}
	drainToConvergence(t, ctx, pool, q, r, "../../examples/lm-test", build.ID)

	if len(b.hosts) == 0 {
		t.Fatal("no hosts deployed")
	}
	// Once the first priority-0 host deploys, no prioritized host may follow:
	// every team's roots are leased ahead of every team's leaves.
	sawLeaf := false
	roots := 0
	for _, h := range b.hosts {
		if h.Priority == 0 {
			sawLeaf = true
			continue
		}
		roots++
		if sawLeaf {
			t.Fatalf("host %s (priority %d, team %s) deployed after a priority-0 host; order: %v", h.DisplayName, h.Priority, h.Team, deployOrder(b.hosts))
		}
	}
	t.Logf("deploy order: %v", deployOrder(b.hosts))
	if roots == 0 {
		t.Fatalf("no host carried a priority; order: %v", deployOrder(b.hosts))
	}
	for _, h := range b.hosts {
		isRoot := strings.Contains(h.DisplayName, "db") || strings.Contains(h.DisplayName, "dc")
		if isRoot != (h.Priority > 0) {
			t.Errorf("host %s priority %d: want >0 only for depends_on roots", h.DisplayName, h.Priority)
		}
	}

	rows, err := pool.Query(ctx, `SELECT message, payload FROM event WHERE build_id = $1 AND kind = 'task.completed' AND message LIKE 'deploy_host%'`, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var msg string
		var raw []byte
		if err := rows.Scan(&msg, &raw); err != nil {
			t.Fatal(err)
		}
		var p struct {
			Queued     string           `json:"queued"`
			Total      string           `json:"total"`
			PhaseOrder []string         `json:"phase_order"`
			PhasesMS   map[string]int64 `json:"phases_ms"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatalf("payload %s: %v", raw, err)
		}
		if p.Queued != "1.5s" || p.PhasesMS["queued"] != 1500 || len(p.PhaseOrder) != 2 || p.PhaseOrder[0] != "queued" || p.PhaseOrder[1] != "create" {
			t.Errorf("task.completed payload = %s, want queued 1.5s then create", raw)
		}
		if !strings.HasPrefix(msg, "deploy_host in ") {
			t.Errorf("task.completed message = %q, want it to carry the total", msg)
		}
		n++
	}
	if n != len(b.hosts) {
		t.Fatalf("%d deploy_host task.completed events, want %d", n, len(b.hosts))
	}
}

func deployOrder(hosts []builder.HostSpec) []string {
	out := make([]string, len(hosts))
	for i, h := range hosts {
		out[i] = h.DisplayName
	}
	return out
}

func TestPhasePayloadEmptyIsNil(t *testing.T) {
	if phasePayload(nil) != nil {
		t.Fatal("no phases should log an empty payload")
	}
}
