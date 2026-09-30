package incuspool

import (
	"testing"

	"github.com/globalcptc/laforge/internal/builder/incus"
)

// newTestHosts builds n distinct, unconnected *incus.Builder values --
// enough to prove hostForTeam's own routing, since no real dialing
// happens until a call is actually made against one.
func newTestHosts(n int) []*incus.Builder {
	hosts := make([]*incus.Builder, n)
	for i := range hosts {
		hosts[i] = incus.New(&incus.Client{BaseURL: "https://unused.invalid"}, incus.Config{})
	}
	return hosts
}

// TestHostForTeamIsDeterministicModuloAssignment is the whole point of
// this package: "round-robin style... spread the load," made concrete as
// team N always resolving to host (N-1)%len(Hosts), with no persisted
// assignment -- a repeated call for the same team must always return the
// same host.
func TestHostForTeamIsDeterministicModuloAssignment(t *testing.T) {
	hosts := newTestHosts(3)
	p := New(hosts, nil)

	cases := []struct {
		team      string
		wantIndex int
	}{
		{"1", 0}, {"2", 1}, {"3", 2}, {"4", 0}, {"5", 1}, {"6", 2}, {"7", 0},
	}
	for _, c := range cases {
		h, err := p.hostForTeam(c.team)
		if err != nil {
			t.Fatalf("hostForTeam(%q): %v", c.team, err)
		}
		if h != hosts[c.wantIndex] {
			t.Errorf("hostForTeam(%q) = host %d's builder, want host %d's", c.team, indexOf(hosts, h), c.wantIndex)
		}
	}
}

func TestHostForTeamIsStableAcrossRepeatedCalls(t *testing.T) {
	hosts := newTestHosts(4)
	p := New(hosts, nil)
	first, err := p.hostForTeam("3")
	if err != nil {
		t.Fatalf("hostForTeam: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := p.hostForTeam("3")
		if err != nil {
			t.Fatalf("hostForTeam (call %d): %v", i, err)
		}
		if again != first {
			t.Fatalf("hostForTeam(\"3\") returned a different host on call %d -- must be a pure function of team+host count", i)
		}
	}
}

func TestHostForTeamRejectsNonNumericTeam(t *testing.T) {
	p := New(newTestHosts(2), nil)
	if _, err := p.hostForTeam("not-a-number"); err == nil {
		t.Fatal("expected an error for a non-numeric team")
	}
	if _, err := p.hostForTeam("0"); err == nil {
		t.Fatal("expected an error for team 0 (teams are 1-indexed)")
	}
}

func TestHostForTeamRejectsAnEmptyPool(t *testing.T) {
	p := New(nil, nil)
	if _, err := p.hostForTeam("1"); err == nil {
		t.Fatal("expected an error for a pool with no hosts configured")
	}
}

func TestSingleHostPoolAlwaysReturnsThatHost(t *testing.T) {
	// The degenerate case every real single-box test in this session runs
	// against: a "pool" of one host must behave exactly like that one
	// host for every team.
	hosts := newTestHosts(1)
	p := New(hosts, nil)
	for _, team := range []string{"1", "2", "5", "100"} {
		h, err := p.hostForTeam(team)
		if err != nil {
			t.Fatalf("hostForTeam(%q): %v", team, err)
		}
		if h != hosts[0] {
			t.Errorf("hostForTeam(%q) did not return the pool's only host", team)
		}
	}
}

func TestImageNamesReflectsThePoolWideCatalog(t *testing.T) {
	p := New(newTestHosts(1), map[string]incus.ImageRef{
		"ubuntu22": {Alias: "ubuntu22"},
		"kali":     {Alias: "kali-rolling"},
	})
	names := p.ImageNames()
	if len(names) != 2 || !names["ubuntu22"] || !names["kali"] {
		t.Errorf("ImageNames() = %+v, want {ubuntu22, kali}", names)
	}
}

func indexOf(hosts []*incus.Builder, target *incus.Builder) int {
	for i, h := range hosts {
		if h == target {
			return i
		}
	}
	return -1
}
