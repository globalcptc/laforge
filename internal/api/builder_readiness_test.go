package api

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

// TestComputeDemand checks the per-environment demand estimate: VPCs per team,
// EIPs for public hosts across teams, and host-instance counts per size across
// teams, counting one instance per (network, object, copy) and excluding
// containers.
func TestComputeDemand(t *testing.T) {
	c := &loader.Content{
		Hosts: []loader.Host{
			{Name: "web", Size: "small", Public: &loader.Ports{}},
			{Name: "db", Size: "large"},
			{Name: "jump", Size: "small", Public: &loader.Ports{}},
		},
		Containers: []loader.Container{{Name: "sidecar"}},
	}
	env := loader.Environment{
		Name:  "range",
		Teams: 3,
		Networks: map[string]map[string][]loader.Copy{
			"dmz": {
				"web":     {{As: "web1"}, {As: "web2"}}, // 2 copies
				"sidecar": {{As: "s1"}},                 // container -> ignored
			},
			"internal": {
				"db":   {{As: "db1"}},
				"jump": {{As: "jump1"}},
			},
		},
	}

	d := computeDemand(c, env)

	if d.Teams != 3 {
		t.Errorf("Teams = %d, want 3", d.Teams)
	}
	// Public host instances per team: web (×2) + jump (×1) = 3; × 3 teams = 9.
	if d.PublicHosts != 9 {
		t.Errorf("PublicHosts = %d, want 9", d.PublicHosts)
	}
	// small: web(2) + jump(1) = 3 per team × 3 = 9; large: db(1) × 3 = 3.
	if d.InstanceCounts["small"] != 9 {
		t.Errorf("InstanceCounts[small] = %d, want 9", d.InstanceCounts["small"])
	}
	if d.InstanceCounts["large"] != 3 {
		t.Errorf("InstanceCounts[large] = %d, want 3", d.InstanceCounts["large"])
	}
	// The container must not appear as a host instance under any size.
	total := 0
	for _, n := range d.InstanceCounts {
		total += n
	}
	if total != 12 {
		t.Errorf("total host instances = %d, want 12 (container excluded)", total)
	}
}

// TestComputeDemandDefaultsTeams ensures an unspecified team count still estimates
// one team's worth rather than zeroing the demand.
func TestComputeDemandDefaultsTeams(t *testing.T) {
	c := &loader.Content{Hosts: []loader.Host{{Name: "box", Size: "small"}}}
	env := loader.Environment{
		Teams:    0,
		Networks: map[string]map[string][]loader.Copy{"n": {"box": {{As: "b"}}}},
	}
	d := computeDemand(c, env)
	if d.InstanceCounts["small"] != 1 {
		t.Errorf("InstanceCounts[small] = %d, want 1", d.InstanceCounts["small"])
	}
}
