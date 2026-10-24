package render

import (
	"fmt"
	"net"
)

// Address implements exactly what the Environment spec
// states: "The IP is the network's CIDR plus that octet." No extra
// validation that last_octet keeps the address inside the network's usable
// range for prefixes smaller than /24 -- authors are expected to use a
// last_octet that makes sense for their CIDR, same as the spec's own
// examples do; this does the literal arithmetic it describes, nothing more.
func Address(cidr string, lastOctet int) (string, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid cidr %q: %w", cidr, err)
	}
	v4 := ip.To4()
	if v4 == nil {
		return "", fmt.Errorf("cidr %q is not IPv4 (only IPv4 networks are modeled today)", cidr)
	}
	if lastOctet < 0 || lastOctet > 255 {
		return "", fmt.Errorf("last_octet %d out of range 0-255", lastOctet)
	}
	out := net.IPv4(v4[0], v4[1], v4[2], byte(lastOctet))
	return out.String(), nil
}
