// Package openstack is a DRAFT builder.Builder backed by real Nova/Neutron
// calls via gophercloud v2. Like the AWS draft next to it, it satisfies the
// full internal/builder.Builder contract and follows the shared conventions
// (deterministic ExternalName "ensure" semantics, per-team access control,
// power as an infrastructure operation) but has NOT been run against a real
// OpenStack cloud -- there is no test infrastructure for it yet. Read it as a
// reviewed starting point, not a proven path; genuinely account-specific
// decisions (floating IPs for external reachability, terminating established
// flows on close, Designate for DNS) are marked TODO(untested).
//
// Mapping from LaForge concepts to OpenStack:
//   - network   -> a Neutron network + subnet with the network's CIDR
//   - host      -> a Nova server on that network
//   - container -> a Zun capsule (see zun.go); Zun is a separate service from
//     Nova, so if the cloud's catalog has no Zun endpoint DeployContainer fails
//     with a clear error rather than a capability opt-out
//   - team access -> a per-team Neutron security group whose ingress rule is
//     added on OpenAccess and removed on CloseAccess; servers join it by name
//
// Credentials come from the standard OS_* environment (clouds via
// AuthOptionsFromEnv), so nothing secret is stored on the config. Every
// server carries laforge:* metadata, which is how "ensure" and Inspect find
// what already exists.
package openstack

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	osclient "github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/secgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"

	"github.com/globalcptc/laforge/internal/builder"
)

const (
	managedKey      = "laforge:managed"
	externalNameKey = "laforge:external-name"
	teamKey         = "laforge:team"
	displayKey      = "laforge:display"
)

// Config is what the resolver hands the builder. Region selects the endpoint
// catalog entry; Images maps a LaForge `os` name to a Glance image id, Sizes
// a LaForge `size` name to a Nova flavor id.
type Config struct {
	Region string
	Images map[string]string // os name -> image id
	Sizes  map[string]string // size name -> flavor id
}

// Builder is the draft Nova/Neutron/Zun builder.
type Builder struct {
	compute   *gophercloud.ServiceClient
	network   *gophercloud.ServiceClient
	container *gophercloud.ServiceClient // Zun; nil if the cloud has no container service
	cfg       Config
}

// New authenticates from the ambient OS_* environment and opens the compute
// and network service clients.
func New(ctx context.Context, cfg Config) (*Builder, error) {
	authOpts, err := osclient.AuthOptionsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("reading OpenStack auth from environment: %w", err)
	}
	provider, err := osclient.AuthenticatedClient(ctx, authOpts)
	if err != nil {
		return nil, fmt.Errorf("authenticating to OpenStack: %w", err)
	}
	eo := gophercloud.EndpointOpts{Region: cfg.Region}
	compute, err := osclient.NewComputeV2(provider, eo)
	if err != nil {
		return nil, fmt.Errorf("opening Nova client: %w", err)
	}
	network, err := osclient.NewNetworkV2(provider, eo)
	if err != nil {
		return nil, fmt.Errorf("opening Neutron client: %w", err)
	}
	// Zun is optional in a deployment's catalog; a nil client makes
	// DeployContainer fail with a clear error at deploy, not a capability flag.
	container := newContainerClient(provider, eo)
	return &Builder{compute: compute, network: network, container: container, cfg: cfg}, nil
}

func teamGroupName(team string) string { return "laforge-team-" + team }

// --- Networks -------------------------------------------------------------

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	if id, err := b.findNetwork(ctx, spec.ExternalName); err != nil {
		return "", err
	} else if id != "" {
		return id, nil // ensure: adopt
	}
	if spec.CIDR == "" {
		return "", fmt.Errorf("network %q has no CIDR -- Neutron requires one for the subnet", spec.ExternalName)
	}
	up := true
	net, err := networks.Create(ctx, b.network, networks.CreateOpts{
		Name:         spec.ExternalName,
		AdminStateUp: &up,
	}).Extract()
	if err != nil {
		return "", fmt.Errorf("creating network %q: %w", spec.ExternalName, err)
	}
	if _, err := subnets.Create(ctx, b.network, subnets.CreateOpts{
		NetworkID: net.ID,
		CIDR:      spec.CIDR,
		IPVersion: gophercloud.IPv4,
		Name:      spec.ExternalName,
	}).Extract(); err != nil {
		return "", fmt.Errorf("creating subnet for %q: %w", spec.ExternalName, err)
	}
	// TODO(untested): external reachability needs a Neutron router linking
	// this network to the external provider network, plus floating IPs on
	// the hosts that should be reachable -- deferred until testable.
	return net.ID, nil
}

