package incus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/render"
)

// Builder provisions native Linux bridges or legacy OVN networks on one Incus host.
// Bridge networks route and NAT through the host and require non-overlapping subnets.
// agentUserData renders the agent-install user-data for an instance from the
// neutral download URL, choosing the format this image consumes: cloud-init for
// Linux, cloudbase-init for Windows. tempPass, when set, is the first-boot
// temporary admin password the user-data applies so the host is reachable before
// its agent runs. An empty url (agent delivery off) yields no user-data. Whether
// the result rides a config-drive ISO vs the lxd socket is decided by the caller
// (DeployHost), since it also depends on whether the agent binary is embedded.
func agentUserData(url, os, tempPass string) string {
	if url == "" {
		return ""
	}
	if agentdelivery.PlatformFor(os) == agentdelivery.Windows {
		return agentdelivery.WindowsCloudbaseUserData(url, tempPass)
	}
	return agentdelivery.LinuxUserData(url, tempPass)
}

// Builder implements internal/builder.Builder against one real Incus
// server. Team isolation is network-level: every NetworkSpec becomes its
// own Incus OVN logical network.
//
// OVN, not a plain Incus bridge, because "every team gets exactly the
// same network" (same CIDR, same
// addresses, for every team) was tested live against two plain bridges
// carrying the identical CIDR and does not work: a *managed* bridge with
// the same ipv4.address on two different Incus networks fails outright
// ("dnsmasq: failed to create listening socket for 10.0.1.1: Address
// already in use" -- both networks' dnsmasq trying to bind the same
// gateway IP), and the alternative -- unmanaged bridges (no
// ipv4.address) with the IP assigned by hand -- creates two identical
// routes in the HOST's own single routing table, one per bridge; `ip
// route get 10.0.1.11` on the host resolves to whichever bridge was
// created first, regardless of which team's traffic it actually is, so
// return/NAT traffic for the second team's identically-addressed host
// silently goes to the first team's bridge instead. Neither is a config
// mistake fixable by tuning; both are the host having exactly one
// routing table for however many teams share it.
//
// OVN sidesteps this by design: each OVN network is its own isolated
// logical switch/router with its own address space, and Incus's own docs
// name identical subnets across multiple OVN networks as a supported
// case ("useful for labs and multi-tenant environments where the same
// logical subnets are used in multiple discrete networks"). Not
// live-verified end to end in this session's own test daemon: the
// Docker-Desktop-on-Mac Linux kernel this environment runs under has no
// openvswitch.ko (confirmed: `modprobe openvswitch` fails, "Module
// openvswitch not found"), so no real OVN control plane could be brought
// up here to test against, only the plain-bridge collision above and the
// OVN network-create call's own "OVN isn't currently available" error
// with no OVN control plane behind it (see builder_test.go). A real
// MicroCloud cluster's own OVN setup is what this now depends on.
type Builder struct {
	Client *Client
	Config Config
	// buildScope, when set, limits team-scoped operations (OpenAccess/
	// CloseAccess) to instances of one build -- an Incus project is shared across
	// environments, so team "5" of two environments otherwise collide. Set per
	// task by the runner (see builder.BuildScoper); "" is team-only.
	buildScope string
}

// ScopeToBuild implements builder.BuildScoper: limit team operations to one
// build's instances, so a team's open/close doesn't reach another environment's
// same-numbered team sharing the project.
func (b *Builder) ScopeToBuild(buildID string) { b.buildScope = buildID }

func New(client *Client, config Config) *Builder {
	return &Builder{Client: client, Config: config}
}

// ImageNames satisfies internal/orchestrator's own small imageCatalog
// interface: the set of os/image names this builder has a real Incus
// image configured for, from Config.Images -- what
// checkBuilderCompatibility checks every referenced os/image against
// before a build starts (the exact "windows-server-2022 not listed"
// example).
func (b *Builder) ImageNames() map[string]bool {
	names := make(map[string]bool, len(b.Config.Images))
	for name := range b.Config.Images {
		names[name] = true
	}
	return names
}

// shortName maps an arbitrarily long, human-readable ExternalName (as
// internal/runner's deterministic naming produces -- 80+ characters is
// typical) into a name Incus will actually accept. Verified against a
// live daemon rather than assumed from documentation: instance names are
// capped at 63 characters, network names at 15 -- a network name becomes
// a real Linux network interface name, confirmed by the exact error a
// live daemon returns for a longer one ("Network interface is too long
// (maximum 15 characters)"). "lf-" + 12 hex characters of a SHA-256 hash
// is 15 characters exactly -- the tightest of the two constraints,
// applied uniformly to both instances and networks rather than
// maintaining two different naming schemes. Deterministic: the same
// ExternalName always produces the same short name, which "ensure"
// idempotency requires -- a retry has to compute the exact same name a
// previous attempt did, not a new one.
func shortName(externalName string) string {
	sum := sha256.Sum256([]byte(externalName))
	return "lf-" + hex.EncodeToString(sum[:])[:12]
}

