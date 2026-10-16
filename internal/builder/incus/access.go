package incus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Team access open/close.
//
// The old mechanism detached eth0 entirely to close a team. That was wrong: on
// this hoster eth0 is also how a guest's agent reaches the gateway (and how the
// host sits behind its NATs), so detaching it cut all telemetry the moment a
// team closed and tore the host off its network -- exactly through the
// pre-competition window when we most need the agents checking in. Removing a
// NIC is also irreversible from the guest's point of view (DHCP leases, routes).
//
// Closing is now done with network ACLs on the NIC that STAYS attached:
//
//   - Egress default-deny, allowing ONLY the pinned gateway (host:port the
//     agents dial) and the team's own environment CIDRs. That keeps agent
//     telemetry and intra-environment traffic (internal DNS resolves the gateway
//     name, internal services still talk) alive, while denying a team any other
//     outbound path -- so a planted implant cannot phone home or pivot out. The
//     gateway is the ONLY off-environment egress a closed team keeps.
//   - Ingress default-deny with the public-port ACLs detached, so no new inbound
//     from the public NAT or another team is accepted. The ACLs are stateful, so
//     replies to the agent's own gateway/intra connections still return; flipping
//     the matching allow away also drops packets on already-established external
//     flows, satisfying "terminate established connections, not merely stop new
//     ones" without ripping out the interface.
//
// Opening restores the exact NIC state saved at close (the original ACL list and
// default actions), so the team's `ports:`/`visible_from`/`public:` policy comes
// back as it was.

// closedAccessKey holds, as JSON, the NIC state CloseAccess must restore on open
// -- the original security.acls list and default ingress/egress actions -- in a
// user.* config key on the instance itself, so OpenAccess needs no durable state
// of its own. Distinct from the legacy removedNICKey (device masking), which
// OpenAccess still understands so a team closed by an older build still reopens.
const closedAccessKey = "user.laforge_closed_access"

// closedAccess is what gets saved under closedAccessKey.
type closedAccess struct {
	// ACLs is the NIC's security.acls value (comma-separated ACL names) as it
	// was before close -- restored verbatim on open.
	ACLs string `json:"acls"`
	// IngressAction/EgressAction are the NIC's default action overrides before
	// close ("" when unset) -- restored (or removed when "") on open.
	IngressAction string `json:"ingress_action"`
	EgressAction  string `json:"egress_action"`
	// FromProfile is true when eth0 had no instance-level device before close
	// (it came from a profile); close created an instance-level override, so
	// open deletes that override to fall back to the profile rather than
	// writing a half-remembered device back.
	FromProfile bool `json:"from_profile"`
}

// closedACLName is the deterministic name of an instance's closed-egress ACL --
// a sibling of hostACLName/publicACLName so all three are easy to tell apart on
// the hoster (lfhacl = host ports, lfpacl = public ports, lfcacl = closed).
func closedACLName(name string) string {
	return "lfcacl" + strings.TrimPrefix(hostACLName(name), "lfhacl")
}

// CloseAccess blocks every one of a team's instances from all external ingress
// and all egress except the pinned gateway and the team's own environment, best
// effort across the whole team (see forEachTeamInstance). eth0 stays attached.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
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
	return b.forEachTeamInstance(ctx, team, func(name string, _ incusInstance) error {
		return b.openNIC(ctx, name)
	})
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

