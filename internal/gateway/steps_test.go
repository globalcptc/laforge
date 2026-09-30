package gateway

import (
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/loader"
)

// TestExpandStepsOnRealWebserverHost exercises every interesting
// expansion path against real content in one shot:
// examples/lm-test/hosts/webserver.yaml's steps are
// [script, download, extract, script] -- two script expansions
// (write_file + execute each) and a download and an extract, both real
// commands the agent runs (no warning notes). Its `schedule:` entry
// (a sibling field, not a step) is untouched by ExpandSteps entirely --
// schedule is no longer a step kind at all.
func TestExpandStepsOnRealWebserverHost(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("loader.Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("content has schema errors: %+v", c.Errors)
	}

	cmds, notes, err := ExpandSteps("../../examples/lm-test", c, "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("ExpandSteps: %v", err)
	}

	wantCommands := []string{
		agentproto.CmdWriteFile, agentproto.CmdExecute, // script: base
		agentproto.CmdDownload,
		agentproto.CmdExtract,
		agentproto.CmdWriteFile, agentproto.CmdExecute, // script: vuln-sqli
	}
	if len(cmds) != len(wantCommands) {
		t.Fatalf("got %d commands, want %d: %+v", len(cmds), len(wantCommands), cmds)
	}
	for i, want := range wantCommands {
		if cmds[i].Command != want {
			t.Errorf("cmds[%d].Command = %q, want %q", i, cmds[i].Command, want)
		}
	}

	// ignore_errors rides from a script onto its execute command: vuln-sqli sets
	// it (cmds[5]), base does not (cmds[1]).
	if cmds[1].IgnoreErrors {
		t.Errorf("base's execute should not ignore errors")
	}
	if !cmds[5].IgnoreErrors {
		t.Errorf("vuln-sqli sets ignore_errors: true, but its execute command didn't carry it")
	}

	// The first script's rendered content actually made it into the
	// write_file payload -- not empty, not a template artifact.
	wf, ok := cmds[0].Payload.(agentproto.WriteFilePayload)
	if !ok {
		t.Fatalf("cmds[0].Payload is %T, want agentproto.WriteFilePayload", cmds[0].Payload)
	}
	if wf.Content == "" {
		t.Fatal("rendered script content is empty")
	}
	if strings.Contains(wf.Content, "{{") {
		t.Fatalf("rendered script content still has an unrendered template marker:\n%s", wf.Content)
	}

	exec, ok := cmds[1].Payload.(agentproto.ExecutePayload)
	if !ok {
		t.Fatalf("cmds[1].Payload is %T, want agentproto.ExecutePayload", cmds[1].Payload)
	}
	if exec.Command != "/bin/bash" {
		t.Fatalf("exec.Command = %q, want /bin/bash (base.yaml's language is bash)", exec.Command)
	}
	if len(exec.Args) != 1 || exec.Args[0] != wf.Path {
		t.Fatalf("exec.Args = %v, want exactly [%q] (the path the script was just written to)", exec.Args, wf.Path)
	}

	dl, ok := cmds[2].Payload.(agentproto.DownloadPayload)
	if !ok {
		t.Fatalf("cmds[2].Payload is %T, want agentproto.DownloadPayload", cmds[2].Payload)
	}
	if dl.From != "https://files.cp.tc/site.tar.gz" || dl.To != "/var/www/site.tar.gz" {
		t.Fatalf("download payload = %+v, want matching webserver.yaml's from/to", dl)
	}

	// download and extract are both real now -- neither emits a warning note.
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none (download and extract are implemented)", notes)
	}
}

// TestExpandStepsRendersValidateBlock proves a step's validate: block
// becomes its own trailing CmdValidate command, immediately after the
// step it checks -- no host in examples/lm-test carries one today, so
// this grafts one onto a real, already-loaded host's first step in
// memory rather than depending on a from-scratch fixture, still
// exercising the real ExpandSteps/render.Resolve path underneath it.
func TestExpandStepsRendersValidateBlock(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("loader.Load: %v", err)
	}
	// Graft a validate block onto database's first step in memory --
	// real content, minimally modified, rather than a from-scratch fixture.
	for i := range c.Hosts {
		if c.Hosts[i].Name == "database" && len(c.Hosts[i].Steps) > 0 {
			c.Hosts[i].Steps[0]["validate"] = []interface{}{
				map[string]interface{}{"service_running": map[string]interface{}{"value": "mysql"}},
				map[string]interface{}{"user_exists": "dbadmin"},
			}
		}
	}

	cmds, _, err := ExpandSteps("../../examples/lm-test", c, "lm-test", "db01", 1)
	if err != nil {
		t.Fatalf("ExpandSteps: %v", err)
	}
	var validateCmd *agentproto.ValidatePayload
	for _, cmd := range cmds {
		if cmd.Command == agentproto.CmdValidate {
			p := cmd.Payload.(agentproto.ValidatePayload)
			validateCmd = &p
			break
		}
	}
	if validateCmd == nil {
		t.Fatal("no validate command was produced")
	}
	if len(validateCmd.Checks) != 2 {
		t.Fatalf("got %d checks, want 2: %+v", len(validateCmd.Checks), validateCmd.Checks)
	}
	kinds := map[string]bool{}
	for _, chk := range validateCmd.Checks {
		kinds[chk.Kind] = true
	}
	if !kinds["service_running"] || !kinds["user_exists"] {
		t.Fatalf("checks = %+v, want service_running and user_exists", validateCmd.Checks)
	}
	// The bare-value form (`user_exists: dbadmin`, not a map) must be
	// normalized to {"value": "dbadmin"} so the agent always sees a map.
	for _, chk := range validateCmd.Checks {
		if chk.Kind == "user_exists" && chk.Args["value"] != "dbadmin" {
			t.Fatalf("user_exists args = %+v, want {value: dbadmin}", chk.Args)
		}
	}
}

// TestScriptArgsAndValidateAreWired proves a script's own `args:` reach the
// execute command and its own `validate:` block becomes a trailing validate,
// grafted onto a real script the webserver already runs.
func TestScriptArgsAndValidateAreWired(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("loader.Load: %v", err)
	}
	for i := range c.Scripts {
		if c.Scripts[i].Name == "base" {
			c.Scripts[i].Args = []string{"--seed", "7"}
			c.Scripts[i].Validate = []map[string]interface{}{
				{"service_running": map[string]interface{}{"value": "nginx"}},
			}
		}
	}

	cmds, _, err := ExpandSteps("../../examples/lm-test", c, "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("ExpandSteps: %v", err)
	}

	// The execute for `base` must end with the script path then --seed 7.
	var found bool
	for _, cmd := range cmds {
		if cmd.Command != agentproto.CmdExecute {
			continue
		}
		p := cmd.Payload.(agentproto.ExecutePayload)
		n := len(p.Args)
		if n >= 2 && p.Args[n-2] == "--seed" && p.Args[n-1] == "7" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no execute command carried the script's args --seed 7: %+v", cmds)
	}

	// A validate command from the script's own validate: block (nginx) exists.
	var sawNginx bool
	for _, cmd := range cmds {
		if cmd.Command != agentproto.CmdValidate {
			continue
		}
		for _, chk := range cmd.Payload.(agentproto.ValidatePayload).Checks {
			if chk.Kind == "service_running" && chk.Args["value"] == "nginx" {
				sawNginx = true
			}
		}
	}
	if !sawNginx {
		t.Fatalf("script's own validate (service_running nginx) was not emitted: %+v", cmds)
	}
}
