package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// TestRenderScriptFoldsScriptPeople proves a script's own `people:` reaches
// `{{ .people }}` when the script renders -- added to the host's assigned
// people (here the host has none), and deduped against them by username.
func TestRenderScriptFoldsScriptPeople(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "mk.sh"),
		[]byte(`{{ range .people }}{{ .username }} {{ end }}`), 0o644); err != nil {
		t.Fatal(err)
	}
	people := []loader.PeopleSource{{Name: "employees", People: []loader.Person{
		{Username: "jdoe"}, {Username: "awhite"},
	}}}

	// Host with NO people of its own; the script pulls in the whole file.
	c := &loader.Content{
		Environments: []loader.Environment{{Name: "e", Teams: 1,
			Networks: map[string]map[string][]loader.Copy{"n": {"h": {{As: "h01", LastOctet: 5}}}}}},
		Networks: []loader.Network{{Name: "n", CIDR: "10.0.0.0/24"}},
		Hosts:    []loader.Host{{Name: "h", OS: "ubuntu22", Size: "small"}},
		Scripts: []loader.Script{{
			Name: "mk", Language: "bash", Source: "mk.sh", SourceFile: "scripts/mk.yaml",
			People: []loader.PeopleRef{{File: "employees"}},
		}},
		People: people,
	}
	ctx, err := render.Resolve(c, "e", "h01", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	out, err := render.RenderScript(root, &c.Scripts[0], ctx, c)
	if err != nil {
		t.Fatalf("RenderScript: %v", err)
	}
	if strings.Fields(out) == nil || out != "jdoe awhite " {
		t.Errorf("script .people = %q, want jdoe awhite from the script's own people:", out)
	}

	// Now the host already has jdoe; the script adds employees -- jdoe must not
	// be listed twice.
	c.Hosts[0].People = []loader.PeopleRef{{File: "employees", Filter: []string{"jdoe"}}}
	ctx, _ = render.Resolve(c, "e", "h01", 1)
	out, err = render.RenderScript(root, &c.Scripts[0], ctx, c)
	if err != nil {
		t.Fatalf("RenderScript dedup: %v", err)
	}
	if out != "jdoe awhite " {
		t.Errorf("deduped .people = %q, want jdoe once then awhite", out)
	}
}

func TestAddress(t *testing.T) {
	cases := []struct {
		cidr string
		oct  int
		want string
	}{
		{"10.0.1.0/24", 12, "10.0.1.12"},
		{"10.0.254.0/24", 201, "10.0.254.201"},
		{"10.0.1.0/24", 0, "10.0.1.0"},
	}
	for _, c := range cases {
		got, err := render.Address(c.cidr, c.oct)
		if err != nil {
			t.Fatalf("Address(%q, %d): %v", c.cidr, c.oct, err)
		}
		if got != c.want {
			t.Errorf("Address(%q, %d) = %q, want %q", c.cidr, c.oct, got, c.want)
		}
	}
	if _, err := render.Address("not-a-cidr", 1); err == nil {
		t.Error("expected an error for an invalid CIDR")
	}
	if _, err := render.Address("10.0.1.0/24", 300); err == nil {
		t.Error("expected an error for an out-of-range last_octet")
	}
}

func loadExample(t *testing.T) *loader.Content {
	t.Helper()
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("example repo should load clean: %+v", c.Errors)
	}
	return c
}

