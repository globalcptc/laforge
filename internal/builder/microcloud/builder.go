package microcloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/render"
)

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

// instanceName is shortName's readable counterpart for hosts and
// containers (not networks, which are stuck with the 15-char cap and so
// keep the pure hash): "lf-<branch>-t<team>-<host>-<6 hex>". The hex
// suffix is a short digest of the full ExternalName -- it keeps the name
// deterministic (idempotent adopt-on-retry recomputes the exact same name)
// and collision-free across two live builds of the same branch+host on one
// hoster, which the readable part alone can't guarantee. Falls back to
// shortName when there's no display hint at all. Capped so the derived
// cidata volume name ("<name>-cidata") still fits Incus's 63-char limit.
func instanceName(displayName, externalName string) string {
	base := sanitizeName(displayName)
	if base == "" {
		return shortName(externalName)
	}
	sum := sha256.Sum256([]byte(externalName))
	suffix := hex.EncodeToString(sum[:])[:6]
	const maxBase = 48 // 63 - len("-cidata") - len("-<6 hex>"), rounded down
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
	bridgeAddr, err := bridgeAddressCIDR(spec.CIDR)
	if err != nil {
		return "", fmt.Errorf("computing gateway address for %s: %w", spec.CIDR, err)
	}
	// Adopt-if-present, by an explicit existence check rather than relying on the
	// create call's error shape. On a CLUSTERED LXD (a real MicroCloud), re-POSTing
	// an OVN network that already exists does NOT return a clean 409 -- it returns
	// a 500 "Network is not in pending state" from the cluster's two-phase network
	// path, which AlreadyExists() can't recognize. Found live against the 4-node
	// cluster: without this pre-check the second (retried) DeployNetwork of an
	// existing network errored instead of adopting, breaking the "ensure" contract.
	if exists, err := b.networkExists(ctx, name); err != nil {
		return "", fmt.Errorf("checking whether network %s exists: %w", name, err)
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
			return name, nil // adopt (single-node/incus-lineage error shape)
		}
		return "", fmt.Errorf("creating network %s (from %s): %w", name, spec.ExternalName, err)
	}
	return name, nil
}

// networkExists reports whether a managed network of this exact name is already
// present, via GET /1.0/networks/<name> (404 -> false). Used by DeployNetwork to
// adopt idempotently on a clustered LXD, where a duplicate create doesn't surface
// as a recognizable "already exists" error.
func (b *Builder) networkExists(ctx context.Context, name string) (bool, error) {
	_, err := b.Client.get(ctx, "/1.0/networks/"+name)
	if err == nil {
		return true, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.NotFound() {
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
	// Tear down the visible_from fabric (peers + ACL) first -- a peered OVN
	// network can't be deleted, nor an attached ACL. Best-effort. See
	// netaccess.go and the incus builder's identical teardown.
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
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, instanceType, img, size, spec.DiskGB, false, spec.CloudInitUserData, spec.CloudInitViaISO)
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
	name, err := b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, "container", base, size, 0, true, "", false)
	if err != nil {
		return name, err
	}
	if err := b.runNestedContainer(ctx, name, spec); err != nil {
		return name, err
	}
	return name, nil
}

// teamConfigKey is the Incus instance config key OpenAccess/CloseAccess
// use to find every instance belonging to one team -- set once at
// deploy time, from spec.Team, since nothing about "which team does this
// belong to" is otherwise recoverable from an instance after the fact
// (the short hashed name deliberately carries no semantic information --
// see shortName's own doc comment).
const teamConfigKey = "user.laforge_team"