// LXD socket-path budget (issue #63, ported from the MicroCloud builder). When a
// VM starts, the daemon opens a Unix control socket at
//
//	<incusLogsDir><project>_<instance>/qemu.monitor
//
// and Go's Linux sun_path buffer is 108 bytes INCLUDING the trailing NUL, so the
// whole path must be <= 107 bytes or the connect fails with a bare EINVAL long
// after the instance name passed the 63-char limit. We budget the generated name
// against it. incusLogsDir is the Incus package-install default; a snap install's
// path is longer, so this can under-budget there -- same "assume the standard
// install" tradeoff MicroCloud makes with the LXD snap path.
const (
	incusLogsDir      = "/var/lib/incus/logs/"
	qemuMonitorSuffix = "/qemu.monitor"
	unixSocketPathMax = 107             // 108-byte sun_path minus the trailing NUL
	minInstanceName   = len("lf-") + 12 // shortName is always "lf-"+12 hex = 15 bytes
)

// maxInstanceNameLen is the longest instance name that keeps the qemu-monitor
// socket path within the Unix-socket limit for this project, also honoring the
// 63-char cidata-volume cap.
func maxInstanceNameLen(project string) int {
	socketCap := unixSocketPathMax - len(incusLogsDir) - len(project) - len("_") - len(qemuMonitorSuffix)
	cidataCap := 63 - len("-cidata")
	if socketCap < cidataCap {
		return socketCap
	}
	return cidataCap
}

// validateInstanceNameBudget rejects, before any resource is created, a project
// so long that not even the minimal hashed name fits the socket path -- a
// permanent configuration error (retrying can't shorten the project).
func validateInstanceNameBudget(project string) error {
	if maxInstanceNameLen(project) < minInstanceName {
		minPath := len(incusLogsDir) + len(project) + len("_") + minInstanceName + len(qemuMonitorSuffix)
		return fmt.Errorf("%w: Incus project %q (%d chars) leaves no room for a valid instance name: even the shortest %d-char name yields a %d-byte VM socket path, over the %d-byte limit. Use a shorter project name.",
			builder.ErrPermanentConfig, project, len(project), minInstanceName, minPath, unixSocketPathMax)
	}
	return nil
}

// instanceName is shortName's readable counterpart for hosts and
// containers (not networks, which are stuck with the 15-char cap and so
// keep the pure hash): "lf-<branch>-t<team>-<host>-<6 hex>". The hex
// suffix is a short digest of the full ExternalName -- it keeps the name
// deterministic (idempotent adopt-on-retry recomputes the exact same name)
// and collision-free across two live builds of the same branch+host on one
// hoster, which the readable part alone can't guarantee. Falls back to
// shortName when there's no display hint at all. Capped to fit BOTH the cidata
// volume's 63-char limit AND the full qemu-monitor socket path for project.
func instanceName(displayName, externalName, project string) string {
	base := sanitizeName(displayName)
	if base == "" {
		return shortName(externalName)
	}
	sum := sha256.Sum256([]byte(externalName))
	suffix := hex.EncodeToString(sum[:])[:6]
	maxBase := maxInstanceNameLen(project) - len("-") - len(suffix)
	if maxBase < len("lf-") {
		return shortName(externalName) // project ate the budget; the 15-byte hash is the floor
	}
	if len(base) > maxBase {
		base = strings.TrimRight(base[:maxBase], "-")
	}
	return base + "-" + suffix
}

// HostReference reconstructs the same instance name DeployHost produced, so a
// destroy whose recorded external_ref was cleared (a rebuild) still targets the
// real `lf-...` instance instead of the logical external name. Deterministic from
// the identical inputs -- the optional hook the runner probes for on destroy.
func (b *Builder) HostReference(externalName, displayName string) string {
	return instanceName(displayName, externalName, b.Client.Project)
}

// networkName is instanceName's counterpart for networks, which are held
// to a much tighter 15-char cap (an Incus network name becomes a real
// Linux interface name). It keeps the "lf-" prefix -- Inspect identifies
// laforge's own networks by exactly that prefix -- and spends the
// remaining 12 characters on a readable, hyphen-free label plus a 4-hex
// disambiguator: "lf-t1vdi3a9f". Same determinism/collision guarantees as
// instanceName, and both the network's own deploy and every instance NIC
// that references it compute this from the identical inputs, so they
// always agree on the name. Falls back to shortName when there's no hint.
func networkName(displayName, externalName string) string {
	base := sanitizeCompact(displayName)
	if base == "" {
		return shortName(externalName)
	}
	sum := sha256.Sum256([]byte(externalName))
	suffix := hex.EncodeToString(sum[:])[:4]
	const maxBase = 15 - len("lf-") - 4 // 8
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return "lf-" + base + suffix
}

