package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"

	"github.com/globalcptc/laforge/internal/builder"
)

type allocatedNetwork struct {
	Name    string            `json:"name"`
	Project string            `json:"project"`
	Type    string            `json:"type"`
	Managed bool              `json:"managed"`
	Status  string            `json:"status"`
	Config  map[string]string `json:"config"`
}

// validateNetworkAllocation runs under the networking lock, on both creation
// and adoption. A successful POST or a Created status alone does not establish
// a usable OVN allocation (issue #65).
func (b *Builder) validateNetworkAllocation(ctx context.Context, name, gateway string, n allocatedNetwork) error {
	if n.Status != "Created" {
		return fmt.Errorf("network %s is in state %q; cluster network creation must finish successfully before deploying guests", name, n.Status)
	}
	nat, _ := strconv.ParseBool(n.Config["ipv4.nat"])
	if n.Name != name || !n.Managed || n.Type != "ovn" || n.Config["network"] != b.Config.OVNUplinkNetwork || !nat || (gateway != "" && n.Config["ipv4.address"] != gateway) {
		return fmt.Errorf("%w: network %s does not match the expected managed OVN network on uplink %s with IPv4 NAT and gateway %s; inspect its configuration before retrying", builder.ErrPermanentConfig, name, b.Config.OVNUplinkNetwork, gateway)
	}
	address, err := netip.ParseAddr(n.Config["volatile.network.ipv4.address"])
	if err != nil || !address.Is4() || !address.IsGlobalUnicast() {
		return fmt.Errorf("network %s on uplink %s has no usable volatile.network.ipv4.address (%q); refusing guest deployment", name, b.Config.OVNUplinkNetwork, n.Config["volatile.network.ipv4.address"])
	}
	// Uplink addresses are shared across projects, including networks not owned
	// by LaForge. Use a copy so the all-projects request has no project query.
	client := *b.Client
	client.Project = ""
	raw, err := client.get(ctx, "/1.0/networks?recursion=1&all-projects=true")
	if err != nil {
		return fmt.Errorf("checking uplink allocation for network %s: %w", name, err)
	}
	var networks []allocatedNetwork
	if err := json.Unmarshal(raw, &networks); err != nil {
		return fmt.Errorf("decoding uplink allocations for network %s: %w", name, err)
	}
	// LXD reports the requested project on an exact GET, even if its networks
	// are inherited from default. Resolve that before comparing identities.
	if n.Project == "" {
		return fmt.Errorf("network %s read-back omitted its project; cannot verify its uplink allocation", name)
	}
	if n.Project != "default" {
		raw, err := client.get(ctx, "/1.0/projects/"+url.PathEscape(n.Project))
		if err != nil {
			return fmt.Errorf("resolving network project %s: %w", n.Project, err)
		}
		var project struct {
			Config map[string]string `json:"config"`
		}
		if err := json.Unmarshal(raw, &project); err != nil || project.Config == nil {
			return fmt.Errorf("cannot read network features for project %s", n.Project)
		}
		ownNetworks, _ := strconv.ParseBool(project.Config["features.networks"])
		if !ownNetworks {
			n.Project = "default"
		}
	}
	found := false
	for _, other := range networks {
		if other.Type == "ovn" && (other.Project == "" || other.Config["network"] == "") {
			return fmt.Errorf("cannot read uplink configuration for OVN network %s/%s; the builder identity needs access to network configuration across projects", other.Project, other.Name)
		}
		if other.Project == n.Project && other.Name == name {
			if other.Status != "Created" || other.Config["network"] != n.Config["network"] || other.Config["volatile.network.ipv4.address"] != address.String() {
				return fmt.Errorf("network %s changed during uplink allocation read-back; retry after cluster networking settles", name)
			}
			found = true
			continue
		}
		if other.Type != "ovn" || other.Config["network"] != n.Config["network"] {
			continue
		}
		otherAddress, _ := netip.ParseAddr(other.Config["volatile.network.ipv4.address"])
		if otherAddress == address {
			return fmt.Errorf("%w: OVN networks %s/%s and %s/%s share uplink %s IPv4 address %s; refusing guest deployment. Repair or recreate the conflicting networks with distinct uplink allocations before retrying", builder.ErrPermanentConfig, n.Project, name, other.Project, other.Name, n.Config["network"], address)
		}
	}
	if !found {
		return fmt.Errorf("network %s/%s is missing from the all-projects allocation read-back; cannot verify uplink %s IPv4 address %s", n.Project, name, n.Config["network"], address)
	}
	return nil
}
