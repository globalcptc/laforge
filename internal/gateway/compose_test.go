package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/loader"
)

// composeRepo is a one-team environment with a compose container that also has
// an authored step of its own.
func composeRepo(t *testing.T) (string, *loader.Content) {
	t.Helper()
	root := t.TempDir()
	for rel, content := range map[string]string{
		".laforgeignore":                    "containers/flaky/\n",
		"env.yaml":                          "environment:\n  name: e\n  teams: 1\n  networks:\n    lan:\n      Flaky Frames:\n        - as: flaky01\n          last_octet: 10\n",
		"networks/lan.yaml":                 "network:\n  name: lan\n  cidr: 10.0.1.0/24\n",
		"containers/flaky.yaml":             "container:\n  name: Flaky Frames\n  compose: flaky/compose.yaml\n  size: small\n  steps:\n    - run: echo after\n",
		"containers/flaky/compose.yaml":     "services:\n  db:\n    image: postgres:16-alpine\n  app:\n    build: .\n    image: registry.example/team/app:1.0\n",
		"containers/flaky/nginx/nginx.conf": "events {}\n",
	} {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loader.Load(root)
	if err != nil || len(c.Errors) > 0 {
		t.Fatalf("fixture should load clean: %v %+v", err, c.Errors)
	}
	return root, c
}

// scripts is every shell script the commands run, in order.
func scripts(cmds []PlannedCommand) []string {
	var out []string
	for _, c := range cmds {
		if p, ok := c.Payload.(agentproto.ExecutePayload); ok {
			out = append(out, p.Args[len(p.Args)-1])
		}
	}
	return out
}

func TestComposeContainerStartsItsProjectBeforeItsSteps(t *testing.T) {
	root, c := composeRepo(t)
	cmds, _, err := ExpandSteps(root, c, "e", "flaky01", 1)
	if err != nil {
		t.Fatalf("ExpandSteps: %v", err)
	}
	if cmds[1].Command != agentproto.CmdWriteFile {
		t.Fatalf("second command = %s, want the archive write_file", cmds[1].Command)
	}
	sh := scripts(cmds)
	all := strings.Join(sh, "\n---\n")
	for _, want := range []string{
		"mkdir -p /opt/laforge/compose/flaky-frames",
		"tar -xzf - -C /opt/laforge/compose/flaky-frames",
		"get.docker.com",
		"docker compose -p flaky-frames --project-directory /opt/laforge/compose/flaky-frames -f /opt/laforge/compose/flaky-frames/'compose.yaml' pull",
		"up -d --no-build --remove-orphans --wait",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("commands missing %q:\n%s", want, all)
		}
	}
	// No credential source, no login -- what a preview gets.
	if strings.Contains(all, "docker login") {
		t.Errorf("expanded a docker login with no registry credentials supplied:\n%s", all)
	}
	// The project is its own group, ahead of the authored step.
	last := cmds[len(cmds)-1]
	if sh[len(sh)-1] != "echo after" || last.Group != 1 || last.GroupLabel != "Run command" {
		t.Errorf("authored step should run last as group 1, got %q group %d %q", sh[len(sh)-1], last.Group, last.GroupLabel)
	}
	for _, cmd := range cmds[:len(cmds)-1] {
		if cmd.Group != 0 || cmd.GroupLabel != ComposeGroupLabel {
			t.Errorf("compose command in group %d %q, want group 0 %q", cmd.Group, cmd.GroupLabel, ComposeGroupLabel)
		}
	}
}

func TestComposeContainerLogsInOnlyWhereACredentialExists(t *testing.T) {
	root, c := composeRepo(t)
	asked := map[string]bool{}
	auth := WithRegistryAuth(func(host string) (RegistryAuth, bool) {
		asked[host] = true
		if host == "registry.example" {
			return RegistryAuth{Username: "robot", Secret: "it's secret"}, true
		}
		return RegistryAuth{}, false
	})
	cmds, _, err := ExpandSteps(root, c, "e", "flaky01", 1, auth)
	if err != nil {
		t.Fatalf("ExpandSteps: %v", err)
	}
	all := strings.Join(scripts(cmds), "\n---\n")
	if !asked["docker.io"] || !asked["registry.example"] {
		t.Errorf("credential lookups = %v, want docker.io and registry.example", asked)
	}
	if want := `printf %s 'it'\''s secret' | docker login 'registry.example' -u 'robot' --password-stdin`; !strings.Contains(all, want) {
		t.Errorf("missing login %q in:\n%s", want, all)
	}
	if strings.Count(all, "docker login") != 1 {
		t.Errorf("want exactly one login (none for Docker Hub without a credential):\n%s", all)
	}
	if strings.Index(all, "docker login") > strings.Index(all, " pull") {
		t.Errorf("login must come before pull:\n%s", all)
	}
}
