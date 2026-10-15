package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// writeWaitForRepo is a two-team environment: dc marks milestone "ready" on its
// second step; ws runs one step, then waits on that milestone; vdi runs one
// step, then waits for dc to finish everything.
func writeWaitForRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"lm-test.yaml": `environment:
  name: lm-test
  teams: 2
  networks:
    lan:
      dc:
        - { as: dc01, last_octet: 10 }
      ws:
        - { as: ws01, last_octet: 20 }
      vdi:
        - { as: vdi01, last_octet: 30 }
`,
		"networks/lan.yaml": "network:\n  name: lan\n  cidr: 10.0.1.0/24\n",
		"hosts/dc.yaml": `host:
  name: dc
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - script: one
    - script: two
      milestone: ready
    - script: three
`,
		"hosts/ws.yaml": `host:
  name: ws
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - script: one
    - script: two
      wait_for:
        host: dc
        milestone: ready
    - script: three
`,
		"hosts/vdi.yaml": `host:
  name: vdi
  os: ubuntu22
  size: small
  disk: 20
  steps:
    - script: one
    - script: two
      wait_for: { host: dc, rebuild: false }
`,
	}
	for _, n := range []string{"one", "two", "three"} {
		files["scripts/"+n+".yaml"] = "script:\n  name: " + n + "\n  language: bash\n  source: " + n + ".sh\n"
		files["scripts/"+n+".sh"] = "true\n"
	}
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loader.Load(root)
	if err != nil || len(c.Errors) > 0 {
		t.Fatalf("test repo does not load clean: %v %v", err, c.Errors)
	}
	return root
}

