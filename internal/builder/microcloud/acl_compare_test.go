package microcloud

import "testing"

// TestSameACLRules: an ACL the hoster returns with every field filled in (empty
// strings for the ones LaForge doesn't set) matches the rules LaForge would
// write, so an unchanged ACL isn't rewritten; any real difference -- a port, a
// source, a rule more or fewer, the order -- does not match.
func TestSameACLRules(t *testing.T) {
	want := []map[string]interface{}{
		{"action": "allow", "state": "enabled", "source": "0.0.0.0/0", "protocol": "tcp", "destination_port": "3389"},
	}
	stored := []map[string]interface{}{
		{"action": "allow", "state": "enabled", "source": "0.0.0.0/0", "protocol": "tcp", "destination_port": "3389",
			"destination": "", "source_port": "", "icmp_type": "", "icmp_code": "", "description": ""},
	}
	if !sameACLRules(stored, want) {
		t.Fatal("an unchanged ACL was seen as different")
	}
	changed := []map[string]interface{}{
		{"action": "allow", "state": "enabled", "source": "0.0.0.0/0", "protocol": "tcp", "destination_port": "22"},
	}
	if sameACLRules(changed, want) {
		t.Fatal("a different port was seen as the same")
	}
	if sameACLRules(append(stored, stored[0]), want) || sameACLRules(nil, want) {
		t.Fatal("a different number of rules was seen as the same")
	}
	if !sameACLRules(nil, []map[string]interface{}{}) {
		t.Fatal("two empty rule sets differ")
	}
}

// TestDedupRules: duplicate egress rules (two environments' same-numbered teams
// share a per-team project, so their identical CIDRs collide) are collapsed to
// one, preserving order -- LXD rejects an ACL with a duplicate rule, which was
// failing every open/close with "Duplicate of egress rule N".
func TestDedupRules(t *testing.T) {
	gw := map[string]interface{}{"action": "allow", "state": "enabled", "destination": "10.0.0.1/32", "protocol": "tcp", "destination_port": "8445"}
	cidr := map[string]interface{}{"action": "allow", "state": "enabled", "destination": "10.250.5.0/24"}
	drop := map[string]interface{}{"action": "drop", "state": "disabled", "destination": "0.0.0.0/0"}
	// cidr appears twice (both environments' team-5 networks have the same CIDR).
	in := []map[string]interface{}{gw, cidr, cidr, drop}
	out := dedupRules(in)
	if len(out) != 3 {
		t.Fatalf("expected 3 unique rules, got %d: %v", len(out), out)
	}
	// Order preserved: gateway, cidr (once), drop.
	if out[0]["destination"] != "10.0.0.1/32" || out[1]["destination"] != "10.250.5.0/24" || out[2]["action"] != "drop" {
		t.Fatalf("dedup reordered or dropped the wrong rule: %v", out)
	}
}

// TestInBuildScope: team operations must act only on the current build's
// instances. Two environments' same-numbered teams share a MicroCloud project,
// so the build-prefixed external name is what separates them; an empty scope
// (tests, legacy) matches everything.
func TestInBuildScope(t *testing.T) {
	const build = "08a0455b-5355-4e77-8a81-77d1c1c0fd06"
	mine := "build-" + build + "-team-5-host-vdi-win-01"
	other := "build-397169b1-bdee-4642-a5d2-c9fe8953b0ee-team-5-host-vdi-win-01"
	if !inBuildScope(mine, build) {
		t.Error("this build's instance should be in scope")
	}
	if inBuildScope(other, build) {
		t.Error("another build's same-numbered team must be excluded")
	}
	if !inBuildScope(other, "") {
		t.Error("an empty scope must match everything (team-only)")
	}
	// A near-miss build id must not match on a shared prefix.
	if inBuildScope("build-"+build+"extra-team-5-host-x", build) {
		t.Error("prefix match must be bounded by the trailing dash")
	}
}
