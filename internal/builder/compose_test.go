package builder

import "testing"

// A compose container's machine is the host its spec describes: same place on
// the network, the builder's compose-host image, the agent by cloud-init.
func TestComposeHostSpec(t *testing.T) {
	c := ContainerSpec{
		ExternalName: "ext", DisplayName: "flaky01", Team: "3", Network: "net", NetworkDisplayName: "t3-prod",
		Address: "10.0.1.30", Size: "medium", DiskGB: 60, TCPPorts: []string{"80", "443"}, UDPPorts: []string{"53"},
		CloudInitUserData: "#cloud-config", ComposeHost: true,
	}
	h := c.ComposeHostSpec()
	if h.OS != ComposeHostImage || h.ExternalName != "ext" || h.DisplayName != "flaky01" || h.Team != "3" ||
		h.Network != "net" || h.NetworkDisplayName != "t3-prod" || h.Address != "10.0.1.30" ||
		h.Size != "medium" || h.DiskGB != 60 || len(h.TCPPorts) != 2 || len(h.UDPPorts) != 1 ||
		h.CloudInitUserData != "#cloud-config" {
		t.Errorf("ComposeHostSpec() = %+v", h)
	}
}
