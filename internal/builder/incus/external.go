package incus

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/globalcptc/laforge/internal/builder"
)

// Default external-port window and per-team span when the builder config leaves
// them unset: 40000-50000, 100 ports reserved to each team within it.
const (
	defaultExternalPortMin = 40000
	defaultExternalPortMax = 50000
	externalPortPerTeam    = 100
)

func (b *Builder) externalPortRange() (min, max int) {
	min, max = b.Config.ExternalPortMin, b.Config.ExternalPortMax
	if min <= 0 {
		min = defaultExternalPortMin
	}
	if max <= min {
		max = defaultExternalPortMax
	}
	return min, max
}

// ConfigureExternalAccess realizes content `public:` ports on Incus/OVN by
// adding a NAT proxy device to each instance: one of the cluster's external IPs
// (Config.ExternalAccessIP, which may be several) listens on a per-(team,host,
// port) external port and DNATs to the instance's own lab address:port -- the
// "flagged open ports NAT'd in on external ports" model, automated.
//
// Each team is assigned ONE external IP, round-robin by team number
// (ips[(team-1) % len(ips)]). With at least as many IPs as teams every team
// gets its own public address; otherwise teams share an IP. On its IP a team's
// external ports come from a DETERMINISTIC sub-window -- slot = (team-1)/len(ips),
// base = min + slot*span, then sequentially over the team's hosts/ports in a
// stable order. Two distinct teams on the same IP always land in different
// slots, so there is no collision and no global coordination, and re-asserting
// an unchanged set yields the same IP and ports. With a single IP this is
// exactly the previous per-team-window behaviour. PATCH merges devices by key,
// so this is idempotent per instance.
//
// VERIFY ON CLUSTER: whether a NAT proxy device on an external IP behaves as
// intended for OVN-networked instances (vs. an OVN network forward on the
// uplink) is the one thing this dev environment can't exercise -- OVN has no
// control plane here (see Builder's own doc comment). The allocation, recording
// and surfacing are all exercised by the fake builder; this is the real-fabric
// half to confirm live.
func (b *Builder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	if b.Config.ExternalAccessIP == "" {
		return nil, fmt.Errorf("this builder has no external_access_ip configured -- set one before content can expose public: ports")
	}
	ips, err := parseExternalIPs(b.Config.ExternalAccessIP)
	if err != nil {
		return nil, fmt.Errorf("external_access_ip %q: %w", b.Config.ExternalAccessIP, err)
	}
	teamNum, err := strconv.Atoi(team)
	if err != nil || teamNum < 1 {
		return nil, fmt.Errorf("external access needs a positive team number, got %q", team)
	}
	// One IP per team, round-robin; teams that share an IP (more teams than IPs)
	// each get a distinct port sub-window (slot) on it.
	min, max := b.externalPortRange()
	teamIP, base, limit := teamExternalWindow(ips, teamNum, min, max)
	if base >= limit {
		return nil, fmt.Errorf("team %d has no external-port window left on %s -- widen external_port_min/max or add more external IPs", teamNum, teamIP)
	}

	// Stable order so the deterministic allocation is reproducible.
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].ExternalRef < hosts[j].ExternalRef })

	next := base
	var out []builder.ExternalEndpoint
	for _, h := range hosts {
		devices := map[string]interface{}{}
		addPorts := func(proto string, ports []string) error {
			ps := append([]string(nil), ports...)
			sort.Strings(ps)
			for _, p := range ps {
				if next >= limit {
					return fmt.Errorf("team %d exhausted its external-port window (%d ports) on %s -- widen external_port_min/max or add more external IPs", teamNum, externalPortPerTeam, teamIP)
				}
				ext := next
				next++
				devices["lfx-"+proto+"-"+p] = map[string]string{
					"type":    "proxy",
					"nat":     "true",
					"listen":  proto + ":" + teamIP + ":" + strconv.Itoa(ext),
					"connect": proto + ":" + h.Address + ":" + p,
				}
				out = append(out, builder.ExternalEndpoint{
					ExternalRef: h.ExternalRef, Protocol: proto, InternalPort: p,
					ExternalPort: strconv.Itoa(ext), PublicAddress: teamIP + ":" + strconv.Itoa(ext),
				})
			}
			return nil
		}
		if err := addPorts("tcp", h.TCPPorts); err != nil {
			return out, err
		}
		if err := addPorts("udp", h.UDPPorts); err != nil {
			return out, err
		}
		if len(devices) == 0 {
			continue
		}
		if _, err := b.Client.patch(ctx, "/1.0/instances/"+h.ExternalRef, map[string]interface{}{"devices": devices}); err != nil {
			return out, fmt.Errorf("adding external-access proxy devices to %s: %w", h.ExternalRef, err)
		}
	}
	return out, nil
}

