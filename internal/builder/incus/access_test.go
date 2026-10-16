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

// TestClosedACLNameDistinct: the closed-egress ACL name is deterministic and
// distinct from the host/public ACL names for the same ref, so all three can
// coexist on a NIC and be told apart on the hoster.
func TestClosedACLNameDistinct(t *testing.T) {
	const ref = "lf-team1-web-01"
	closed, host, public := closedACLName(ref), hostACLName(ref), publicACLName(ref)
	if closed == host || closed == public || host == public {
		t.Fatalf("ACL names collide: closed=%q host=%q public=%q", closed, host, public)
	}
	if closedACLName(ref) != closed {
		t.Fatal("closedACLName is not deterministic")
	}
	if got := closedACLName(ref); got[:6] != "lfcacl" {
		t.Fatalf("closedACLName prefix = %q, want lfcacl*", got)
	}
}

// TestSetOrDelete: restoring an empty field removes the key (back to unset),
// a non-empty field sets it.
func TestSetOrDelete(t *testing.T) {
	m := map[string]interface{}{"a": "1", "b": "2"}
	setOrDelete(m, "a", "")
	if _, ok := m["a"]; ok {
		t.Error("setOrDelete with empty value should delete the key")
	}
	setOrDelete(m, "b", "9")
	if m["b"] != "9" {
		t.Errorf("setOrDelete should set b=9, got %v", m["b"])
	}
}