// sanitizeCompact reduces a hint to just [a-z0-9] (no hyphens, no prefix)
// -- the hyphen-free core networkName packs into its 15-char budget.
func sanitizeCompact(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeName turns an arbitrary display hint into a valid Incus instance
// name: lowercase, only [a-z0-9-], no repeated or edge hyphens, always
// starting with a letter (the "lf-" prefix guarantees that). Returns ""
// for an empty hint so instanceName can fall back to the hash.
func sanitizeName(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("lf-")
	lastHyphen := true // avoid a leading hyphen right after "lf-"
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// bridgeAddressCIDR turns a content network's own CIDR (e.g.
// "10.0.1.0/24" -- the network address itself) into the address the network gateway
// needs, in CIDR notation (e.g. "10.0.1.1/24"). Found by testing the
// equivalent plain-bridge case against a live daemon (OVN itself isn't
// available in this session's environment -- see Builder's own doc
// comment): Incus rejects the bare network address outright ("Not a
// usable IPv4 address"), so the network needs a real gateway address on
// the subnet -- last_octet 1 by convention, reusing internal/render.Address
// (the exact same "CIDR + last_octet -> IP" arithmetic
// Address() implements for content) rather
// than a second copy of it.
func bridgeAddressCIDR(cidr string) (string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	ones, _ := ipnet.Mask.Size()
	addr, err := render.Address(cidr, 1)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d", addr, ones), nil
}

func (b *Builder) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	// A peered OVN network can't be deleted while its peers exist, and its ACL
	// can't be deleted while attached -- tear down the visible_from fabric
	// (ConfigureNetworkAccess's peers + ACL) first. Best-effort: a missing
	// peer/ACL is fine, this only has to leave the network deletable.
	b.teardownNetworkAccess(ctx, externalRef)
	_, err := b.Client.delete(ctx, "/1.0/networks/"+externalRef)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			return nil
		}
		return fmt.Errorf("deleting network %s: %w", externalRef, err)
	}
	return nil
}

func (b *Builder) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	img, ok := b.Config.Images[spec.OS]
	if !ok {
		return "", fmt.Errorf("no image configured for os %q -- this builder config doesn't support it", spec.OS)
	}
	size, ok := b.Config.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no size configured for %q", spec.Size)
	}
	instanceType := "container"
	if img.VM {
		instanceType = "virtual-machine"
	}
	// A custom compose-host system image still runs Docker inside Incus.
	// VMs and ordinary host images do not need container nesting enabled.
	nesting := !img.VM && spec.OS == builder.ComposeHostImage
	win := agentdelivery.PlatformFor(spec.OS) == agentdelivery.Windows
	cloudInit := agentUserData(spec.AgentDownloadURL, spec.OS, spec.TempAdminPassword)
	// Embed-first: when we have the patched binary, write it onto the config
	// drive so the user-data copies it locally (the download URL stays as the
	// fallback). Windows always needs the config drive (cloudbase-init can't read
	// the LXD socket); Linux switches to it only when embedding, else the socket.
	var agentBinary []byte
	agentFile := ""
	if len(spec.AgentBinary) > 0 {
		agentBinary = spec.AgentBinary
		agentFile = agentdelivery.LinuxAgentFile
		if win {
			agentFile = agentdelivery.WindowsAgentFile
		}
	}
	viaISO := win || len(agentBinary) > 0
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, instanceType, img, size, spec.DiskGB, nesting, cloudInit, viaISO, agentBinary, agentFile, spec.Priority)
}

// deployComposeHost deploys the machine a compose container's project runs on:
// a nesting system container booted from this builder's docker base image
// (Docker and the compose plugin already installed), with the agent delivered
// by cloud-init like a host's. A builder image named builder.ComposeHostImage
// overrides that -- an operator's own image, VM or container, deployed exactly
// as a host would be. The base is named by alias, not
// fingerprint: each host in a pool built its own copy.
func (b *Builder) deployComposeHost(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	if _, ok := b.Config.Images[builder.ComposeHostImage]; ok {
		return b.DeployHost(ctx, spec.ComposeHostSpec())
	}
	if b.Config.DockerBaseFingerprint == "" {
		return "", fmt.Errorf("this builder has no docker base image yet -- build it first (Infrastructure → Base image), or add an image named %q", builder.ComposeHostImage)
	}
	size, ok := b.Config.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no size configured for %q", spec.Size)
	}
	// A compose host is always this Linux docker-base container, so its agent is
	// delivered by cloud-init over the lxd socket (never the ISO path; no embed).
	cloudInit := agentUserData(spec.AgentDownloadURL, "linux", "")
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, "container", ImageRef{Alias: builder.DockerBaseAlias}, size, spec.DiskGB, true, cloudInit, false, nil, "", spec.Priority)
}

// laforgeAgentPath is where the LaForge agent is planted inside a native OCI
// container and made its entrypoint (see installAgentSupervisor).
const laforgeAgentPath = "/usr/local/bin/laforge-agent"

