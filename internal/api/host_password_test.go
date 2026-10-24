package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/runner"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestCurrentCredentialsPicksWhatIsOnTheBox seeds real agent_task rows and
// checks each host resolves to the admin password actually on the box: the
// last APPLIED admin set_password (by finish time), else the next planned one,
// else (no admin set_password at all) the environment root password. Failed
// steps and non-admin accounts never count.
func TestCurrentCredentialsPicksWhatIsOnTheBox(t *testing.T) {
	env := setupAPITest(t)
	build := realBuildFixture(t, env)
	ctx := context.Background()

	r := &runner.Runner{
		Pool: env.server.Pool, Builder: fake.New(env.server.Pool), RepoRoot: "../../examples/m7-two-team", ID: "test-runner",
		LeaseDuration: 30 * time.Second, HeartbeatInterval: 5 * time.Second, BuildID: &build.ID,
	}
	for i := 0; i < 20; i++ {
		did, err := r.LeaseAndExecuteOne(ctx)
		if err != nil {
			t.Fatalf("LeaseAndExecuteOne: %v", err)
		}
		if !did {
			break
		}
	}
	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	var hosts []db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			hosts = append(hosts, o)
		}
	}
	if len(hosts) < 2 {
		t.Fatalf("fixture produced %d hosts, need 2", len(hosts))
	}
	applied, planned := hosts[0].ID, hosts[1].ID
	// A host with no admin set_password rows at all.
	none := pgtype.UUID{Bytes: [16]byte{0xde, 0xad}, Valid: true}

	pool := env.server.Pool
	for _, id := range []pgtype.UUID{applied, planned} {
		if _, err := pool.Exec(ctx, `DELETE FROM agent_task WHERE deployed_object_id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	t0 := time.Now().Add(-time.Hour)
	add := func(id pgtype.UUID, idx int, user, pw, status string, adHoc bool, finished *time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO agent_task (deployed_object_id, step_index, command, payload, status, ad_hoc, finished_at)
			VALUES ($1, $2, 'set_password', jsonb_build_object('username', $3::text, 'password', $4::text), $5, $6, $7)`,
			id, idx, user, pw, status, adHoc, finished); err != nil {
			t.Fatalf("seeding agent_task: %v", err)
		}
	}
	at := func(d time.Duration) *time.Time { v := t0.Add(d); return &v }

	// applied: the auto step, then a later authored change, then an operator's
	// ad-hoc change (latest finish wins); a failed attempt, a non-admin account
	// and a still-pending step must all be ignored.
	add(applied, 0, "Administrator", "env-root", "done", false, at(0))
	add(applied, 5, "Administrator", "authored", "done", false, at(10*time.Minute))
	add(applied, 6, "pentester", "not-admin", "done", false, at(20*time.Minute))
	add(applied, 7, "Administrator", "broken", "failed", false, at(25*time.Minute))
	add(applied, 9, "Administrator", "not-yet", "pending", false, nil)
	add(applied, 10, "administrator", "operator", "done", true, at(30*time.Minute))
	// planned: nothing applied yet -- the earliest pending authored step.
	add(planned, 0, "root", "first-planned", "blocked", false, nil)
	add(planned, 4, "root", "second-planned", "pending", false, nil)

	got, err := currentCredentials(ctx, env.q, build.ID)
	if err != nil {
		t.Fatalf("currentCredentials: %v", err)
	}
	if hp := passwordFor(got, applied, "env-root"); hp != (hostPassword{Password: "operator", Applied: true, Source: "step"}) {
		t.Errorf("applied host = %+v, want the operator's later ad-hoc change, applied", hp)
	}
	if hp := passwordFor(got, planned, "env-root"); hp != (hostPassword{Password: "first-planned", Applied: false, Source: "step"}) {
		t.Errorf("planned host = %+v, want the earliest planned step, not yet applied", hp)
	}
	if hp := passwordFor(got, none, "env-root"); hp != (hostPassword{Password: "env-root", Applied: false, Source: "environment"}) {
		t.Errorf("host with no admin set_password = %+v, want the environment root password", hp)
	}
	// The full credential list keeps every account (admin first, the
	// case-insensitive "administrator" rows folded into one) -- what the host
	// Info tab shows.
	want := []hostCredential{
		{Username: "administrator", Password: "operator", Applied: true, Source: "step", Admin: true},
		{Username: "pentester", Password: "not-admin", Applied: true, Source: "step"},
	}
	if c := got[applied]; len(c) != len(want) || c[0] != want[0] || c[1] != want[1] {
		t.Errorf("applied host credentials = %+v, want %+v", c, want)
	}

	// The build-wide endpoint (`laforge passwords`) lists the same accounts,
	// readable at read access.
	client := signedInClient(t, env, "read-only-viewer")
	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/credentials")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /credentials status = %d, want 200", resp.StatusCode)
	}
	var rows []struct {
		Host     string `json:"host"`
		Username string `json:"username"`
		Password string `json:"password"`
		Applied  bool   `json:"applied"`
		Admin    bool   `json:"admin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, r := range rows {
		found[r.Username+"="+r.Password] = true
	}
	for _, k := range []string{"administrator=operator", "pentester=not-admin", "root=first-planned"} {
		if !found[k] {
			t.Errorf("GET /credentials missing %s; got %+v", k, rows)
		}
	}
}
