package microcloud

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

// Team access open/close -- the MicroCloud sibling of the Incus builder's
// access.go, ported to MicroCloud's network model. We never touch an instance's
// NIC devices (rewriting devices forces Incus/LXD to rebuild the host-side
// veth/tap -- disruptive, and a hard failure on some VMs). Everything is done by
// editing network and ACL OBJECTS, which re-apply in place.
//
// Two paths, because a MicroCloud guest has two kinds of NIC:
//   - eth0 on the team's own OVN networks (intra). Egress is locked at the
//     NETWORK level, exactly like the Incus builder: default egress drop + an
//     lfnacl ACL whose broad "allow 0.0.0.0/0" egress rule is the open/close
//     switch, with the gateway and team CIDRs always allowed.
//   - lf-public (device eth1), the separate routed public NIC on the shared
//     public network. It can't be gated per-team at the network level (the
//     public network is shared), so its per-host public ACL (lfpub*) is the
//     lever: close disables its inbound port allows AND enables a broad egress
//     drop, leaving only the gateway + team CIDRs reachable. Because lf-public is
//     a routed NIC (not a nat-proxy), its ACL actually filters its traffic.
//
// Everything is rules-only: open/close flips rule state and the network default
// egress action, never an instance device, so no NIC is restarted.

// networkInfo is a team network's real Incus name and IPv4 CIDR.
type networkInfo struct {
	name string
	cidr string
}

// networkAccessACLName is the per-network egress access ACL (lfnacl), distinct
// from the per-host port (lfhacl) and public (lfpub) ACLs.
func networkAccessACLName(netName string) string {
	sum := sha256.Sum256([]byte(netName))
	return "lfnacl" + hex.EncodeToString(sum[:])[:12]
}

// CloseAccess locks a team to gateway + intra egress and disables its inbound
// published ports, best effort across the team's networks and public hosts. No
// instance NIC device is touched.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	return b.setAccess(ctx, team, true)
}

// OpenAccess restores a team's full egress and re-enables its inbound published
// ports.
func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	return b.setAccess(ctx, team, false)
}

// No lock is held across the team: each network change (PATCH /1.0/networks/…)
// takes the networking lock for itself in Client.do, and each ACL write in
// ensureACLObjectFull. Holding it across the whole loop -- reads of every
// instance included -- stalled every deploy on the cluster for as long as an
// open/close took, and at competition start that's every team at once.
func (b *Builder) setAccess(ctx context.Context, team string, closed bool) error {
	instances, err := b.teamInstances(ctx, team)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return nil
	}
	// Gateway allow is mandatory to close (losing it cuts telemetry); opening
	// re-enables the broad allow anyway, so a missing gateway must never wedge a
	// team shut -- tolerate it on open.
	gw, err := b.gatewayEgressRules(ctx)
	if err != nil {
		if closed {
			return err
		}
		gw = nil
	}
	nets := b.teamNetworks(ctx, instances)
	// Dedup CIDRs: with two environments running at once, same-numbered teams
	// share a per-team LXD project, so teamNetworks can gather two networks with
	// the SAME CIDR. An ACL built from duplicate CIDRs has duplicate egress rules,
	// which LXD rejects ("Duplicate of egress rule N") -- failing every open/close.
	cidrs := make([]string, 0, len(nets))
	seenCIDR := make(map[string]bool, len(nets))
	for _, n := range nets {
		if n.cidr != "" && !seenCIDR[n.cidr] {
			seenCIDR[n.cidr] = true
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
		if err := b.setPublicAccess(ctx, inst.Name, gw, cidrs, closed); err != nil {
			errs = append(errs, fmt.Errorf("instance %s public access: %w", inst.Name, err))
		}
	}
	return errors.Join(errs...)
}