func (b *Builder) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	// Deleting the network cascades its subnets once no ports remain (hosts
	// are torn down first by the orchestrator). Already-gone is success.
	if err := networks.Delete(ctx, b.network, externalRef).ExtractErr(); err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			return nil
		}
		return fmt.Errorf("deleting network %s: %w", externalRef, err)
	}
	return nil
}

func (b *Builder) findNetwork(ctx context.Context, externalName string) (string, error) {
	pages, err := networks.List(b.network, networks.ListOpts{Name: externalName}).AllPages(ctx)
	if err != nil {
		return "", fmt.Errorf("listing networks for %q: %w", externalName, err)
	}
	found, err := networks.ExtractNetworks(pages)
	if err != nil {
		return "", fmt.Errorf("extracting networks for %q: %w", externalName, err)
	}
	for _, n := range found {
		if n.Name == externalName {
			return n.ID, nil
		}
	}
	return "", nil
}

// --- Hosts ----------------------------------------------------------------

func (b *Builder) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	if id, err := b.findServer(ctx, spec.ExternalName); err != nil {
		return "", err
	} else if id != "" {
		return id, nil // ensure: adopt
	}

	image, ok := b.cfg.Images[spec.OS]
	if !ok {
		return "", fmt.Errorf("no image mapping for os %q (configure the builder's image map)", spec.OS)
	}
	flavor, ok := b.cfg.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no flavor mapping for size %q (configure the builder's size map)", spec.Size)
	}
	netID, err := b.findNetwork(ctx, spec.Network)
	if err != nil {
		return "", err
	}
	if netID == "" {
		return "", fmt.Errorf("network %q not found -- its DeployNetwork may not have run yet", spec.Network)
	}
	if err := b.ensureTeamGroup(ctx, spec.Team); err != nil {
		return "", err
	}

	opts := servers.CreateOpts{
		Name:           spec.ExternalName,
		ImageRef:       image,
		FlavorRef:      flavor,
		Networks:       []servers.Network{{UUID: netID}},
		SecurityGroups: []string{teamGroupName(spec.Team)},
		Metadata: map[string]string{
			managedKey:      "true",
			externalNameKey: spec.ExternalName,
			teamKey:         spec.Team,
			displayKey:      spec.DisplayName,
		},
	}
	if spec.CloudInitUserData != "" {
		opts.UserData = []byte(spec.CloudInitUserData)
	}
	srv, err := servers.Create(ctx, b.compute, opts, nil).Extract()
	if err != nil {
		return "", fmt.Errorf("creating server for %q: %w", spec.ExternalName, err)
	}
	return srv.ID, nil
}

// DeployContainer is implemented in zun.go (OpenStack containers are Zun
// capsules, a separate service from Nova).

func (b *Builder) DestroyHost(ctx context.Context, team, externalRef string) error {
	if err := servers.Delete(ctx, b.compute, externalRef).ExtractErr(); err != nil {
		if gophercloud.ResponseCodeIs(err, 404) {
			return nil
		}
		return fmt.Errorf("deleting server %s: %w", externalRef, err)
	}
	return nil
}

// DestroyContainer is implemented in zun.go.

func (b *Builder) findServer(ctx context.Context, externalName string) (string, error) {
	pages, err := servers.List(b.compute, servers.ListOpts{Name: externalName}).AllPages(ctx)
	if err != nil {
		return "", fmt.Errorf("listing servers for %q: %w", externalName, err)
	}
	found, err := servers.ExtractServers(pages)
	if err != nil {
		return "", fmt.Errorf("extracting servers for %q: %w", externalName, err)
	}
	for _, s := range found {
		if s.Name == externalName {
			return s.ID, nil
		}
	}
	return "", nil
}

