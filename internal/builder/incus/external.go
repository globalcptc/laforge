package incus

import (
	"context"
	"fmt"
	"sort"
	"strconv"

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
// adding a NAT proxy device to each instance: the cluster's single external IP
// (Config.ExternalAccessIP) listens on a per-(team,host,port) external port and
// DNATs to the instance's own lab address:port. One shared IP, a distinct
// external port per mapping -- exactly the "flagged open ports NAT'd in on
// random external ports" model, automated.
//
// External ports are allocated DETERMINISTICALLY from a per-team sub-window
// (teamBase = min + (team-1)*span, then sequentially over the team's hosts/ports
// in a stable order), so there is no cross-team collision and no global
// coordination, and re-asserting an unchanged set yields the same ports.
// Changing a team's public-port set may re-pack its ports within its window.
// PATCH merges devices by key, so this is idempotent per instance.
//
// VERIFY ON CLUSTER: whether a NAT proxy device on a shared external IP behaves
// as intended for OVN-networked instances (vs. an OVN network forward on the
// uplink) is the one thing this dev environment can't exercise -- OVN has no
// control plane here (see Builder's own doc comment). The allocation, recording
// and surfacing are all exercised by the fake builder; this is the real-fabric
// half to confirm live.
func (b *Builder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	if b.Config.ExternalAccessIP == "" {
		return nil, fmt.Errorf("this builder has no external_access_ip configured -- set one before content can expose public: ports")
	}
	teamNum, err := strconv.Atoi(team)
	if err != nil || teamNum < 1 {
		return nil, fmt.Errorf("external access needs a positive team number, got %q", team)
	}
	min, max := b.externalPortRange()
	base := min + (teamNum-1)*externalPortPerTeam
	limit := base + externalPortPerTeam
	if limit > max+1 {
		limit = max + 1
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
					return fmt.Errorf("team %d exhausted its external-port window (%d ports) -- widen external_port_min/max", teamNum, externalPortPerTeam)
				}
				ext := next
				next++
				devices["lfx-"+proto+"-"+p] = map[string]string{
					"type":    "proxy",
					"nat":     "true",
					"listen":  proto + ":" + b.Config.ExternalAccessIP + ":" + strconv.Itoa(ext),
					"connect": proto + ":" + h.Address + ":" + p,
				}
				out = append(out, builder.ExternalEndpoint{
					ExternalRef: h.ExternalRef, Protocol: proto, InternalPort: p,
					ExternalPort: strconv.Itoa(ext), PublicAddress: b.Config.ExternalAccessIP + ":" + strconv.Itoa(ext),
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
