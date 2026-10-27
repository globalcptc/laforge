package microcloud

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
	"sync"
	"time"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/render"
)

// agentUserData renders the agent-install user-data for an instance from the
// neutral download URL, choosing the format this image consumes: cloud-init for
// Linux, cloudbase-init for Windows. viaISO is true for Windows, where
// cloudbase-init can't read the lxd socket so the user-data is delivered as a
// NoCloud config-drive ISO. An empty url (agent delivery off) yields nothing.
func agentUserData(url, os, tempPass string) (userData string, viaISO bool) {
	if url == "" {
		return "", false
	}
	if agentdelivery.PlatformFor(os) == agentdelivery.Windows {
		return agentdelivery.WindowsCloudbaseUserData(url, tempPass), true
	}
	return agentdelivery.LinuxUserData(url, tempPass), false
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
	Client          *Client
	Config          Config
	PublicAddresses PublicAddressAllocator
	retryWait       func(context.Context, time.Duration) error
}

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

// LXD socket-path budget. When LXD starts a VM it opens a Unix control socket at
//
//	<lxdLogsDir><project>_<instance>/qemu.monitor
//
// and Go's Linux Unix-socket address buffer (sun_path) is 108 bytes INCLUDING a
// trailing NUL, so that whole path must be <= 107 bytes or the connect fails with
// a bare "invalid argument" (EINVAL) -- long after the instance name passed the
// 63-char instance limit. MicroCloud is always the LXD snap, so its runtime logs
// live under the snap's common dir. We budget the generated instance name against
// this so a long project + long placement name can't produce a VM that passes
// creation but can never start. See issue #63.
const (
	lxdSnapLogsDir    = "/var/snap/lxd/common/lxd/logs/"
	qemuMonitorSuffix = "/qemu.monitor"
	unixSocketPathMax = 107             // 108-byte sun_path minus the trailing NUL
	minInstanceName   = len("lf-") + 12 // shortName is always "lf-"+12 hex = 15 bytes
)

// maxInstanceNameLen is the longest instance name that keeps the qemu-monitor
// socket path within the Unix-socket limit for this project, also honoring the
// 63-char cap the derived cidata volume ("<name>-cidata") needs. Can go negative
// for an absurdly long project; validateInstanceNameBudget catches that.
func maxInstanceNameLen(project string) int {
	socketCap := unixSocketPathMax - len(lxdSnapLogsDir) - len(project) - len("_") - len(qemuMonitorSuffix)
	cidataCap := 63 - len("-cidata")
	if socketCap < cidataCap {
		return socketCap
	}
	return cidataCap
}

