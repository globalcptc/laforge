package microcloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/globalcptc/laforge/internal/builder"
)

// ConfigureNetworkAccess enforces a team's `visible_from` + `ports:` policy on
// MicroCloud/LXD's OVN, identically to the Incus builder (LXD and Incus share the
// OVN network/peer/ACL API) -- see the incus
// package's ConfigureNetworkAccess for the proven, live-verified mechanics:
// full-mesh peer the team's OVN networks (peering only within the team is the
// isolation boundary), set each network's default ingress to drop, then attach a
// per-host ACL at each instance's own NIC allowing only its declared ports from
// its own network (same-net) and its visible_from networks (cross-net). The NIC
// (to-lport) attachment is what filters same-switch traffic too, not just routed
// cross-network ingress. Idempotent. Mirrors incus; needs its own live smoke-test
// against a real MicroCloud (the mechanics are Incus-verified with real packets).
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	incusName := make(map[string]string, len(networks))
	cidr := make(map[string]string, len(networks))
	for _, na := range networks {
		incusName[na.ExternalName] = networkName(na.DisplayName, na.ExternalName)
		cidr[na.ExternalName] = na.CIDR
	}

	for _, a := range networks {
		for _, bnet := range networks {
			if a.ExternalName == bnet.ExternalName {
				continue
			}
			if err := b.ensurePeer(ctx, incusName[a.ExternalName], incusName[bnet.ExternalName]); err != nil {
				return fmt.Errorf("peering %s -> %s for team %s: %w", incusName[a.ExternalName], incusName[bnet.ExternalName], team, err)
			}
		}
	}

	for _, na := range networks {
		var allowCIDRs []string
		for _, from := range na.VisibleFrom {
			if c, ok := cidr[from]; ok && c != "" {
				allowCIDRs = append(allowCIDRs, c)
			}
		}
		// Network default-deny ingress (applies to every port, same-switch
		// included), allow egress; disable IPv6. No network-level ACL rules --
		// the allows live on each host's NIC below.
		if _, err := b.Client.patch(ctx, "/1.0/networks/"+incusName[na.ExternalName], map[string]interface{}{
			"config": map[string]string{
				"security.acls.default.ingress.action": "drop",
				"security.acls.default.egress.action":  "allow",
				"ipv6.address":                         "none",
			},
		}); err != nil {
			return fmt.Errorf("setting default-deny on %s for team %s: %w", incusName[na.ExternalName], team, err)
		}
		for _, h := range na.Hosts {
			if err := b.ensureHostPortACL(ctx, na.CIDR, allowCIDRs, h); err != nil {
				return fmt.Errorf("applying host port ACL for %s (team %s): %w", h.ExternalRef, team, err)
			}
		}
	}
	return nil
}

// teardownNetworkAccess removes a network's peers and clears its default-deny so
// it can be deleted (a peered OVN network can't be). The per-host NIC ACLs are
// removed with their instances in destroyInstance. Best-effort. Mirrors incus.
func (b *Builder) teardownNetworkAccess(ctx context.Context, netIncusName string) {
	if raw, err := b.Client.get(ctx, "/1.0/networks/"+netIncusName+"/peers"); err == nil {
		var urls []string
		if json.Unmarshal(raw, &urls) == nil {
			for _, u := range urls {
				b.Client.delete(ctx, u)
			}
		}
	}
	b.Client.patch(ctx, "/1.0/networks/"+netIncusName, map[string]interface{}{
		"config": map[string]string{
			"security.acls.default.ingress.action": "",
			"security.acls.default.egress.action":  "",
		},
	})
}

func peerName(targetIncusName string) string {
	sum := sha256.Sum256([]byte(targetIncusName))
	return "lfp" + hex.EncodeToString(sum[:])[:12]
}

