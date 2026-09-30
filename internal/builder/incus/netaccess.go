package incus

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

// ConfigureNetworkAccess enforces a team's `visible_from` policy on Incus/OVN,
// exactly as proven live: full-mesh peer the team's OVN networks so they route (peering only
// within the team is the team-isolation boundary -- CIDRs are identical across
// teams, so nothing else may be), then give each network an ACL that denies
// cross-network ingress by default and allows ingress only from the CIDRs of the
// networks in its VisibleFrom, further scoped to each host's declared `ports:` (a
// host is reachable from an allowed sibling only on the TCP/UDP ports it lists; a
// host with no ports on none). Same-network traffic and egress (internet, the
// LaForge gateway) are unaffected -- verified: OVN's default-deny ingress does not
// touch same-switch traffic, so the port firewall applies to cross-network
// (routed) ingress; egress defaults to allow. Idempotent: peers and ACLs are
// ensured, re-runs converge.
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	// The real Incus names + CIDRs, keyed by the ExternalName the caller used.
	incusName := make(map[string]string, len(networks))
	cidr := make(map[string]string, len(networks))
	for _, na := range networks {
		incusName[na.ExternalName] = networkName(na.DisplayName, na.ExternalName)
		cidr[na.ExternalName] = na.CIDR
	}

	// 1. Full-mesh peering: every ordered pair, so both half-peers exist and the
	//    peering completes (OVN needs a peer object on each side).
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

	// 2. Per-network ACL: default-deny ingress, then per-host port allow rules
	//    scoped to each VisibleFrom source -- a host is reachable from an allowed
	//    sibling network only on its declared TCP/UDP ports (and a host with no
	//    ports on none).
	for _, na := range networks {
		var allowCIDRs []string
		for _, from := range na.VisibleFrom {
			c, ok := cidr[from]
			if !ok || c == "" {
				// A visible_from naming a network not in this set is a content
				// error the loader already reports; skip it here rather than fail
				// the whole team's access config.
				continue
			}
			allowCIDRs = append(allowCIDRs, c)
		}
		// Set the network's DEFAULT ingress to drop (so every instance port
		// default-denies, same-switch included), but attach no network-level ACL
		// rules: the allow rules live on each host's own NIC below. Egress
		// default allow; disable IPv6 (see DeployNetwork).
		if _, err := b.Client.patch(ctx, "/1.0/networks/"+incusName[na.ExternalName], map[string]interface{}{
			"config": map[string]string{
				"security.acls.default.ingress.action": "drop",
				"security.acls.default.egress.action":  "allow",
				"ipv6.address":                         "none",
			},
		}); err != nil {
			return fmt.Errorf("setting default-deny on %s for team %s: %w", incusName[na.ExternalName], team, err)
		}
		// Per-host NIC ACL: the port firewall at each host's own instance port
		// (to-lport), so BOTH routed cross-network and same-switch same-network
		// ingress are filtered. Same-net siblings may reach declared ports (own
		// CIDR is an allowed source); everything else at the port is dropped.
		for _, h := range na.Hosts {
			if err := b.ensureHostPortACL(ctx, na.CIDR, allowCIDRs, h); err != nil {
				return fmt.Errorf("applying host port ACL for %s (team %s): %w", h.ExternalRef, team, err)
			}
		}
	}
	return nil
}

// teardownNetworkAccess removes a network's visible_from fabric so it can be
// deleted: every peer on it, then its ACL (detached first). Best-effort -- each
// step ignores not-found, since teardown must converge even from a partial
// state. Called by DestroyNetwork.
func (b *Builder) teardownNetworkAccess(ctx context.Context, netIncusName string) {
	// Delete every peer this network has.
	if raw, err := b.Client.get(ctx, "/1.0/networks/"+netIncusName+"/peers"); err == nil {
		var urls []string
		if json.Unmarshal(raw, &urls) == nil {
			for _, u := range urls {
				// u is a URL like /1.0/networks/<net>/peers/<name>; delete it directly.
				b.Client.delete(ctx, u)
			}
		}
	}
	// Clear the network's default-deny config so it can be deleted cleanly (the
	// per-host NIC ACLs are removed with their instances in destroyInstance).
	b.Client.patch(ctx, "/1.0/networks/"+netIncusName, map[string]interface{}{
		"config": map[string]string{
			"security.acls.default.ingress.action": "",
			"security.acls.default.egress.action":  "",
		},
	})
}

// peerName is the deterministic, per-source-network name of the peer object that
// points at target -- unique among a network's peers and stable across re-runs.
func peerName(targetIncusName string) string {
	sum := sha256.Sum256([]byte(targetIncusName))
	return "lfp" + hex.EncodeToString(sum[:])[:12]
}

// nicIngressRules is the per-host port firewall as OVN/LXD ACL ingress rules,
// attached at the host's own instance port (to-lport): one allow rule per source
// per protocol, scoping ingress to the host's declared ports. No `destination`
// (the instance port itself is the destination). A host with no ports yields no
// rule, so with the network's default-deny it is reachable on nothing.
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

// ensurePeer creates the half-peer on `from` pointing at `to`, idempotently.
func (b *Builder) ensurePeer(ctx context.Context, from, to string) error {
	_, err := b.Client.post(ctx, "/1.0/networks/"+from+"/peers", map[string]interface{}{
		"name":           peerName(to),
		"target_network": to,
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			return nil // already peered
		}
		return err
	}
	return nil
}

// ensureACLObject creates or replaces (idempotently) a network-ACL object with
// the given ingress allow rules and an empty egress (the default-allow egress is
// set where the ACL is attached). PUT is a full replace, so a changed policy
// converges on re-run.
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

// hostACLName is the deterministic ACL name for a single host's own port
// firewall, applied at its instance NIC (to-lport), so it filters same-switch
// traffic the network-level ACL can't reach.
func hostACLName(externalRef string) string {
	sum := sha256.Sum256([]byte(externalRef))
	return "lfhacl" + hex.EncodeToString(sum[:])[:12]
}

// ensureHostPortACL enforces a host's `ports:` at its own instance port, which
// is what makes SAME-network (same-switch) traffic subject to the firewall --
// a network-level ACL only filters routed (cross-network) ingress. The per-host
// ACL allows the host's declared ports from its own network (same-net siblings)
// and from the visible_from networks (cross-net), and is attached to eth0 with a
// NIC-level default-deny ingress, so everything else -- including a same-net
// sibling hitting an undeclared port -- is dropped at delivery. A host that
// declared no ports gets an empty allow set: reachable on nothing.
func (b *Builder) ensureHostPortACL(ctx context.Context, ownCIDR string, visibleCIDRs []string, h builder.HostAccess) error {
	if h.ExternalRef == "" {
		return nil // not yet deployed / no instance to attach to; skip this pass
	}
	name := hostACLName(h.ExternalRef)
	// Same-net (own network) + cross-net (visible_from) sources, scoped to this
	// host's declared ports. At a NIC (to-lport) the instance's own port IS the
	// destination, so the rules carry no `destination` -- only source + port.
	sources := append([]string{ownCIDR}, visibleCIDRs...)
	if err := b.ensureACLObject(ctx, name, "LaForge port firewall for "+h.ExternalRef, nicIngressRules(sources, h)); err != nil {
		return err
	}
	// Attach at the instance's eth0 with a NIC-level default-deny ingress. GET
	// the current instance-level eth0 device and merge the ACL keys in, so the
	// network/ipv4.address config deployInstance set stays intact.
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
