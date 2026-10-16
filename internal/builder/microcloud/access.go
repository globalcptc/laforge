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
// access.go, and for the same reason: the old mechanism detached eth0 (and the
// public NIC) to close a team, which cut agent telemetry and tore the host off
// its networks through the pre-competition window. Closing is now done with
// network ACLs on the NICs that STAY attached.
//
// A MicroCloud guest can have two NICs: eth0 on the team's OVN network, and
// lf-public (the separate public NIC) carrying the internet default route when
// the host declares `public:` ports. Either may be the agent's path to the
// gateway, so BOTH are firewalled identically on close: egress default-deny
// allowing ONLY the pinned gateway and the team's own environment CIDRs, ingress
// default-deny with the public-port ACL detached. eth0 keeps its intra host ACL
// so same-environment traffic (internal DNS, internal services) survives. The
// pre-close NIC state is saved per device and restored on open.
const (
	// closedAccessKey saves eth0's pre-close NIC state (ACL list + default
	// actions) as JSON; publicClosedAccessKey does the same for lf-public.
	// Distinct from the legacy removedNICKey/publicClosedKey (device masking),
	// which openNIC still understands so a team closed by an older build reopens.
	closedAccessKey       = "user.laforge_closed_access"
	publicClosedAccessKey = "user.laforge_closed_public_access"
)

// closedNICDevices maps each NIC device to the config key holding its saved
// pre-close state, in a deterministic order (eth0 first).
func closedNICDevices() []struct{ device, key string } {
	return []struct{ device, key string }{
		{"eth0", closedAccessKey},
		{publicNICName, publicClosedAccessKey},
	}
}

// closedAccess is what gets saved under a closed-access key: enough of a NIC's
// pre-close state to put it back exactly on open.
type closedAccess struct {
	ACLs          string `json:"acls"`
	IngressAction string `json:"ingress_action"`
	EgressAction  string `json:"egress_action"`
	// FromProfile is true when the device had no instance-level entry before
	// close (it came from a profile); open deletes the override close created
	// rather than writing a half-remembered device back.
	FromProfile bool `json:"from_profile"`
}

// closedACLName is the deterministic name of an instance's closed-egress ACL.
func closedACLName(name string) string {
	sum := sha256.Sum256([]byte(name))
	return "lfcacl" + hex.EncodeToString(sum[:])[:12]
}

// CloseAccess firewalls every one of a team's instances: no new external
// ingress, and no egress except the pinned gateway and the team's own
// environment. Best effort across the whole team; the NICs stay attached.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	ctx, release, err := b.Client.lockNetworking(ctx)
	if err != nil {
		return err
	}
	defer release()
	instances, err := b.teamInstances(ctx, team)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return nil
	}
	egress, err := b.closedEgressRules(ctx, instances)
	if err != nil {
		return err
	}
	var errs []error
	for _, inst := range instances {
		if err := b.closeNIC(ctx, inst.Name, egress); err != nil {
			errs = append(errs, fmt.Errorf("instance %s: %w", inst.Name, err))
		}
	}
	return errors.Join(errs...)
}

// OpenAccess restores every one of a team's instances to its pre-close NIC
// state, best effort across the whole team.
func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	ctx, release, err := b.Client.lockNetworking(ctx)
	if err != nil {
		return err
	}
	defer release()
	return b.forEachTeamInstance(ctx, team, func(name string, _ mcInstance) error {
		return b.openNIC(ctx, name)
	})
}

// teamInstances is every instance tagged with this team.
func (b *Builder) teamInstances(ctx context.Context, team string) ([]mcInstance, error) {
	all, err := b.ourInstances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]mcInstance, 0, len(all))
	for _, inst := range all {
		if inst.Config[teamConfigKey] == team {
			out = append(out, inst)
		}
	}
	return out, nil
}

