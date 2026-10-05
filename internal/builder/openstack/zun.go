// Containers on OpenStack are Zun capsules, not Nova servers -- a different
// service and shape from the rest of this builder. This file is a DRAFT: it has
// never run against a real OpenStack cloud with Zun enabled. The model mirrors
// the AWS Fargate path (the LaForge agent runs as the container's supervising
// entrypoint so a container checks in and runs steps/validators like a host),
// adapted to Zun's constraints:
//
//   - A Zun capsule is a pod-like group of containers sharing a network namespace,
//     but it has NO init-container ordering (unlike ECS's dependsOn) -- so rather
//     than a separate init container, the app container's own command is overridden
//     with a small shell that DOWNLOADS the patched agent by its one-time-token URL
//     and execs it in supervisor mode. That needs `sh` + `wget` present in the
//     image; images without them must set `command:` (which also skips agent
//     supervision, same tradeoff as everywhere).
//   - Zun, like Fargate, doesn't hand back the image's own entrypoint, so the
//     command the agent supervises comes from the content `command:` if set, else
//     is fetched from the image's registry config (imagemeta.Entrypoint).
//
// Known DRAFT gaps: a static Address isn't applied (the capsule gets the Neutron
// network's dynamic addressing); the one-time download token won't survive a
// capsule restart; imagemeta only does anonymous Docker Hub pulls; per-team
// network attachment for a capsule (nets: in the template) is best-effort and
// unverified against a live Zun.
package openstack

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	osclient "github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/container/v1/capsules"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/imagemeta"
)

const agentBinInCapsule = "/laforge-agent"

// DeployContainer runs a LaForge `container:` as a Zun capsule. externalRef is
// the capsule UUID. DRAFT -- see the file header. Requires the container service
// client (Zun); if the cloud has no Zun endpoint, New leaves it nil and this
// returns a clear error rather than a capability opt-out.
func (b *Builder) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	if spec.ComposeHost {
		return "", fmt.Errorf("compose containers are not supported on this builder yet")
	}
	if b.container == nil {
		return "", fmt.Errorf("this OpenStack cloud has no Zun (container) service -- containers cannot be deployed here")
	}
	if id, err := b.findCapsule(ctx, spec.ExternalName); err != nil {
		return "", err
	} else if id != "" {
		return id, nil // ensure: adopt
	}
	if err := b.ensureTeamGroup(ctx, spec.Team); err != nil {
		return "", err
	}

	// The command the agent supervises: content Command if set, else the image's
	// own entrypoint (Zun doesn't expose it, so fetch it from the registry).
	supervise := strings.Join(spec.Command, " ")
	if supervise == "" && spec.AgentDownloadURL != "" {
		ep, err := imagemeta.Entrypoint(ctx, spec.Image)
		if err != nil {
			return "", fmt.Errorf("resolving entrypoint of image %q (set command: on the container to avoid this): %w", spec.Image, err)
		}
		supervise = ep
	}

	env := map[string]string{}
	for k, v := range spec.Env {
		env[k] = v
	}

	container := map[string]any{
		"image":           spec.Image,
		"imagePullPolicy": "ifnotpresent",
	}

	if spec.AgentDownloadURL != "" {
		// No init-container ordering in Zun: override the command with a shell that
		// plants the agent and execs it in supervisor mode, supervising `supervise`.
		env["LAFORGE_SUPERVISE"] = supervise
		container["command"] = []string{
			"sh", "-c",
			fmt.Sprintf("set -e; wget -O %s %q; chmod +x %s; exec %s",
				agentBinInCapsule, spec.AgentDownloadURL, agentBinInCapsule, agentBinInCapsule),
		}
	} else if len(spec.Command) > 0 {
		container["command"] = spec.Command
	}

	if len(env) > 0 {
		container["env"] = env
	}
	if ports := capsulePorts(spec.TCPPorts, spec.UDPPorts); len(ports) > 0 {
		container["ports"] = ports
	}
	if cpu, mem, ok := zunResources(b.cfg.Sizes[spec.Size]); ok {
		container["resources"] = map[string]any{
			"requests": map[string]any{"cpu": cpu, "memory": mem},
		}
	}

	template := map[string]any{
		"capsuleVersion": "beta",
		"kind":           "capsule",
		"metadata": map[string]any{
			"name": spec.ExternalName,
			"labels": map[string]string{
				managedKey:      "true",
				externalNameKey: spec.ExternalName,
				teamKey:         spec.Team,
				displayKey:      spec.DisplayName,
			},
		},
		"spec": map[string]any{
			"containers": []map[string]any{container},
		},
	}
	// Best-effort per-team network attachment. Unverified against a live Zun --
	// the capsule template's `nets` shape varies by Zun version, so a failure here
	// shouldn't be silent, but the network's own DeployNetwork must have run first.
	if netID, err := b.findNetwork(ctx, spec.Network); err == nil && netID != "" {
		template["spec"].(map[string]any)["nets"] = []map[string]any{{"network": netID}}
	}

	raw, err := json.Marshal(template)
	if err != nil {
		return "", fmt.Errorf("encoding capsule template for %q: %w", spec.ExternalName, err)
	}
	res, err := capsules.Create(ctx, b.container, capsules.CreateOpts{
		TemplateOpts: &capsules.Template{Bin: raw},
	}).ExtractBase()
	if err != nil {
		return "", fmt.Errorf("creating capsule for %q: %w", spec.ExternalName, err)
	}
	if res.UUID == "" {
		return "", fmt.Errorf("capsule for %q created without a UUID", spec.ExternalName)
	}
	return res.UUID, nil
}

