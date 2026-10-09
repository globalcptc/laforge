package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// This file is the engine behind the per-builder "docker base image" build
// job (docs: the Docker-container design). On a MicroCloud/Incus cluster
// there is no OCI runtime, so a LaForge `container:` runs as a thin, nesting-
// enabled LXD system container that itself runs Docker. That LXD container
// boots from a docker-ready base image, which this builds ON the cluster and
// publishes into its own image store:
//
//	import an ubuntu container base (from the builder's configured image
//	server -- cloud-images.ubuntu.com for Canonical LXD, NOT
//	images.linuxcontainers.org) -> create a nesting container from it ->
//	install Docker -> publish the result as the alias below -> clean up.
//
// Every step calls the caller's log function so the builder-config page can
// stream a live console. Proven end to end against a real MicroCloud cluster
// (see TestDockerInLXDSpike and TestBuildDockerBaseLive).

// DockerBaseAlias is the published image the container runtime boots from.
const DockerBaseAlias = builder.DockerBaseAlias

// tempBuildInstance is the throwaway container the base is built in. Fixed
// name so a re-run adopts/cleans a leftover from an interrupted build.
const tempBuildInstance = "laforge-docker-base-build"

// BaseImageSource is where the ubuntu container base is pulled from -- a
// per-builder setting, so an internet-connected cluster can use cloud-images
// and a locked-down one an internal mirror. Defaults live in builderconfig.
type BaseImageSource struct {
	Server   string // e.g. https://cloud-images.ubuntu.com/releases
	Protocol string // simplestreams
	Alias    string // e.g. 22.04
}

// BuildDockerBase builds (or rebuilds) the docker-ready base image on this
// cluster and returns its published fingerprint. log receives one
// human-readable progress line at a time (step markers plus the real command
// output), for the live console. It always cleans up the temp build instance.
//
// storagePool is the builder's pool. In a project other than `default` the
// build container gets its root disk there explicitly, since such a project's
// default profile may have none; it still needs that profile to give it a
// network with internet access, and fails clearly if it doesn't.
func BuildDockerBase(ctx context.Context, c *Client, src BaseImageSource, storagePool string, log func(string)) (string, error) {
	if src.Server == "" || src.Alias == "" {
		return "", fmt.Errorf("base image source is not configured (server and alias are required)")
	}
	if log == nil {
		log = func(string) {}
	}
	// Clean any leftover temp instance from a prior interrupted build first.
	c.forceStopDelete(ctx, tempBuildInstance)
	defer c.forceStopDelete(ctx, tempBuildInstance)

	log(fmt.Sprintf("Importing base image %q from %s …", src.Alias, src.Server))
	baseFP, err := c.importImagePull(ctx, src)
	if err != nil {
		return "", fmt.Errorf("importing base image: %w", err)
	}
	log("Imported base image " + short(baseFP))

	log("Creating nesting-enabled build container …")
	create := map[string]any{
		"name": tempBuildInstance, "type": "container",
		"source": map[string]any{"type": "image", "fingerprint": baseFP},
		"config": map[string]any{"security.nesting": "true"},
	}
	if c.Project != "" {
		if storagePool == "" {
			storagePool = "default"
		}
		create["devices"] = map[string]any{"root": map[string]string{"type": "disk", "path": "/", "pool": storagePool}}
	}
	if _, err := c.post(ctx, "/1.0/instances", create); err != nil {
		return "", fmt.Errorf("creating build container: %w", err)
	}
	if !c.hasNIC(ctx, tempBuildInstance) {
		return "", fmt.Errorf("the build container has no network device: project %q's default profile has no NIC, and the build needs internet access -- add one to that profile (e.g. on the cluster's default OVN network)", c.Project)
	}
	if _, err := c.put(ctx, "/1.0/instances/"+tempBuildInstance+"/state", map[string]any{"action": "start", "timeout": 60}); err != nil {
		return "", fmt.Errorf("starting build container: %w", err)
	}
	if ip := c.waitForInstanceIPv4(ctx, tempBuildInstance, 60*time.Second); ip != "" {
		log("Build container is up (" + ip + ")")
	} else {
		log("Build container started (no IPv4 yet; continuing)")
	}

	log("Installing Docker and the compose plugin …")
	_, out, err := c.execRecord(ctx, tempBuildInstance, builder.DockerBaseInstallScript)
	logLines(log, out)
	if err != nil {
		return "", fmt.Errorf("installing docker: %w", err)
	}
	if !strings.Contains(out, "DOCKER_OK") {
		return "", fmt.Errorf("docker and the compose plugin did not come up under nesting")
	}

	// Trim what shouldn't ship in the image.
	_, _, _ = c.execRecord(ctx, tempBuildInstance, builder.DockerBaseCleanupScript)

	log("Stopping build container to publish …")
	if _, err := c.put(ctx, "/1.0/instances/"+tempBuildInstance+"/state", map[string]any{"action": "stop", "timeout": 30}); err != nil {
		return "", fmt.Errorf("stopping build container: %w", err)
	}

	// A rebuild reuses the alias: drop the old alias first so publish can
	// reclaim it (the old image is left unaliased; the caller may prune it by
	// the previously recorded fingerprint).
	_, _ = c.delete(ctx, "/1.0/images/aliases/"+DockerBaseAlias)

	log("Publishing image as " + DockerBaseAlias + " …")
	fp, err := c.publishInstanceAsImage(ctx, tempBuildInstance, DockerBaseAlias)
	if err != nil {
		return "", fmt.Errorf("publishing image: %w", err)
	}
	log("Published " + DockerBaseAlias + " (" + short(fp) + ")")
	return fp, nil
}

