package incus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// Team access open/close.
//
// Closing a team is a NETWORK-BOUNDARY policy -- what may cross in/out of the
// team's environment -- so it is applied with ACLs on the team's NETWORKS and on
// each host's existing public-port ACL, never by touching an instance's eth0
// device. (Rewriting devices.eth0 forces Incus to stop/restart the host-side
// veth/tap, which is disruptive on bridge networks and failed outright on VMs
// with a null host-side device name. Network-ACL and ACL-object edits re-apply
// the firewall in place -- no device is ever reconfigured or stopped.)
//
// The rule set is PROVISIONED once and then only toggled by flipping rule state,
// never added/removed/deleted:
//
//   - Each team network's default egress action is set to drop (a fail-closed
//     baseline), and an access ACL (lfnacl) is attached carrying egress ALLOWs
//     for the pinned gateway and each team CIDR (always enabled) plus one broad
//     "allow 0.0.0.0/0" egress rule that is the open/close switch.
//   - Each public host's existing public-port ACL (lfpacl) carries the inbound
//     allows; its rules are the inbound switch.
//
// Close disables the broad egress allow and the inbound allows: unmatched egress
// then hits the network's default drop (only the gateway + intra stay reachable,
// so an implant cannot phone home or pivot) and inbound published ports are no
// longer allowed, so the network's default-deny ingress drops them. Open
// re-enables both. No allow-vs-drop precedence is relied on (every enabled rule
// is an allow; the default action does the denying), which matters because this
// Incus (7.5.1) has no per-rule priority.

// closedProxiesKey holds, as JSON, the public-forward proxy devices removed on
// close, so open re-adds them exactly. Kept on the instance itself (an opaque
// user.* config value) so this builder needs no durable state of its own.
const closedProxiesKey = "user.laforge_closed_proxies"

// networkInfo is a team network's real Incus name and IPv4 CIDR.
type networkInfo struct {
	name string
	cidr string
}

// networkAccessACLName is the deterministic egress access ACL for one network --
// "lfnacl" so it's distinct from the per-host port (lfhacl) and public (lfpacl)
// ACLs.
func networkAccessACLName(netName string) string {
	sum := sha256.Sum256([]byte(netName))
	return "lfnacl" + hex.EncodeToString(sum[:])[:12]
}

// CloseAccess locks a team to gateway + intra egress and disables its inbound
// published ports, best effort across the team's networks and hosts.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	return b.setAccess(ctx, team, true)
}

// OpenAccess restores a team's full egress and re-enables its inbound published
// ports.
func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	return b.setAccess(ctx, team, false)
}

func (b *Builder) setAccess(ctx context.Context, team string, closed bool) error {
	instances, err := b.teamInstances(ctx, team)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return nil
	}
	// The gateway allow is mandatory to close (losing it would cut telemetry).
	// Opening doesn't need it -- the broad allow is re-enabled anyway -- so a
	// missing gateway address must never wedge a team shut: tolerate it on open.
	gw, err := b.gatewayEgressRules(ctx)
	if err != nil {
		if closed {
			return err
		}
		gw = nil
	}
	nets := b.teamNetworks(ctx, instances)
	cidrs := make([]string, 0, len(nets))
	for _, n := range nets {
		if n.cidr != "" {
			cidrs = append(cidrs, n.cidr)
		}
	}
	var errs []error
	for _, n := range nets {
		if err := b.applyNetworkEgress(ctx, n.name, gw, cidrs, closed); err != nil {
			errs = append(errs, fmt.Errorf("network %s: %w", n.name, err))
		}
	}
	for _, inst := range instances {
		if err := b.setInstanceInbound(ctx, inst.Name, !closed); err != nil {
			errs = append(errs, fmt.Errorf("instance %s inbound: %w", inst.Name, err))
		}
	}
	return errors.Join(errs...)
}

// applyNetworkEgress (re)writes the network's access ACL and asserts the
// fail-closed default-drop egress baseline. The gateway + intra allows are
// always enabled; the broad 0.0.0.0/0 allow is enabled when open, disabled when
// closed -- that single rule is the egress switch.
func (b *Builder) applyNetworkEgress(ctx context.Context, netName string, gatewayAllows []map[string]interface{}, teamCIDRs []string, closed bool) error {
	egress := make([]map[string]interface{}, 0, len(gatewayAllows)+len(teamCIDRs)+1)
	egress = append(egress, gatewayAllows...)
	for _, c := range teamCIDRs {
		egress = append(egress, map[string]interface{}{"action": "allow", "state": "enabled", "destination": c})
	}
	broad := "enabled"
	if closed {
		broad = "disabled"
	}
	egress = append(egress, map[string]interface{}{"action": "allow", "state": broad, "destination": "0.0.0.0/0"})

	name := networkAccessACLName(netName)
	if err := b.ensureACLObjectFull(ctx, name, "LaForge access control for "+netName, nil, egress); err != nil {
		return err
	}
	if err := b.attachNetworkACL(ctx, netName, name); err != nil {
		return err
	}
	// Fail-closed baseline: unmatched egress is dropped, so a broken toggle locks
	// a team down (gateway-only) rather than leaving it wide open. Idempotent.
	_, err := b.Client.patch(ctx, "/1.0/networks/"+url.PathEscape(netName), map[string]interface{}{
		"config": map[string]string{"security.acls.default.egress.action": "drop"},
	})
	return err
}