// DeployContainer runs a LaForge `container:` as a NATIVE OCI application
// container. Incus 6.3+ speaks the OCI protocol directly, so spec.Image is
// pulled straight from the registry (Config.OCIRegistry, Docker Hub by default)
// and run as a real container -- no nested Docker, no docker-base image.
// spec.Env becomes the container's environment, spec.Command overrides its
// command, and it lands on the team's own network with spec.Address, exactly
// like a host's NIC.
//
// When spec.AgentBinary is provided (agent delivery on), the LaForge agent is
// planted inside and made the container's entrypoint: it starts the image's real
// command as a child and supervises it while checking in over the gateway, so a
// container runs its `steps:`/validators exactly like a host (see the Rust
// agent's supervisor mode). Verified against a real Incus 7.5.1 daemon.
func (b *Builder) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	// A compose container is a Docker-capable machine the agent starts the
	// project on -- deployed, and later destroyed, like any other instance.
	if spec.ComposeHost {
		return b.deployComposeHost(ctx, spec)
	}
	if err := validateInstanceNameBudget(b.Client.Project); err != nil {
		return "", err
	}
	name := instanceName(spec.DisplayName, spec.ExternalName, b.Client.Project)
	size, ok := b.Config.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no size configured for %q", spec.Size)
	}

	config := map[string]string{teamConfigKey: spec.Team, externalNameKey: spec.ExternalName}
	if size.CPU != "" {
		config["limits.cpu"] = size.CPU
	}
	if size.Memory != "" {
		config["limits.memory"] = size.Memory
	}
	for k, v := range spec.Env {
		config["environment."+k] = v
	}
	// Incus runs the OCI image directly under LXD -- no Docker daemon, so no
	// native log driver. When the environment set container_logs, the agent
	// instead forwards this container's output to the gateway; LAFORGE_LOG_FORWARD
	// turns that on (see run_container in the agent). Native-driver builders
	// (MicroCloud, Fargate) never set this.
	if spec.LogDriver != "" {
		config["environment.LAFORGE_LOG_FORWARD"] = "1"
	}
	// Without the agent, a Command override is the container's entrypoint
	// directly. With the agent (below), the agent supervises that command
	// instead, so it isn't set here.
	if len(spec.Command) > 0 && len(spec.AgentBinary) == 0 {
		config["oci.entrypoint"] = strings.Join(spec.Command, " ")
	}

	devices := map[string]interface{}{}
	if spec.Network != "" {
		nic := map[string]string{"type": "nic", "network": networkName(spec.NetworkDisplayName, spec.Network)}
		if spec.Address != "" {
			nic["ipv4.address"] = spec.Address
		}
		devices["eth0"] = nic
	}

	source := map[string]string{
		"type": "image", "protocol": "oci",
		"server": b.Config.ociRegistryOrDefault(), "alias": spec.Image,
	}
	release, err := b.admit(ctx, spec.Priority)
	if err != nil {
		return "", err
	}
	defer release()
	// Create (stopped) -- the agent has to be planted before first start.
	done := builder.StartPhase(ctx, "create")
	err = b.Client.createInstance(ctx, name, map[string]interface{}{
		"name": name, "type": "container", "source": source, "config": config, "devices": devices,
	}, instanceOwner{externalName: spec.ExternalName, team: spec.Team})
	done()
	if err != nil {
		return name, fmt.Errorf("creating OCI container %s (image %q from %s): %w", name, spec.Image, b.Config.ociRegistryOrDefault(), err)
	}

	if len(spec.AgentBinary) > 0 {
		done := builder.StartPhase(ctx, "agent_install")
		err := b.installAgentSupervisor(ctx, name, spec)
		done()
		if err != nil {
			return name, err
		}
	}

	done = builder.StartPhase(ctx, "start")
	err = b.startInstance(ctx, name)
	done()
	if err != nil {
		return name, err
	}
	return name, nil
}

// installAgentSupervisor plants the LaForge agent inside a just-created (stopped)
// OCI container and makes it the container's entrypoint, handing it the image's
// real command to supervise via LAFORGE_SUPERVISE. Idempotent: it re-pushes the
// binary but only captures the original command once (guarded by
// LAFORGE_SUPERVISE already being set), so a retried/adopted deploy doesn't
// mistake the agent path for the app command.
func (b *Builder) installAgentSupervisor(ctx context.Context, name string, spec builder.ContainerSpec) error {
	cfg, err := b.instanceConfig(ctx, name)
	if err != nil {
		return fmt.Errorf("reading %s config: %w", name, err)
	}
	if err := b.Client.pushFile(ctx, name, laforgeAgentPath, 0o755, spec.AgentBinary); err != nil {
		return fmt.Errorf("planting agent in %s: %w", name, err)
	}
	if cfg["environment.LAFORGE_SUPERVISE"] != "" {
		return nil // already configured on a previous attempt -- don't re-capture
	}
	// The command the agent supervises: a content Command override if given,
	// else the image's own entrypoint (auto-populated by Incus at create).
	supervise := cfg["oci.entrypoint"]
	if len(spec.Command) > 0 {
		supervise = strings.Join(spec.Command, " ")
	}
	if _, err := b.Client.patch(ctx, "/1.0/instances/"+name, map[string]interface{}{
		"config": map[string]string{
			"environment.LAFORGE_SUPERVISE": supervise,
			"oci.entrypoint":                laforgeAgentPath,
		},
	}); err != nil {
		return fmt.Errorf("configuring agent supervisor on %s: %w", name, err)
	}
	return nil
}

// instanceConfig reads one instance's current config map.
func (b *Builder) instanceConfig(ctx context.Context, name string) (map[string]string, error) {
	raw, err := b.Client.get(ctx, "/1.0/instances/"+name)
	if err != nil {
		return nil, err
	}
	var inst struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &inst); err != nil {
		return nil, fmt.Errorf("decoding instance %s: %w", name, err)
	}
	return inst.Config, nil
}

// teamConfigKey is the Incus instance config key OpenAccess/CloseAccess
// use to find every instance belonging to one team -- set once at
// deploy time, from spec.Team, since nothing about "which team does this
// belong to" is otherwise recoverable from an instance after the fact
// (the short hashed name deliberately carries no semantic information --
// see shortName's own doc comment).
const teamConfigKey = "user.laforge_team"