// hostACLName is the deterministic ACL name for a single host's port firewall,
// applied at its instance NIC (to-lport) so it filters same-switch traffic.
func hostACLName(externalRef string) string {
	sum := sha256.Sum256([]byte(externalRef))
	return "lfhacl" + hex.EncodeToString(sum[:])[:12]
}

func (b *Builder) ensurePeer(ctx context.Context, from, to string) error {
	_, err := b.Client.post(ctx, "/1.0/networks/"+from+"/peers", map[string]interface{}{
		"name":           peerName(to),
		"target_network": to,
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			return nil
		}
		return err
	}
	return nil
}

// nicIngressRules is the per-host port firewall as ACL ingress rules attached at
// the host's own instance port (to-lport): one allow rule per source per
// protocol, scoping ingress to the declared ports. No `destination` (the port is
// the destination). A host with no ports yields nothing -> reachable on nothing.
func nicIngressRules(sources []string, h builder.HostAccess) []map[string]interface{} {
	rules := make([]map[string]interface{}, 0)
	for _, src := range sources {
		if len(h.TCPPorts) > 0 {
			rules = append(rules, map[string]interface{}{
				"action": "allow", "state": "enabled", "source": src,
				"protocol": "tcp", "destination_port": strings.Join(h.TCPPorts, ","),
			})
		}
		if len(h.UDPPorts) > 0 {
			rules = append(rules, map[string]interface{}{
				"action": "allow", "state": "enabled", "source": src,
				"protocol": "udp", "destination_port": strings.Join(h.UDPPorts, ","),
			})
		}
	}
	return rules
}

// ensureACLObject creates or replaces (idempotently) a network-ACL object.
func (b *Builder) ensureACLObject(ctx context.Context, name, description string, ingress []map[string]interface{}) error {
	acl := map[string]interface{}{
		"name":        name,
		"description": description,
		"ingress":     ingress,
		"egress":      []map[string]interface{}{},
		"config":      map[string]string{},
	}
	if _, err := b.Client.post(ctx, "/1.0/network-acls", acl); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			if _, err := b.Client.put(ctx, "/1.0/network-acls/"+name, acl); err != nil {
				return fmt.Errorf("updating ACL %s: %w", name, err)
			}
		} else {
			return fmt.Errorf("creating ACL %s: %w", name, err)
		}
	}
	return nil
}

// ensureHostPortACL enforces a host's `ports:` at its instance NIC so same-switch
// traffic is filtered too. Allows the declared ports from the host's own network
// (same-net) and the visible_from networks (cross-net); the network's default-deny
// drops everything else, including a same-net sibling on an undeclared port. A
// host that declared no ports gets an empty allow set: reachable on nothing.
func (b *Builder) ensureHostPortACL(ctx context.Context, ownCIDR string, visibleCIDRs []string, h builder.HostAccess) error {
	if h.ExternalRef == "" {
		return nil
	}
	name := hostACLName(h.ExternalRef)
	sources := append([]string{ownCIDR}, visibleCIDRs...)
	if err := b.ensureACLObject(ctx, name, "LaForge port firewall for "+h.ExternalRef, nicIngressRules(sources, h)); err != nil {
		return err
	}
	full, _, err := b.getInstancePut(ctx, h.ExternalRef)
	if err != nil {
		return fmt.Errorf("reading instance %s: %w", h.ExternalRef, err)
	}
	eth0 := map[string]interface{}{}
	if raw, ok := full.Devices["eth0"]; ok {
		if err := json.Unmarshal(raw, &eth0); err != nil {
			return fmt.Errorf("decoding eth0 of %s: %w", h.ExternalRef, err)
		}
	}
	if eth0["type"] == nil {
		eth0["type"] = "nic"
	}
	eth0["security.acls"] = name
	return retryOnBusy(ctx, func() error {
		_, err := b.Client.patch(ctx, "/1.0/instances/"+h.ExternalRef, map[string]interface{}{
			"devices": map[string]interface{}{"eth0": eth0},
		})
		return err
	})
}