// validateInstanceNameBudget rejects, before any resource is created, a project
// so long that not even the minimal hashed instance name fits the socket path --
// a permanent configuration error (retrying can't shorten the project), reported
// with the computed length and the limit so the operator can act.
func validateInstanceNameBudget(project string) error {
	if maxInstanceNameLen(project) < minInstanceName {
		minPath := len(lxdSnapLogsDir) + len(project) + len("_") + minInstanceName + len(qemuMonitorSuffix)
		return fmt.Errorf("%w: LXD project %q (%d chars) leaves no room for a valid instance name: even the shortest %d-char name yields a %d-byte VM socket path, over the %d-byte limit. Use a shorter project name.",
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
// shortName when there's no display hint at all. Capped to fit BOTH the
// cidata volume's 63-char limit AND the full qemu-monitor socket path for
// project (see maxInstanceNameLen), so the generated name can never be a VM
// that creates fine but can't start.
func instanceName(displayName, externalName, project string) string {
	base := sanitizeName(displayName)
	if base == "" {
		return shortName(externalName)
	}
	sum := sha256.Sum256([]byte(externalName))
	suffix := hex.EncodeToString(sum[:])[:6]
	maxBase := maxInstanceNameLen(project) - len("-") - len(suffix)
	if maxBase < len("lf-") {
		// No room for a readable base once the project eats the budget; the pure
		// hash (15 bytes) is the shortest deterministic name. A project too long
		// for even that is rejected up front by validateInstanceNameBudget.
		return shortName(externalName)
	}
	if len(base) > maxBase {
		base = strings.TrimRight(base[:maxBase], "-")
	}
	return base + "-" + suffix
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

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	if b.Config.OVNUplinkNetwork == "" {
		return "", fmt.Errorf("this builder config has no OVNUplinkNetwork set -- every DeployNetwork call needs an uplink to route OVN logical networks through")
	}
	name := networkName(spec.DisplayName, spec.ExternalName)
	slog.InfoContext(ctx, "MicroCloud ensuring primary OVN network", "project", b.Client.Project, "network", name, "uplink", b.Config.OVNUplinkNetwork, "cidr", spec.CIDR)
	bridgeAddr, err := bridgeAddressCIDR(spec.CIDR)
	if err != nil {
		return "", fmt.Errorf("computing gateway address for %s: %w", spec.CIDR, err)
	}
	ctx, release, err := b.Client.lockNetworking(ctx)
	if err != nil {
		return name, err
	}
	defer release()
	// Adopt-if-present, by an explicit existence check rather than relying on the
	// create call's error shape. On a CLUSTERED LXD (a real MicroCloud), re-POSTing
	// an OVN network that already exists does NOT return a clean 409 -- it returns
	// a 500 "Network is not in pending state" from the cluster's two-phase network
	// path, which AlreadyExists() can't recognize. Found live against the 4-node
	// cluster: without this pre-check the second (retried) DeployNetwork of an
	// existing network errored instead of adopting, breaking the "ensure" contract.
	if exists, err := b.networkExists(ctx, name, bridgeAddr); err != nil {
		return name, fmt.Errorf("checking whether network %s exists: %w", name, err)
	} else if exists {
		return name, nil // adopt
	}
	_, err = b.Client.post(ctx, "/1.0/networks", map[string]interface{}{
		"name": name,
		"type": "ovn",
		"config": map[string]string{
			"network":      b.Config.OVNUplinkNetwork,
			"ipv4.address": bridgeAddr,
			"ipv4.nat":     "true",
			// IPv6 off -- LaForge is IPv4-only; see the incus builder's
			// DeployNetwork for why leaving OVN to auto-assign IPv6 breaks a later
			// ACL attach when the uplink carries no IPv6 range.
			"ipv6.address": "none",
		},
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			if exists, readErr := b.networkExists(ctx, name, bridgeAddr); readErr != nil {
				return name, fmt.Errorf("validating existing network %s: %w", name, readErr)
			} else if exists {
				return name, nil
			}
		}
		slog.ErrorContext(ctx, "MicroCloud primary network creation failed", "project", b.Client.Project, "network", name, "error", err)
		return name, fmt.Errorf("creating network %s (from %s): %w", name, spec.ExternalName, err)
	}
	if exists, err := b.networkExists(ctx, name, bridgeAddr); err != nil {
		return name, fmt.Errorf("validating created network %s: %w", name, err)
	} else if !exists {
		return name, fmt.Errorf("created network %s is missing on read-back", name)
	}
	slog.InfoContext(ctx, "MicroCloud primary OVN network created", "project", b.Client.Project, "network", name)
	return name, nil
}

// networkExists reports whether a managed network of this exact name is already
// present and has a valid, unique uplink allocation (404 -> false). Both new
// networks and adoption paths must pass the same check before a network task
// can finish and release dependent guests. Call with the networking lock held.
func (b *Builder) networkExists(ctx context.Context, name, gateway string) (bool, error) {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+name)
	if err == nil {
		var state allocatedNetwork
		if err := json.Unmarshal(raw, &state); err != nil {
			return false, err
		}
		if err := b.validateNetworkAllocation(ctx, name, gateway, state); err != nil {
			return false, err
		}
		return true, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus == 404 {
		return false, nil
	}
	return false, err
}

// bridgeAddressCIDR turns a content network's own CIDR (e.g.
// "10.0.1.0/24" -- the network address itself) into the address the OVN network's own logical router/gateway
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
	ctx, release, err := b.Client.lockNetworking(ctx)
	if err != nil {
		return err
	}
	defer release()
	// Tear down the visible_from fabric (peers + ACL) first -- a peered OVN
	// network can't be deleted, nor an attached ACL. Best-effort. See
	// netaccess.go and the incus builder's identical teardown.
	b.teardownNetworkAccess(ctx, externalRef)
	_, err = b.Client.delete(ctx, "/1.0/networks/"+externalRef)
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
	win := agentdelivery.PlatformFor(spec.OS) == agentdelivery.Windows
	cloudInit, _ := agentUserData(spec.AgentDownloadURL, spec.OS, spec.TempAdminPassword)
	// Embed-first: write the patched agent onto the custom config-drive ISO so
	// the user-data copies it locally (download URL stays as fallback). Windows
	// always takes the ISO path; Linux switches to it only when embedding. (The
	// public-NIC path uses LXD's own config drive, which we don't own, so those
	// hosts stay download-only -- see DeployHostWithPublicPorts.)
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
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, instanceType, img, size, spec.DiskGB, false, cloudInit, viaISO, agentBinary, agentFile)
}

// DeployContainer runs a LaForge `container:` as a nested Docker container. LXD
// has no native OCI support, so this deploys a thin, nesting-enabled LXD system
// container from this builder's docker-ready base image (the "Docker host",
// built by the image-build job) and then `docker run`s spec.Image inside it.
// spec.Image is the OCI ref, NOT an entry in the builder's LXD image map -- the
// base image is always the docker base, so authors never pick the runtime.
//
// The Docker host itself runs NO LaForge agent: it is pure infrastructure, as
// invisible to content as a native-OCI host's own OS. The agent that checks in
// and runs the container's steps/validators is planted INSIDE the nested
// application container (runNestedContainer makes it the container's entrypoint
// and has it supervise the image's real command). So a `container:` runs its
// steps from inside the app on MicroCloud exactly as it does on a native-OCI
// builder -- the Docker runtime never leaks into content.
func (b *Builder) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	// A compose container is a Docker-capable machine the agent starts the
	// project on -- deployed, and later destroyed, like any other instance.
	if spec.ComposeHost {
		return b.deployComposeHost(ctx, spec)
	}
	if b.Config.DockerBaseFingerprint == "" {
		return "", fmt.Errorf("this builder has no docker base image yet -- build it first (Infrastructure → Base image)")
	}
	size, ok := b.Config.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no size configured for %q", spec.Size)
	}
	base := ImageRef{Fingerprint: b.Config.DockerBaseFingerprint}
	// No cloud-init on the Docker host: it needs no agent of its own. The docker
	// base image already brings dockerd up on boot.
	name, err := b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, "container", base, size, 0, true, "", false, nil, "")
	if err != nil {
		return name, err
	}
	if err := b.runNestedContainer(ctx, name, spec); err != nil {
		return name, err
	}
	return name, nil
}

