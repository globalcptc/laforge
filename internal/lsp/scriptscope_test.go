package lsp

import "testing"

// TestScriptScopeUnionsVarsAndFlagsPartialAvailability is the plan's own
// worked example, against real content: "Completion offers the union of
// every context the script runs in, marking anything that is not
// available everywhere: `db_name` shows as available on 2 of the 5
// hosts that run this script". examples/lm-test's
// `base` script runs on both `database` and `webserver` hosts (5 teams
// each, so 10 real (environment, team, as) triples); `db_admin_pw` is
// set only in database.yaml's own vars, so it must show up as available
// on strictly fewer triples than a var set at environment or network
// level, which every one of those 10 triples sees.
func TestScriptScopeUnionsVarsAndFlagsPartialAvailability(t *testing.T) {
	w := newTestWorkspace(t)
	content, _ := w.Snapshot()
	if content == nil {
		t.Fatal("nil content")
	}

	vars, people := scriptScope(content, "base")

	byKey := make(map[string]varUsage, len(vars))
	for _, v := range vars {
		byKey[v.Key] = v
	}

	dbPW, ok := byKey["db_admin_pw"]
	if !ok {
		t.Fatalf("db_admin_pw not found in scope; got %+v", vars)
	}
	company, ok := byKey["company"]
	if !ok {
		t.Fatalf("company (environment-level var) not found in scope; got %+v", vars)
	}
	if dbPW.Total != company.Total {
		t.Fatalf("dbPW.Total=%d, company.Total=%d, want equal (same set of triples run \"base\")", dbPW.Total, company.Total)
	}
	if dbPW.SeenIn >= dbPW.Total {
		t.Fatalf("db_admin_pw.SeenIn=%d of Total=%d, want strictly fewer -- it's set only on the database host, not every host running base", dbPW.SeenIn, dbPW.Total)
	}
	if company.SeenIn != company.Total {
		t.Fatalf("company.SeenIn=%d, Total=%d, want equal -- an environment-level var is visible to every context", company.SeenIn, company.Total)
	}
	if dbPW.Source != "host/container" {
		t.Fatalf("db_admin_pw.Source = %q, want \"host/container\"", dbPW.Source)
	}
	if company.Source != "environment" {
		t.Fatalf("company.Source = %q, want \"environment\"", company.Source)
	}

	if len(people) == 0 {
		t.Fatal("scriptScope returned no people sources, want examples/lm-test's real employees.csv")
	}
	var sawEmployees bool
	for _, p := range people {
		if p.Name == "employees" {
			sawEmployees = true
		}
	}
	if !sawEmployees {
		t.Fatalf("people sources = %+v, want to include \"employees\"", people)
	}
}

// TestScriptForSourceFileFindsRealScript proves the file->script lookup
// itself against a real file, the thing every other script-completion
// feature depends on.
func TestScriptForSourceFileFindsRealScript(t *testing.T) {
	w := newTestWorkspace(t)
	content, _ := w.Snapshot()
	s := scriptForSourceFile(content, "scripts/base.sh")
	if s == nil {
		t.Fatal("scriptForSourceFile(scripts/base.sh) = nil, want the real \"base\" script")
	}
	if s.Name != "base" {
		t.Fatalf("found script name = %q, want \"base\"", s.Name)
	}
}
