package microcloud

import (
	"context"
	"testing"
)

// TestClusterTarget pins the cluster-group targeting: LaForge never picks a
// member; it hands LXD the configured group as "@<group>" (or no target when
// unset) and lets LXD schedule.
func TestClusterTarget(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		group string
		want  string
	}{
		{"default", "@default"}, // the whole cluster, addressed as a group
		{"gpu", "@gpu"},         // a named group
		{"  edge  ", "@edge"},   // trimmed
		{"", ""},                // unset -> no explicit target (LXD default)
		{"   ", ""},             // whitespace-only -> unset
	}
	for _, c := range cases {
		b := &Builder{Config: Config{ClusterGroup: c.group}}
		if got := b.clusterTarget(ctx); got != c.want {
			t.Errorf("clusterTarget(%q) = %q, want %q", c.group, got, c.want)
		}
	}
}
