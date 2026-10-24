package microcloud

import (
	"reflect"
	"testing"
)

func TestParseExternalIPs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
		err  bool
	}{
		{name: "single", in: "203.0.113.10", want: []string{"203.0.113.10"}},
		{name: "list", in: "203.0.113.10, 203.0.113.20", want: []string{"203.0.113.10", "203.0.113.20"}},
		{name: "full range", in: "192.0.2.10-192.0.2.12", want: []string{"192.0.2.10", "192.0.2.11", "192.0.2.12"}},
		{name: "short range", in: "192.0.2.10-12", want: []string{"192.0.2.10", "192.0.2.11", "192.0.2.12"}},
		{name: "list and range mixed", in: "10.0.0.1, 10.0.0.5-6", want: []string{"10.0.0.1", "10.0.0.5", "10.0.0.6"}},
		{name: "dedup preserves order", in: "10.0.0.1, 10.0.0.1-2", want: []string{"10.0.0.1", "10.0.0.2"}},
		{name: "whitespace tolerated", in: "  10.0.0.1 ,10.0.0.2  ", want: []string{"10.0.0.1", "10.0.0.2"}},
		{name: "empty", in: "", err: true},
		{name: "not an ip", in: "nope", err: true},
		{name: "ipv6 rejected", in: "2001:db8::1", err: true},
		{name: "range backwards", in: "10.0.0.5-4", err: true},
		{name: "range too large", in: "10.0.0.0-10.255.255.255", err: true},
		{name: "bad final octet", in: "10.0.0.1-999", err: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseExternalIPs(c.in)
			if c.err {
				if err == nil {
					t.Fatalf("parseExternalIPs(%q) = %v, want error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseExternalIPs(%q) unexpected error: %v", c.in, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseExternalIPs(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestTeamExternalWindow pins the per-team IP + port-window assignment: teams
// spread one-per-IP round-robin, and teams that share an IP get non-overlapping
// port windows.
func TestTeamExternalWindow(t *testing.T) {
	const (
		min = 40000
		max = 50000
	)

	// Three IPs, teams 1..4: teams 1-3 each get their own IP at the base window;
	// team 4 wraps back to IP[0] but in the next slot, so it never collides with
	// team 1.
	ips := []string{"a", "b", "c"}
	want := []struct {
		team int
		ip   string
		base int
	}{
		{1, "a", min},
		{2, "b", min},
		{3, "c", min},
		{4, "a", min + externalPortPerTeam},
		{5, "b", min + externalPortPerTeam},
	}
	for _, w := range want {
		ip, base, limit := teamExternalWindow(ips, w.team, min, max)
		if ip != w.ip || base != w.base || limit != w.base+externalPortPerTeam {
			t.Errorf("team %d -> (ip=%s base=%d limit=%d), want (ip=%s base=%d limit=%d)",
				w.team, ip, base, limit, w.ip, w.base, w.base+externalPortPerTeam)
		}
	}

	// A single IP must reproduce the original per-team windowing exactly.
	for team := 1; team <= 3; team++ {
		ip, base, limit := teamExternalWindow([]string{"only"}, team, min, max)
		wantBase := min + (team-1)*externalPortPerTeam
		if ip != "only" || base != wantBase || limit != wantBase+externalPortPerTeam {
			t.Errorf("single-IP team %d -> (ip=%s base=%d limit=%d), want (only, %d, %d)",
				team, ip, base, limit, wantBase, wantBase+externalPortPerTeam)
		}
	}
}
