package microcloud

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"strconv"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder"
	"gopkg.in/yaml.v3"
)

const publicNICName = "lf-public"
const publicNICKey = "user.laforge_public_nic"
const publicReadyKey = "user.laforge_public_ready"
const publicClosedKey = "user.laforge_closed_public"

type publicNICPlan struct {
	ready         bool
	ExternalName  string             `json:"external_name"`
	Address       string             `json:"address"`
	Settings      PublicAccessConfig `json:"settings"`
	Device        map[string]string  `json:"device"`
	PrimaryMAC    string             `json:"primary_mac"`
	NetworkConfig string             `json:"network_config"`
}

func (b *Builder) PublicNICEnabled() bool { return b.Config.PublicAccess.Type == "nic" }

// Only MicroCloud implements this optional deployment entry point. Provisioning
// before first boot is required for Windows (there is no LXD Windows agent).
func (b *Builder) DeployHostWithPublicPorts(ctx context.Context, spec builder.HostSpec, tcp, udp []string) (string, error) {
	if !b.PublicNICEnabled() || len(tcp)+len(udp) == 0 {
		return b.DeployHost(ctx, spec)
	}
	name := instanceName(spec.DisplayName, spec.ExternalName)
	img, ok := b.Config.Images[spec.OS]
	if !ok {
		return "", fmt.Errorf("no image configured for os %q", spec.OS)
	}
	size, ok := b.Config.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no size configured for %q", spec.Size)
	}
	windows := spec.CloudInitViaISO || agentdelivery.PlatformFor(spec.OS) == agentdelivery.Windows
	if windows && !img.VM {
		return name, fmt.Errorf("Windows public NIC requires a VM image")
	}
	var plan *publicNICPlan
	err := b.retryInstanceStep(ctx, name, "preparing public NIC", func() error {
		var err error
		plan, err = b.preparePublicNIC(ctx, name, spec, tcp, udp)
		return err
	})
	if err != nil {
		return name, err
	}
	if plan.ready {
		slog.InfoContext(ctx, "MicroCloud public NIC already configured", "project", b.publicProject(), "instance", name, "network", plan.Settings.Network, "public_ip", plan.Address)
		return name, b.startInstance(ctx, name)
	}
	instanceType := "container"
	if img.VM {
		instanceType = "virtual-machine"
	}
	return b.deployInstance(ctx, spec.ExternalName, spec.DisplayName, spec.Team, spec.Network, spec.NetworkDisplayName, spec.Address, instanceType, img, size, spec.DiskGB, false, spec.CloudInitUserData, windows, plan)
}

func (b *Builder) publicProject() string {
	if b.Client.Project == "" {
		return "default"
	}
	return b.Client.Project
}

func publicMAC(project, name, nic string) string {
	sum := sha256.Sum256([]byte(project + "/" + name + "/" + nic))
	return net.HardwareAddr{0x02, sum[0], sum[1], sum[2], sum[3], sum[4]}.String()
}

type publicNetwork struct {
	Type   string            `json:"type"`
	Status string            `json:"status"`
	Config map[string]string `json:"config"`
}

func (b *Builder) readPublicNetwork(ctx context.Context, name string) (publicNetwork, error) {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+url.PathEscape(name))
	if err != nil {
		return publicNetwork{}, fmt.Errorf("reading network %s: %w", name, err)
	}
	var n publicNetwork
	if err := json.Unmarshal(raw, &n); err != nil {
		return n, err
	}
	if n.Type != "ovn" || n.Status != "Created" {
		return n, fmt.Errorf("network %s must be a Created OVN network", name)
	}
	return n, nil
}

