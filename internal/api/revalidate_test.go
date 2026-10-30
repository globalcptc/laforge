package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestRevalidateQueuesValidatorsAndReports: Revalidate queues a host's
// validators (and only those), skips a host whose agent isn't in with the
// reason, and the report shows each check -- pending, then its result.
func TestRevalidateQueuesValidatorsAndReports(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	ctx := context.Background()
	pool := env.server.Pool
	q := db.New(pool)

	client := signedInClient(t, env, "good-token")
	adminClient := signedInClient(t, env, "the-owner")
	grantAdmin(t, env, "the-owner")
	grantManage(t, env, adminClient, "exchanged-good-token")

	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "finished"}); err != nil {
		t.Fatal(err)
	}
	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	var hosts []db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			hosts = append(hosts, o)
		}
	}
	if len(hosts) != 2 {
		t.Fatalf("fixture has %d hosts, want 2", len(hosts))
	}
	live, noAgent := hosts[0], hosts[1]
	for _, o := range hosts {
		if _, err := pool.Exec(ctx, "UPDATE deployed_object SET status = 'finished', deployed_at = now() - interval '1 hour', steps_materialized_at = now() WHERE id = $1", o.ID); err != nil {
			t.Fatal(err)
		}
		for i, cmd := range []string{"execute", "validate"} {
			if _, err := pool.Exec(ctx, `INSERT INTO agent_task (deployed_object_id, step_index, command, payload, status) VALUES ($1, $2, $3, $4, 'done')`,
				o.ID, i, cmd, `{"checks":[{"kind":"port_open","args":{"port":22}}]}`); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO agent_session (deployed_object_id, cert_fingerprint) VALUES ($1, 'fp')", live.ID); err != nil {
		t.Fatal(err)
	}

	body := `{"target":{"ids":["` + live.ID.String() + `","` + noAgent.ID.String() + `"]}}`
	resp, err := client.Post(env.httpURL+"/builds/"+build.ID.String()+"/revalidate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("revalidate status = %d, want 200", resp.StatusCode)
	}
	var res revalidateResponse
	json.NewDecoder(resp.Body).Decode(&res)
	if res.ReportID == "" || len(res.Started) != 1 || res.Started[0].ObjectID != live.ID.String() {
		t.Fatalf("started = %+v (report %q), want just the live host", res.Started, res.ReportID)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].ObjectID != noAgent.ID.String() || !strings.Contains(res.Skipped[0].Reason, "agent") {
		t.Fatalf("skipped = %+v, want the agentless host with an agent reason", res.Skipped)
	}

	getReport := func() validationReportView {
		r, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/validation-reports/" + res.ReportID)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Fatalf("get report status = %d", r.StatusCode)
		}
		var rep validationReportView
		json.NewDecoder(r.Body).Decode(&rep)
		return rep
	}
	rep := getReport()
	if rep.Running != 1 || len(rep.Runs) != 1 || len(rep.Runs[0].Steps) != 1 {
		t.Fatalf("report = %+v, want one running run with one validator step", rep)
	}
	step := rep.Runs[0].Steps[0]
	if step.Status != "pending" || len(step.Checks) != 1 || step.Checks[0].Kind != "port_open" || step.Checks[0].Passed != nil {
		t.Fatalf("pending step = %+v, want its port_open check listed, not run yet", step)
	}

	// The Steps tab doesn't show the copy.
	sr, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/objects/" + live.ID.String() + "/steps")
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Body.Close()
	var stepsBody json.RawMessage
	json.NewDecoder(sr.Body).Decode(&stepsBody)
	if n := strings.Count(string(stepsBody), `"command":"validate"`); n != 1 {
		t.Fatalf("Steps tab shows %d validate rows, want 1 (the build's own, not the revalidation copy)", n)
	}

	// The agent reports a failing check.
	var taskID string
	pool.QueryRow(ctx, "SELECT id FROM agent_task WHERE deployed_object_id = $1 AND validation_run_id IS NOT NULL", live.ID).Scan(&taskID)
	if _, err := pool.Exec(ctx, "UPDATE agent_task SET status = 'failed' WHERE id = $1", taskID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO validator_result (agent_task_id, kind, args, passed, message) VALUES ($1, 'port_open', '{\"port\":22}', false, 'port 22 is not listening')", taskID); err != nil {
		t.Fatal(err)
	}
	rep = getReport()
	got := rep.Runs[0].Steps[0]
	if len(got.Checks) != 1 || got.Checks[0].Passed == nil || *got.Checks[0].Passed || got.Checks[0].Message != "port 22 is not listening" {
		t.Fatalf("reported step = %+v, want the failed check with its message", got)
	}
}