// DestroyContainer deletes the Zun capsule. Idempotent: a missing capsule is not
// an error.
func (b *Builder) DestroyContainer(ctx context.Context, team, externalRef string) error {
	if b.container == nil || externalRef == "" {
		return nil
	}
	if err := capsules.Delete(ctx, b.container, externalRef).ExtractErr(); err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			return nil
		}
		return fmt.Errorf("deleting capsule %s: %w", externalRef, err)
	}
	return nil
}

// findCapsule returns the UUID of a LaForge-managed capsule with this external
// name, or "" if none exists -- the "ensure" adopt check.
func (b *Builder) findCapsule(ctx context.Context, externalName string) (string, error) {
	pages, err := capsules.List(b.container, capsules.ListOpts{}).AllPages(ctx)
	if err != nil {
		return "", fmt.Errorf("listing capsules for %q: %w", externalName, err)
	}
	all, err := capsules.ExtractCapsulesBase(pages)
	if err != nil {
		return "", fmt.Errorf("extracting capsules for %q: %w", externalName, err)
	}
	for _, c := range all {
		if c.MetaLabels[managedKey] == "true" && c.MetaLabels[externalNameKey] == externalName {
			return c.UUID, nil
		}
	}
	return "", nil
}

// capsulePorts maps content TCP/UDP ports to the capsule template's port shape.
func capsulePorts(tcp, udp []string) []map[string]any {
	var out []map[string]any
	add := func(ports []string, proto string) {
		for _, p := range ports {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				out = append(out, map[string]any{"containerPort": n, "protocol": proto})
			}
		}
	}
	add(tcp, "TCP")
	add(udp, "UDP")
	return out
}

// zunResources maps a content size to a capsule's cpu (vCPUs) / memory (MB)
// request. The shared Sizes map holds Nova flavor ids for VMs, which don't apply
// to a capsule, so a "cpu/memory" value (e.g. "1/1024") is honored and anything
// else yields ok=false (let Zun apply its own default).
func zunResources(s string) (cpu float64, mem int, ok bool) {
	a, b, found := strings.Cut(s, "/")
	if !found {
		return 0, 0, false
	}
	c, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	m, err2 := strconv.Atoi(strings.TrimSpace(b))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return c, m, true
}

// newContainerClient opens the Zun (container) service client, or returns nil if
// the cloud's catalog has no such endpoint -- containers then fail at deploy with
// a clear error rather than a capability flag. Called from New.
func newContainerClient(provider *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) *gophercloud.ServiceClient {
	c, err := osclient.NewContainerV1(provider, eo)
	if err != nil {
		return nil
	}
	return c
}