// applyNetworkEgress sets a team OVN network's boundary policy -- identical to
// the Incus builder. Attaching a network ACL engages network-level ingress
// enforcement, so an egress-only ACL would drop ALL inbound; hence open DETACHES
// the ACL and restores default-allow egress (normal state), and close attaches
// an ACL allowing egress to gateway + team CIDRs and ingress from team CIDRs
// (intra; the agent reply is stateful), flipping the default egress to drop.
func (b *Builder) applyNetworkEgress(ctx context.Context, netName string, gatewayAllows []map[string]interface{}, teamCIDRs []string, closed bool) error {
	name := networkAccessACLName(netName)
	if !closed {
		if err := b.detachNetworkACL(ctx, netName, name); err != nil {
			return err
		}
		_, err := b.Client.patch(ctx, "/1.0/networks/"+url.PathEscape(netName), map[string]interface{}{
			"config": map[string]string{"security.acls.default.egress.action": "allow"},
		})
		return err
	}
	ingress := make([]map[string]interface{}, 0, len(teamCIDRs))
	egress := make([]map[string]interface{}, 0, len(gatewayAllows)+len(teamCIDRs))
	egress = append(egress, gatewayAllows...)
	for _, c := range teamCIDRs {
		ingress = append(ingress, map[string]interface{}{"action": "allow", "state": "enabled", "source": c})
		egress = append(egress, map[string]interface{}{"action": "allow", "state": "enabled", "destination": c})
	}
	if err := b.ensureACLObjectFull(ctx, name, "LaForge access lockdown for "+netName, dedupRules(ingress), dedupRules(egress)); err != nil {
		return err
	}
	if err := b.attachNetworkACL(ctx, netName, name); err != nil {
		return err
	}
	_, err := b.Client.patch(ctx, "/1.0/networks/"+url.PathEscape(netName), map[string]interface{}{
		"config": map[string]string{"security.acls.default.egress.action": "drop"},
	})
	return err
}

// detachNetworkACL removes name from the network's security.acls (idempotent),
// disengaging network-level ingress enforcement so inbound flows normally.
func (b *Builder) detachNetworkACL(ctx context.Context, netName, name string) error {
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
	kept := []string{}
	for _, a := range strings.Split(have, ",") {
		if a = strings.TrimSpace(a); a != "" && a != name {
			kept = append(kept, a)
		}
	}
	newVal := strings.Join(kept, ",")
	if newVal == strings.TrimSpace(have) {
		return nil
	}
	_, err = b.Client.patch(ctx, "/1.0/networks/"+url.PathEscape(netName), map[string]interface{}{
		"config": map[string]string{"security.acls": newVal},
	})
	return err
}

// setPublicAccess toggles a public host's lf-public NIC via its per-host public
// ACL (lfpub) -- rules only, the NIC is never touched. Close disables the inbound
// port allows and enables a broad egress drop (so only gateway + intra leave);
// open re-enables inbound and disables the broad drop. A host with no public NIC
// has no such ACL and is skipped.
func (b *Builder) setPublicAccess(ctx context.Context, name string, gatewayAllows []map[string]interface{}, teamCIDRs []string, closed bool) error {
	aclName := publicACLName(b.publicProject(), name)
	raw, err := b.Client.get(ctx, "/1.0/network-acls/"+aclName)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && (apiErr.HTTPStatus == 404 || apiErr.NotFound()) {
			return nil // no public NIC on this host
		}
		return err
	}
	var acl struct {
		Description string                   `json:"description"`
		Ingress     []map[string]interface{} `json:"ingress"`
		Egress      []map[string]interface{} `json:"egress"`
		Config      map[string]string        `json:"config"`
	}
	if err := json.Unmarshal(raw, &acl); err != nil {
		return err
	}
	// Inbound: flip the existing port-allow rules' state (preserve the ports).
	want := "enabled"
	if closed {
		want = "disabled"
	}
	for _, r := range acl.Ingress {
		r["state"] = want
	}
	// Egress: gateway + intra always allowed, broad drop toggled on when closed.
	acl.Egress = publicEgressRules(gatewayAllows, teamCIDRs, closed)
	_, err = b.Client.do(ctx, "PUT", "/1.0/network-acls/"+aclName, map[string]interface{}{
		"name":        aclName,
		"description": acl.Description,
		"ingress":     acl.Ingress,
		"egress":      acl.Egress,
		"config":      acl.Config,
	}, requestOptions{})
	return err
}