func TestResolveComputesAddressAndPeers(t *testing.T) {
	c := loadExample(t)
	ctx, err := render.Resolve(c, "lm-test", "db01", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ctx.Address != "10.0.1.11" {
		t.Errorf("expected address 10.0.1.11 (prod=10.0.1.0/24 + last_octet 11), got %s", ctx.Address)
	}
	if ctx.ObjectKind != "host" || ctx.ObjectName != "database" {
		t.Errorf("expected object database/host, got %s/%s", ctx.ObjectName, ctx.ObjectKind)
	}
	// prod has db01, web01, scoreboard, dc01, ws01 -- db01 should see the
	// other four as peers, not itself.
	if len(ctx.Peers) != 4 {
		t.Errorf("expected 4 peers on prod, got %d: %+v", len(ctx.Peers), ctx.Peers)
	}
	for _, p := range ctx.Peers {
		if p.As == "db01" {
			t.Error("a copy should not appear in its own peer list")
		}
	}
}

func TestResolveRejectsBadTeam(t *testing.T) {
	c := loadExample(t)
	if _, err := render.Resolve(c, "lm-test", "db01", 0); err == nil {
		t.Error("expected an error for team 0 (environment has 5 teams, 1-indexed)")
	}
	if _, err := render.Resolve(c, "lm-test", "db01", 6); err == nil {
		t.Error("expected an error for team 6 (environment only has 5 teams)")
	}
}

func TestResolveRejectsUnknownHost(t *testing.T) {
	c := loadExample(t)
	if _, err := render.Resolve(c, "lm-test", "nope01", 1); err == nil {
		t.Error("expected an error for a placement that doesn't exist")
	}
}

func TestVarCascadeEnvironmentNetworkHost(t *testing.T) {
	c := loadExample(t)
	// prod network sets authoritative_dns_ip; lm-test environment sets
	// company; neither host nor network in this fixture override the
	// other's keys, so this proves the merge includes both levels
	// without asserting an override case the fixture doesn't have.
	ctx, err := render.Resolve(c, "lm-test", "db01", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	vars := ctx.EffectiveVars()
	if vars["company"] != "Allports" {
		t.Errorf("expected environment-level var 'company' to be visible, got %q", vars["company"])
	}
	if vars["authoritative_dns_ip"] != "10.0.1.6" {
		t.Errorf("expected network-level var to be visible, got %q", vars["authoritative_dns_ip"])
	}

	var companyEntry *render.VarEntry
	for i := range ctx.Vars {
		if ctx.Vars[i].Key == "company" {
			companyEntry = &ctx.Vars[i]
		}
	}
	if companyEntry == nil || companyEntry.Source != "environment" {
		t.Errorf("expected 'company' to be attributed to the environment level, got %+v", companyEntry)
	}
}

func TestVarOverrideIsRecordedWithShadowedValue(t *testing.T) {
	c := &loader.Content{
		Environments: []loader.Environment{{Name: "e", Teams: 1, Vars: map[string]string{"x": "from-env"},
			Networks: map[string]map[string][]loader.Copy{"n": {"h": {{As: "h01", LastOctet: 5}}}}}},
		Networks: []loader.Network{{Name: "n", CIDR: "10.0.0.0/24"}},
		Hosts:    []loader.Host{{Name: "h", OS: "ubuntu22", Size: "small", Vars: map[string]string{"x": "from-host"}}},
	}
	ctx, err := render.Resolve(c, "e", "h01", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ctx.EffectiveVars()["x"] != "from-host" {
		t.Fatalf("expected host to win over environment, got %q", ctx.EffectiveVars()["x"])
	}
	var entry *render.VarEntry
	for i := range ctx.Vars {
		if ctx.Vars[i].Key == "x" {
			entry = &ctx.Vars[i]
		}
	}
	if entry == nil || entry.Source != "host/container" {
		t.Fatalf("expected 'x' attributed to host/container, got %+v", entry)
	}
	if len(entry.Shadowed) != 1 || entry.Shadowed[0].Value != "from-env" {
		t.Fatalf("expected the environment's shadowed value to be recorded, got %+v", entry.Shadowed)
	}
}

func TestRenderObjectPeopleResolvesAndFilters(t *testing.T) {
	c := &loader.Content{
		Environments: []loader.Environment{{Name: "e", Teams: 1,
			Networks: map[string]map[string][]loader.Copy{"n": {
				"all":  {{As: "all01", LastOctet: 5}},
				"some": {{As: "some01", LastOctet: 6}},
			}}}},
		Networks: []loader.Network{{Name: "n", CIDR: "10.0.0.0/24"}},
		Hosts: []loader.Host{
			{Name: "all", OS: "ubuntu22", Size: "small", People: []loader.PeopleRef{{File: "employees"}}},
			{Name: "some", OS: "ubuntu22", Size: "small", People: []loader.PeopleRef{{File: "employees", Filter: []string{"jdoe"}}}},
		},
		People: []loader.PeopleSource{{Name: "employees", People: []loader.Person{
			{Username: "jdoe", Attributes: map[string]string{"email": "jdoe@x"}},
			{Username: "awhite", Attributes: map[string]string{"email": "awhite@x"}},
		}}},
	}
	tmpl := `{{ range .people }}{{ .username }}:{{ .email }} {{ end }}`

	all, err := render.Resolve(c, "e", "all01", 1)
	if err != nil {
		t.Fatalf("Resolve all: %v", err)
	}
	out, err := render.RenderString("t", tmpl, all, c)
	if err != nil {
		t.Fatalf("render all: %v", err)
	}
	if out != "jdoe:jdoe@x awhite:awhite@x " {
		t.Errorf("unfiltered .people = %q, want every row of employees.csv", out)
	}

	some, err := render.Resolve(c, "e", "some01", 1)
	if err != nil {
		t.Fatalf("Resolve some: %v", err)
	}
	out, err = render.RenderString("t", tmpl, some, c)
	if err != nil {
		t.Fatalf("render some: %v", err)
	}
	if out != "jdoe:jdoe@x " {
		t.Errorf("filtered .people = %q, want only jdoe", out)
	}
}

func TestRenderStringDotContextForm(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "web01", 3)
	out, err := render.RenderString("t", "host={{ .host.hostname }} team={{ .build.team }}", ctx, c)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if out != "host=web01 team=3" {
		t.Errorf("got %q", out)
	}
}

func TestRenderStringBareFunctionChainForm(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "web01", 1)
	// Confirms the doc's own (informally written but valid) syntax works:
	// a niladic function call chained with a field, no leading dot.
	out, err := render.RenderString("t", "company={{ vars.company }}", ctx, c)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if out != "company=Allports" {
		t.Errorf("got %q", out)
	}
}

func TestRenderStringPeopleFunction(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "db01", 1)
	out, err := render.RenderString("t", `{{ range people "employees" }}{{ .username }} {{ end }}`, ctx, c)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	for _, want := range []string{"arivera3", "aroberts", "jdoe"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got %q", want, out)
		}
	}
}