// --- Inspect --------------------------------------------------------------

func (b *Builder) Inspect(ctx context.Context) ([]builder.Resource, error) {
	pages, err := servers.List(b.compute, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}
	all, err := servers.ExtractServers(pages)
	if err != nil {
		return nil, fmt.Errorf("extracting servers: %w", err)
	}
	var out []builder.Resource
	for _, s := range all {
		if s.Metadata[managedKey] != "true" {
			continue // not ours
		}
		out = append(out, builder.Resource{ExternalRef: s.ID, Kind: "host", State: mapServerStatus(s.Status)})
	}
	// TODO(untested): Neutron networks carry no metadata the way Nova servers
	// do; identifying LaForge-managed networks for drift would use the
	// resource-tags extension. Deferred -- host liveness (the load-bearing
	// part of Inspect, for the power-state poll) is covered above.
	return out, nil
}

func mapServerStatus(status string) string {
	switch status {
	case "ACTIVE":
		return builder.PowerStateRunning
	case "SHUTOFF":
		return builder.PowerStateStopped
	default:
		return builder.PowerStateOther
	}
}

// --- Access (per-team Neutron security group) -----------------------------

func (b *Builder) ensureTeamGroup(ctx context.Context, team string) error {
	if id, err := b.findGroup(ctx, team); err != nil {
		return err
	} else if id != "" {
		return nil
	}
	_, err := groups.Create(ctx, b.network, groups.CreateOpts{
		Name:        teamGroupName(team),
		Description: "LaForge team " + team + " ingress control",
	}).Extract()
	if err != nil {
		return fmt.Errorf("creating security group for team %s: %w", team, err)
	}
	return nil
}

func (b *Builder) findGroup(ctx context.Context, team string) (string, error) {
	pages, err := groups.List(b.network, groups.ListOpts{Name: teamGroupName(team)}).AllPages(ctx)
	if err != nil {
		return "", fmt.Errorf("listing security groups for team %s: %w", team, err)
	}
	found, err := groups.ExtractGroups(pages)
	if err != nil {
		return "", fmt.Errorf("extracting security groups for team %s: %w", team, err)
	}
	if len(found) > 0 {
		return found[0].ID, nil
	}
	return "", nil
}

func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	gid, err := b.findGroup(ctx, team)
	if err != nil {
		return err
	}
	if gid == "" {
		return nil // nothing deployed for this team yet
	}
	_, err = rules.Create(ctx, b.network, rules.CreateOpts{
		Direction:      rules.DirIngress,
		EtherType:      rules.EtherType4,
		SecGroupID:     gid,
		RemoteIPPrefix: "0.0.0.0/0",
	}).Extract()
	if err != nil {
		// A duplicate rule (already open) is convergence, not failure.
		if gophercloud.ResponseCodeIs(err, 409) {
			return nil
		}
		return fmt.Errorf("opening access for team %s: %w", team, err)
	}
	return nil
}

func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	gid, err := b.findGroup(ctx, team)
	if err != nil {
		return err
	}
	if gid == "" {
		return nil
	}
	pages, err := rules.List(b.network, rules.ListOpts{SecGroupID: gid, Direction: string(rules.DirIngress)}).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("listing ingress rules for team %s: %w", team, err)
	}
	existing, err := rules.ExtractRules(pages)
	if err != nil {
		return fmt.Errorf("extracting ingress rules for team %s: %w", team, err)
	}
	for _, r := range existing {
		if err := rules.Delete(ctx, b.network, r.ID).ExtractErr(); err != nil {
			if gophercloud.ResponseCodeIs(err, 404) {
				continue
			}
			return fmt.Errorf("revoking rule %s for team %s: %w", r.ID, team, err)
		}
	}
	// NOTE(untested): removing the ingress rule blocks NEW connections;
	// OpenStack security groups are stateful, so established flows can persist
	// until they idle. Fully terminating them (the contract's requirement)
	// would need conntrack flushing on the compute hosts -- deferred until
	// testable.
	return nil
}

// --- Power ----------------------------------------------------------------