// forEachTeamInstance applies fn to every instance of a team, BEST EFFORT: it
// does not stop at the first failure, so one wedged instance can't leave the
// rest of a team open (on close) or stranded (on open).
func (b *Builder) forEachTeamInstance(ctx context.Context, team string, fn func(name string, inst mcInstance) error) error {
	instances, err := b.teamInstances(ctx, team)
	if err != nil {
		return err
	}
	var errs []error
	for _, inst := range instances {
		if err := fn(inst.Name, inst); err != nil {
			errs = append(errs, fmt.Errorf("instance %s: %w", inst.Name, err))
		}
	}
	return errors.Join(errs...)
}

// closedEgressRules is the egress allow-list a closed team keeps: the pinned
// gateway (so agents still check in) plus every environment CIDR the team's
// instances sit on (internal DNS + internal services). Everything else is denied
// by the NIC's default-deny egress.
func (b *Builder) closedEgressRules(ctx context.Context, instances []mcInstance) ([]map[string]interface{}, error) {
	gw, err := b.gatewayEgressRules(ctx)
	if err != nil {
		return nil, err
	}
	cidrs := b.teamNetworkCIDRs(ctx, instances)
	rules := make([]map[string]interface{}, 0, len(gw)+len(cidrs))
	rules = append(rules, gw...)
	for _, c := range cidrs {
		rules = append(rules, map[string]interface{}{
			"action": "allow", "state": "enabled", "destination": c,
		})
	}
	return rules, nil
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

// teamNetworkCIDRs is the set of environment CIDRs the given instances' eth0
// NICs sit on. Best-effort: an instance or network we can't read is skipped.
func (b *Builder) teamNetworkCIDRs(ctx context.Context, instances []mcInstance) []string {
	nets := map[string]bool{}
	for _, inst := range instances {
		_, expanded, err := b.getInstancePut(ctx, inst.Name)
		if err != nil {
			continue
		}
		if eth0, ok := expanded["eth0"].(map[string]interface{}); ok {
			if n, _ := eth0["network"].(string); n != "" {
				nets[n] = true
			}
		}
	}
	var cidrs []string
	for n := range nets {
		if c := b.networkCIDR(ctx, n); c != "" {
			cidrs = append(cidrs, c)
		}
	}
	sort.Strings(cidrs)
	return cidrs
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

// closeNIC firewalls one instance's NICs. It ensures the instance's
// closed-egress ACL exists, then (ETag-guarded via retryOnBusy) rewrites each
// active NIC -- eth0 and, when present, lf-public -- to deny new ingress and all
// egress except that ACL's allow-list. The public-port ACL is stripped from the
// public NIC; eth0 keeps its intra host ACL. Original NIC state is saved per
// device on the FIRST close only.
func (b *Builder) closeNIC(ctx context.Context, name string, egress []map[string]interface{}) error {
	aclName := closedACLName(name)
	if err := b.ensureACLObjectFull(ctx, aclName, "LaForge closed access for "+name, nil, egress); err != nil {
		return err
	}
	return retryOnBusy(ctx, func() error {
		full, expanded, err := b.getInstancePut(ctx, name)
		if err != nil {
			return err
		}
		config := map[string]string{}
		devices := map[string]interface{}{}
		for _, d := range closedNICDevices() {
			nic, fromProfile, ok := effectiveNIC(full, expanded, d.device)
			if !ok {
				continue // no such active NIC on this instance
			}
			if _, already := full.Config[d.key]; !already {
				saved := closedAccess{
					ACLs:          nic["security.acls"],
					IngressAction: nic["security.acls.default.ingress.action"],
					EgressAction:  nic["security.acls.default.egress.action"],
					FromProfile:   fromProfile,
				}
				encoded, err := json.Marshal(saved)
				if err != nil {
					return err
				}
				config[d.key] = string(encoded)
			}
			// Keep the intra host ACL (lfhacl*), drop the public-port ACL
			// (lfpub*), add the closed-egress ACL; default-deny both directions.
			kept := []string{}
			for _, a := range strings.Split(nic["security.acls"], ",") {
				a = strings.TrimSpace(a)
				if a == "" || a == aclName || strings.HasPrefix(a, "lfpub") {
					continue
				}
				kept = append(kept, a)
			}
			nic["security.acls"] = strings.Join(append(kept, aclName), ",")
			nic["security.acls.default.ingress.action"] = "drop"
			nic["security.acls.default.egress.action"] = "drop"
			devices[d.device] = nic
		}
		if len(devices) == 0 {
			return nil // no NIC to firewall
		}
		body := map[string]interface{}{"devices": devices}
		if len(config) > 0 {
			body["config"] = config
		}
		_, err = b.Client.do(ctx, "PATCH", "/1.0/instances/"+name, body, requestOptions{ifMatch: full.etag})
		return err
	})
}

// openNIC restores one instance's pre-close NIC state for both devices. It
// understands the current ACL-based close (closedAccessKey/publicClosedAccessKey)
// and the legacy device-masking close (removedNICKey/publicClosedKey), so a team
// closed by an older build still reopens. A full PUT is used so config keys can
// be deleted outright.
func (b *Builder) openNIC(ctx context.Context, name string) error {
	return retryOnBusy(ctx, func() error {
		full, _, err := b.getInstancePut(ctx, name)
		if err != nil {
			return err
		}
		changed := false

		// Current ACL-based close.
		for _, d := range closedNICDevices() {
			rawSaved, ok := full.Config[d.key]
			if !ok {
				continue
			}
			var saved closedAccess
			if err := json.Unmarshal([]byte(rawSaved), &saved); err != nil {
				return fmt.Errorf("decoding closed access state for %s: %w", d.device, err)
			}
			if saved.FromProfile {
				delete(full.Devices, d.device)
			} else if dev, ok := full.Devices[d.device]; ok {
				var m map[string]interface{}
				if err := json.Unmarshal(dev, &m); err != nil {
					return fmt.Errorf("decoding %s for restore: %w", d.device, err)
				}
				setOrDelete(m, "security.acls", saved.ACLs)
				setOrDelete(m, "security.acls.default.ingress.action", saved.IngressAction)
				setOrDelete(m, "security.acls.default.egress.action", saved.EgressAction)
				restored, err := json.Marshal(m)
				if err != nil {
					return err
				}
				full.Devices[d.device] = restored
			}
			delete(full.Config, d.key)
			changed = true
		}

		// Legacy device-masking close.
		for device, key := range map[string]string{"eth0": removedNICKey, publicNICName: publicClosedKey} {
			rawSaved, ok := full.Config[key]
			if !ok {
				continue
			}
			var saved savedNIC
			if err := json.Unmarshal([]byte(rawSaved), &saved); err != nil {
				return fmt.Errorf("decoding legacy saved nic state for %s: %w", device, err)
			}
			if saved.Instance {
				raw, err := json.Marshal(saved.Device)
				if err != nil {
					return err
				}
				full.Devices[device] = raw
			} else {
				delete(full.Devices, device)
			}
			delete(full.Config, key)
			changed = true
		}

		if !changed {
			return nil // already open -- idempotent
		}
		_, err = b.Client.do(ctx, "PUT", "/1.0/instances/"+name, full, requestOptions{ifMatch: full.etag})
		return err
	})
}

// effectiveNIC returns the device's in-effect NIC config (instance-level if
// present, else the profile's cloned into an instance override), whether it came
// from a profile, and whether it's a real active NIC. ok is false when there's
// no such device or it isn't a nic.
func effectiveNIC(full instancePut, expanded map[string]interface{}, device string) (nic map[string]string, fromProfile bool, ok bool) {
	if raw, has := full.Devices[device]; has {
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, false, false
		}
		if m["type"] != "nic" {
			return nil, false, false
		}
		return m, false, true
	}
	if e, has := expanded[device].(map[string]interface{}); has {
		m := map[string]string{}
		for k, v := range e {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
		if m["type"] != "nic" {
			return nil, false, false
		}
		return m, true, true
	}
	return nil, false, false
}

// setOrDelete sets key to val, or removes it when val is "".
func setOrDelete(m map[string]interface{}, key, val string) {
	if val == "" {
		delete(m, key)
		return
	}
	m[key] = val
}
