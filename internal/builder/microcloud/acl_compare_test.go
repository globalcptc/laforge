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