// setInstanceInbound blocks (open=false) or restores (open=true) a host's public
// inbound by removing or re-adding its NAT proxy forward devices (lfx-*). On
// this hoster the nat-proxy forward is NOT filtered by NIC or network ACLs
// (verified live), so removing the forward device is the only thing that
// actually stops inbound to a published port. The removed devices are saved on
// the instance (closedProxiesKey) so open re-adds them exactly. eth0 is never
// touched, and a VM is not restarted (verified live: status stayed Running).
func (b *Builder) setInstanceInbound(ctx context.Context, name string, open bool) error {
	return retryOnBusy(ctx, func() error {
		full, _, err := b.getInstancePut(ctx, name)
		if err != nil {
			return err
		}
		if full.Devices == nil {
			full.Devices = map[string]json.RawMessage{}
		}
		if open {
			raw, ok := full.Config[closedProxiesKey]
			if !ok {
				return nil // not closed -- idempotent
			}
			var saved map[string]json.RawMessage
			if err := json.Unmarshal([]byte(raw), &saved); err != nil {
				return fmt.Errorf("decoding saved proxy forwards: %w", err)
			}
			for dn, dv := range saved {
				full.Devices[dn] = dv
			}
			delete(full.Config, closedProxiesKey)
		} else {
			if _, already := full.Config[closedProxiesKey]; already {
				return nil // already closed -- don't overwrite the saved set
			}
			removed := map[string]json.RawMessage{}
			for dn, dv := range full.Devices {
				if !strings.HasPrefix(dn, "lfx-") {
					continue
				}
				var d map[string]interface{}
				if err := json.Unmarshal(dv, &d); err != nil || d["type"] != "proxy" {
					continue
				}
				removed[dn] = dv
			}
			if len(removed) == 0 {
				return nil // no public forwards on this host -- nothing to block
			}
			for dn := range removed {
				delete(full.Devices, dn)
			}
			savedJSON, err := json.Marshal(removed)
			if err != nil {
				return err
			}
			if full.Config == nil {
				full.Config = map[string]string{}
			}
			full.Config[closedProxiesKey] = string(savedJSON)
		}
		_, err = b.Client.put(ctx, "/1.0/instances/"+name, full)
		return err
	})
}

// attachNetworkACL adds name to the network's security.acls (idempotent merge),
// leaving the network's default actions untouched beyond applyNetworkEgress's own
// drop baseline. A network config PATCH does not restart instance NICs.
func (b *Builder) attachNetworkACL(ctx context.Context, netName, name string) error {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+url.PathEscape(netName))
	if err != nil {
		return err
	}
	var net struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &net); err != nil {
		return err
	}
	have := net.Config["security.acls"]
	for _, a := range strings.Split(have, ",") {
		if strings.TrimSpace(a) == name {
			return nil // already attached
		}
	}
	names := name
	if strings.TrimSpace(have) != "" {
		names = have + "," + name
	}
	_, err = b.Client.patch(ctx, "/1.0/networks/"+url.PathEscape(netName), map[string]interface{}{
		"config": map[string]string{"security.acls": names},
	})
	return err
}

// teamInstances is every instance tagged with this team.
func (b *Builder) teamInstances(ctx context.Context, team string) ([]incusInstance, error) {
	all, err := b.ourInstances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]incusInstance, 0, len(all))
	for _, inst := range all {
		if inst.Config[teamConfigKey] == team {
			out = append(out, inst)
		}
	}
	return out, nil
}

// teamNetworks is the set of networks the given instances' eth0 NICs sit on,
// with each network's IPv4 CIDR. Best-effort: an instance or network we can't
// read is skipped.
func (b *Builder) teamNetworks(ctx context.Context, instances []incusInstance) []networkInfo {
	seen := map[string]bool{}
	var out []networkInfo
	for _, inst := range instances {
		_, expanded, err := b.getInstancePut(ctx, inst.Name)
		if err != nil {
			continue
		}
		eth0, ok := expanded["eth0"].(map[string]interface{})
		if !ok {
			continue
		}
		n, _ := eth0["network"].(string)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, networkInfo{name: n, cidr: b.networkCIDR(ctx, n)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// networkCIDR is a managed network's IPv4 subnet in canonical form
// ("10.0.1.0/24"), "" when it has no IPv4 subnet or can't be read.
func (b *Builder) networkCIDR(ctx context.Context, name string) string {
	raw, err := b.Client.get(ctx, "/1.0/networks/"+url.PathEscape(name))
	if err != nil {
		return ""
	}
	var n struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		return ""
	}
	addr := n.Config["ipv4.address"]
	if addr == "" || addr == "none" {
		return ""
	}
	_, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		return ""
	}
	return ipnet.String()
}

// gatewayEgressRules resolves the pinned gateway host:port to one TCP egress
// allow per resolved IPv4 address. Errors rather than returning nothing: a close
// that silently dropped the gateway allow would cut all telemetry.
func (b *Builder) gatewayEgressRules(ctx context.Context) ([]map[string]interface{}, error) {
	addr := strings.TrimSpace(b.Config.GatewayAddr)
	if addr == "" {
		return nil, fmt.Errorf("cannot close access: no gateway address configured (GATEWAY_PUBLIC_ADDR) -- refusing to close a team without preserving the agent's path to the gateway")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, "8443" // no :port given; the gateway's default
	}
	var ips []string
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			ips = append(ips, host)
		}
	} else {
		resolved, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolving gateway host %q for the closed-access egress allow: %w", host, err)
		}
		for _, a := range resolved {
			if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
				ips = append(ips, a)
			}
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("gateway host %q resolved to no IPv4 address for the closed-access egress allow", host)
	}
	sort.Strings(ips)
	rules := make([]map[string]interface{}, 0, len(ips))
	for _, ip := range ips {
		rules = append(rules, map[string]interface{}{
			"action": "allow", "state": "enabled",
			"destination": ip + "/32", "protocol": "tcp", "destination_port": port,
		})
	}
	return rules, nil
}
