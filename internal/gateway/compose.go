package gateway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/compose"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// composeRoot is where a project lives on its machine: /opt/laforge/compose/<name>.
const composeRoot = "/opt/laforge/compose"

// composeWaitSeconds is how long `up --wait` gives the stack's health checks
// before the step fails.
const composeWaitSeconds = 600

// installDockerScript installs Docker Engine with the compose plugin when the
// machine doesn't have both already. A machine booted from the builder's docker
// base image has them and skips straight past it; this is for an operator's own
// compose-host image that doesn't. Docker's own convenience script covers
// the Linux distributions LaForge images use.
const installDockerScript = `set -e
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo "docker and the compose plugin are already installed"
  exit 0
fi
# Docker without the compose plugin (a distribution's docker.io package): add
# the distribution's plugin rather than replacing its Docker.
if command -v docker >/dev/null 2>&1 && command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update && apt-get install -y docker-compose-v2 && docker compose version && exit 0
fi
if command -v curl >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com -o /tmp/laforge-get-docker.sh
else
  wget -qO /tmp/laforge-get-docker.sh https://get.docker.com
fi
sh /tmp/laforge-get-docker.sh
rm -f /tmp/laforge-get-docker.sh
systemctl enable --now docker 2>/dev/null || service docker start
docker compose version`

// expandCompose is the commands that bring a container's Compose project up
// on its machine: unpack the project's directory, make sure Docker is there,
// log in to any registry there is a stored credential for, pull, and start.
// Each is its own command so a failure names the stage it happened in. Linux
// only. file is the compose file's repo-relative path; name is the container's.
func expandCompose(repoRoot string, ctx *render.Context, c *loader.Content, o expandOptions) ([]PlannedCommand, error) {
	file, name := ctx.Compose, ctx.ObjectName
	// The compose file gets the same template pass as a `run:`/script, with this
	// object's context, so it can vary per team/object. compose.Load renders it
	// before validating and archiving.
	renderCompose := func(raw []byte) ([]byte, error) {
		s, err := render.RenderString("compose:"+file, string(raw), ctx, c)
		return []byte(s), err
	}
	b, err := compose.Load(repoRoot, file, renderCompose)
	if err != nil {
		return nil, err
	}
	project := compose.ProjectName(name)
	dir := composeRoot + "/" + project
	sh := func(script string) PlannedCommand {
		return PlannedCommand{Command: agentproto.CmdExecute, Payload: agentproto.ExecutePayload{
			Command: "/bin/sh", Args: []string{"-c", script},
		}}
	}

	// The directory goes over as one base64 archive: a write_file's content is
	// text, and this keeps subdirectories, file modes and any binary file
	// (a certificate, say) intact in a single command.
	archivePath := dir + ".tgz.b64"
	out := []PlannedCommand{
		sh(fmt.Sprintf("rm -rf %s && mkdir -p %s", dir, dir)),
		{Command: agentproto.CmdWriteFile, Payload: agentproto.WriteFilePayload{
			Path: archivePath, Content: base64.StdEncoding.EncodeToString(b.Archive), Mode: "0600",
		}},
		sh(fmt.Sprintf("base64 -d %s | tar -xzf - -C %s && rm -f %s", archivePath, dir, archivePath)),
	}

	out = append(out, sh(installDockerScript))

	// Native container-log forwarding for a compose project: set the Docker
	// daemon's default log driver on this host (from the environment's
	// container_logs) and restart Docker before `docker compose up`, so every
	// service inherits it. A service that declares its own `logging:` still
	// wins. The daemon ships directly -- the gateway never sees these logs.
	if o.containerLogs != nil && o.containerLogs.Driver != "" {
		out = append(out, sh(dockerDaemonLogScript(o.containerLogs)))
	}

	if o.registryAuth != nil {
		seen := map[string]bool{}
		for _, image := range b.Images {
			host := compose.RegistryHost(image)
			key := host
			if key == "" {
				key = "docker.io"
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			auth, ok := o.registryAuth(key)
			if !ok {
				continue
			}
			// Docker Hub is `docker login` with no server argument.
			server := ""
			if host != "" {
				server = " " + shellQuote(host)
			}
			out = append(out, sh(fmt.Sprintf("printf %%s %s | docker login%s -u %s --password-stdin",
				shellQuote(auth.Secret), server, shellQuote(auth.Username))))
		}
	}

	dc := fmt.Sprintf("docker compose -p %s --project-directory %s -f %s/%s", project, dir, dir, shellQuote(b.File))
	out = append(out,
		sh(dc+" pull"),
		// --no-build: an image that couldn't be pulled is an error here, never a
		// reason to build on the host. --wait holds until every service is
		// running and, where it has a healthcheck, healthy.
		sh(fmt.Sprintf("%s up -d --no-build --remove-orphans --wait --wait-timeout %d", dc, composeWaitSeconds)),
	)
	return out, nil
}

// shellQuote single-quotes s for /bin/sh.
// dockerDaemonLogScript writes /etc/docker/daemon.json with container_logs'
// driver as the Docker daemon default and restarts Docker, so compose services
// started afterward inherit it. log-opts values are strings (what the drivers
// expect), so the options map serializes directly.
func dockerDaemonLogScript(cl *loader.ContainerLogs) string {
	daemon := map[string]any{"log-driver": cl.Driver}
	if len(cl.Options) > 0 {
		daemon["log-opts"] = cl.Options
	}
	b, _ := json.Marshal(daemon)
	return fmt.Sprintf(`set -e
mkdir -p /etc/docker
cat > /etc/docker/daemon.json <<'LAFORGE_EOF'
%s
LAFORGE_EOF
systemctl restart docker 2>/dev/null || service docker restart || true`, string(b))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
