package loader

import (
	"reflect"
	"testing"
)

func TestDependents(t *testing.T) {
	// Chain: web depends_on app, app depends_on db. So db is the foundation;
	// rebuilding db must cascade up to app and web.
	c := &Content{
		Hosts: []Host{
			{Name: "db"},
			{Name: "app", DependsOn: []string{"db"}},
			{Name: "web", DependsOn: []string{"app"}},
			{Name: "lonely"},
		},
		Containers: []Container{
			{Name: "cache", DependsOn: []string{"db"}},
		},
	}

	cases := []struct {
		name  string
		roots []string
		want  []string
	}{
		{"foundation cascades to all above it", []string{"db"}, []string{"app", "cache", "db", "web"}},
		{"middle cascades up only", []string{"app"}, []string{"app", "web"}},
		{"leaf is just itself", []string{"web"}, []string{"web"}},
		{"independent host is just itself", []string{"lonely"}, []string{"lonely"}},
		{"container dependent included", []string{"cache"}, []string{"cache"}},
		{"unknown root returns itself", []string{"ghost"}, []string{"ghost"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Dependents(tc.roots)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Dependents(%v) = %v, want %v", tc.roots, got, tc.want)
			}
		})
	}
}

func TestDependentsIgnoresNetworkEdges(t *testing.T) {
	// A host depends_on a network name: networks are foundational and excluded
	// from the cascade (they have no depends_on and aren't rebuilt this way).
	c := &Content{
		Networks: []Network{{Name: "lan"}},
		Hosts: []Host{
			{Name: "h1", DependsOn: []string{"lan"}},
		},
	}
	// Rebuilding the network should not drag h1 in via this graph (network edges
	// are skipped), so the closure of "lan" is just "lan" itself.
	if got := c.Dependents([]string{"lan"}); !reflect.DeepEqual(got, []string{"lan"}) {
		t.Errorf("Dependents([lan]) = %v, want [lan]", got)
	}
}

func TestDependentsDiamond(t *testing.T) {
	// base <- left, base <- right, left/right <- top. Rebuilding base hits all.
	c := &Content{
		Hosts: []Host{
			{Name: "base"},
			{Name: "left", DependsOn: []string{"base"}},
			{Name: "right", DependsOn: []string{"base"}},
			{Name: "top", DependsOn: []string{"left", "right"}},
		},
	}
	got := c.Dependents([]string{"base"})
	want := []string{"base", "left", "right", "top"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dependents([base]) = %v, want %v", got, want)
	}
}
