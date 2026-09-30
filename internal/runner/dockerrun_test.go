package runner

import (
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

func TestRegistryHost(t *testing.T) {
	cases := map[string]string{
		"nginx":                              "",
		"nginx:alpine":                       "",
		"library/nginx":                      "",
		"myuser/app:1.2":                     "",
		"registry.internal/team/app:tag":     "registry.internal",
		"registry.internal:5000/app":         "registry.internal:5000",
		"localhost:5000/app":                 "localhost:5000",
		"ghcr.io/globalcptc/scoreboard:main": "ghcr.io",
	}
	for image, want := range cases {
		if got := registryHost(image); got != want {
			t.Errorf("registryHost(%q) = %q, want %q", image, got, want)
		}
	}
}

func TestDockerRunCommand(t *testing.T) {
	ct := &loader.Container{
		Image:   "nginx:alpine",
		Env:     map[string]string{"B": "2", "A": "1"},
		Command: []string{"-g", "daemon off;"},
	}
	p := dockerRunCommand(ct, nil)
	if p.Command != "/bin/sh" || len(p.Args) != 2 || p.Args[0] != "-c" {
		t.Fatalf("unexpected execute shape: %+v", p)
	}
	script := p.Args[1]
	for _, want := range []string{
		"docker pull 'nginx:alpine'",
		"docker rm -f laforge",
		"docker run -d --restart=always --network host --name laforge",
		"-e 'A=1'", "-e 'B=2'", // env sorted for determinism
		"'nginx:alpine' '-g' 'daemon off;'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("generated command missing %q\n got: %s", want, script)
		}
	}
	if strings.Contains(script, "docker login") {
		t.Errorf("no credential given, but a docker login was emitted: %s", script)
	}
}

func TestDockerRunCommandWithLogin(t *testing.T) {
	ct := &loader.Container{Image: "registry.internal/team/app:1.0"}
	cred := &db.RegistryCredential{RegistryHost: "registry.internal", Username: "bot", Secret: "s3cr3t"}
	script := dockerRunCommand(ct, cred).Args[1]
	if !strings.HasPrefix(script, "docker login 'registry.internal' -u 'bot' -p 's3cr3t' && docker pull") {
		t.Errorf("login not emitted correctly: %s", script)
	}
}
