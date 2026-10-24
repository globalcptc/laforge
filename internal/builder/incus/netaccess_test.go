package incus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// TestConfigureNetworkAccessLive drives the real ConfigureNetworkAccess against a
// live Incus box: it deploys three OVN networks for a team, applies a policy where only
// netB is visible from netA, and verifies the builder produced exactly the
// configuration proven (separately, by hand) to enforce it -- a full peering mesh
// between the team's networks, and on netB a default-deny-ingress ACL whose one
// allow rule is netA's CIDR. Skips without a live daemon.
func TestConfigureNetworkAccessLive(t *testing.T) {
	client := liveClient(t)
	b := New(client, testConfig())
	ctx := context.Background()

	// Three networks for one team, distinct subnets.
	type net struct {
		ext, disp, cidr, incus string
	}
	suffix := time.Now().UnixNano()
	nets := []net{
		{ext: fmt.Sprintf("na-%d", suffix), disp: "t97a", cidr: "10.66.1.0/24"},
		{ext: fmt.Sprintf("nb-%d", suffix), disp: "t97b", cidr: "10.66.2.0/24"},
		{ext: fmt.Sprintf("nc-%d", suffix), disp: "t97c", cidr: "10.66.3.0/24"},
	}
	for i := range nets {
		ref, err := b.DeployNetwork(ctx, builder.NetworkSpec{
			ExternalName: nets[i].ext, DisplayName: nets[i].disp, Team: "97", CIDR: nets[i].cidr,
		})
		if err != nil {
			if strings.Contains(err.Error(), "OVN") {
				t.Skipf("OVN not available on this daemon: %v", err)
			}
			t.Fatalf("DeployNetwork %s: %v", nets[i].ext, err)
		}
		nets[i].incus = ref
	}
	t.Cleanup(func() {
		// DestroyNetwork now tears down peers + ACL itself, so this is enough.
		for _, n := range nets {
			b.DestroyNetwork(context.Background(), "97", n.incus)
		}
	})

	// Policy: netB visible only from netA; netA and netC visible from nobody.
	// (No hosts here -- this checks the network fabric: peering + default-deny.
	// The per-host NIC port firewall is proven end to end, with real packets, by
	// TestPortFirewallReachabilityLive.)
	access := []builder.NetworkAccess{
		{ExternalName: nets[0].ext, DisplayName: nets[0].disp, CIDR: nets[0].cidr},
		{ExternalName: nets[1].ext, DisplayName: nets[1].disp, CIDR: nets[1].cidr, VisibleFrom: []string{nets[0].ext}},
		{ExternalName: nets[2].ext, DisplayName: nets[2].disp, CIDR: nets[2].cidr},
	}
	if err := b.ConfigureNetworkAccess(ctx, "97", access); err != nil {
		t.Fatalf("ConfigureNetworkAccess: %v", err)
	}

	// Idempotent: a second call must not error.
	if err := b.ConfigureNetworkAccess(ctx, "97", access); err != nil {
		t.Fatalf("ConfigureNetworkAccess (second, idempotent): %v", err)
	}

	// Full mesh: each network has a peer to each of the other two.
	for _, n := range nets {
		peers := listPeerNames(t, b, n.incus)
		if len(peers) < 2 {
			t.Errorf("network %s has %d peers, want >=2 (full mesh): %v", n.incus, len(peers), peers)
		}
	}

	// Every network default-denies ingress (the base the per-host NIC ACLs open
	// specific ports over); egress stays allow.
	for _, n := range nets {
		cfg := networkConfig(t, b, n.incus)
		if cfg["security.acls.default.ingress.action"] != "drop" {
			t.Errorf("network %s default ingress action = %q, want drop", n.incus, cfg["security.acls.default.ingress.action"])
		}
		if cfg["security.acls.default.egress.action"] != "allow" {
			t.Errorf("network %s default egress action = %q, want allow", n.incus, cfg["security.acls.default.egress.action"])
		}
	}
}

func listPeerNames(t *testing.T, b *Builder, netIncus string) []string {
	t.Helper()
	raw, err := b.Client.get(context.Background(), "/1.0/networks/"+netIncus+"/peers")
	if err != nil {
		t.Fatalf("GET peers for %s: %v", netIncus, err)
	}
	var urls []string
	if err := json.Unmarshal(raw, &urls); err != nil {
		t.Fatalf("decode peers: %v", err)
	}
	return urls
}

func networkConfig(t *testing.T, b *Builder, netIncus string) map[string]string {
	t.Helper()
	raw, err := b.Client.get(context.Background(), "/1.0/networks/"+netIncus)
	if err != nil {
		t.Fatalf("GET network %s: %v", netIncus, err)
	}
	var n struct {
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &n); err != nil {
		t.Fatalf("decode network: %v", err)
	}
	return n.Config
}