// deployInstance records each of its phases (builder.StartPhase) -- queued for
// a slot on this host, config drive, create/copy, device replacement, start --
// so the runner can show where a slow deploy's time went. priority orders the
// admission queue (see admission.go).
func (b *Builder) deployInstance(ctx context.Context, externalName, displayName, team, network, networkDisplayName, address, instanceType string, img ImageRef, size SizeSpec, diskGB int, nesting bool, cloudInit string, cloudInitViaISO bool, agentBinary []byte, agentFile string, priority int) (string, error) {
	if err := validateInstanceNameBudget(b.Client.Project); err != nil {
		return "", err
	}
	name := instanceName(displayName, externalName, b.Client.Project)

	release, err := b.admit(ctx, priority)
	if err != nil {
		return "", err
	}
	defer release()

	config := map[string]string{teamConfigKey: team, externalNameKey: externalName}
	if nesting {
		// A LaForge Docker container runs Docker INSIDE this LXD system
		// container, which needs nested-container support enabled.
		config["security.nesting"] = "true"
	}
	if instanceType == "virtual-machine" {
		// Lab VMs boot from images whose bootloader shim isn't signed for
		// Incus's Secure Boot (Ubuntu cloud images loop on "prohibited by
		// secure boot policy" and never reach cloud-init) -- found live.
		// These are throwaway competition hosts, so Secure Boot buys
		// nothing here; turning it off lets any image boot.
		config["security.secureboot"] = "false"
	}
	if cloudInit != "" && !cloudInitViaISO {
		// Linux: cloud-init reads user-data from the /dev/lxd/sock
		// datasource, so setting the config key is enough.
		config["cloud-init.user-data"] = cloudInit
	}
	if size.CPU != "" {
		config["limits.cpu"] = size.CPU
	}
	if size.Memory != "" {
		config["limits.memory"] = size.Memory
	}

	devices := map[string]interface{}{}
	// Attach to the team's own network rather than falling back to
	// whatever the "default" profile provides -- found missing while
	// proving a real two-team build end to end: without this, every
	// instance landed on the hoster's shared default bridge regardless of
	// which team's network DeployNetwork had created for it, so there was
	// no actual isolation between teams despite each having its own
	// network (see builder.HostSpec.Network's doc comment). networkName
	// is the same deterministic mapping DeployNetwork used for this same
	// network (identical DisplayName + ExternalName inputs) -- if that
	// network's own deploy task hasn't run yet, this
	// legitimately fails with a real "network not found" error, which the
	// existing task retry/backoff resolves on its own once it has.
	if network != "" {
		nic := map[string]string{"type": "nic", "network": networkName(networkDisplayName, network)}
		// address is content's own last_octet address for this copy
		// (builder.HostSpec.Address's doc comment) -- an OVN NIC's own
		// ipv4.address key is documented to statically assign it via
		// OVN's per-network DHCP, rather than this builder needing to
		// configure the guest OS directly. Not live-verified (OVN isn't
		// available in this session's environment -- see Builder's own
		// doc comment); if address is empty (a builder config still
		// using a plain bridge type, or a spec that genuinely has none),
		// this is simply omitted and Incus's own default applies, same
		// as before this field existed.
		if address != "" {
			nic["ipv4.address"] = address
		}
		devices["eth0"] = nic
	}
	if diskGB > 0 {
		// "type" is required on every Incus device, root disks included --
		// found live: omitting it (an earlier version of this code) is
		// accepted by JSON marshaling (Go doesn't know it's required) but
		// rejected by the daemon at instance-create time with "Invalid
		// device type," surfaced only once a real test exercised a
		// nonzero DiskGB (the package's
		// own tests never had, since neither TestContainerDeployAdoptDestroy
		// nor TestDeployHostVMSurfacesRealKVMError sets a disk size).
		devices["root"] = map[string]string{"type": "disk", "path": "/", "pool": b.Config.storagePoolOrDefault(), "size": fmt.Sprintf("%dGB", diskGB)}
	}

	if cloudInit != "" && cloudInitViaISO {
		// The user-data (and, when embedding, the agent binary itself) is
		// delivered as a NoCloud config-drive ISO attached to the instance
		// (imported as a custom volume first -- see configdrive.go). Windows
		// always takes this path (cloudbase-init can't read the LXD socket);
		// Linux takes it when the agent binary is embedded.
		done := builder.StartPhase(ctx, "config_drive")
		iso, err := buildCidataISO(name, cloudInit, agentBinary, agentFile)
		if err != nil {
			done()
			return name, fmt.Errorf("building config drive for %s: %w", name, err)
		}
		volName := cidataVolName(name)
		err = b.importConfigDrive(ctx, name, iso)
		done()
		if err != nil {
			return name, err
		}
		devices["cidata"] = map[string]string{"type": "disk", "pool": b.Config.storagePoolOrDefault(), "source": volName}
	}

	if _, ok := devices["root"]; !ok && img.IsSnapshot() {
		// A copy's disk lands on whatever pool its root device names; without
		// one it would follow the template's, and then disagree with the
		// default profile once the template's devices are replaced. Naming the
		// builder's pool also keeps the copy a fast clone when the template
		// lives there.
		devices["root"] = map[string]string{"type": "disk", "path": "/", "pool": b.Config.storagePoolOrDefault()}
	}
	var source map[string]interface{}
	switch {
	case img.IsSnapshot():
		// A copy of a template's snapshot. instance_only: the template's other
		// snapshots aren't wanted on every team's copy.
		source = map[string]interface{}{"type": "copy", "source": img.Instance + "/" + img.Snapshot, "instance_only": true}
		if img.SourceProject != "" {
			source["project"] = img.SourceProject
		}
	case img.Fingerprint != "":
		source = map[string]interface{}{"type": "image", "fingerprint": img.Fingerprint}
	default:
		source = map[string]interface{}{"type": "image", "alias": img.Alias, "server": img.Server, "protocol": img.Protocol}
	}
	// "copy" for a snapshot clone, "create" for an image unpack -- named apart
	// because they cost very differently on a non-CoW storage pool.
	createPhase := "create"
	if img.IsSnapshot() {
		createPhase = "copy"
	}
	done := builder.StartPhase(ctx, createPhase)
	err = b.Client.createInstance(ctx, name, map[string]interface{}{
		"name":    name,
		"type":    instanceType,
		"source":  source,
		"config":  config,
		"devices": devices,
	}, instanceOwner{externalName: externalName, team: team})
	done()
	if err != nil {
		// A copy whose requested root disk is smaller than the source snapshot's
		// block volume fails here with "Block volumes cannot be shrunk" -- a
		// configuration mismatch retrying can't fix (preflight catches it when the
		// source size is readable; this guards the sources whose size it couldn't
		// determine). Mark it permanent so the runner stops retrying.
		if diskGB > 0 && img.IsSnapshot() && isShrinkError(err) {
			return name, fmt.Errorf("%w: host disk %dGB is smaller than the %s source it copies from, whose block volume cannot be shrunk; raise this host's disk: to at least the source's size: %w",
				builder.ErrPermanentConfig, diskGB, externalName, err)
		}
		// Keep the deterministic ref for cleanup when creation is uncertain.
		return name, fmt.Errorf("creating instance %s (from %s): %w", name, externalName, err)
	}

	if img.IsSnapshot() {
		done := builder.StartPhase(ctx, "configure_devices")
		err := b.replaceCopiedDevices(ctx, name, devices)
		done()
		if err != nil {
			return name, err
		}
	}

	done = builder.StartPhase(ctx, "start")
	err = b.startInstance(ctx, name)
	done()
	if err != nil {
		return name, err
	}
	return name, nil
}