// deployComposeHost deploys the machine a compose container's project runs on:
// a nesting system container booted from this builder's docker base image
// (Docker and the compose plugin already installed), with the agent delivered
// by cloud-init like a host's. A builder image named builder.ComposeHostImage
// overrides that -- an operator's own image, VM or container, deployed exactly
// as a host would be.
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
	// delivered by cloud-init (never the Windows ISO path).
	cloudInit, _ := agentUserData(spec.AgentDownloadURL, "linux", "")
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, "container", ImageRef{Fingerprint: b.Config.DockerBaseFingerprint}, size, spec.DiskGB, true, cloudInit, false, nil, "")
}

// teamConfigKey is the Incus instance config key OpenAccess/CloseAccess
// use to find every instance belonging to one team -- set once at
// deploy time, from spec.Team, since nothing about "which team does this
// belong to" is otherwise recoverable from an instance after the fact
// (the short hashed name deliberately carries no semantic information --
// see shortName's own doc comment).
const teamConfigKey = "user.laforge_team"

func (b *Builder) deployInstance(ctx context.Context, externalName, displayName, team, network, networkDisplayName, address, instanceType string, img ImageRef, size SizeSpec, diskGB int, nesting bool, cloudInit string, cloudInitViaISO bool, agentBinary []byte, agentFile string, publicPlans ...*publicNICPlan) (string, error) {
	if err := validateInstanceNameBudget(b.Client.Project); err != nil {
		return "", err
	}
	name := instanceName(displayName, externalName, b.Client.Project)
	slog.InfoContext(ctx, "MicroCloud deploying instance", "project", b.Client.Project, "instance", name, "external_name", externalName, "team", team, "instance_type", instanceType, "snapshot", img.IsSnapshot())

	var public *publicNICPlan
	if len(publicPlans) > 0 {
		public = publicPlans[0]
	}
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
	if cloudInit != "" && (!cloudInitViaISO || public != nil) {
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
		if public != nil {
			nic["name"] = "eth0"
			nic["hwaddr"] = public.PrimaryMAC
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
	} else if b.Client.Project != "" || public != nil {
		// A project other than `default` often has its own, empty default
		// profile (features.profiles), so it can't be relied on for a root
		// disk. In `default` the profile's root disk is left alone, as before.
		devices["root"] = map[string]string{"type": "disk", "path": "/", "pool": b.Config.storagePoolOrDefault()}
	}

	if public != nil {
		devices[publicNICName] = public.Device
		raw, err := json.Marshal(public)
		if err != nil {
			return name, err
		}
		config[publicNICKey] = string(raw)
		config["cloud-init.network-config"] = public.NetworkConfig
		config["user.network-config"] = public.NetworkConfig
		// LXD 5.21 supplies a native NoCloud config drive. It includes the
		// network-config file and works even without agent user-data.
		if cloudInitViaISO {
			devices["cidata"] = map[string]string{"type": "disk", "source": "cloud-init:config"}
		}
	}

	if cloudInit != "" && cloudInitViaISO && public == nil {
		// Windows (and embedding Linux): the user-data, and the agent binary
		// itself when embedded, ride a NoCloud config-drive ISO attached to the
		// instance (imported as a custom volume first -- see configdrive.go).
		iso, err := buildCidataISO(name, cloudInit, agentBinary, agentFile)
		if err != nil {
			return name, fmt.Errorf("building config drive for %s: %w", name, err)
		}
		volName := cidataVolName(name)
		if err := b.Client.importISOVolume(ctx, b.Config.storagePoolOrDefault(), volName, iso); err != nil {
			// A config-drive volume left behind by an earlier attempt that
			// timed out mid-create is still attached to that partial
			// instance, so importISOVolume's own delete-then-create couldn't
			// replace it ("volume already exists"). The drive carries a
			// fresh one-time agent token every attempt, so it must be
			// REPLACED, never adopted -- tear the partial instance and its
			// volume down (destroyInstance removes both), then re-import.
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
				_ = b.destroyInstance(ctx, name)
				if err := b.Client.importISOVolume(ctx, b.Config.storagePoolOrDefault(), volName, iso); err != nil {
					return name, fmt.Errorf("importing config drive for %s (after clearing a stale one): %w", name, err)
				}
			} else {
				return name, fmt.Errorf("importing config drive for %s: %w", name, err)
			}
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
	request := map[string]interface{}{
		"name":    name,
		"type":    instanceType,
		"source":  source,
		"config":  config,
		"devices": devices,
	}
	if public != nil {
		// Both NICs and the root disk are explicit. Do not inherit an extra
		// management/public NIC or network metadata from a shared profile.
		request["profiles"] = []string{}
	}
	if img.IsSnapshot() {
		// LXD merges copy-time devices with the snapshot's devices and
		// validates that merged set before it creates the destination. Mask
		// obsolete NICs/disks now: replacing devices after the copy is too
		// late when a template refers to a network or volume that is gone.
		if err := b.retryInstanceStep(ctx, name, "reading snapshot devices", func() error {
			createDevices, err := b.snapshotCreateDevices(ctx, img, devices)
			if err == nil {
				request["devices"] = createDevices
			}
			return err
		}); err != nil {
			return name, err
		}
	}
	if err := b.createInstance(ctx, name, request, b.clusterTarget(ctx), instanceOwner{externalName: externalName, team: team}); err != nil {
		// A copy whose requested root disk is smaller than the source snapshot's
		// block volume fails here with "Block volumes cannot be shrunk" -- a
		// configuration mismatch retrying can't fix (and preflight catches when
		// the source size is readable; this guards the sources whose size it
		// couldn't determine). Mark it permanent so the runner stops retrying,
		// and point at the fix.
		if diskGB > 0 && img.IsSnapshot() && isShrinkError(err) {
			return name, fmt.Errorf("%w: host disk %dGB is smaller than the %s source it copies from, whose block volume cannot be shrunk; raise this host's disk: to at least the source's size: %w",
				builder.ErrPermanentConfig, diskGB, externalName, err)
		}
		// Keep the deterministic ref for cleanup even if creation is uncertain.
		return name, fmt.Errorf("creating instance from %s: %w", externalName, err)
	}

	if img.IsSnapshot() {
		if err := b.retryInstanceStep(ctx, name, "configuring snapshot devices", func() error { return b.replaceCopiedDevices(ctx, name, devices) }); err != nil {
			return name, err
		}
	}

	if public != nil {
		if err := b.retryInstanceStep(ctx, name, "recording public NIC readiness", func() error {
			_, err := b.Client.patch(ctx, "/1.0/instances/"+name, map[string]interface{}{"config": map[string]string{publicReadyKey: "true"}})
			return err
		}); err != nil {
			return name, err
		}
	}
	if err := b.startInstance(ctx, name); err != nil {
		return name, err
	}
	return name, nil
}

// ConsoleLog reads the instance's console log through LXD's
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
	lxdAction := map[string]string{
		builder.PowerStart:  "start",
		builder.PowerStop:   "stop",
		builder.PowerReboot: "restart",
	}[action]
	if lxdAction == "" {
		return fmt.Errorf("unsupported power action %q", action)
	}
	body := map[string]interface{}{"action": lxdAction, "force": force}
	if !force && lxdAction != "start" {
		// Give a graceful shutdown/reboot a real window to complete before
		// Incus gives up; a hard action needs none.
		body["timeout"] = 30
	}
	_, err := b.Client.put(ctx, "/1.0/instances/"+externalRef+"/state", body)
	if err != nil {
		var apiErr *APIError
		// Starting something already running, or stopping something already
		// stopped, is the desired end state -- treat it as success rather
		// than surfacing a confusing error for a no-op.
		if errors.As(err, &apiErr) && (apiErr.AlreadyRunning() || apiErr.NotFound()) && lxdAction != "restart" {
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

// DestroyHostForRebuild keeps the public reservation while replacing a box.
func (b *Builder) DestroyHostForRebuild(ctx context.Context, team, externalRef string) error {
	return b.destroyInstance(ctx, externalRef, destroyOptions{keepPublicAddress: true})
}

// The runner knows the full deployment identity, unlike a shortened LXD name.
// Use it to guard teardown against adopting/deleting another build's public box.
func (b *Builder) DestroyHostForDeployment(ctx context.Context, team, externalRef, externalName string, rebuild bool) error {
	return b.destroyInstance(ctx, externalRef, destroyOptions{keepPublicAddress: rebuild, externalName: externalName})
}

// HostReference resolves the same name as DeployHost when a rebuild reset has
// already cleared external_ref. Only this builder exposes this optional hook.
func (b *Builder) HostReference(externalName, displayName string) string {
	return instanceName(displayName, externalName, b.Client.Project)
}

type destroyOptions struct {
	keepPublicAddress bool
	externalName      string
}

func (b *Builder) destroyInstance(ctx context.Context, name string, options ...destroyOptions) error {
	var opts destroyOptions
	if len(options) > 0 {
		opts = options[0]
	}
	retain := opts.keepPublicAddress
	destroyStarted := time.Now()

	// A create the cluster is still carrying out (an interrupted deploy's) is
	// waited out first: destroying around it would either miss the instance
	// it's about to produce or fail against a busy one.
	if err := b.Client.waitForInstanceCreation(ctx, name); err != nil {
		slog.WarnContext(ctx, "MicroCloud destroy: couldn't check for an unfinished creation; going ahead", "instance", name, "error", err)
	}

	// One read first, always: it guards against deleting another deployment's
	// box, and it says what this instance actually has -- a public NIC with
	// its ACL, a per-host port ACL, a config-drive volume -- so the cleanup
	// after the delete removes exactly those, not a blind list of everything
	// it might have had. A destroy is otherwise six or seven hoster requests
	// whatever the host; this makes a plain Linux host three.
	var leftovers []leftover
	exists := true
	full, expanded, err := b.getInstancePut(ctx, name)
	if err != nil {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 404 {
			return err
		}
		exists = false
		if opts.externalName != "" {
			// Nothing exists at this exact name. Do not delete later by name:
			// a different colliding deployment could be creating it right now.
			slog.InfoContext(ctx, "MicroCloud destroy: no instance by this name; nothing to delete", "project", b.Client.Project, "instance", name)
			if b.PublicAddresses != nil && !retain {
				return b.PublicAddresses.Release(ctx, b.publicProject(), name, opts.externalName)
			}
			return nil
		}
		// A destroy by deterministic name for something whose creation may
		// have half-happened: nothing to stop or delete, but anything it left
		// behind is removed blind, since there's no instance to ask.
		slog.InfoContext(ctx, "MicroCloud destroy: no instance by this name; removing anything a partial create left behind", "project", b.Client.Project, "instance", name)
		leftovers = []leftover{b.publicACLLeftover(name), b.hostACLLeftover(name), b.cidataLeftover(name)}
	} else {
		if opts.externalName != "" {
			if raw := full.Config[publicNICKey]; raw != "" {
				var plan publicNICPlan
				if err := json.Unmarshal([]byte(raw), &plan); err != nil {
					return err
				}
				if plan.ExternalName != opts.externalName {
					return fmt.Errorf("instance %s belongs to another deployment; refusing to delete it", name)
				}
			}
		}
		leftovers = b.leftoversOf(name, full, expanded)
	}

	if exists {
		slog.InfoContext(ctx, "MicroCloud destroying instance: stopping it (force, 30s)", "project", b.Client.Project, "instance", name, "retain_public_ip_for_rebuild", retain, "leftovers_to_remove", leftoverNames(leftovers))
		_, err = b.Client.put(ctx, "/1.0/instances/"+name+"/state", map[string]interface{}{
			"action": "stop", "timeout": 30, "force": true,
		})
		if err != nil {
			var apiErr *APIError
			alreadyStopped := errors.As(err, &apiErr) && (apiErr.NotFound() ||
				strings.Contains(apiErr.Message, "already stopped") || strings.Contains(apiErr.Message, "not running"))
			if !alreadyStopped {
				slog.ErrorContext(ctx, "MicroCloud destroy: stop failed", "instance", name, "elapsed", time.Since(destroyStarted).Round(time.Millisecond), "error", err)
				return fmt.Errorf("stopping instance %s: %w", name, err)
			}
			slog.InfoContext(ctx, "MicroCloud destroy: instance already stopped or gone", "instance", name, "detail", err.Error())
		} else {
			slog.InfoContext(ctx, "MicroCloud destroy: instance stopped", "instance", name, "elapsed", time.Since(destroyStarted).Round(time.Millisecond))
		}
		deleteStarted := time.Now()
		slog.InfoContext(ctx, "MicroCloud destroy: deleting the instance", "instance", name)
		_, err = b.Client.delete(ctx, "/1.0/instances/"+name)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.NotFound() {
				slog.InfoContext(ctx, "MicroCloud destroy: instance was already gone", "instance", name)
			} else {
				slog.ErrorContext(ctx, "MicroCloud destroy: delete failed", "instance", name, "elapsed", time.Since(deleteStarted).Round(time.Millisecond), "error", err)
				return fmt.Errorf("deleting instance %s: %w", name, err)
			}
		} else {
			slog.InfoContext(ctx, "MicroCloud destroy: instance deleted", "instance", name, "elapsed", time.Since(deleteStarted).Round(time.Millisecond))
		}
	}

	// The delete operation's own result is the confirmation (a 404 on delete
	// means it was already gone); no second read. The public address is only
	// released once the instance is known gone, which is now.
	if b.PublicAddresses != nil {
		if !retain {
			if err := b.PublicAddresses.Release(ctx, b.publicProject(), name, opts.externalName); err != nil {
				slog.ErrorContext(ctx, "MicroCloud destroy: releasing the public address failed", "instance", name, "error", err)
				return fmt.Errorf("releasing public address: %w", err)
			}
			slog.InfoContext(ctx, "MicroCloud destroy: public address released", "instance", name)
		} else {
			slog.InfoContext(ctx, "MicroCloud public IP reservation retained for rebuild", "project", b.publicProject(), "instance", name)
		}
	}

	// What the instance left behind goes concurrently -- the ACLs and the
	// volume are independent of each other, and all belong to this one task,
	// so this stays within the builder's concurrency limit. Best-effort, and
	// logged by the client request by request; a failure never fails the
	// destroy (the next deploy's own ensure/replace handles a straggler).
	if len(leftovers) > 0 {
		slog.InfoContext(ctx, "MicroCloud destroy: removing what it left behind", "instance", name, "leftovers", leftoverNames(leftovers))
		var wg sync.WaitGroup
		for _, l := range leftovers {
			wg.Add(1)
			go func(l leftover) {
				defer wg.Done()
				if err := l.remove(ctx); err != nil {
					slog.WarnContext(ctx, "MicroCloud destroy: leftover not removed", "instance", name, "leftover", l.name, "error", err)
				}
			}(l)
		}
		wg.Wait()
	}
	slog.InfoContext(ctx, "MicroCloud destroy: done", "instance", name, "elapsed", time.Since(destroyStarted).Round(time.Millisecond))
	return nil
}

// leftover is one thing a destroyed instance leaves behind to remove.
type leftover struct {
	name   string
	remove func(ctx context.Context) error
}

func leftoverNames(ls []leftover) []string {
	out := make([]string, 0, len(ls))
	for _, l := range ls {
		out = append(out, l.name)
	}
	return out
}

// leftoversOf reads, from the instance as the hoster describes it, which of
// the things LaForge attaches at deploy this instance actually has: the public
// NIC's ACL (eth1), the per-host port ACL on eth0, and a config-drive volume
// (Windows: the cidata disk is a custom volume in a pool; Linux's is the
// built-in cloud-init:config and has no volume).
func (b *Builder) leftoversOf(name string, full instancePut, expanded map[string]interface{}) []leftover {
	device := func(dev string) map[string]interface{} {
		if raw, ok := full.Devices[dev]; ok {
			var d map[string]interface{}
			if json.Unmarshal(raw, &d) == nil {
				return d
			}
		}
		if d, ok := expanded[dev].(map[string]interface{}); ok {
			return d
		}
		return nil
	}
	acls := func(dev string) string {
		if d := device(dev); d != nil {
			if v, ok := d["security.acls"].(string); ok {
				return v
			}
		}
		return ""
	}
	var out []leftover
	if strings.Contains(acls(publicNICName), publicACLName(b.publicProject(), name)) {
		out = append(out, b.publicACLLeftover(name))
	}
	if strings.Contains(acls("eth0"), hostACLName(name)) {
		out = append(out, b.hostACLLeftover(name))
	}
	if d := device("cidata"); d != nil {
		if pool, _ := d["pool"].(string); pool != "" {
			out = append(out, b.cidataLeftover(name))
		}
	}
	return out
}

func (b *Builder) publicACLLeftover(name string) leftover {
	acl := publicACLName(b.publicProject(), name)
	return leftover{name: "public ACL " + acl, remove: func(ctx context.Context) error {
		_, err := b.Client.delete(ctx, "/1.0/network-acls/"+acl)
		return ignoreNotFound(err)
	}}
}

func (b *Builder) hostACLLeftover(name string) leftover {
	acl := hostACLName(name)
	return leftover{name: "host port ACL " + acl, remove: func(ctx context.Context) error {
		_, err := b.Client.delete(ctx, "/1.0/network-acls/"+acl)
		return ignoreNotFound(err)
	}}
}

func (b *Builder) cidataLeftover(name string) leftover {
	vol := cidataVolName(name)
	return leftover{name: "config drive " + vol, remove: func(ctx context.Context) error {
		return b.Client.deleteVolume(ctx, b.Config.storagePoolOrDefault(), vol)
	}}
}

func ignoreNotFound(err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.NotFound() {
		return nil
	}
	return err
}

type mcInstance struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Status string            `json:"status"` // Incus's live state: Running, Stopped, Frozen, ...
	Config map[string]string `json:"config"`
}

// powerStateFromLXDState maps Incus's own instance status onto the normalized
// builder.PowerState* vocabulary.
func powerStateFromLXDState(status string) string {
	switch status {
	case "Running":
		return builder.PowerStateRunning
	case "Stopped":
		return builder.PowerStateStopped
	default:
		return builder.PowerStateOther
	}
}

type mcNetwork struct {
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
	var inst mcInstance
	if err := json.Unmarshal(raw, &inst); err != nil {
		return "", false, fmt.Errorf("decoding instance %s: %w", externalRef, err)
	}
	return powerStateFromLXDState(inst.Status), true, nil
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
		out = append(out, builder.Resource{ExternalRef: inst.Name, Kind: kind, State: powerStateFromLXDState(inst.Status)})
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

// Team access open/close lives in access.go. Like the Incus builder it no
// longer detaches eth0 (which cut agent telemetry and tore the host off its
// NATs) -- it firewalls the NICs that stay attached. See access.go.

// instancePut is the full body Incus/LXD's PUT .../instances/<name> requires --
// PUT replaces the whole resource, PATCH merges key by key. getInstancePut reads
// it (plus the in-effect expanded_devices) for callers that need a host's real
// NIC/network config, e.g. teamNetworks and ensureHostPortACL.
type instancePut struct {
	etag         string
	Architecture string                     `json:"architecture"`
	Config       map[string]string          `json:"config"`
	Devices      map[string]json.RawMessage `json:"devices"`
	Ephemeral    bool                       `json:"ephemeral"`
	Profiles     []string                   `json:"profiles"`
	Stateful     bool                       `json:"stateful"`
	Description  string                     `json:"description"`
}

func (b *Builder) getInstancePut(ctx context.Context, name string) (instancePut, map[string]interface{}, error) {
	var etag string
	raw, err := b.Client.do(ctx, "GET", "/1.0/instances/"+name, nil, requestOptions{etag: &etag})
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
	full.etag = etag
	return full, expanded.ExpandedDevices, nil
}

// retryOnBusy retries fn a bounded number of times against a real,
// transient "this exact instance still has another operation in
// flight" race (see APIError.Busy's own doc comment) -- the same
// bounded-retry-with-backoff idiom startInstance already uses for its
// own AlreadyRunning tolerance, generalized here since removeNIC/
// restoreNIC hit the identical class of race on a config update instead
// of a state change (found live: an open_access immediately following a
// close_access on the same instance).
func retryOnBusy(ctx context.Context, fn func() error) error {
	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		var apiErr *APIError
		if !(errors.As(err, &apiErr) && (apiErr.Busy() || apiErr.HTTPStatus == 412)) {
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

func (b *Builder) snapshotCreateDevices(ctx context.Context, img ImageRef, devices map[string]interface{}) (map[string]interface{}, error) {
	scoped := *b.Client
	if img.SourceProject != "" {
		scoped.Project = img.SourceProject
	}
	raw, err := scoped.get(ctx, "/1.0/instances/"+url.PathEscape(img.Instance)+"/snapshots/"+url.PathEscape(img.Snapshot))
	if err != nil {
		return nil, fmt.Errorf("reading source snapshot devices: %w", err)
	}
	var snapshot struct {
		Devices         map[string]json.RawMessage `json:"devices"`
		ExpandedDevices map[string]json.RawMessage `json:"expanded_devices"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decoding source snapshot devices: %w", err)
	}
	createDevices := make(map[string]interface{}, len(devices)+len(snapshot.ExpandedDevices))
	for _, inherited := range []map[string]json.RawMessage{snapshot.Devices, snapshot.ExpandedDevices} {
		for key := range inherited {
			createDevices[key] = map[string]string{"type": "none"}
		}
	}
	for key, device := range devices {
		createDevices[key] = device
	}
	return createDevices, nil
}

// replaceCopiedDevices makes a copied instance's devices exactly LaForge's own
// -- the team NIC, the root disk, a config drive -- and its profiles just
// "default" (or empty for explicit public-NIC deployments). A copy otherwise keeps
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
		if _, public := devices[publicNICName]; public {
			full.Profiles = []string{}
		}
		if _, err := b.Client.do(ctx, "PUT", "/1.0/instances/"+name, full, requestOptions{ifMatch: full.etag}); err != nil {
			return fmt.Errorf("replacing the devices %s copied from its template: %w", name, err)
		}
		return nil
	})
}

// ourInstances is LaForge's own instances in this builder's project -- see
// Client.laforgeInstances for why they're read one by one.
func (b *Builder) ourInstances(ctx context.Context) ([]mcInstance, error) {
	raws, err := b.Client.laforgeInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	out := make([]mcInstance, 0, len(raws))
	for _, raw := range raws {
		var inst mcInstance
		if err := json.Unmarshal(raw, &inst); err != nil {
			return nil, fmt.Errorf("decoding instance: %w", err)
		}
		out = append(out, inst)
	}
	return out, nil
}
