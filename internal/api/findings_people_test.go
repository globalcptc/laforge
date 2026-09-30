package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ingest"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// realIngestedBuild runs the actual ingest pipeline (internal/ingest.
// ValidateAndStore -- the same code path a real push webhook drives)
// against examples/lm-test, which has real findings on a network, two
// hosts, and a script (webserver's own steps run that script, so its
// finding should resolve as inherited), and a real people CSV. Then
// Reconciles a real build from the resulting content_revision. Used by
// both the findings and people tests below since both need real,
// non-synthetic content rows to resolve against.
func realIngestedBuild(t *testing.T, env *apiTestEnv) (revisionID string, build db.Build) {
	t.Helper()
	ctx := context.Background()
	result, err := ingest.ValidateAndStore(ctx, env.server.Pool, env.repo.ID, "lm-test-real-ingest-sha", "refs/heads/main", "../../examples/lm-test")
	if err != nil {
		t.Fatalf("ValidateAndStore: %v", err)
	}
	if !result.Valid {
		t.Fatalf("examples/lm-test failed validation: %+v", result.Issues)
	}
	b, err := env.q.CreateBuild(ctx, db.CreateBuildParams{
		ContentRevisionID: result.Revision.ID, EnvironmentName: "lm-test",
	})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	if err := orchestrator.Reconcile(ctx, env.server.Pool, "../../examples/lm-test", b.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return result.Revision.ID.String(), b
}

func TestFindingsResolveDirectAndInheritedFromScript(t *testing.T) {
	env := setupAPITest(t)
	_, build := realIngestedBuild(t, env)
	client := signedInClient(t, env, "read-only-viewer")

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/findings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var findings []findingInstance
	json.NewDecoder(resp.Body).Decode(&findings)
	if len(findings) == 0 {
		t.Fatal("expected real findings from examples/lm-test, got none")
	}

	var sawDirectOnDatabase, sawInheritedFromScript, sawNetworkFinding bool
	teamsSeen := map[int32]bool{}
	for _, f := range findings {
		if f.ObjectKind == "environment" && f.TeamNumber != nil {
			t.Fatal("an environment-level finding must NOT carry a team number -- it's not instantiated per team")
		}
		if f.ObjectKind == "host" && f.ObjectName == "database" && f.Source == "direct" {
			sawDirectOnDatabase = true
		}
		if f.ObjectKind == "host" && f.ObjectName == "webserver" && f.Source == "script:vuln-sqli" {
			sawInheritedFromScript = true
			if f.TeamNumber == nil {
				t.Fatal("an inherited host finding must carry a team number")
			}
			teamsSeen[*f.TeamNumber] = true
		}
		if f.ObjectKind == "network" {
			sawNetworkFinding = true
		}
	}
	if !sawDirectOnDatabase {
		t.Error("expected a direct finding on host database (database.yaml has one)")
	}
	if !sawInheritedFromScript {
		t.Error("expected webserver to inherit vuln-sqli's finding via its own steps")
	}
	if !sawNetworkFinding {
		t.Error("expected a finding on network prod")
	}
	// lm-test is a 5-team environment (examples/lm-test/lm-test.yaml) --
	// confirms the inherited finding genuinely appears once per team,
	// not once for the whole build.
	if len(teamsSeen) != 5 {
		t.Fatalf("inherited script finding seen for %d distinct teams, want 5 (lm-test's team count)", len(teamsSeen))
	}
}

// TestBuildDetailIncludesRealAccessSchedule proves the environment's own
// authored access windows (examples/lm-test/lm-test.yaml has two real
// ones) flow through to GET /builds/{id} -- what the top bar's countdown
// and the Access screen need alongside each team's live access_state.
func TestBuildDetailIncludesRealAccessSchedule(t *testing.T) {
	env := setupAPITest(t)
	_, build := realIngestedBuild(t, env)
	client := signedInClient(t, env, "read-only-viewer")

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var detail buildDetail
	json.NewDecoder(resp.Body).Decode(&detail)
	if len(detail.Access) != 2 {
		t.Fatalf("Access = %+v, want the 2 real windows from lm-test.yaml", detail.Access)
	}
	if detail.Access[0].Open != "2026-10-01T09:00:00Z" || detail.Access[0].Close != "2026-10-01T18:00:00Z" {
		t.Fatalf("Access[0] = %+v, want the real first window", detail.Access[0])
	}
}

func TestFindingsCSVExport(t *testing.T) {
	env := setupAPITest(t)
	_, build := realIngestedBuild(t, env)
	client := signedInClient(t, env, "read-only-viewer")

	resp, err := client.Get(env.httpURL + "/builds/" + build.ID.String() + "/findings?format=csv")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("Content-Type = %q, want text/csv", ct)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV response: %v", err)
	}
	if len(rows) < 2 { // header + at least one real finding row
		t.Fatalf("CSV has %d row(s), want a header plus real findings", len(rows))
	}
	wantHeader := []string{"team", "object_kind", "object_name", "as_name", "source", "severity", "difficulty", "description"}
	for i, col := range wantHeader {
		if rows[0][i] != col {
			t.Fatalf("CSV header[%d] = %q, want %q", i, rows[0][i], col)
		}
	}
}

func TestListPeopleRealCSVAndSearch(t *testing.T) {
	env := setupAPITest(t)
	realIngestedBuild(t, env) // ingests examples/lm-test, including people/employees.csv
	client := signedInClient(t, env, "read-only-viewer")

	resp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/people")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var people []personView
	json.NewDecoder(resp.Body).Decode(&people)
	if len(people) == 0 {
		t.Fatal("expected real people from examples/lm-test/people/employees.csv, got none")
	}
	found := false
	for _, p := range people {
		if p.Username == "arivera3" {
			found = true
			if p.Attributes["department"] != "Administration" {
				t.Fatalf("arivera3.department = %q, want Administration", p.Attributes["department"])
			}
			if p.Attributes["password"] == "" {
				t.Fatal("password attribute missing -- \"shows passwords, because during an event that is the point\"")
			}
		}
	}
	if !found {
		t.Fatal("did not find the real employee arivera3 from the CSV")
	}

	// Search by an attribute value (department), not just username --
	// "searchable by name, username, department, title."
	searchResp, err := client.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/people?q=administration")
	if err != nil {
		t.Fatal(err)
	}
	defer searchResp.Body.Close()
	var matched []personView
	json.NewDecoder(searchResp.Body).Decode(&matched)
	if len(matched) == 0 {
		t.Fatal("searching by department \"administration\" matched nobody")
	}
	for _, p := range matched {
		if p.Attributes["department"] != "Administration" {
			t.Fatalf("search for \"administration\" matched %s with department %q", p.Username, p.Attributes["department"])
		}
	}
}

func TestPeopleEndpointsRequireAtLeastRead(t *testing.T) {
	env := setupAPITest(t)
	resp, err := http.Get(env.httpURL + "/repos/" + env.repo.ID.String() + "/people")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want 401", resp.StatusCode)
	}
}