func (b *Builder) PowerAction(ctx context.Context, team, externalRef, action string, force bool) error {
	switch action {
	case builder.PowerStart:
		return wrapPower(action, externalRef, servers.Start(ctx, b.compute, externalRef).ExtractErr())
	case builder.PowerStop:
		return wrapPower(action, externalRef, servers.Stop(ctx, b.compute, externalRef).ExtractErr())
	case builder.PowerReboot:
		how := servers.SoftReboot
		if force {
			how = servers.HardReboot
		}
		return wrapPower(action, externalRef, servers.Reboot(ctx, b.compute, externalRef, servers.RebootOpts{Type: how}).ExtractErr())
	default:
		return fmt.Errorf("unknown power action %q", action)
	}
}

func wrapPower(action, ref string, err error) error {
	if err != nil {
		return fmt.Errorf("power %s on %s: %w", action, ref, err)
	}
	return nil
}

// --- Network access (visible_from) ----------------------------------------

// ConfigureNetworkAccess enforces a team's `visible_from` + `ports:` policy with
// per-host Neutron security groups -- the OpenStack analogue of the
// Incus/MicroCloud per-host NIC ACL (see the incus builder's proven
// mechanics). For each host it ensures a security group
// whose ingress rules allow exactly its declared TCP/UDP ports, and only from its
// own network's CIDR (same-network siblings) plus its `visible_from` networks'
// CIDRs (cross-network); it then makes that group the server's group. A host that
// declared no ports gets a group with no ingress rules -- reachable on nothing.
// Neutron security groups are stateful and applied per-port, so this filters
// same-network traffic too (no separate same-net handling needed).
//
// DRAFT: never run against a real account. Known gaps, documented rather than
// guessed at:
//   - Cross-network reachability still needs ROUTING between the team's networks
//     (the intended design is a per-team Neutron router the networks attach to);
//     until that exists the visible_from CIDR rules are correct but cross-network
//     packets won't route to be evaluated.
//   - Making the per-host group the server's authoritative group means the legacy
//     per-team OpenAccess/CloseAccess group no longer governs these hosts;
//     reconciling the access schedule with the port firewall is an open question.
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	cidrByName := make(map[string]string, len(networks))
	for _, na := range networks {
		cidrByName[na.ExternalName] = na.CIDR
	}
	for _, na := range networks {
		sources := []string{na.CIDR}
		for _, from := range na.VisibleFrom {
			if c := cidrByName[from]; c != "" {
				sources = append(sources, c)
			}
		}
		for _, h := range na.Hosts {
			if h.ExternalRef == "" {
				continue // not yet deployed; a later pass will configure it
			}
			if err := b.ensureHostFirewall(ctx, team, h, sources); err != nil {
				return fmt.Errorf("configuring host firewall for %s (team %s): %w", h.ExternalRef, team, err)
			}
		}
	}
	return nil
}

// hostGroupName is the deterministic per-host firewall group name.
func hostGroupName(externalRef string) string { return "laforge-hostfw-" + externalRef }

// ensureHostFirewall creates/updates the host's own security group (ingress =
// its declared ports from the allowed source CIDRs, default-deny otherwise) and
// makes it the server's group. Idempotent: ingress rules are rebuilt each pass.
func (b *Builder) ensureHostFirewall(ctx context.Context, team string, h builder.HostAccess, sources []string) error {
	name := hostGroupName(h.ExternalRef)
	gid, err := b.ensureNamedGroup(ctx, name, "LaForge host firewall for "+h.ExternalRef)
	if err != nil {
		return err
	}

	// Converge ingress: delete existing ingress rules, then add the current
	// declared-port-from-allowed-source set. Egress defaults are left intact.
	if err := b.clearIngressRules(ctx, gid); err != nil {
		return err
	}
	for _, src := range sources {
		if err := b.addPortRules(ctx, gid, rules.ProtocolTCP, h.TCPPorts, src); err != nil {
			return err
		}
		if err := b.addPortRules(ctx, gid, rules.ProtocolUDP, h.UDPPorts, src); err != nil {
			return err
		}
	}

	// Make the per-host group authoritative on the server: add it, drop the
	// legacy team group. Both best-effort against already-in-the-desired-state.
	if err := secgroups.AddServer(ctx, b.compute, h.ExternalRef, name).ExtractErr(); err != nil && !gophercloud.ResponseCodeIs(err, 409) {
		return fmt.Errorf("attaching firewall group to server %s: %w", h.ExternalRef, err)
	}
	_ = secgroups.RemoveServer(ctx, b.compute, h.ExternalRef, teamGroupName(team)).ExtractErr()
	return nil
}