func (b *Builder) deployInstance(ctx context.Context, externalName, displayName, team, network, networkDisplayName, address, instanceType string, img ImageRef, size SizeSpec, diskGB int, nesting bool, cloudInit string, cloudInitViaISO bool) (string, error) {
	name := instanceName(displayName, externalName)

	config := map[string]string{teamConfigKey: team}
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
		// Windows: cloudbase-init can't read the socket, so the user-data
		// is delivered as a NoCloud config-drive ISO attached to the
		// instance (imported as a custom volume first -- see configdrive.go).
		iso, err := buildCidataISO(name, cloudInit)
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

	source := map[string]string{"type": "image", "alias": img.Alias, "server": img.Server, "protocol": img.Protocol}
	if img.Fingerprint != "" {
		source = map[string]string{"type": "image", "fingerprint": img.Fingerprint}
	}
	_, err := b.Client.post(ctx, "/1.0/instances", map[string]interface{}{
		"name":    name,
		"type":    instanceType,
		"source":  source,
		"config":  config,
		"devices": devices,
	})
	if err != nil {
		var apiErr *APIError
		if !(errors.As(err, &apiErr) && apiErr.AlreadyExists()) {
			// Return the (deterministic) name even on failure: the instance
			// may have been partially created at the hoster, and the runner
			// records this ref so teardown can still destroy the orphan --
			// otherwise a config-drive/create failure leaves an untracked
			// instance nothing can clean up (found live on MicroCloud/Ceph).
			return name, fmt.Errorf("creating instance %s (from %s): %w", name, externalName, err)
		}
		// "adopt if already there" -- fall through to ensure it's started.
	}

	if err := b.startInstance(ctx, name); err != nil {
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

func (b *Builder) destroyInstance(ctx context.Context, name string) error {
	_, err := b.Client.put(ctx, "/1.0/instances/"+name+"/state", map[string]interface{}{
		"action": "stop", "timeout": 30, "force": true,
	})
	if err != nil {
		var apiErr *APIError
		alreadyStopped := errors.As(err, &apiErr) && (apiErr.NotFound() ||
			strings.Contains(apiErr.Message, "already stopped") || strings.Contains(apiErr.Message, "not running"))
		if !alreadyStopped {
			return fmt.Errorf("stopping instance %s: %w", name, err)
		}
	}
	_, err = b.Client.delete(ctx, "/1.0/instances/"+name)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			// Still clean up the config-drive volume below.
		} else {
			return fmt.Errorf("deleting instance %s: %w", name, err)
		}
	}
	// Remove the config-drive volume, if this instance had one (Windows).
	// A no-op when there wasn't one; never fails the destroy.
	_ = b.Client.deleteVolume(ctx, b.Config.storagePoolOrDefault(), cidataVolName(name))
	// Remove this instance's per-host port-firewall ACL now that its only user
	// is gone, so it doesn't linger orphaned. Best-effort. Mirrors incus.
	_, _ = b.Client.delete(ctx, "/1.0/network-acls/"+hostACLName(name))
	return nil
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

func (b *Builder) Inspect(ctx context.Context) ([]builder.Resource, error) {
	raw, err := b.Client.get(ctx, "/1.0/instances?recursion=1")
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	var instances []mcInstance
	if err := json.Unmarshal(raw, &instances); err != nil {
		return nil, fmt.Errorf("decoding instance list: %w", err)
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

	rawNets, err := b.Client.get(ctx, "/1.0/networks?recursion=1")
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	var nets []mcNetwork
	if err := json.Unmarshal(rawNets, &nets); err != nil {
		return nil, fmt.Errorf("decoding network list: %w", err)
	}
	for _, n := range nets {
		if !n.Managed || !strings.HasPrefix(n.Name, "lf-") {
			continue // the project's own pre-existing networks, not ours
		}
		out = append(out, builder.Resource{ExternalRef: n.Name, Kind: "network"})
	}
	return out, nil
}

// OpenAccess and CloseAccess are the required, behavioural operations:
// "closing a team must block all external ingress and terminate
// established connections, not merely stop new ones." This
// implementation removes (CloseAccess) or re-adds (OpenAccess) every
// team-tagged instance's `eth0` network device entirely, rather than
// e.g. an iptables rule -- verified against a live daemon
// (builder_test.go) to be the mechanism that actually satisfies the
// "terminate established connections" requirement: removing the device
// tears down the interface itself, which drops any connection using it,
// not just future ones. The device's own config (which network, what
// IP) is remembered so OpenAccess restores the exact same attachment,
// matching "port reservations survive" for the equivalent network-level
// case.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	return b.forEachTeamInstance(ctx, team, func(name string, inst mcInstance) error {
		return b.removeNIC(ctx, name)
	})
}

func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	return b.forEachTeamInstance(ctx, team, func(name string, inst mcInstance) error {
		return b.restoreNIC(ctx, name)
	})
}

func (b *Builder) forEachTeamInstance(ctx context.Context, team string, fn func(name string, inst mcInstance) error) error {
	raw, err := b.Client.get(ctx, "/1.0/instances?recursion=1")
	if err != nil {
		return fmt.Errorf("listing instances: %w", err)
	}
	var instances []mcInstance
	if err := json.Unmarshal(raw, &instances); err != nil {
		return fmt.Errorf("decoding instance list: %w", err)
	}
	for _, inst := range instances {
		if inst.Config[teamConfigKey] != team {
			continue
		}
		if err := fn(inst.Name, inst); err != nil {
			return fmt.Errorf("instance %s: %w", inst.Name, err)
		}
	}
	return nil
}

