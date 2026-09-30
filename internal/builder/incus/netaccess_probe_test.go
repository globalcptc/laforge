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

// TestPortFirewallReachabilityLive is the real packet-level proof of the
// `ports:` ingress firewall, against a live Incus box. It deploys three containers on
// two peered OVN networks -- a probe on netA, and a target plus a same-network
// sibling on netB -- applies a policy where netB is visible from netA and the
// target declares ONLY tcp 80, then measures reachability from inside the
// guests. It distinguishes allow from deny by TIMING, no listeners needed: a
// DROPPED port makes `nc -w 3` wait the full timeout (~3s), while an
// allowed-but-closed port gets an instant RST (<1s). Cross-network enforcement
// is asserted; same-network is measured and logged (it reveals whether the
// network-level ACL filters same-switch traffic). Skips without a live daemon.
func TestPortFirewallReachabilityLive(t *testing.T) {
	client := liveClient(t)
	b := New(client, testConfig())
	ctx := context.Background()
	alpine := testConfig().Images["alpine"]
	small := testConfig().Sizes["small"]

	suffix := time.Now().UnixNano()
	type netdef struct{ ext, disp, cidr, incus string }
	netA := &netdef{ext: fmt.Sprintf("pa-%d", suffix), disp: "t96a", cidr: "10.77.1.0/24"}
	netB := &netdef{ext: fmt.Sprintf("pb-%d", suffix), disp: "t96b", cidr: "10.77.2.0/24"}
	for _, n := range []*netdef{netA, netB} {
		ref, err := b.DeployNetwork(ctx, builder.NetworkSpec{ExternalName: n.ext, DisplayName: n.disp, Team: "96", CIDR: n.cidr})
		if err != nil {
			if strings.Contains(err.Error(), "OVN") {
				t.Skipf("OVN not available on this daemon: %v", err)
			}
			t.Fatalf("DeployNetwork %s: %v", n.ext, err)
		}
		n.incus = ref
	}

	type instdef struct{ ext, disp, netExt, netDisp, wantAddr, name, ip string }
	probeA := &instdef{ext: fmt.Sprintf("pra-%d", suffix), disp: "probea", netExt: netA.ext, netDisp: netA.disp, wantAddr: "10.77.1.10"}
	targetB := &instdef{ext: fmt.Sprintf("tgb-%d", suffix), disp: "targetb", netExt: netB.ext, netDisp: netB.disp, wantAddr: "10.77.2.10"}
	siblingB := &instdef{ext: fmt.Sprintf("sbb-%d", suffix), disp: "siblingb", netExt: netB.ext, netDisp: netB.disp, wantAddr: "10.77.2.11"}
	insts := []*instdef{probeA, targetB, siblingB}

	t.Cleanup(func() {
		bg := context.Background()
		for _, in := range insts {
			if in.name != "" {
				b.destroyInstance(bg, in.name)
			}
		}
		b.DestroyNetwork(bg, "96", netA.incus)
		b.DestroyNetwork(bg, "96", netB.incus)
	})

	for _, in := range insts {
		name, err := b.deployInstance(ctx, in.ext, in.disp, "96", in.netExt, in.netDisp, in.wantAddr,
			"container", alpine, small, 0, false, "", false)
		if err != nil {
			t.Fatalf("deployInstance %s: %v", in.ext, err)
		}
		in.name = name
	}

	// Wait for each container's eth0 to get an IPv4, and record the REAL IP
	// (so the ACL rule and the probe both use what the guest actually has,
	// not just the requested static address).
	for _, in := range insts {
		in.ip = waitForIPv4(t, b, in.name)
		t.Logf("%s (%s) up at %s", in.disp, in.name, in.ip)
	}

	// Policy: netB visible from netA; the target declares ONLY tcp 80.
	access := []builder.NetworkAccess{
		{ExternalName: netA.ext, DisplayName: netA.disp, CIDR: netA.cidr},
		{ExternalName: netB.ext, DisplayName: netB.disp, CIDR: netB.cidr, VisibleFrom: []string{netA.ext},
			Hosts: []builder.HostAccess{{Address: targetB.ip, ExternalRef: targetB.name, TCPPorts: []string{"80"}}}},
	}
	if err := b.ConfigureNetworkAccess(ctx, "96", access); err != nil {
		t.Fatalf("ConfigureNetworkAccess: %v", err)
	}
	// Give OVN a moment to program the ACL flows.
	time.Sleep(3 * time.Second)

	probe := func(from *instdef, ip string, port int) time.Duration {
		start := time.Now()
		code, err := b.Client.exec(ctx, from.name, "sh", "-c", fmt.Sprintf("nc -w 3 %s %d </dev/null", ip, port))
		d := time.Since(start)
		t.Logf("  %-9s -> %s:%-3d  exit=%d err=%v  took=%s", from.disp, ip, port, code, err, d.Round(50*time.Millisecond))
		return d
	}

	const dropThreshold = 2500 * time.Millisecond // ~nc -w 3 timeout => the port was DROPPED
	const allowCeiling = 1500 * time.Millisecond  // instant RST => the port was ALLOWED (reached, closed)

	t.Log("cross-network (probeA on netA -> targetB on netB):")
	dAllow := probe(probeA, targetB.ip, 80) // declared + visible_from -> allowed
	dDeny := probe(probeA, targetB.ip, 22)  // undeclared -> dropped

	if dAllow > allowCeiling {
		t.Errorf("cross-network declared port 80 took %s, want < %s (allowed by the firewall)", dAllow, allowCeiling)
	}
	if dDeny < dropThreshold {
		t.Errorf("cross-network UNDECLARED port 22 took %s, want > %s (dropped by the firewall)", dDeny, dropThreshold)
	}

	t.Log("same-network (siblingB on netB -> targetB on netB), with the per-host NIC ACL:")
	dSameAllow := probe(siblingB, targetB.ip, 80) // declared -> allowed even same-net (own CIDR is a source)
	dSameDeny := probe(siblingB, targetB.ip, 22)  // undeclared -> dropped at the target's port

	if dSameAllow > allowCeiling {
		t.Errorf("same-network declared port 80 took %s, want < %s (a sibling may reach a declared port)", dSameAllow, allowCeiling)
	}
	if dSameDeny < dropThreshold {
		t.Errorf("same-network UNDECLARED port 22 took %s, want > %s (dropped by the per-host NIC ACL -- same-switch filtering)", dSameDeny, dropThreshold)
	}
	t.Logf("SAME-NETWORK RESULT: :80 %s (allowed), :22 %s (%s)",
		dSameAllow.Round(50*time.Millisecond), dSameDeny.Round(50*time.Millisecond),
		map[bool]string{true: "DROPPED -- same-switch filtering ACTIVE", false: "reachable -- NOT filtered"}[dSameDeny >= dropThreshold])
}

// waitForIPv4 polls an instance's state until eth0 reports an IPv4 address,
// returning it. Fails after a bounded wait.
func waitForIPv4(t *testing.T, b *Builder, name string) string {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := b.Client.get(context.Background(), "/1.0/instances/"+name+"/state")
		if err == nil {
			var st struct {
				Network map[string]struct {
					Addresses []struct {
						Family  string `json:"family"`
						Address string `json:"address"`
						Scope   string `json:"scope"`
					} `json:"addresses"`
				} `json:"network"`
			}
			if json.Unmarshal(raw, &st) == nil {
				if eth0, ok := st.Network["eth0"]; ok {
					for _, a := range eth0.Addresses {
						if a.Family == "inet" && a.Scope == "global" {
							return a.Address
						}
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("instance %s never got an eth0 IPv4 within the deadline", name)
	return ""
}