// forEachTeamInstance applies fn to every instance of a team, BEST EFFORT: it
// does not stop at the first failure, so one wedged instance can't leave the
// rest of a team open (on close) or stranded (on open). Every instance is
// attempted and the errors are joined into one.
func (b *Builder) forEachTeamInstance(ctx context.Context, team string, fn func(name string, inst incusInstance) error) error {
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
// instances sit on (so internal DNS and internal services survive). Everything
// else is denied by the NIC's default-deny egress.
func (b *Builder) closedEgressRules(ctx context.Context, instances []incusInstance) ([]map[string]interface{}, error) {
	gw, err := b.gatewayEgressRules(ctx)
	if err != nil {
		return nil, err
	}
	// Intra-environment egress is best-effort: a network we can't read just
	// isn't added (the gateway allow, the one that must not be lost, already
	// errored above if it couldn't be pinned).
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

// gatewayEgressRules resolves the pinned gateway host:port to one allow rule per
// resolved IPv4 address, on that TCP port. Re-resolved on every close so a
// gateway DNS change is picked up. Errors rather than returning nothing: a close
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

// teamNetworkCIDRs is the set of environment CIDRs the given instances' eth0
// NICs sit on. Best-effort: an instance or network we can't read is skipped,
// since the gateway allow is what close must not lose, not intra reachability.
func (b *Builder) teamNetworkCIDRs(ctx context.Context, instances []incusInstance) []string {
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

// networkCIDR is a managed network's IPv4 subnet in canonical form
// ("10.0.1.0/24"), derived from its ipv4.address ("10.0.1.1/24"). "" when the
// network has no IPv4 subnet or can't be read.
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

// closeNIC closes one instance: it ensures the instance's closed-egress ACL
// exists, then (ETag-guarded, like attachNICACL, so a concurrent access pass
// can't clobber it) rewrites eth0 to deny new ingress and all egress except that
// ACL's allow-list. The original NIC state is saved on the FIRST close only, so
// a re-close after an external-access pass transiently re-opened the NIC still
// restores the true open state on open.
func (b *Builder) closeNIC(ctx context.Context, name string, egress []map[string]interface{}) error {
	aclName := closedACLName(name)
	if err := b.ensureACLObjectFull(ctx, aclName, "LaForge closed access for "+name, nil, egress); err != nil {
		return err
	}
	path := "/1.0/instances/" + url.PathEscape(name)
	return retryOnBusy(ctx, func() error {
		for attempt := 0; attempt < 5; attempt++ {
			var headers http.Header
			raw, err := b.Client.doWithHeaders(ctx, http.MethodGet, path, nil, nil, &headers)
			if err != nil {
				return err
			}
			var instance struct {
				Config          map[string]string            `json:"config"`
				Devices         map[string]map[string]string `json:"devices"`
				ExpandedDevices map[string]map[string]string `json:"expanded_devices"`
			}
			if err := json.Unmarshal(raw, &instance); err != nil {
				return err
			}
			nic := instance.Devices["eth0"]
			fromProfile := false
			if nic == nil {
				if e := instance.ExpandedDevices["eth0"]; e != nil {
					nic = cloneStringMap(e)
					fromProfile = true
				}
			}
			if nic == nil || nic["type"] != "nic" {
				return nil // no active NIC to firewall -- nothing to close
			}

			body := map[string]interface{}{}
			if _, already := instance.Config[closedAccessKey]; !already {
				saved := closedAccess{
					ACLs:          nic["security.acls"],
					IngressAction: nic["security.acls.default.ingress.action"],
					EgressAction:  nic["security.acls.default.egress.action"],
					FromProfile:   fromProfile,
				}
				savedJSON, err := json.Marshal(saved)
				if err != nil {
					return err
				}
				body["config"] = map[string]string{closedAccessKey: string(savedJSON)}
			}

			// Keep the intra host ACL (lfhacl*, same-environment port firewall),
			// drop the public-port ACLs (lfpacl*, external ingress), add the
			// closed-egress ACL; default-deny both directions.
			kept := []string{}
			for _, a := range strings.Split(nic["security.acls"], ",") {
				a = strings.TrimSpace(a)
				if a == "" || a == aclName || strings.HasPrefix(a, "lfpacl") {
					continue
				}
				kept = append(kept, a)
			}
			nic["security.acls"] = strings.Join(append(kept, aclName), ",")
			nic["security.acls.default.ingress.action"] = "drop"
			nic["security.acls.default.egress.action"] = "drop"

			etag := headers.Get("ETag")
			if etag == "" {
				return fmt.Errorf("instance %s response has no ETag; cannot safely close NIC", name)
			}
			body["devices"] = map[string]interface{}{"eth0": nic}
			_, err = b.Client.doWithHeaders(ctx, http.MethodPatch, path, body, http.Header{"If-Match": []string{etag}}, nil)
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatus == http.StatusPreconditionFailed {
				continue
			}
			return err
		}
		return fmt.Errorf("instance %s kept changing while closing; retry the task", name)
	})
}

// openNIC restores one instance's pre-close NIC state. It understands both the
// current ACL-based close (closedAccessKey) and the legacy device-masking close
// (removedNICKey), so a team closed by an older build still reopens. A full PUT
// is used because a config key must be deleted outright (PATCH only merges).
func (b *Builder) openNIC(ctx context.Context, name string) error {
	return retryOnBusy(ctx, func() error {
		full, _, err := b.getInstancePut(ctx, name)
		if err != nil {
			return err
		}
		changed := false

		if rawSaved, ok := full.Config[closedAccessKey]; ok {
			var saved closedAccess
			if err := json.Unmarshal([]byte(rawSaved), &saved); err != nil {
				return fmt.Errorf("decoding closed access state: %w", err)
			}
			if saved.FromProfile {
				// Close created the instance-level override; drop it so the
				// profile's own eth0 takes back over.
				delete(full.Devices, "eth0")
			} else if dev, ok := full.Devices["eth0"]; ok {
				var m map[string]interface{}
				if err := json.Unmarshal(dev, &m); err != nil {
					return fmt.Errorf("decoding eth0 for restore: %w", err)
				}
				setOrDelete(m, "security.acls", saved.ACLs)
				setOrDelete(m, "security.acls.default.ingress.action", saved.IngressAction)
				setOrDelete(m, "security.acls.default.egress.action", saved.EgressAction)
				restored, err := json.Marshal(m)
				if err != nil {
					return err
				}
				full.Devices["eth0"] = restored
			}
			delete(full.Config, closedAccessKey)
			changed = true
		}

		if rawSaved, ok := full.Config[removedNICKey]; ok {
			var saved savedNIC
			if err := json.Unmarshal([]byte(rawSaved), &saved); err != nil {
				return fmt.Errorf("decoding legacy saved nic state: %w", err)
			}
			if saved.Instance {
				deviceJSON, err := json.Marshal(saved.Device)
				if err != nil {
					return err
				}
				full.Devices["eth0"] = deviceJSON
			} else {
				delete(full.Devices, "eth0")
			}
			delete(full.Config, removedNICKey)
			changed = true
		}

		if !changed {
			return nil // already open -- idempotent
		}
		_, err = b.Client.put(ctx, "/1.0/instances/"+name, full)
		return err
	})
}

// setOrDelete sets key to val, or removes it when val is "" -- so a NIC field
// that was unset before close is restored to unset, not to an empty string.
func setOrDelete(m map[string]interface{}, key, val string) {
	if val == "" {
		delete(m, key)
		return
	}
	m[key] = val
}

// cloneStringMap copies a NIC's string map so edits don't mutate the decoded
// expanded-devices view we're reading from.
func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
