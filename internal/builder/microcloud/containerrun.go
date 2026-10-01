package microcloud

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/builder"
)

// nestedContainerName is the fixed Docker name of the one application container
// LaForge runs inside each `container:` object's LXD Docker host. Fixed (not
// per-object) because there is exactly one per host, and a stable name makes
// the run idempotent -- a redeploy/retry removes and recreates it cleanly.
const nestedContainerName = "laforge"

// nestedAgentPath is where the per-object LaForge agent is planted on the Docker
// host and bind-mounted into the nested container as its `--entrypoint`. The
// agent then runs as PID 1 inside the application container and supervises the
// image's real command, so the container's steps/validators run INSIDE the app
// -- the Docker runtime is never visible to content, exactly as on a native-OCI
// builder. Mounted read-only; /laforge-agent is the path it appears at inside.
const nestedAgentPath = "/opt/laforge-agent"

// runNestedContainer drives the just-deployed LXD Docker host (boxName) to pull
// and run the application image as a nested Docker container. When an agent
// binary is present it is planted and made the nested container's entrypoint
// (supervisor mode); otherwise the image runs its own entrypoint, unmanaged.
// This is the MicroCloud equivalent of the native-OCI builder planting the
// agent as oci.entrypoint -- the difference is confined to the builder, so
// content stays builder-agnostic.
func (b *Builder) runNestedContainer(ctx context.Context, boxName string, spec builder.ContainerSpec) error {
	if err := b.waitDockerReady(ctx, boxName); err != nil {
		return fmt.Errorf("docker runtime on %s: %w", boxName, err)
	}
	if len(spec.AgentBinary) > 0 {
		if err := b.Client.pushFile(ctx, boxName, nestedAgentPath, 0o755, spec.AgentBinary); err != nil {
			return fmt.Errorf("planting agent on %s: %w", boxName, err)
		}
	}
	code, err := b.Client.exec(ctx, boxName, "/bin/sh", "-c", dockerRunScript(spec))
	if err != nil {
		return fmt.Errorf("running container image on %s: %w", boxName, err)
	}
	if code != 0 {
		return fmt.Errorf("running container image %q on %s: docker run exited %d", spec.Image, boxName, code)
	}
	return nil
}

// waitDockerReady blocks until the Docker daemon on a freshly-booted host
// answers, so the pull/run below doesn't race first boot. A bounded in-guest
// retry loop (not a fixed sleep), surfaced as a real error if Docker never
// comes up rather than letting the run fail with a confusing "docker: not
// found"/"cannot connect to the daemon".
func (b *Builder) waitDockerReady(ctx context.Context, boxName string) error {
	const probe = `for i in $(seq 1 60); do docker info >/dev/null 2>&1 && exit 0; sleep 2; done; exit 1`
	code, err := b.Client.exec(ctx, boxName, "/bin/sh", "-c", probe)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("docker daemon did not come up within the timeout")
	}
	return nil
}

// dockerRunScript builds the `/bin/sh -c` command the Docker host runs to bring
// up the application container: an optional `docker login`, a `docker pull`, a
// clean `docker rm -f`, then `docker run -d --restart=always --network host` so
// the app binds the host's own lab IP (the team's OVN address). With an agent,
// it is bind-mounted in and made the entrypoint, supervising the image's real
// command (content's `command:` override, or the image's own entrypoint+cmd
// resolved in-host via `docker inspect` -- the Docker equivalent of the
// oci.entrypoint a native-OCI builder reads from image metadata). The registry
// secret travels to the host over the same mTLS API as every other call.
func dockerRunScript(spec builder.ContainerSpec) string {
	img := shq(spec.Image)
	var b strings.Builder
	if spec.RegistryHost != "" || spec.RegistryUser != "" {
		host := spec.RegistryHost
		if host == "docker.io" {
			host = "" // default Docker Hub endpoint
		}
		fmt.Fprintf(&b, "docker login %s -u %s -p %s && ", shq(host), shq(spec.RegistryUser), shq(spec.RegistrySecret))
	}
	fmt.Fprintf(&b, "docker pull %s && docker rm -f %s >/dev/null 2>&1; docker run -d --restart=always --network host --name %s",
		img, nestedContainerName, nestedContainerName)
	for _, k := range sortedKeys(spec.Env) {
		fmt.Fprintf(&b, " -e %s", shq(k+"="+spec.Env[k]))
	}
	if len(spec.AgentBinary) > 0 {
		fmt.Fprintf(&b, " -v %s:/laforge-agent:ro --entrypoint /laforge-agent", shq(nestedAgentPath))
		if len(spec.Command) > 0 {
			// Content gave the command explicitly -- quote it literally, no
			// in-host resolution needed.
			fmt.Fprintf(&b, " -e %s %s", shq("LAFORGE_SUPERVISE="+strings.Join(spec.Command, " ")), img)
		} else {
			// No command override: resolve the image's own entrypoint+cmd on the
			// host at run time, so nothing has to be known ahead of time.
			// Double-quoted so the command substitution runs and its (possibly
			// space-bearing) result stays a single -e value. Image refs carry no
			// shell metacharacters, so inlining spec.Image raw here is safe.
			fmt.Fprintf(&b, ` -e "LAFORGE_SUPERVISE=$(docker inspect --format '{{range .Config.Entrypoint}}{{.}} {{end}}{{range .Config.Cmd}}{{.}} {{end}}' %s)" %s`, spec.Image, img)
		}
	} else {
		// No agent delivery: run the image as-is, content command as its args.
		fmt.Fprintf(&b, " %s", img)
		for _, arg := range spec.Command {
			fmt.Fprintf(&b, " %s", shq(arg))
		}
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// shq single-quotes a value for /bin/sh, escaping embedded single quotes.
func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