func (b *Builder) preparePublicNIC(ctx context.Context, name string, spec builder.HostSpec, tcp, udp []string) (*publicNICPlan, error) {
	cfg := b.Config.PublicAccess
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if b.PublicAddresses == nil {
		return nil, fmt.Errorf("public NIC requires a persistent address allocator")
	}
	if spec.Network == "" || spec.Address == "" {
		return nil, fmt.Errorf("public NIC requires a primary network and static lab address")
	}
	primaryName := networkName(spec.NetworkDisplayName, spec.Network)
	if primaryName == cfg.Network {
		return nil, fmt.Errorf("public network must differ from the primary network")
	}
	primary, err := b.readPublicNetwork(ctx, primaryName)
	if err != nil {
		return nil, err
	}
	public, err := b.readPublicNetwork(ctx, cfg.Network)
	if err != nil {
		return nil, err
	}
	primaryCIDR, err := netip.ParsePrefix(primary.Config["ipv4.address"])
	if err != nil || !primaryCIDR.Addr().Is4() {
		return nil, fmt.Errorf("primary network requires an IPv4 subnet")
	}
	publicCIDR, _ := netip.ParsePrefix(cfg.CIDR)
	if publicCIDR.Overlaps(primaryCIDR) {
		return nil, fmt.Errorf("public and primary subnets overlap")
	}
	publicSubnet, subnetErr := netip.ParsePrefix(public.Config["ipv4.address"])
	if subnetErr == nil {
		if publicSubnet.Masked() != publicCIDR {
			return nil, fmt.Errorf("public NIC cidr %s differs from network subnet %s", cfg.CIDR, publicSubnet.Masked())
		}
		ips, _ := parsePublicIPs(cfg.Ranges) // cfg.Validate already checked the pool.
		for _, ip := range ips {
			if ip == publicSubnet.Addr().String() {
				return nil, fmt.Errorf("public address pool includes network gateway %s", ip)
			}
		}
	}
	if mtu, err := strconv.Atoi(public.Config["bridge.mtu"]); err == nil && cfg.MTU > mtu {
		return nil, fmt.Errorf("public NIC MTU %d exceeds network MTU %d", cfg.MTU, mtu)
	}
	addr, err := netip.ParseAddr(spec.Address)
	if err != nil || !primaryCIDR.Contains(addr) || addr == primaryCIDR.Addr() {
		return nil, fmt.Errorf("invalid primary address %q", spec.Address)
	}
	// Exact instance reads are authoritative. Never adopt a private instance and
	// assume it will consume new first-boot metadata when public is enabled later.
	full, _, err := b.getInstancePut(ctx, name)
	var existing *publicNICPlan
	if err == nil {
		if raw := full.Config[publicNICKey]; raw != "" {
			existing = &publicNICPlan{}
			if err := json.Unmarshal([]byte(raw), existing); err != nil {
				return nil, err
			}
			if existing.ExternalName != spec.ExternalName {
				return nil, fmt.Errorf("instance name %s belongs to a different deployment", name)
			}
		} else {
			return nil, fmt.Errorf("instance %s already exists without a public NIC; rebuild it to enable separate-NIC access", name)
		}
	} else {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 404 {
			return nil, err
		}
	}
	address, err := b.PublicAddresses.Reserve(ctx, b.publicProject(), name, spec.ExternalName, cfg, existing == nil)
	if err != nil {
		return nil, err
	}
	plan := &publicNICPlan{ready: existing != nil && full.Config[publicReadyKey] == "true", ExternalName: spec.ExternalName, Address: address, Settings: cfg, PrimaryMAC: publicMAC(b.publicProject(), name, "eth0")}
	plan.Device = map[string]string{
		"type": "nic", "name": "eth1", "network": cfg.Network,
		"hwaddr":                               publicMAC(b.publicProject(), name, publicNICName),
		"security.acls":                        publicACLName(b.publicProject(), name),
		"security.acls.default.ingress.action": "drop",
		"security.acls.default.egress.action":  "allow",
	}
	// DHCP-disabled networks (GUEST_GUAC_WAN) reject ipv4.address on
	// the LXD device. Guest metadata always assigns the static address.
	if subnetErr == nil && public.Config["ipv4.dhcp"] != "false" {
		plan.Device["ipv4.address"] = address
	}
	plan.NetworkConfig, err = publicGuestNetwork(plan, spec.Address, primaryCIDR, primary.Config["bridge.mtu"], public.Config["bridge.mtu"])
	if err != nil {
		return nil, err
	}
	if existing != nil && (existing.Address != address || existing.NetworkConfig != plan.NetworkConfig || existing.Device["network"] != cfg.Network) {
		return nil, fmt.Errorf("instance %s has different public NIC settings; rebuild it", name)
	}
	if err := b.ensurePublicACL(ctx, name, tcp, udp); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "MicroCloud public NIC prepared", "project", b.publicProject(), "instance", name, "network", cfg.Network, "public_ip", address, "primary_ip", spec.Address, "dhcp_reservation", plan.Device["ipv4.address"] != "", "tcp_ports", tcp, "udp_ports", udp)
	return plan, nil
}

