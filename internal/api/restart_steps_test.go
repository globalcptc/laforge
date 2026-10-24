package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/runner"
)

// stepsBuildFixture is a deploying build of examples/lm-test (hosts with
// real `steps:`), its boxes brought up by the real runner against the fake
// builder and its steps materialized -- what restart-from-step acts on.
func stepsBuildFixture(t *testing.T, env *apiTestEnv) db.Build {
	t.Helper()
	ctx := context.Background()
	const root = "../../examples/lm-test"
	rev, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: env.repo.ID, CommitSha: "restart-steps-sha"})
	if err != nil {
		t.Fatal(err)
	}
	build, err := env.q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "lm-test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatal(err)
	}
	build.Status = "deploying"
	reconcile := func() {
		if err := orchestrator.Reconcile(ctx, env.server.Pool, root, build.ID); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	}
	reconcile()
	if _, err := env.server.Pool.Exec(ctx, "UPDATE deployed_object SET status='running' WHERE kind='network' AND team_id IN (SELECT id FROM team WHERE build_id=$1)", build.ID); err != nil {
		t.Fatal(err)
	}
	reconcile()
	r := &runner.Runner{Pool: env.server.Pool, Builder: fake.New(env.server.Pool), RepoRoot: root, ID: "restart-test-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: 5 * time.Second, BuildID: &build.ID}
	for i := 0; i < 500; i++ {
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !did {
			break
		}
	}
	reconcile() // boxes are up: steps materialize
	return build
}

