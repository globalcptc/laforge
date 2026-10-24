package incus

import (
	"context"
	"testing"
)

// TestGatewayEgressRulesPinnedIP: a gateway given as a bare IP:port becomes one
// TCP egress allow to that /32 on that port, with no DNS lookup needed.
func TestGatewayEgressRulesPinnedIP(t *testing.T) {
	b := &Builder{Config: Config{GatewayAddr: "203.0.113.9:8443"}}
	rules, err := b.gatewayEgressRules(context.Background())
	if err != nil {
		t.Fatalf("gatewayEgressRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1: %v", len(rules), rules)
	}
	r := rules[0]
	if r["action"] != "allow" || r["state"] != "enabled" || r["protocol"] != "tcp" ||
		r["destination"] != "203.0.113.9/32" || r["destination_port"] != "8443" {
		t.Fatalf("unexpected gateway egress rule: %v", r)
	}
}

// TestGatewayEgressRulesDefaultPort: a gateway with no :port defaults to 8443.
func TestGatewayEgressRulesDefaultPort(t *testing.T) {
	b := &Builder{Config: Config{GatewayAddr: "203.0.113.9"}}
	rules, err := b.gatewayEgressRules(context.Background())
	if err != nil {
		t.Fatalf("gatewayEgressRules: %v", err)
	}
	if len(rules) != 1 || rules[0]["destination_port"] != "8443" {
		t.Fatalf("want a single rule on default port 8443, got %v", rules)
	}
}

// TestGatewayEgressRulesMissingErrors: close must refuse rather than silently
// drop the gateway allow (which would cut all telemetry) when no gateway address
// is configured.
func TestGatewayEgressRulesMissingErrors(t *testing.T) {
	b := &Builder{Config: Config{GatewayAddr: ""}}
	if _, err := b.gatewayEgressRules(context.Background()); err == nil {
		t.Fatal("gatewayEgressRules with no address must error, not return an empty allow-list")
	}
}

// TestNetworkAccessACLNameDistinct: the network access ACL name is
// deterministic and distinct from the per-host port/public ACL names, so all
// three can coexist and be told apart on the hoster (lfnacl = network access).
func TestNetworkAccessACLNameDistinct(t *testing.T) {
	const net = "lf-t1corp7487"
	na := networkAccessACLName(net)
	if na == hostACLName(net) || na == publicACLName(net) {
		t.Fatalf("network ACL name collides: %q", na)
	}
	if networkAccessACLName(net) != na {
		t.Fatal("networkAccessACLName is not deterministic")
	}
	if na[:6] != "lfnacl" {
		t.Fatalf("networkAccessACLName prefix = %q, want lfnacl*", na)
	}
}

