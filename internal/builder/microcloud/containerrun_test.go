package microcloud

import (
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

func TestDockerRunScriptNoAgent(t *testing.T) {
	// No agent delivery: the image runs its own entrypoint, content command as
	// its args -- no --entrypoint override, no LAFORGE_SUPERVISE.
	spec := builder.ContainerSpec{
		Image:   "nginx:alpine",
		Env:     map[string]string{"B": "2", "A": "1"},
		Command: []string{"-g", "daemon off;"},
	}
	script := dockerRunScript(spec)
	for _, want := range []string{
		"docker pull 'nginx:alpine'",
		"docker rm -f laforge",
		"docker run -d --restart=always --network host --name laforge",
		"-e 'A=1'", "-e 'B=2'", // env sorted for determinism
		"'nginx:alpine' '-g' 'daemon off;'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("missing %q\n got: %s", want, script)
		}
	}
	for _, unwanted := range []string{"docker login", "--entrypoint", "LAFORGE_SUPERVISE"} {
		if strings.Contains(script, unwanted) {
			t.Errorf("unexpected %q with no agent/credential: %s", unwanted, script)
		}
	}
}

func TestDockerRunScriptWithAgentAndCommand(t *testing.T) {
	// With an agent and an explicit command: the agent is bind-mounted and made
	// the entrypoint, supervising the given command (quoted literally, no
	// in-host inspect). The command is NOT also appended as docker args.
	spec := builder.ContainerSpec{
		Image:       "nginx:alpine",
		Command:     []string{"nginx", "-g", "daemon off;"},
		AgentBinary: []byte("ELF..."),
	}
	script := dockerRunScript(spec)
	for _, want := range []string{
		"-v '/opt/laforge-agent':/laforge-agent:ro",
		"--entrypoint /laforge-agent",
		"-e 'LAFORGE_SUPERVISE=nginx -g daemon off;'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("missing %q\n got: %s", want, script)
		}
	}
	// The image is the final positional arg; the app command rides in
	// LAFORGE_SUPERVISE, not as trailing docker args.
	if !strings.HasSuffix(script, "'nginx:alpine'") {
		t.Errorf("image should be the final run arg: %s", script)
	}
	if strings.Contains(script, "docker inspect") {
		t.Errorf("explicit command should not trigger an in-host inspect: %s", script)
	}
}

func TestDockerRunScriptWithAgentNoCommand(t *testing.T) {
	// With an agent and no command override: resolve the image's own
	// entrypoint+cmd on the host via docker inspect, inside a double-quoted -e
	// so the substitution runs and stays a single value.
	spec := builder.ContainerSpec{
		Image:       "nginx:alpine",
		AgentBinary: []byte("ELF..."),
	}
	script := dockerRunScript(spec)
	for _, want := range []string{
		"--entrypoint /laforge-agent",
		`-e "LAFORGE_SUPERVISE=$(docker inspect --format`,
		".Config.Entrypoint",
		".Config.Cmd",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("missing %q\n got: %s", want, script)
		}
	}
}

func TestDockerRunScriptWithLogin(t *testing.T) {
	spec := builder.ContainerSpec{
		Image:          "registry.internal/team/app:1.0",
		RegistryHost:   "registry.internal",
		RegistryUser:   "bot",
		RegistrySecret: "s3cr3t",
	}
	script := dockerRunScript(spec)
	if !strings.HasPrefix(script, "docker login 'registry.internal' -u 'bot' -p 's3cr3t' && docker pull") {
		t.Errorf("login not emitted correctly: %s", script)
	}
}

func TestDockerRunScriptDockerHubLoginUsesDefaultEndpoint(t *testing.T) {
	// A docker.io credential logs in with an empty host (the default Docker Hub
	// endpoint), not the literal "docker.io".
	spec := builder.ContainerSpec{
		Image:        "myuser/app:1.0",
		RegistryHost: "docker.io",
		RegistryUser: "bot",
	}
	script := dockerRunScript(spec)
	if !strings.HasPrefix(script, "docker login '' -u 'bot'") {
		t.Errorf("docker.io should log in against the default endpoint: %s", script)
	}
}