// teamExternalWindow picks a team's external IP and its external-port window.
// Team t takes ips[(t-1) % len(ips)] (round-robin, so teams spread across the
// IPs); within that IP its ports come from slot (t-1)/len(ips), base =
// min + slot*span, limit = base + span (clamped to max+1). Distinct teams on the
// same IP always have different slots, so their windows never overlap. base >=
// limit means the window is exhausted (too many teams for the range on this IP).
func teamExternalWindow(ips []string, teamNum, min, max int) (ip string, base, limit int) {
	ip = ips[(teamNum-1)%len(ips)]
	slot := (teamNum - 1) / len(ips)
	base = min + slot*externalPortPerTeam
	limit = base + externalPortPerTeam
	if limit > max+1 {
		limit = max + 1
	}
	return ip, base, limit
}

// maxExternalIPs caps how many addresses an external_access_ip value may expand
// to, so a mistyped range ("10.0.0.0-10.255.255.255") fails loudly instead of
// trying to enumerate millions of addresses.
const maxExternalIPs = 1024

// parseExternalIPs turns the external_access_ip config into an ordered,
// de-duplicated list of IPv4 addresses. It accepts a comma-separated list, an
// inclusive dashed range ("192.0.2.10-192.0.2.20", or the short last-octet form
// "192.0.2.10-20"), or any mix of the two. IPv6 is not supported -- the OVN
// proxy `listen` syntax this feeds is IPv4-only (no bracketed address). Order is
// preserved so the per-team, round-robin IP assignment is stable.
func parseExternalIPs(s string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	add := func(ip string) {
		if !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		loStr, hiStr, isRange := strings.Cut(tok, "-")
		if !isRange {
			ip := net.ParseIP(tok)
			if ip == nil || ip.To4() == nil {
				return nil, fmt.Errorf("%q is not a valid IPv4 address", tok)
			}
			add(ip.To4().String())
			continue
		}
		loStr, hiStr = strings.TrimSpace(loStr), strings.TrimSpace(hiStr)
		lo := net.ParseIP(loStr)
		if lo == nil || lo.To4() == nil {
			return nil, fmt.Errorf("range %q: %q is not a valid IPv4 address", tok, loStr)
		}
		lo4 := lo.To4()
		var hi4 net.IP
		if strings.Contains(hiStr, ".") {
			hi := net.ParseIP(hiStr)
			if hi == nil || hi.To4() == nil {
				return nil, fmt.Errorf("range %q: %q is not a valid IPv4 address", tok, hiStr)
			}
			hi4 = hi.To4()
		} else {
			// Short form: "a.b.c.d-e" means a.b.c.d through a.b.c.e.
			oct, err := strconv.Atoi(hiStr)
			if err != nil || oct < 0 || oct > 255 {
				return nil, fmt.Errorf("range %q: %q is not a valid final octet (0-255)", tok, hiStr)
			}
			hi4 = net.IPv4(lo4[0], lo4[1], lo4[2], byte(oct)).To4()
		}
		start, end := binary.BigEndian.Uint32(lo4), binary.BigEndian.Uint32(hi4)
		if end < start {
			return nil, fmt.Errorf("range %q: end is before start", tok)
		}
		if count := uint64(end) - uint64(start) + 1; count > maxExternalIPs {
			return nil, fmt.Errorf("range %q spans %d addresses (max %d) -- narrow it", tok, count, maxExternalIPs)
		}
		for v := start; ; v++ {
			var b4 [4]byte
			binary.BigEndian.PutUint32(b4[:], v)
			add(net.IPv4(b4[0], b4[1], b4[2], b4[3]).To4().String())
			if v == end {
				break
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no IPv4 address found")
	}
	if len(out) > maxExternalIPs {
		return nil, fmt.Errorf("%d external IPs is more than the maximum %d", len(out), maxExternalIPs)
	}
	return out, nil
}