func TestRenderStringUnknownKeyFailsStrict(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "web01", 1)
	_, err := render.RenderString("host.yaml", "{{ .hostnme }}", ctx, c)
	if err == nil {
		t.Fatal("expected a typo'd field reference to fail in strict mode: \"{{ .hostnme }} is an error\"")
	}
	if !strings.Contains(err.Error(), "host.yaml") {
		t.Errorf("expected the error to name the template (file), got %q", err.Error())
	}
}

func TestRenderStringUnknownPeopleSourceFails(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "db01", 1)
	_, err := render.RenderString("t", `{{ range people "nosuchsource" }}{{ end }}`, ctx, c)
	if err == nil {
		t.Fatal("expected an error for an unknown people source")
	}
}

func TestRenderScriptRealFile(t *testing.T) {
	c := loadExample(t)
	ctx, _ := render.Resolve(c, "lm-test", "web01", 2)
	script := findScriptForTest(c, "base")
	out, err := render.RenderScript("../../examples/lm-test", script, ctx, c)
	if err != nil {
		t.Fatalf("RenderScript: %v", err)
	}
	if !strings.Contains(out, "web01") || !strings.Contains(out, "team 2") {
		t.Errorf("rendered script missing expected substitutions: %q", out)
	}
}

func findScriptForTest(c *loader.Content, name string) *loader.Script {
	for i := range c.Scripts {
		if c.Scripts[i].Name == name {
			return &c.Scripts[i]
		}
	}
	return nil
}

func TestCheckAllOnRealExampleRepoIsClean(t *testing.T) {
	c := loadExample(t)
	errs := render.CheckAll("../../examples/lm-test", c)
	if len(errs) != 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.String())
		}
		t.Fatalf("expected zero render errors across every host/team/script, got %d:\n%s", len(errs), strings.Join(msgs, "\n"))
	}
}

func TestCheckAllCatchesABrokenTemplate(t *testing.T) {
	c := &loader.Content{
		Environments: []loader.Environment{{Name: "e", Teams: 1,
			Networks: map[string]map[string][]loader.Copy{"n": {"h": {{As: "h01", LastOctet: 5}}}}}},
		Networks: []loader.Network{{Name: "n", CIDR: "10.0.0.0/24"}},
		Hosts: []loader.Host{{Name: "h", OS: "ubuntu22", Size: "small",
			Steps: []loader.Step{{"run": "{{ .nope.nope }}"}}}},
	}
	errs := render.CheckAll(".", c)
	if len(errs) == 0 {
		t.Fatal("expected CheckAll to catch the broken template reference")
	}
}