// startInstance issues the start call, retrying a bounded number of times
// on failure. A live daemon under deeply nested cgroup delegation (Incus
// running inside Docker inside Docker Desktop's own VM) was
// observed failing this exact call with a
// transient EBUSY race in LXC's own cgroup controller delegation during
// setup -- confirmed by hand against a live daemon: the *same* start
// call, simply repeated, reliably succeeds within a couple of attempts
// (a bare status check right after the first failure isn't enough --
// the instance was confirmed still genuinely Stopped at that point, not
// just slow to report; a fresh start call is what actually resolves it).
// Trusting the first failure here would make DeployHost/DeployContainer
// wrongly non-idempotent in that environment: a caller's own retry would
// hit "already exists" on create and then fail again on this same racy
// start, forever. "Ensure" means converging on the desired state, so a
// handful of retries against a real, observed transient condition is the
// correct, honest fix -- not a blanket "ignore all start errors."
func (b *Builder) startInstance(ctx context.Context, name string) error {
	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := b.Client.put(ctx, "/1.0/instances/"+name+"/state", map[string]interface{}{
			"action": "start", "timeout": 60, "force": false,
		})
		if err == nil {
			return nil
		}
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyRunning() {
			return nil
		}
		lastErr = err
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return fmt.Errorf("starting instance %s (after %d attempts): %w", name, maxAttempts, lastErr)
}

// ConsoleLog reads the instance's console log through Incus's
// GET /1.0/instances/<name>/console (a raw body, not the JSON envelope).
func (b *Builder) ConsoleLog(ctx context.Context, team, externalRef string, maxBytes int) (string, error) {
	data, err := b.Client.getRaw(ctx, "/1.0/instances/"+externalRef+"/console")
	if err != nil {
		return "", fmt.Errorf("reading console log of %s: %w", externalRef, err)
	}
	return builder.TailBytes(data, maxBytes), nil
}

// PowerAction maps LaForge's start/stop/reboot onto Incus's own instance
// state actions (start / stop / restart). force=true is a hard power
// operation (Incus's own `force`, an immediate kill/reset); force=false is
// graceful -- Incus sends the guest a shutdown/reboot signal and waits up
// to `timeout` for it to comply, which is the OS-level variant. A graceful
// stop/reboot on an unresponsive guest genuinely times out; that's the
// real signal the caller wanted force, surfaced rather than hidden.
func (b *Builder) PowerAction(ctx context.Context, team, externalRef, action string, force bool) error {
	incusAction := map[string]string{
		builder.PowerStart:  "start",
		builder.PowerStop:   "stop",
		builder.PowerReboot: "restart",
	}[action]
	if incusAction == "" {
		return fmt.Errorf("unsupported power action %q", action)
	}
	body := map[string]interface{}{"action": incusAction, "force": force}
	if !force && incusAction != "start" {
		// Give a graceful shutdown/reboot a real window to complete before
		// Incus gives up; a hard action needs none.
		// Windows guests regularly need well over 30 seconds to shut down cleanly.
		body["timeout"] = 120
	}
	_, err := b.Client.put(ctx, "/1.0/instances/"+externalRef+"/state", body)
	if err != nil {
		var apiErr *APIError
		// Starting something already running, or stopping something already
		// stopped, is the desired end state -- treat it as success rather
		// than surfacing a confusing error for a no-op.
		if errors.As(err, &apiErr) && (apiErr.AlreadyRunning() || apiErr.NotFound()) && incusAction != "restart" {
			return nil
		}
		return fmt.Errorf("%s instance %s: %w", action, externalRef, err)
	}
	return nil
}

