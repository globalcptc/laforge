package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// copyModifiedLmTest copies examples/lm-test into a fresh temp dir with
// one edit: the entire `vpn:` network section (one network, one host copy
// -- wireguard) removed from lm-test.yaml. Used to prove the other half of
// Reconcile's diff that TestReconcileOnRealExampleRepo doesn't cover:
// "diffs desired vs observed state ... and creates tasks (deploy or
// destroy...)" for something REMOVED from content, not just something new
// or changed.
func copyModifiedLmTest(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.Walk("../../examples/lm-test", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel("../../examples/lm-test", path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == "lm-test.yaml" {
			content = []byte(removeVPNNetwork(string(content)))
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, content, info.Mode())
	})
	if err != nil {
		t.Fatalf("copying modified examples/lm-test: %v", err)
	}
	return dst
}

// removeVPNNetwork strips the `    vpn:` topology block (per
// examples/lm-test/lm-test.yaml's current shape: `networks:` sits one
// level inside `environment:`, so its own entries -- `prod:`, `dev:`,
// `vpn:`, ... -- are at 4-space indent, not 2) from the environment
// file's raw text. A crude text edit rather than a YAML-aware one, but
// this test only needs the *result* (one fewer network, one fewer host
// in the resolved topology), not to demonstrate YAML editing.
func removeVPNNetwork(yaml string) string {
	lines := strings.Split(yaml, "\n")
	var out []string
	skip := false
	for _, line := range lines {
		if strings.HasPrefix(line, "    vpn:") {
			skip = true
			continue
		}
		if skip {
			// Still inside the vpn: block as long as the line is indented
			// deeper than "    vpn:" itself (6+ spaces) -- the next
			// sibling ("    dev:", "    vdi:", etc.) or lesser-indented
			// line ends it.
			if strings.HasPrefix(line, "      ") {
				continue
			}
			skip = false
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// TestReconcileAndRunnerRemoveDroppedObjects proves the removal half of
// "the orchestrator diffs desired vs observed state ... and creates tasks
// (deploy or destroy...)": deploy the real lm-test content fully, then
// reconcile the SAME build against content with the vpn network (and its
// one wireguard host) removed, and confirm a real runner destroying that
// object actually deletes its deployed_object row -- not just resets it
// for redeploy, which is what a fingerprint-changed (but still desired)
// object gets instead.
func TestReconcileAndRunnerRemoveDroppedObjects(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "runner-removal")

	r := &Runner{
		Pool: pool, Builder: fake.New(pool), RepoRoot: "../../examples/lm-test",
		ID: "test-runner-removal", LeaseDuration: 10 * time.Second, HeartbeatInterval: 2 * time.Second,
		BuildID: &build.ID,
	}
	// Interleave reconcile + drain to convergence; every box deploys regardless
	// of depends_on (which now gates step execution, not the deploy).
	if n := drainToConvergence(t, ctx, pool, q, r, "../../examples/lm-test", build.ID); n != 65 {
		t.Fatalf("initial full deploy drained %d tasks, want 65", n)
	}

	before, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (before removal): %v", err)
	}
	if len(before) != 65 {
		t.Fatalf("len(before) = %d, want 65", len(before))
	}

	modifiedRepo := copyModifiedLmTest(t)
	// RepoRoot for both Reconcile and the runner must point at the
	// modified content from here on -- a real webhook-driven rebuild
	// would have fetched a new commit into a new checkout the same way.
	r.RepoRoot = modifiedRepo
	if err := orchestrator.Reconcile(ctx, pool, modifiedRepo, build.ID); err != nil {
		t.Fatalf("Reconcile (modified content, vpn removed): %v", err)
	}

	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	var destroyTasks int
	for _, tk := range tasks {
		if tk.Status == "pending" && strings.HasPrefix(tk.Kind, "destroy_") {
			destroyTasks++
		}
	}
	// One per team (5 teams) x one object removed (wireguard; the vpn
	// network itself has no other copies so it's removed too) = 10.
	if destroyTasks != 10 {
		t.Fatalf("pending destroy tasks after removing vpn = %d, want 10 (5 teams x {vpn network, wireguard host})", destroyTasks)
	}

	if n := drainQueue(t, ctx, r, 50); n != 10 {
		t.Fatalf("drained %d removal tasks, want 10", n)
	}

	after, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (after removal): %v", err)
	}
	if len(after) != 55 {
		// 65 - 10: the vpn network and its one wireguard host, removed
		// from EVERY team (5 teams x 2 objects each), not just once --
		// every team gets its own independent copy of the topology.
		t.Fatalf("len(after) = %d, want 55 (65 - 10: vpn network + wireguard host, removed from all 5 teams)", len(after))
	}
	for _, o := range after {
		if o.NetworkName != nil && *o.NetworkName == "vpn" {
			t.Fatalf("object %s is still on the removed vpn network", o.ObjectName)
		}
		if o.Kind == "network" && o.ObjectName == "vpn" {
			t.Fatal("the vpn network's deployed_object row is still present after removal")
		}
	}

	// A further reconcile pass against the modified content must be a
	// true no-op now -- nothing left to remove, nothing new to deploy.
	if err := orchestrator.Reconcile(ctx, pool, modifiedRepo, build.ID); err != nil {
		t.Fatalf("Reconcile (modified content, second pass): %v", err)
	}
	finalTasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild (final): %v", err)
	}
	for _, tk := range finalTasks {
		if tk.Status == "pending" || tk.Status == "leased" {
			t.Fatalf("unexpected open task after removal fully converged: %s %s", tk.Kind, tk.Status)
		}
	}
}
