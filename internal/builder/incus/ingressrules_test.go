package incus

import "testing"

import "github.com/globalcptc/laforge/internal/builder"

// TestNICIngressRules covers the per-host NIC port-firewall rule generation
// offline (the live probe needs a daemon): one allow rule per source per
// protocol, no `destination` (the instance port is the destination), and a
// host with no ports yields nothing.
func TestNICIngressRules(t *testing.T) {
	h := builder.HostAccess{Address: "10.0.1.10", TCPPorts: []string{"80", "443"}, UDPPorts: []string{"53"}}
	rules := nicIngressRules([]string{"10.0.1.0/24", "10.0.9.0/24"}, h)

	// 2 sources x (1 tcp + 1 udp) = 4 rules; no destination field.
	if len(rules) != 4 {
		t.Fatalf("got %d rules, want 4: %+v", len(rules), rules)
	}
	for _, r := range rules {
		if r["action"] != "allow" {
			t.Errorf("rule action = %v, want allow", r["action"])
		}
		if _, hasDest := r["destination"]; hasDest {
			t.Errorf("NIC rule must not carry a destination (the port is the destination): %+v", r)
		}
		if r["source"] != "10.0.1.0/24" && r["source"] != "10.0.9.0/24" {
			t.Errorf("rule source = %v, want one of the allowed CIDRs", r["source"])
		}
		switch r["protocol"] {
		case "tcp":
			if r["destination_port"] != "80,443" {
				t.Errorf("tcp dport = %v, want 80,443", r["destination_port"])
			}
		case "udp":
			if r["destination_port"] != "53" {
				t.Errorf("udp dport = %v, want 53", r["destination_port"])
			}
		default:
			t.Errorf("unexpected protocol %v", r["protocol"])
		}
	}

	// A host with no declared ports -> no allow rules -> reachable on nothing.
	if got := nicIngressRules([]string{"10.0.1.0/24"}, builder.HostAccess{Address: "10.0.1.11"}); len(got) != 0 {
		t.Errorf("no-ports host should yield no rules, got %+v", got)
	}
}