func (b *Builder) DestroyHost(ctx context.Context, team, externalRef string) error {
	return b.destroyInstance(ctx, externalRef)
}

func (b *Builder) DestroyContainer(ctx context.Context, team, externalRef string) error {
	return b.destroyInstance(ctx, externalRef)
}

func (b *Builder) destroyInstance(ctx context.Context, name string) error {
	started := time.Now()
	slog.InfoContext(ctx, "Incus destroying instance: waiting for any in-flight creation", "project", b.Client.Project, "instance", name)
	if err := b.Client.waitForInstanceCreation(ctx, name); err != nil {
		slog.ErrorContext(ctx, "Incus destroy: creation wait failed", "instance", name, "error", err)
		return err
	}
	slog.InfoContext(ctx, "Incus destroy: stopping the instance (force, 30s)", "instance", name)
	_, err := b.Client.put(ctx, "/1.0/instances/"+name+"/state", map[string]interface{}{
		"action": "stop", "timeout": 30, "force": true,
	})
	if err != nil {
		var apiErr *APIError
		alreadyStopped := errors.As(err, &apiErr) && (apiErr.NotFound() ||
			strings.Contains(apiErr.Message, "already stopped") || strings.Contains(apiErr.Message, "not running"))
		if !alreadyStopped {
			slog.ErrorContext(ctx, "Incus destroy: stop failed", "instance", name, "elapsed", time.Since(started).Round(time.Millisecond), "error", err)
			return fmt.Errorf("stopping instance %s: %w", name, err)
		}
		slog.InfoContext(ctx, "Incus destroy: instance already stopped or gone", "instance", name, "detail", err.Error())
	}
	slog.InfoContext(ctx, "Incus destroy: deleting the instance", "instance", name)
	_, err = b.Client.delete(ctx, "/1.0/instances/"+name)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			slog.InfoContext(ctx, "Incus destroy: instance was already gone", "instance", name)
			// Still clean up the config-drive volume below.
		} else {
			slog.ErrorContext(ctx, "Incus destroy: delete failed", "instance", name, "elapsed", time.Since(started).Round(time.Millisecond), "error", err)
			return fmt.Errorf("deleting instance %s: %w", name, err)
		}
	} else {
		slog.InfoContext(ctx, "Incus destroy: instance deleted; removing leftovers (config drive, host ACL)", "instance", name, "elapsed", time.Since(started).Round(time.Millisecond))
	}
	// Remove the config-drive volume, if this instance had one (Windows).
	// A missing volume is a no-op; other errors must remain visible so
	// deployment retries cannot import over a drive that still exists.
	if err := b.Client.deleteVolume(ctx, b.Config.storagePoolOrDefault(), cidataVolName(name)); err != nil {
		return fmt.Errorf("deleting config drive for %s: %w", name, err)
	}
	// Remove this instance's per-host port-firewall ACL now that the instance
	// (its only user) is gone -- otherwise it lingers orphaned. Best-effort.
	_, _ = b.Client.delete(ctx, "/1.0/network-acls/"+hostACLName(name))
	_, _ = b.Client.delete(ctx, "/1.0/network-acls/"+publicACLName(name))
	return nil
}

type incusInstance struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Status string            `json:"status"` // Incus's live state: Running, Stopped, Frozen, ...
	Config map[string]string `json:"config"`
}

// powerStateFromIncus maps Incus's own instance status onto the normalized
// builder.PowerState* vocabulary.
func powerStateFromIncus(status string) string {
	switch status {
	case "Running":
		return builder.PowerStateRunning
	case "Stopped":
		return builder.PowerStateStopped
	default:
		return builder.PowerStateOther
	}
}

type incusNetwork struct {
	Name    string `json:"name"`
	Managed bool   `json:"managed"`
}

// InstanceState reads one instance's power state directly (builder.InstanceStater),
// for confirming an instance the full listing left out.
func (b *Builder) InstanceState(ctx context.Context, externalRef string) (string, bool, error) {
	raw, err := b.Client.get(ctx, "/1.0/instances/"+url.PathEscape(externalRef))
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			return "", false, nil
		}
		return "", false, err
	}
	var inst incusInstance
	if err := json.Unmarshal(raw, &inst); err != nil {
		return "", false, fmt.Errorf("decoding instance %s: %w", externalRef, err)
	}
	return powerStateFromIncus(inst.Status), true, nil
}

func (b *Builder) Inspect(ctx context.Context) ([]builder.Resource, error) {
	instances, err := b.ourInstances(ctx)
	if err != nil {
		return nil, err
	}
	var out []builder.Resource
	for _, inst := range instances {
		// Incus only distinguishes container/virtual-machine, not
		// LaForge's host/container -- and a Linux Host deploys as an
		// Incus container too (see DeployHost), so this can't always
		// tell the two apart precisely. Not load-bearing right now:
		// nothing consumes Inspect's Kind field for anything other than
		// "does a resource with this ref still exist," which needs no
		// kind at all.
		kind := "container"
		if inst.Type == "virtual-machine" {
			kind = "host"
		}
		out = append(out, builder.Resource{ExternalRef: inst.Name, Kind: kind, State: powerStateFromIncus(inst.Status)})
	}

	// Names only: with hundreds of networks, a recursive listing can outlast
	// the request (see listNames), and only the names are needed here.
	netNames, err := b.Client.listNames(ctx, "/1.0/networks")
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	for _, n := range netNames {
		if !strings.HasPrefix(n, "lf-") {
			continue // the project's own pre-existing networks, not ours
		}
		out = append(out, builder.Resource{ExternalRef: n, Kind: "network"})
	}
	return out, nil
}

