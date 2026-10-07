package incus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// This file builds the "docker base image" on one Incus host: an Ubuntu system
// container image with Docker and the compose plugin installed, published into
// the host's own image store as builder.DockerBaseAlias. A `container:` that
// runs a Docker Compose project boots a nesting container from it (see
// DeployContainer); a single-image `container:` doesn't use it -- Incus runs
// those as native OCI.
//
// It is the Incus counterpart of internal/builder/microcloud's BuildDockerBase,
// running the same recipe (builder.DockerBaseInstallScript) against the Incus
// API. An Incus builder is a pool of independent hosts with no shared image
// store, so the orchestrator runs this once per host; instances then boot from
// the alias, which every host resolves to its own copy.

// tempBuildInstance is the throwaway container the base is built in. Fixed
// name so a re-run cleans up a leftover from an interrupted build.
const tempBuildInstance = "laforge-docker-base-build"

// DefaultBaseImageSource is the Ubuntu container image the base is built from
// when a builder config doesn't name one Incus can use: the linuxcontainers.org
// cloud variant, which ships cloud-init (the agent is delivered through it).
var DefaultBaseImageSource = BaseImageSource{
	Server: "https://images.linuxcontainers.org", Protocol: "simplestreams", Alias: "ubuntu/24.04/cloud",
}

// BaseImageSource is where the Ubuntu container base is pulled from.
type BaseImageSource struct {
	Server   string
	Protocol string
	Alias    string
}

// BuildDockerBase builds (or rebuilds) the docker base image on this host and
// returns its published fingerprint. log receives one human-readable progress
// line at a time, for the live console. It always cleans up the temp build
// instance. storagePool is the pool the build container's root disk goes on
// ("" for the default profile's).
func BuildDockerBase(ctx context.Context, c *Client, src BaseImageSource, storagePool string, log func(string)) (string, error) {
	if src.Server == "" || src.Alias == "" {
		src = DefaultBaseImageSource
	}
	if src.Protocol == "" {
		src.Protocol = "simplestreams"
	}
	if log == nil {
		log = func(string) {}
	}
	c.forceStopDelete(ctx, tempBuildInstance)
	defer c.forceStopDelete(ctx, tempBuildInstance)

	log(fmt.Sprintf("Creating nesting-enabled build container from %q (%s) …", src.Alias, src.Server))
	create := map[string]any{
		"name": tempBuildInstance, "type": "container",
		"source": map[string]any{"type": "image", "mode": "pull", "alias": src.Alias, "server": src.Server, "protocol": src.Protocol},
		"config": map[string]any{"security.nesting": "true"},
	}
	if storagePool != "" {
		create["devices"] = map[string]any{"root": map[string]string{"type": "disk", "path": "/", "pool": storagePool}}
	}
	if _, err := c.post(ctx, "/1.0/instances", create); err != nil {
		return "", fmt.Errorf("creating build container: %w", err)
	}
	if _, err := c.put(ctx, "/1.0/instances/"+tempBuildInstance+"/state", map[string]any{"action": "start", "timeout": 60}); err != nil {
		return "", fmt.Errorf("starting build container: %w", err)
	}
	if ip := c.waitForInstanceIPv4(ctx, tempBuildInstance, 90*time.Second); ip != "" {
		log("Build container is up (" + ip + ")")
	} else {
		log("Build container started (no IPv4 yet; continuing)")
	}

	log("Installing Docker and the compose plugin …")
	out, err := c.execRecord(ctx, tempBuildInstance, builder.DockerBaseInstallScript)
	logLines(log, out)
	if err != nil {
		return "", fmt.Errorf("installing docker: %w", err)
	}
	if !strings.Contains(out, "DOCKER_OK") {
		return "", fmt.Errorf("docker and the compose plugin did not come up under nesting")
	}
	_, _ = c.execRecord(ctx, tempBuildInstance, builder.DockerBaseCleanupScript)

	log("Stopping build container to publish …")
	if _, err := c.put(ctx, "/1.0/instances/"+tempBuildInstance+"/state", map[string]any{"action": "stop", "timeout": 30}); err != nil {
		return "", fmt.Errorf("stopping build container: %w", err)
	}

	// A rebuild reuses the alias: drop it first so publish can reclaim it. The
	// previous image is left unaliased; instances already running from it are
	// unaffected.
	_, _ = c.delete(ctx, "/1.0/images/aliases/"+builder.DockerBaseAlias)

	log("Publishing image as " + builder.DockerBaseAlias + " …")
	fp, err := c.publishInstanceAsImage(ctx, tempBuildInstance, builder.DockerBaseAlias)
	if err != nil {
		return "", fmt.Errorf("publishing image: %w", err)
	}
	log("Published " + builder.DockerBaseAlias + " (" + shortFingerprint(fp) + ")")
	return fp, nil
}

