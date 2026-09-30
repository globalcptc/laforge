package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// writeMaterializeTestRepo is a minimal, synthetic content checkout (not
// examples/lm-test, which uses `script:` for its own real schedule entry
// -- a two-command action materializeSchedule correctly skips today, see
// its own doc comment) with a host whose schedule: entry resolves to
// exactly ONE agent command (reboot), so materialization actually has
// something real to produce.
func writeMaterializeTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"lm-test.yaml": `environment:
  name: lm-test
  teams: 1
  networks:
    prod:
      webserver:
        - as: web01
          last_octet: 10
`,
		"networks/prod.yaml": `network:
  name: prod
  cidr: 10.0.1.0/24
`,
		"hosts/webserver.yaml": `host:
  name: webserver
  os: ubuntu22
  size: small
  disk: 20
  schedule:
    - when: Every 30 minutes
      reboot: {}
`,
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestReconcileMaterializesSingleCommandScheduleEntry is the content
// half of "real execution, dispatched to real agents, not a stub" --
// deploying a host with a `schedule:` entry that resolves to exactly one
// agent command (reboot, no rendering needed) produces a real
// scheduled_task row, ready for the same dispatch loop
// TestDispatchDueScheduledTaskCreatesRealAgentTask already proves fires
// against a live agent.
func TestReconcileMaterializesSingleCommandScheduleEntry(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "schedule-materialize-test")
	repoRoot := writeMaterializeTestRepo(t)

	if err := Reconcile(ctx, pool, repoRoot, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var web01 db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			web01 = o
		}
	}
	if web01.ID.String() == "" {
		t.Fatal("web01 deployed_object not found")
	}

	tasks, err := q.ListScheduledTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListScheduledTasksByBuild: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("scheduled_task rows = %d, want 1", len(tasks))
	}
	st := tasks[0]
	if st.Source != "content" {
		t.Errorf("Source = %q, want \"content\"", st.Source)
	}
	if !st.DeployedObjectID.Valid || st.DeployedObjectID != web01.ID {
		t.Errorf("DeployedObjectID = %v, want %v", st.DeployedObjectID, web01.ID)
	}
	if st.Command != "reboot" {
		t.Errorf("Command = %q, want \"reboot\"", st.Command)
	}
	if st.WhenExpr != "Every 30 minutes" {
		t.Errorf("WhenExpr = %q, want \"Every 30 minutes\"", st.WhenExpr)
	}
	if st.Status != "pending" || !st.NextFireAt.Valid {
		t.Errorf("Status/NextFireAt = %q/%v, want pending with a real next fire time", st.Status, st.NextFireAt)
	}

	// Reconciling again must NOT duplicate the row -- created only gates
	// materialization on a genuinely new deploy task, and even if it
	// fired twice, CreateContentScheduledTask's own ON CONFLICT upserts
	// rather than duplicating.
	if err := Reconcile(ctx, pool, repoRoot, build.ID); err != nil {
		t.Fatalf("Reconcile (second pass): %v", err)
	}
	tasksAfter, err := q.ListScheduledTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListScheduledTasksByBuild (after): %v", err)
	}
	if len(tasksAfter) != 1 {
		t.Fatalf("scheduled_task rows after a second reconcile = %d, want still 1 (no duplication)", len(tasksAfter))
	}
}

// TestDispatchMultiCommandScheduleEntryEnqueuesEveryCommand is the real
// fix for materializeSchedule's earlier single-command-only limit:
// examples/lm-test/hosts/webserver.yaml's real schedule entry uses
// `script: reboot`, which gateway.ExpandOneStep resolves to TWO agent
// commands (write_file the rendered script, then execute it) -- both
// must reach agent_task, in order, not just the first or neither.
// Materialization itself stays cheap (just records which entry, not
// its resolved commands -- see materializeSchedule's own doc comment);
// the real expansion happens at dispatch time, against live content.
func TestDispatchMultiCommandScheduleEntryEnqueuesEveryCommand(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "schedule-dispatch-multicommand-test")

	// webserver (which carries the schedule) depends_on database, so its
	// schedule materializes only once it actually deploys -- drive to
	// convergence rather than a single Reconcile.
	deployToConvergence(t, pool, "../../examples/lm-test", build.ID)
	tasks, err := q.ListScheduledTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListScheduledTasksByBuild: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected at least one materialized scheduled_task (webserver's real script: reboot entry)")
	}
	st := tasks[0]
	if st.Command != "script" {
		t.Fatalf("Command = %q, want \"script\" (materialization records the action key for display; the real expansion happens at dispatch time)", st.Command)
	}

	// The full deploy already materialized webserver's own steps into
	// agent_task rows, so capture the count first and assert the dispatch
	// appends exactly the two the schedule expands to.
	before, err := q.ListAgentTasksByHost(ctx, st.DeployedObjectID)
	if err != nil {
		t.Fatalf("ListAgentTasksByHost (before): %v", err)
	}

	// Force it due, then dispatch for real.
	if _, err := pool.Exec(ctx, "UPDATE scheduled_task SET next_fire_at = now() - interval '1 second' WHERE id = $1", st.ID); err != nil {
		t.Fatalf("forcing due: %v", err)
	}
	if err := DispatchDueScheduledTasks(ctx, pool, FixedRepoRoot("../../examples/lm-test"), 100); err != nil {
		t.Fatalf("DispatchDueScheduledTasks: %v", err)
	}

	agentTasks, err := q.ListAgentTasksByHost(ctx, st.DeployedObjectID)
	if err != nil {
		t.Fatalf("ListAgentTasksByHost: %v", err)
	}
	added := len(agentTasks) - len(before)
	if added != 2 {
		t.Fatalf("dispatch added %d agent_task rows for %s, want 2 (write_file then execute -- script: reboot's real expansion)", added, st.DeployedObjectID)
	}
	// The two newest (appended after the host's existing steps) are the
	// dispatched pair, in order.
	newest := agentTasks[len(agentTasks)-2:]
	if newest[0].Command != "write_file" {
		t.Errorf("newest[0].Command = %q, want \"write_file\"", newest[0].Command)
	}
	if newest[1].Command != "execute" {
		t.Errorf("newest[1].Command = %q, want \"execute\"", newest[1].Command)
	}
	if newest[0].StepIndex >= newest[1].StepIndex {
		t.Errorf("step order wrong: write_file at %d, execute at %d -- must run in order", newest[0].StepIndex, newest[1].StepIndex)
	}

	updated, err := q.GetScheduledTask(ctx, st.ID)
	if err != nil {
		t.Fatalf("GetScheduledTask: %v", err)
	}
	if updated.Status != "pending" || !updated.NextFireAt.Valid {
		t.Errorf("Status/NextFireAt = %q/%v, want still pending with a real next fire time (recurring, not fires-once)", updated.Status, updated.NextFireAt)
	}
}
