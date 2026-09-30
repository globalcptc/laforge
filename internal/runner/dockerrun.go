package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/gateway"
	"github.com/globalcptc/laforge/internal/loader"
)

// dockerRunPlannedCommand wraps dockerRunCommand as the PlannedCommand
// materializeSteps queues -- keeping the agentproto reference out of
// runner.go.
func dockerRunPlannedCommand(ct *loader.Container, cred *db.RegistryCredential) gateway.PlannedCommand {
	return gateway.PlannedCommand{Command: agentproto.CmdExecute, Payload: dockerRunCommand(ct, cred)}
}

// dockerContainerName is the fixed name of the one Docker container LaForge
// runs inside each container object's LXD box. Fixed (not per-object) because
// there is exactly one per box, and a stable name makes the run idempotent
// (a redeploy/retry removes and recreates it cleanly).
const dockerContainerName = "laforge"

// dockerRunCommand builds the agent `execute` command that runs a container's
// image: an optional `docker login`, a `docker pull`, then `docker run -d
// --restart=always --network host` so the app binds the LXD box's own lab IP.
// It's the first materialized step for a container object; the agent (running
// in the LXD box) executes it, giving the Docker container full LaForge
// coverage without an agent inside the image itself.
//
// The registry secret is interpolated into the command with -p; it travels to
// the agent over the mTLS gateway, same trust model as every other step
// payload. (--password-stdin would avoid it appearing in the container's
// process args; a follow-up.)
func dockerRunCommand(ct *loader.Container, cred *db.RegistryCredential) agentproto.ExecutePayload {
	var b strings.Builder
	if cred != nil {
		host := cred.RegistryHost
		if host == "docker.io" {
			host = "" // default Docker Hub endpoint
		}
		fmt.Fprintf(&b, "docker login %s -u %s -p %s && ", shq(host), shq(cred.Username), shq(cred.Secret))
	}
	fmt.Fprintf(&b, "docker pull %s && docker rm -f %s >/dev/null 2>&1; docker run -d --restart=always --network host --name %s",
		shq(ct.Image), dockerContainerName, dockerContainerName)
	for _, k := range sortedKeys(ct.Env) {
		fmt.Fprintf(&b, " -e %s", shq(k+"="+ct.Env[k]))
	}
	fmt.Fprintf(&b, " %s", shq(ct.Image))
	for _, arg := range ct.Command {
		fmt.Fprintf(&b, " %s", shq(arg))
	}
	return agentproto.ExecutePayload{Command: "/bin/sh", Args: []string{"-c", b.String()}, TimeoutSec: 600}
}

// registryHost extracts the registry host from an OCI image ref, or "" for a
// Docker Hub ref. A first path segment counts as a host only if it looks like
// one (has a "." or ":", or is "localhost"); otherwise it's a Docker Hub
// namespace ("library/nginx", "myuser/app").
func registryHost(image string) string {
	slash := strings.IndexByte(image, '/')
	if slash < 0 {
		return "" // "nginx", "nginx:alpine"
	}
	first := image[:slash]
	if strings.ContainsAny(first, ".:") || first == "localhost" {
		return first
	}
	return ""
}

// registryCredFor returns the stored credential for an image's registry, or
// nil when there is none (a public image, or an unauthenticated registry). A
// Docker Hub ref looks up the special "docker.io" host, so a private Docker
// Hub account can still be configured.
func (r *Runner) registryCredFor(ctx context.Context, q *db.Queries, image string) *db.RegistryCredential {
	host := registryHost(image)
	if host == "" {
		host = "docker.io"
	}
	cred, err := q.GetRegistryCredentialByHost(ctx, host)
	if err != nil {
		return nil
	}
	return &cred
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