// removedNICKey stashes what CloseAccess needs to undo, as JSON, in a
// user.* config key on the instance itself -- so OpenAccess can restore
// exactly what was there, without this builder needing any durable state
// of its own beyond what's already on the instance. Incus config values
// are opaque strings from its own point of view, so this is a legitimate,
// real place to keep it.
//
// What gets saved had to change after testing against a live daemon: a
// deployInstance-created instance never sets eth0 at the instance level
// at all (see deployInstance -- devices only carries "root", and only
// when DiskGB > 0). eth0 comes entirely from Incus's "default" profile,
// confirmed live -- GET .../instances/<name> returns an empty top-level
// "devices" and eth0 only inside "expanded_devices". Checking the
// top-level "devices" field for eth0 (an earlier version of this code)
// silently found nothing and treated every real instance as "already
// closed," which is exactly the kind of bug the plan's "close really
// closes... every builder passes this before it is considered done"
// verification rule exists to catch. The fix: read expanded_devices for
// the real, in-effect NIC config, and disable it with an instance-level
// override of type "none" -- Incus's own documented mechanism for
// switching off a profile-provided device per instance, since an
// instance can't delete a device that belongs to its profile, only mask
// it. Restoring removes that override so the profile's device resumes,
// which also correctly handles the (untested-live, but symmetric) case
// of an instance that had its own instance-level eth0 to begin with --
// same save/restore shape either way, the JSON payload just says which.
const removedNICKey = "user.laforge_closed_eth0"

type savedNIC struct {
	// Instance is true when eth0 was defined directly on the instance
	// (not via a profile) -- restoreNIC puts Device back verbatim.
	// False means eth0 only ever existed via a profile, and restoring
	// means removing the "none" override so the profile's device takes
	// effect again, not writing anything back.
	Instance bool                   `json:"instance"`
	Device   map[string]interface{} `json:"device,omitempty"`
}

// instancePut is the full body Incus's PUT .../instances/<name> requires --
// verified live: PUT replaces the whole resource, so omitting a field
// (this code's first attempt sent only config+devices) resets it, and an
// empty Profiles wipes every profile-provided device, not just eth0 --
// surfaced live as "Invalid expanded devices: Failed detecting root disk
// device: No root device could be found" on an instance that never had
// its own root device, only the default profile's. PATCH, by contrast,
// merges config and devices key by key (also verified live) and needs no
// other field -- so this builder uses PATCH wherever a merge is enough
// (removeNIC only ever adds keys) and full PUT only where a key must be
// deleted outright (restoreNIC clearing removedNICKey, or removing an
// eth0 override so a profile's own device takes back over -- PATCHing a
// device to null was tried live and rejected: "Missing device type in
// config").
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

func (b *Builder) removeNIC(ctx context.Context, name string) error {
	full, expandedDevices, err := b.getInstancePut(ctx, name)
	if err != nil {
		return err
	}
	if _, closed := full.Config[removedNICKey]; closed {
		return nil // idempotent
	}
	effective, hasNIC := expandedDevices["eth0"]
	if !hasNIC {
		return nil // no network device at all -- nothing to close
	}
	saved := savedNIC{}
	if instanceDevice, ok := full.Devices["eth0"]; ok {
		saved.Instance = true
		if err := json.Unmarshal(instanceDevice, &saved.Device); err != nil {
			return fmt.Errorf("decoding instance-level eth0 device: %w", err)
		}
	} else if m, ok := effective.(map[string]interface{}); ok {
		saved.Device = m // for reference only; Instance stays false
	}
	savedJSON, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	return retryOnBusy(ctx, func() error {
		_, err := b.Client.patch(ctx, "/1.0/instances/"+name, map[string]interface{}{
			"config":  map[string]string{removedNICKey: string(savedJSON)},
			"devices": map[string]interface{}{"eth0": map[string]interface{}{"type": "none"}},
		})
		return err
	})
}

func (b *Builder) restoreNIC(ctx context.Context, name string) error {
	full, _, err := b.getInstancePut(ctx, name)
	if err != nil {
		return err
	}
	rawSaved, ok := full.Config[removedNICKey]
	if !ok {
		return nil // already open -- idempotent
	}
	var saved savedNIC
	if err := json.Unmarshal([]byte(rawSaved), &saved); err != nil {
		return fmt.Errorf("decoding saved nic state: %w", err)
	}
	if saved.Instance {
		deviceJSON, err := json.Marshal(saved.Device)
		if err != nil {
			return err
		}
		full.Devices["eth0"] = deviceJSON
	} else {
		delete(full.Devices, "eth0") // fall back to the profile's own device
	}
	delete(full.Config, removedNICKey)
	return retryOnBusy(ctx, func() error {
		_, err := b.Client.put(ctx, "/1.0/instances/"+name, full)
		return err
	})
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
