package microcloud

import (
	"fmt"
	"net/netip"
	"strings"
)

// PublicAccessConfig is global to one MicroCloud builder. Content still selects
// public TCP/UDP ports; it never selects the network or allocates an address.
type PublicAccessConfig struct {
	Type    string        `json:"type"` // proxy (default) or nic
	Network string        `json:"network,omitempty"`
	CIDR    string        `json:"cidr,omitempty"`
	Ranges  string        `json:"ranges,omitempty"`
	Gateway string        `json:"gateway,omitempty"`
	DNS     []string      `json:"dns,omitempty"`
	MTU     int           `json:"mtu,omitempty"`
	Routes  []PublicRoute `json:"routes,omitempty"`
}

type PublicRoute struct {
	To  string `json:"to"`
	Via string `json:"via"`
}

func (c PublicAccessConfig) Validate() error {
	if c.Type == "" || c.Type == "proxy" {
		if c.Network != "" || c.CIDR != "" || c.Ranges != "" || c.Gateway != "" || len(c.DNS) != 0 || c.MTU != 0 || len(c.Routes) != 0 {
			return fmt.Errorf("public network settings require microcloud_public_access.type=nic")
		}
		return nil
	}
	if c.Type != "nic" {
		return fmt.Errorf("microcloud_public_access.type must be proxy or nic")
	}
	if strings.TrimSpace(c.Network) != c.Network || c.Network == "" || strings.ContainsAny(c.Network, "/?&#\\\n\r") {
		return fmt.Errorf("public NIC requires an existing OVN network name")
	}
	prefix, err := netip.ParsePrefix(c.CIDR)
	if err != nil || !prefix.Addr().Is4() || prefix != prefix.Masked() || prefix.Bits() > 30 {
		return fmt.Errorf("public NIC cidr must be an IPv4 network CIDR with usable host addresses (for example 10.250.0.0/16)")
	}
	ips, err := parsePublicIPs(c.Ranges)
	if err != nil {
		return fmt.Errorf("public NIC ranges: %w", err)
	}
	usable := func(s string) bool {
		a, err := netip.ParseAddr(s)
		return err == nil && a.Is4() && a.IsGlobalUnicast() && prefix.Contains(a) && a != prefix.Addr() && prefix.Contains(a.Next())
	}
	reserved := map[string]bool{}
	if c.Gateway != "" {
		if !usable(c.Gateway) {
			return fmt.Errorf("public NIC gateway must be a usable IPv4 address in %s", c.CIDR)
		}
		reserved[c.Gateway] = true
	}
	for _, dns := range c.DNS {
		a, err := netip.ParseAddr(dns)
		if err != nil || !a.Is4() || !a.IsGlobalUnicast() {
			return fmt.Errorf("invalid public NIC DNS address %q", dns)
		}
		reserved[dns] = true
	}
	for _, route := range c.Routes {
		p, err := netip.ParsePrefix(route.To)
		if err != nil || !p.Addr().Is4() || p != p.Masked() {
			return fmt.Errorf("invalid public NIC route destination %q", route.To)
		}
		if p.Bits() == 0 {
			return fmt.Errorf("use gateway for the public NIC default route")
		}
		if !usable(route.Via) {
			return fmt.Errorf("public NIC route gateway %q must be in %s", route.Via, c.CIDR)
		}
		reserved[route.Via] = true
	}
	for _, ip := range ips {
		if !usable(ip) || reserved[ip] {
			return fmt.Errorf("public NIC pool address %s is outside the usable subnet or reserved for a gateway/DNS server", ip)
		}
	}
	if c.MTU != 0 && (c.MTU < 576 || c.MTU > 9000) {
		return fmt.Errorf("public NIC mtu must be 576-9000, or 0 to inherit")
	}
	return nil
}

// Public pools can span a /16. Proxy mode retains its existing 1024-IP limit.
func parsePublicIPs(s string) ([]string, error) { return parseIPv4Ranges(s, 65536) }