// publishInstanceAsImage publishes a (stopped) instance as a new image with
// the given alias and returns the new image's fingerprint.
func (c *Client) publishInstanceAsImage(ctx context.Context, instance, alias string) (string, error) {
	raw, err := c.post(ctx, "/1.0/images", map[string]any{
		"source":  map[string]any{"type": "instance", "name": instance},
		"aliases": []map[string]any{{"name": alias}},
	})
	if err != nil {
		return "", err
	}
	var op struct {
		Metadata struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"metadata"`
	}
	if json.Unmarshal(raw, &op) == nil && op.Metadata.Fingerprint != "" {
		return op.Metadata.Fingerprint, nil
	}
	aliasRaw, err := c.get(ctx, "/1.0/images/aliases/"+alias)
	if err != nil {
		return "", err
	}
	var a struct {
		Target string `json:"target"`
	}
	_ = json.Unmarshal(aliasRaw, &a)
	if a.Target == "" {
		return "", fmt.Errorf("published image but its fingerprint could not be resolved")
	}
	return a.Target, nil
}

// execRecord runs one shell script in a container in record-output mode and
// returns its combined stdout+stderr. The script reports its own success in
// what it prints, so the exit code isn't returned.
func (c *Client) execRecord(ctx context.Context, instance, script string) (string, error) {
	raw, err := c.post(ctx, "/1.0/instances/"+instance+"/exec", map[string]any{
		"command":            []string{"sh", "-c", script},
		"wait-for-websocket": false,
		"record-output":      true,
		"interactive":        false,
		"environment":        map[string]string{"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
	})
	if err != nil {
		return "", err
	}
	var op struct {
		Metadata struct {
			Output map[string]string `json:"output"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &op); err != nil {
		return "", fmt.Errorf("decoding exec operation: %w", err)
	}
	var sb strings.Builder
	for _, key := range []string{"1", "2"} {
		if p := op.Metadata.Output[key]; p != "" {
			sb.WriteString(c.rawGet(ctx, p))
		}
	}
	return sb.String(), nil
}

// rawGet fetches a non-JSON endpoint (an exec output log), which do() can't:
// it decodes every response as an API envelope.
func (c *Client) rawGet(ctx context.Context, path string) string {
	u := c.BaseURL + path
	if c.Project != "" && !strings.Contains(path, "project=") {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		u += sep + "project=" + url.QueryEscape(c.Project)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func (c *Client) waitForInstanceIPv4(ctx context.Context, name string, within time.Duration) string {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		raw, err := c.get(ctx, "/1.0/instances/"+name+"/state")
		if err == nil {
			var st struct {
				Network map[string]struct {
					Addresses []struct {
						Family, Address, Scope string
					} `json:"addresses"`
				} `json:"network"`
			}
			if json.Unmarshal(raw, &st) == nil {
				for _, a := range st.Network["eth0"].Addresses {
					if a.Family == "inet" && a.Scope == "global" {
						return a.Address
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return ""
}

// forceStopDelete best-effort removes an instance (stopped or running),
// ignoring "not found" -- used to clean up the temp build container.
func (c *Client) forceStopDelete(ctx context.Context, name string) {
	_, _ = c.put(ctx, "/1.0/instances/"+name+"/state", map[string]any{"action": "stop", "timeout": 30, "force": true})
	time.Sleep(1 * time.Second)
	_, _ = c.delete(ctx, "/1.0/instances/"+name)
}

func shortFingerprint(fp string) string {
	if len(fp) > 12 {
		return fp[:12]
	}
	return fp
}

func logLines(log func(string), out string) {
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			log(line)
		}
	}
}