func publicGuestNetwork(plan *publicNICPlan, primaryAddress string, primary netip.Prefix, primaryMTU, publicMTU string) (string, error) {
	mask := func(bits int) string { return net.IP(net.CIDRMask(bits, 32)).String() }
	pubPrefix, _ := netip.ParsePrefix(plan.Settings.CIDR)
	primarySubnet := map[string]interface{}{"type": "static", "address": primaryAddress, "netmask": mask(primary.Bits())}
	publicSubnet := map[string]interface{}{"type": "static", "address": plan.Address, "netmask": mask(pubPrefix.Bits())}
	if plan.Settings.Gateway != "" {
		publicSubnet["gateway"] = plan.Settings.Gateway
	} else {
		primarySubnet["gateway"] = primary.Addr().String()
	}
	if len(plan.Settings.DNS) > 0 {
		publicSubnet["dns_nameservers"] = plan.Settings.DNS
	} else {
		primarySubnet["dns_nameservers"] = []string{primary.Addr().String()}
	}
	var routes []map[string]string
	for _, route := range plan.Settings.Routes {
		p, _ := netip.ParsePrefix(route.To)
		routes = append(routes, map[string]string{"network": p.Addr().String(), "netmask": mask(p.Bits()), "gateway": route.Via})
	}
	if len(routes) > 0 {
		publicSubnet["routes"] = routes
	}
	primaryNIC := map[string]interface{}{"type": "physical", "name": "eth0", "mac_address": plan.PrimaryMAC, "subnets": []interface{}{primarySubnet}}
	publicNIC := map[string]interface{}{"type": "physical", "name": "eth1", "mac_address": plan.Device["hwaddr"], "subnets": []interface{}{publicSubnet}}
	if mtu, err := strconv.Atoi(primaryMTU); err == nil && mtu > 0 {
		primaryNIC["mtu"] = mtu
	}
	if plan.Settings.MTU > 0 {
		publicNIC["mtu"] = plan.Settings.MTU
	} else if mtu, err := strconv.Atoi(publicMTU); err == nil && mtu > 0 {
		publicNIC["mtu"] = mtu
	}
	// Version 1, with routes nested under the subnet, is supported by both
	// cloud-init and Cloudbase-Init's NoCloud network parser.
	raw, err := yaml.Marshal(map[string]interface{}{"version": 1, "config": []interface{}{primaryNIC, publicNIC}})
	return string(raw), err
}

func publicACLName(project, name string) string {
	sum := sha256.Sum256([]byte(project + "/" + name))
	return fmt.Sprintf("lfpub%x", sum[:10])
}

func (b *Builder) ensurePublicACL(ctx context.Context, name string, tcp, udp []string) error {
	rules := nicIngressRules([]string{"0.0.0.0/0"}, builder.HostAccess{TCPPorts: tcp, UDPPorts: udp})
	if err := b.ensureACLObject(ctx, publicACLName(b.publicProject(), name), "LaForge public ports for "+name, rules); err != nil {
		return err
	}
	slog.InfoContext(ctx, "MicroCloud public port rules applied", "project", b.publicProject(), "instance", name, "tcp_ports", tcp, "udp_ports", udp)
	return nil
}

func (b *Builder) configurePublicNICAccess(ctx context.Context, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	var out []builder.ExternalEndpoint
	for _, host := range hosts {
		full, _, err := b.getInstancePut(ctx, host.ExternalRef)
		if err != nil {
			return nil, err
		}
		raw := full.Config[publicNICKey]
		if raw == "" {
			if len(host.TCPPorts)+len(host.UDPPorts) == 0 {
				continue
			}
			return nil, fmt.Errorf("%s has no public NIC; separate-NIC access requires a host deployed in this mode (rebuild existing hosts)", host.ExternalRef)
		}
		var plan publicNICPlan
		if err := json.Unmarshal([]byte(raw), &plan); err != nil {
			return nil, err
		}
		// Updating the ACL never reattaches a closed NIC. Empty ports revoke
		// public access while preserving the host's address until teardown.
		if err := b.ensurePublicACL(ctx, host.ExternalRef, host.TCPPorts, host.UDPPorts); err != nil {
			return nil, err
		}
		for proto, ports := range map[string][]string{"tcp": host.TCPPorts, "udp": host.UDPPorts} {
			for _, port := range ports {
				out = append(out, builder.ExternalEndpoint{ExternalRef: host.ExternalRef, Protocol: proto, InternalPort: port, ExternalPort: port, PublicAddress: plan.Address + ":" + port})
			}
		}
	}
	return out, nil
}

// HostPublicAddress is recorded before the runner exposes the box as running,
// so script materialization can use it without waiting for endpoint publishing.
func (b *Builder) HostPublicAddress(ctx context.Context, name string) (string, error) {
	if !b.PublicNICEnabled() {
		return "", nil
	}
	full, _, err := b.getInstancePut(ctx, name)
	if err != nil {
		return "", err
	}
	if raw := full.Config[publicNICKey]; raw != "" {
		var plan publicNICPlan
		if err := json.Unmarshal([]byte(raw), &plan); err != nil {
			return "", err
		}
		return plan.Address, nil
	}
	return "", nil
}