// OpenAccess and CloseAccess (team access open/close) live in access.go. They
// no longer detach eth0 -- see access.go's own doc comment for why the NIC must
// stay attached (NAT + agent telemetry) and how close is done with ACLs instead.

// instancePut is the full body Incus's PUT .../instances/<name> requires --
// verified live: PUT replaces the whole resource, so omitting a field resets it.
// PATCH, by contrast, merges config and devices key by key. getInstancePut reads
// it (plus the in-effect expanded_devices) for callers that need a host's real
// NIC/network config, e.g. teamNetworks and HasWorkingEth0.
type instancePut struct {
	Architecture string                     `json:"architecture"`
	Config       map[string]string          `json:"config"`
	Devices      map[string]json.RawMessage `json:"devices"`
	Ephemeral    bool                       `json:"ephemeral"`
	Profiles     []string                   `json:"profiles"`
	Stateful     bool                       `json:"stateful"`
	Description  string                     `json:"description"`
}

func (b *Builder) getInstancePut(ctx context.Context, name string) (instancePut, map[string]interface{}, error) {
	raw, err := b.Client.get(ctx, "/1.0/instances/"+name)
	if err != nil {
		return instancePut{}, nil, fmt.Errorf("reading instance: %w", err)
	}
	var full instancePut
	if err := json.Unmarshal(raw, &full); err != nil {
		return instancePut{}, nil, fmt.Errorf("decoding instance: %w", err)
	}
	var expanded struct {
		ExpandedDevices map[string]interface{} `json:"expanded_devices"`
	}
	if err := json.Unmarshal(raw, &expanded); err != nil {
		return instancePut{}, nil, fmt.Errorf("decoding instance: %w", err)
	}
	return full, expanded.ExpandedDevices, nil
}

// retryOnBusy retries fn a bounded number of times against a real,
// transient "this exact instance still has another operation in
// flight" race (see APIError.Busy's own doc comment) -- the same
// bounded-retry-with-backoff idiom startInstance already uses for its
// own AlreadyRunning tolerance, generalized here since access/NIC config
// updates hit the identical class of race (found live: an open_access
// immediately following a close_access on the same instance).
func retryOnBusy(ctx context.Context, fn func() error) error {
	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		var apiErr *APIError
		if !(errors.As(err, &apiErr) && apiErr.Busy()) {
			return err
		}
		lastErr = err
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return lastErr
}

// HasWorkingEth0 reports whether name's own eth0 (as Incus's
// expanded_devices actually sees it -- the same effective view
// removeNIC/restoreNIC read) is a real network device, not "none" --
// exactly what CloseAccess/OpenAccess toggle. Exported so a real,
// external caller (an operator health check, or a live test verifying
// close/open access against a real cluster) can confirm the effect
// directly through the API this builder already talks to, instead of
// shelling out to a local `incus` CLI the way this package's own live
// tests once had to before this existed.
func (b *Builder) HasWorkingEth0(ctx context.Context, name string) (bool, error) {
	_, expandedDevices, err := b.getInstancePut(ctx, name)
	if err != nil {
		return false, err
	}
	dev, ok := expandedDevices["eth0"]
	if !ok {
		return false, nil
	}
	devMap, ok := dev.(map[string]interface{})
	if !ok {
		return false, nil
	}
	return devMap["type"] != "none", nil
}

// replaceCopiedDevices makes a copied instance's devices exactly LaForge's own
// -- the team NIC, the root disk, a config drive -- and its profiles just
// "default", as an instance created from an image has. A copy otherwise keeps
// the template's own devices and profiles: a second NIC on some management
// network, a data volume, a GPU, all shared by every team's copy. Done before
// the first start, so a copy never boots with them.
func (b *Builder) replaceCopiedDevices(ctx context.Context, name string, devices map[string]interface{}) error {
	return retryOnBusy(ctx, func() error {
		full, _, err := b.getInstancePut(ctx, name)
		if err != nil {
			return err
		}
		full.Devices = map[string]json.RawMessage{}
		for k, d := range devices {
			raw, err := json.Marshal(d)
			if err != nil {
				return err
			}
			full.Devices[k] = raw
		}
		full.Profiles = []string{"default"}
		if _, err := b.Client.put(ctx, "/1.0/instances/"+name, full); err != nil {
			return fmt.Errorf("replacing the devices %s copied from its template: %w", name, err)
		}
		return nil
	})
}

// ourInstances is LaForge's own instances in this builder's project -- see
// Client.laforgeInstances for why they're read one by one.
func (b *Builder) ourInstances(ctx context.Context) ([]incusInstance, error) {
	raws, err := b.Client.laforgeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	out := make([]incusInstance, 0, len(raws))
	for _, raw := range raws {
		var inst incusInstance
		if err := json.Unmarshal(raw, &inst); err != nil {
			return nil, fmt.Errorf("decoding instance: %w", err)
		}
		out = append(out, inst)
	}
	return out, nil
}