// publicEgressRules is the egress rule set for the lf-public NIC's per-host ACL.
// gateway + intra are always-enabled allows; the open/close switch is a broad
// drop-0.0.0.0/0 enabled only when closed. The public network is shared across
// teams, so we can't use a per-team network default here -- the specific
// gateway/intra allows take precedence over the broad drop. (Unverified live --
// MicroCloud has no active hosts.)
func publicEgressRules(gatewayAllows []map[string]interface{}, teamCIDRs []string, closed bool) []map[string]interface{} {
	rules := make([]map[string]interface{}, 0, len(gatewayAllows)+len(teamCIDRs)+1)
	rules = append(rules, gatewayAllows...)
	for _, c := range teamCIDRs {
		rules = append(rules, map[string]interface{}{"action": "allow", "state": "enabled", "destination": c})
	}
	st := "disabled"
	if closed {
		st = "enabled"
	}
	rules = append(rules, map[string]interface{}{"action": "drop", "state": st, "destination": "0.0.0.0/0"})
	return dedupRules(rules)
}

// dedupRules drops byte-identical ACL rules, preserving order. LXD rejects an
// ACL whose rule list contains a duplicate ("Duplicate of egress rule N"), so
// any path that could assemble the same rule twice (a gateway allow equal to a
// CIDR allow, overlapping networks on a shared per-team project) would otherwise
// fail the whole open/close. Keyed on the rule's canonical JSON.
func dedupRules(rules []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(rules))
	seen := make(map[string]bool, len(rules))
	for _, r := range rules {
		key, err := json.Marshal(r)
		if err != nil {
			out = append(out, r) // unmarshalable: keep it rather than drop silently
			continue
		}
		if seen[string(key)] {
			continue
		}
		seen[string(key)] = true
		out = append(out, r)
	}
	return out
}

// attachNetworkACL adds name to the network's security.acls (idempotent merge).
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
			return nil
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

// teamInstances is every instance tagged with this team AND, when a build scope
// is set, belonging to that build. A MicroCloud project is shared across
// environments, so filtering on team alone pulled in another environment's
// same-numbered team -- their identical CIDRs then produced duplicate ACL rules
// that LXD rejected, failing every open/close. Each instance's external-name
// config key is deterministically "build-<buildID>-team-...", so the build
// prefix scopes without a new config key or a redeploy of existing instances.
func (b *Builder) teamInstances(ctx context.Context, team string) ([]mcInstance, error) {
	all, err := b.ourInstances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]mcInstance, 0, len(all))
	for _, inst := range all {
		if inst.Config[teamConfigKey] != team {
			continue
		}
		if !inBuildScope(inst.Config[externalNameKey], b.buildScope) {
			continue
		}
		out = append(out, inst)
	}
	return out, nil
}

// inBuildScope reports whether an instance (by its deterministic external name,
// "build-<buildID>-team-...") belongs to buildScope. An empty scope matches
// everything (team-only, legacy/test behaviour).
func inBuildScope(externalName, buildScope string) bool {
	return buildScope == "" || strings.HasPrefix(externalName, "build-"+buildScope+"-")
}

// teamNetworks is the set of networks the given instances' eth0 NICs sit on,
// with each network's IPv4 CIDR. Best-effort.
func (b *Builder) teamNetworks(ctx context.Context, instances []mcInstance) []networkInfo {
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

// networkCIDR is a managed network's IPv4 subnet in canonical form, "" when none.
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
// allow per resolved IPv4 address. Errors rather than returning nothing.
func (b *Builder) gatewayEgressRules(ctx context.Context) ([]map[string]interface{}, error) {
	addr := strings.TrimSpace(b.Config.GatewayAddr)
	if addr == "" {
		return nil, fmt.Errorf("cannot close access: no gateway address configured (GATEWAY_PUBLIC_ADDR) -- refusing to close a team without preserving the agent's path to the gateway")
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = addr, "8443"
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