// importImagePull server-side pulls a container image and returns its local
// fingerprint. Falls back to scanning the store when the operation doesn't
// echo the fingerprint (older daemons).
func (c *Client) importImagePull(ctx context.Context, src BaseImageSource) (string, error) {
	protocol := src.Protocol
	if protocol == "" {
		protocol = "simplestreams"
	}
	raw, err := c.post(ctx, "/1.0/images", map[string]any{
		"source": map[string]any{
			"type": "image", "mode": "pull",
			"alias": src.Alias, "server": src.Server, "protocol": protocol,
		},
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
	if fp := c.newestContainerImage(ctx); fp != "" {
		return fp, nil
	}
	return "", fmt.Errorf("image imported but its fingerprint could not be resolved")
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
	// Fall back to resolving the alias we just set.
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

// execRecord runs one shell command in a container via the LXD exec API in
// record-output mode, returning its exit code and combined stdout+stderr.
func (c *Client) execRecord(ctx context.Context, instance, script string) (int, string, error) {
	raw, err := c.post(ctx, "/1.0/instances/"+instance+"/exec", map[string]any{
		"command":            []string{"sh", "-c", script},
		"wait-for-websocket": false,
		"record-output":      true,
		"interactive":        false,
		"environment":        map[string]string{"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
	})
	if err != nil {
		return -1, "", err
	}
	var op struct {
		Metadata struct {
			Output map[string]string `json:"output"`
			Return int               `json:"return"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &op); err != nil {
		return -1, "", fmt.Errorf("decoding exec operation: %w", err)
	}
	var sb strings.Builder
	for _, key := range []string{"1", "2"} {
		if p := op.Metadata.Output[key]; p != "" {
			sb.WriteString(c.rawGet(p))
		}
	}
	return op.Metadata.Return, sb.String(), nil
}

// rawGet fetches a non-JSON endpoint (an exec log file) directly, bypassing
// do()'s apiResponse decoding.
func (c *Client) rawGet(path string) string {
	u := c.BaseURL + path
	if c.Project != "" && !strings.Contains(path, "project=") {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		u += sep + "project=" + url.QueryEscape(c.Project)
	}
	resp, err := c.HTTPClient.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func (c *Client) newestContainerImage(ctx context.Context) string {
	raw, err := c.get(ctx, "/1.0/images?recursion=1")
	if err != nil {
		return ""
	}
	var imgs []struct {
		Fingerprint string `json:"fingerprint"`
		Type        string `json:"type"`
	}
	if json.Unmarshal(raw, &imgs) != nil {
		return ""
	}
	for _, im := range imgs {
		if im.Type == "container" {
			return im.Fingerprint
		}
	}
	return ""
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
				if eth0, ok := st.Network["eth0"]; ok {
					for _, a := range eth0.Addresses {
						if a.Family == "inet" && a.Scope == "global" {
							return a.Address
						}
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

func short(fp string) string {
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

// hasNIC reports whether an instance ends up with any network device once its
// profiles are applied.
func (c *Client) hasNIC(ctx context.Context, name string) bool {
	raw, err := c.get(ctx, "/1.0/instances/"+name)
	if err != nil {
		return true // can't tell; let the build itself find out
	}
	var inst struct {
		ExpandedDevices map[string]map[string]string `json:"expanded_devices"`
	}
	if json.Unmarshal(raw, &inst) != nil {
		return true
	}
	for _, d := range inst.ExpandedDevices {
		if d["type"] == "nic" {
			return true
		}
	}
	return false
}