// ensureNamedGroup adopts or creates a Neutron security group by name, returning
// its id. A fresh Neutron group already default-denies ingress (only egress
// defaults), which is the safe base the ingress rules open specific ports over.
func (b *Builder) ensureNamedGroup(ctx context.Context, name, desc string) (string, error) {
	pages, err := groups.List(b.network, groups.ListOpts{Name: name}).AllPages(ctx)
	if err != nil {
		return "", fmt.Errorf("listing security group %s: %w", name, err)
	}
	found, err := groups.ExtractGroups(pages)
	if err != nil {
		return "", fmt.Errorf("extracting security group %s: %w", name, err)
	}
	if len(found) > 0 {
		return found[0].ID, nil
	}
	g, err := groups.Create(ctx, b.network, groups.CreateOpts{Name: name, Description: desc}).Extract()
	if err != nil {
		return "", fmt.Errorf("creating security group %s: %w", name, err)
	}
	return g.ID, nil
}

// clearIngressRules deletes every ingress rule on a security group, so the next
// authorize pass is a clean rebuild.
func (b *Builder) clearIngressRules(ctx context.Context, gid string) error {
	pages, err := rules.List(b.network, rules.ListOpts{SecGroupID: gid, Direction: string(rules.DirIngress)}).AllPages(ctx)
	if err != nil {
		return fmt.Errorf("listing ingress rules of %s: %w", gid, err)
	}
	existing, err := rules.ExtractRules(pages)
	if err != nil {
		return fmt.Errorf("extracting ingress rules of %s: %w", gid, err)
	}
	for _, r := range existing {
		if err := rules.Delete(ctx, b.network, r.ID).ExtractErr(); err != nil && !gophercloud.ResponseCodeIs(err, 404) {
			return fmt.Errorf("deleting ingress rule %s: %w", r.ID, err)
		}
	}
	return nil
}

// addPortRules adds one ingress rule per declared port (range) for a protocol,
// from a single source CIDR. An empty port list adds nothing (default-deny).
func (b *Builder) addPortRules(ctx context.Context, gid string, proto rules.RuleProtocol, ports []string, source string) error {
	for _, p := range ports {
		lo, hi, ok := parsePortRange(p)
		if !ok {
			continue
		}
		_, err := rules.Create(ctx, b.network, rules.CreateOpts{
			Direction:      rules.DirIngress,
			EtherType:      rules.EtherType4,
			SecGroupID:     gid,
			Protocol:       proto,
			PortRangeMin:   lo,
			PortRangeMax:   hi,
			RemoteIPPrefix: source,
		}).Extract()
		if err != nil && !gophercloud.ResponseCodeIs(err, 409) {
			return fmt.Errorf("adding %s %s-%d rule from %s: %w", proto, p, hi, source, err)
		}
	}
	return nil
}

// parsePortRange parses "80" or "8000-8100" into an inclusive [from,to]. Returns
// ok=false for anything it can't parse, which the caller skips.
func parsePortRange(p string) (int, int, bool) {
	p = strings.TrimSpace(p)
	if lo, hi, found := strings.Cut(p, "-"); found {
		l, err1 := strconv.Atoi(strings.TrimSpace(lo))
		h, err2 := strconv.Atoi(strings.TrimSpace(hi))
		if err1 != nil || err2 != nil || l < 0 || h > 65535 || l > h {
			return 0, 0, false
		}
		return l, h, true
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 0 || n > 65535 {
		return 0, 0, false
	}
	return n, n, true
}

// --- DNS ------------------------------------------------------------------

// Compile-time check that the draft satisfies the full Builder contract.
var _ builder.Builder = (*Builder)(nil)