func TestRestartFromStep(t *testing.T) {
	env := setupAPITest(t)
	build := stepsBuildFixture(t, env)
	ctx := context.Background()
	admin := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")

	// Team 1's database host: a step in the middle of its build fails.
	var obj db.DeployedObject
	objs, _ := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	teams, _ := env.q.ListTeamsByBuild(ctx, build.ID)
	teamNum := map[string]int32{}
	for _, tm := range teams {
		teamNum[tm.ID.String()] = tm.TeamNumber
	}
	instances := 0
	for _, o := range objs {
		if o.ObjectName == "database" {
			instances++
			if teamNum[o.TeamID.String()] == 1 {
				obj = o
			}
		}
	}
	if !obj.ID.Valid {
		t.Fatal("no team-1 database host")
	}
	before, err := env.q.ListAgentTasksByHost(ctx, obj.ID)
	if err != nil || len(before) == 0 {
		t.Fatalf("database has %d steps (%v); the fixture should have materialized them", len(before), err)
	}
	for _, tk := range before {
		if tk.StepGroup == nil {
			t.Fatalf("step %d has no step_group; materialization should stamp it", tk.StepIndex)
		}
	}
	// Steps 0 done, step 1 failed, the rest never ran.
	env.server.Pool.Exec(ctx, "UPDATE agent_task SET status = 'done' WHERE deployed_object_id = $1 AND step_group = 0", obj.ID)
	env.server.Pool.Exec(ctx, "UPDATE agent_task SET status = 'failed', last_error = 'mirror timed out' WHERE deployed_object_id = $1 AND step_group = 1", obj.ID)
	env.server.Pool.Exec(ctx, "UPDATE deployed_object SET status = 'build_failed' WHERE id = $1", obj.ID)
	var keep []pgtype.UUID
	for _, tk := range before {
		if *tk.StepGroup < 1 {
			keep = append(keep, tk.ID)
		}
	}

	post := func(body string) (int, map[string]any) {
		resp, err := admin.Post(env.httpURL+"/builds/"+build.ID.String()+"/restart-steps", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	// The dry run widens to every team's database and names what runs again.
	status, out := post(`{"target":{"ids":["` + obj.ID.String() + `"]},"from_group":1,"dry_run":true}`)
	if status != http.StatusOK {
		t.Fatalf("dry run = %d %v", status, out)
	}
	hosts := out["hosts"].([]any)
	if len(hosts) != instances {
		t.Fatalf("dry run listed %d hosts, want every team's database (%d)", len(hosts), instances)
	}
	eligible := 0
	for _, h := range hosts {
		m := h.(map[string]any)
		if m["eligible"] == true && m["skipped"] == nil {
			eligible++
			if m["id"] != obj.ID.String() {
				t.Fatalf("an instance other than the failed one is eligible: %v", m)
			}
			if steps := m["steps"].([]any); len(steps) == 0 || steps[0].(map[string]any)["group"].(float64) != 1 {
				t.Fatalf("eligible host's steps = %v, want from group 1", m["steps"])
			}
		}
	}
	if eligible != 1 {
		t.Fatalf("%d eligible instances, want 1 (only team 1's failed)", eligible)
	}

	// Restart it.
	status, out = post(`{"target":{"ids":["` + obj.ID.String() + `"]},"from_group":1}`)
	if status != http.StatusOK || out["restarted"].(float64) != 1 {
		t.Fatalf("restart = %d %v", status, out)
	}
	after, _ := env.q.ListAgentTasksByHost(ctx, obj.ID)
	kept := map[string]bool{}
	for _, id := range keep {
		kept[id.String()] = true
	}
	var newer int
	for _, tk := range after {
		if *tk.StepGroup < 1 {
			if !kept[tk.ID.String()] || tk.Status != "done" {
				t.Fatalf("step before the restart point changed: %+v", tk)
			}
			continue
		}
		if kept[tk.ID.String()] {
			t.Fatalf("a re-queued step kept its old id: %+v", tk)
		}
		if tk.Status != "pending" || tk.Attempts != 0 || tk.LastError != nil {
			t.Fatalf("re-queued step isn't fresh: %+v", tk)
		}
		newer++
	}
	if newer == 0 || len(after) != len(before) {
		t.Fatalf("after the restart: %d rows (%d new), before: %d -- the same steps should be queued again", len(after), newer, len(before))
	}
	got, _ := env.q.GetDeployedObject(ctx, obj.ID)
	if got.Status != "building" || got.LastError != nil {
		t.Fatalf("host after restart = %s (%v), want building with no error", got.Status, got.LastError)
	}
	var events int
	env.server.Pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE build_id = $1 AND kind = 'steps.restarted'", build.ID).Scan(&events)
	if events != 1 {
		t.Fatalf("steps.restarted events = %d, want 1", events)
	}

	// A step running right now refuses the restart unless forced.
	env.server.Pool.Exec(ctx, "UPDATE agent_task SET status = 'leased', lease_expires_at = now() + interval '5 minutes', attempts = 1 WHERE deployed_object_id = $1 AND step_group = 1", obj.ID)
	env.server.Pool.Exec(ctx, "UPDATE deployed_object SET status = 'build_failed' WHERE id = $1", obj.ID)
	if status, out = post(`{"target":{"ids":["` + obj.ID.String() + `"]},"from_group":1}`); status != http.StatusConflict {
		t.Fatalf("restart while a step runs = %d %v, want 409", status, out)
	}
	if status, out = post(`{"target":{"ids":["` + obj.ID.String() + `"]},"from_group":1,"force":true}`); status != http.StatusOK || out["restarted"].(float64) != 1 {
		t.Fatalf("forced restart = %d %v", status, out)
	}

	// Two host definitions at once are refused.
	var other string
	for _, o := range objs {
		if o.ObjectName != "database" && o.Kind == "host" {
			other = o.ID.String()
			break
		}
	}
	if status, _ = post(`{"target":{"ids":["` + obj.ID.String() + `","` + other + `"]},"from_group":0,"dry_run":true}`); status != http.StatusBadRequest && status != http.StatusConflict {
		t.Fatalf("mixed definitions = %d, want a refusal", status)
	}

	// The steps list attributes rows by step_group.
	resp, err := admin.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + obj.ID.String() + "/steps")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var views []map[string]any
	json.NewDecoder(resp.Body).Decode(&views)
	if len(views) != len(after) {
		t.Fatalf("steps view has %d rows, want %d", len(views), len(after))
	}
	for _, v := range views {
		if v["group_index"].(float64) >= 1 && v["group_label"] == nil {
			t.Fatalf("a re-queued step lost its group label: %v", v)
		}
	}
}