// TestWaitForHoldsFromTheGatedStep walks a build through a step-level gate:
// steps before the gate run at once, the gated step and everything after it
// wait for the target's milestone on the SAME team's copy, a failed target
// fails the waiter, and a whole-host wait releases only when the target is
// finished. (Each script step expands to write_file + execute: 2 rows.)
func TestWaitForHoldsFromTheGatedStep(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	repo := writeWaitForRepo(t)
	build := newTestBuild(t, pool, "wait-for-gates")

	reconcileWithNetworksUp(t, ctx, pool, repo, build.ID)
	if _, err := pool.Exec(ctx,
		"UPDATE deployed_object SET status='running' WHERE kind='host' AND team_id IN (SELECT id FROM team WHERE build_id=$1)",
		build.ID); err != nil {
		t.Fatalf("marking hosts up: %v", err)
	}
	reconcile := func() {
		t.Helper()
		if err := Reconcile(ctx, pool, repo, build.ID); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	}
	reconcile()

	obj := func(team int, name string) db.DeployedObject {
		t.Helper()
		var id pgtype.UUID
		if err := pool.QueryRow(ctx,
			`SELECT o.id FROM deployed_object o JOIN team ON team.id = o.team_id
			 WHERE team.build_id = $1 AND team.team_number = $2 AND o.object_name = $3`,
			build.ID, team, name).Scan(&id); err != nil {
			t.Fatalf("finding team %d %s: %v", team, name, err)
		}
		o, err := q.GetDeployedObject(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	statuses := func(team int, name string) string {
		t.Helper()
		rows, err := q.ListAgentTasksByHost(ctx, obj(team, name).ID)
		if err != nil {
			t.Fatal(err)
		}
		var s []string
		for _, r := range rows {
			s = append(s, r.Status)
		}
		return strings.Join(s, ",")
	}
	blockedOn := func(team int, name string) []string {
		t.Helper()
		var out []string
		if b := obj(team, name).BlockedOn; len(b) > 0 {
			_ = json.Unmarshal(b, &out)
		}
		return out
	}
	setRows := func(team int, name, status string, idx ...int) {
		t.Helper()
		for _, i := range idx {
			if _, err := pool.Exec(ctx, "UPDATE agent_task SET status=$1 WHERE deployed_object_id=$2 AND step_index=$3",
				status, obj(team, name).ID, i); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Materialized: ws's first step runs now; the gated step and the rest wait.
	for team := 1; team <= 2; team++ {
		if got, want := statuses(team, "ws"), "pending,pending,blocked,blocked,blocked,blocked"; got != want {
			t.Fatalf("team %d ws steps = %s, want %s", team, got, want)
		}
		if got := blockedOn(team, "ws"); len(got) != 0 {
			t.Fatalf("team %d ws blocked_on = %v while still working toward the gate, want empty", team, got)
		}
		if got, want := statuses(team, "dc"), "pending,pending,pending,pending,pending,pending"; got != want {
			t.Fatalf("team %d dc steps = %s, want %s", team, got, want)
		}
	}
	// The target side carries its milestone and script on every row of the step.
	var marked int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM agent_task WHERE deployed_object_id=$1 AND milestone='ready' AND step_script='two'",
		obj(1, "dc").ID).Scan(&marked); err != nil || marked != 2 {
		t.Fatalf("dc rows marked with milestone ready = %d (%v), want 2", marked, err)
	}

	// ws team 1 works up to the gate: now it is actually waiting, and says on what.
	setRows(1, "ws", "done", 0, 1)
	reconcile()
	if got := blockedOn(1, "ws"); len(got) != 1 || got[0] != "dc · ready" {
		t.Fatalf("team 1 ws blocked_on = %v, want [dc · ready]", got)
	}

	// dc team 1 reaches the milestone (its later step still pending); dc team 2
	// does not. Only team 1's ws is released -- gates are per team.
	setRows(1, "dc", "done", 0, 1, 2, 3)
	reconcile()
	if got, want := statuses(1, "ws"), "done,done,pending,pending,pending,pending"; got != want {
		t.Fatalf("team 1 ws after dc reached ready = %s, want %s", got, want)
	}
	if got := blockedOn(1, "ws"); len(got) != 0 {
		t.Fatalf("team 1 ws blocked_on = %v after release, want empty", got)
	}
	if got, want := statuses(2, "ws"), "pending,pending,blocked,blocked,blocked,blocked"; got != want {
		t.Fatalf("team 2 ws = %s, want still held (%s)", got, want)
	}
	// The whole-host wait isn't satisfied by a milestone: dc isn't finished.
	if got, want := statuses(1, "vdi"), "pending,pending,blocked,blocked"; got != want {
		t.Fatalf("team 1 vdi = %s, want %s", got, want)
	}

	// dc team 1 finishes: vdi's whole-host wait releases.
	if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status='finished' WHERE id=$1", obj(1, "dc").ID); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if got, want := statuses(1, "vdi"), "pending,pending,pending,pending"; got != want {
		t.Fatalf("team 1 vdi after dc finished = %s, want %s", got, want)
	}

	// dc team 2 fails the milestone step for good: team 2's ws can never
	// continue, so it fails, naming what it waited on.
	setRows(2, "dc", "failed", 2)
	reconcile()
	ws2 := obj(2, "ws")
	if ws2.Status != "build_failed" || !strings.Contains(db.StrOrEmpty(ws2.LastError), "dc · ready, which failed") {
		t.Fatalf("team 2 ws = %s (%q), want build_failed naming dc · ready", ws2.Status, db.StrOrEmpty(ws2.LastError))
	}
}

func TestTeamGatesEveryCopyMustReach(t *testing.T) {
	g := &teamGates{
		placed:   map[string]bool{"dc": true},
		finished: map[string]bool{}, failed: map[string]bool{},
		copies: map[string][]string{"dc": {"a", "b"}},
		milestone: map[string]map[string]map[string]*pointProgress{
			"dc": {"ready": {"a": {total: 2, reached: 2}}},
		},
		script: map[string]map[string]map[string]*pointProgress{},
	}
	w := loader.WaitFor{Host: "dc", Milestone: "ready"}
	if got := g.evaluate(w); got != gateWaiting {
		t.Fatalf("one of two copies reached: %v, want waiting", got)
	}
	g.milestone["dc"]["ready"]["b"] = &pointProgress{total: 2, reached: 2}
	if got := g.evaluate(w); got != gateSatisfied {
		t.Fatalf("both copies reached: %v, want satisfied", got)
	}
	// Reached, then the target failed on a later step: still satisfied.
	g.failed["dc"] = true
	if got := g.evaluate(w); got != gateSatisfied {
		t.Fatalf("reached before a later failure: %v, want satisfied", got)
	}
	// A target not placed in this environment is skipped, not waited on forever.
	if got := g.evaluate(loader.WaitFor{Host: "elsewhere", Milestone: "x"}); got != gateSatisfied {
		t.Fatalf("unplaced target: %v, want satisfied", got)
	}
	// Not reached and the target failed: the waiter fails.
	g.milestone["dc"]["ready"]["b"] = &pointProgress{total: 2, reached: 1}
	if got := g.evaluate(w); got != gateFailed {
		t.Fatalf("unreached with failed target: %v, want failed", got)
	}
}
