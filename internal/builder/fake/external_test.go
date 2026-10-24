package fake

import (
	"context"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

func TestFakeConfigureExternalAccess(t *testing.T) {
	b := &Builder{}
	hosts := []builder.ExternalHost{
		{ExternalRef: "h1", Address: "10.0.1.10", TCPPorts: []string{"3389"}},
		{ExternalRef: "h2", Address: "10.0.1.11", TCPPorts: []string{"22"}, UDPPorts: []string{"53"}},
	}
	eps, err := b.ConfigureExternalAccess(context.Background(), "1", hosts)
	if err != nil {
		t.Fatalf("ConfigureExternalAccess: %v", err)
	}
	if len(eps) != 3 { // 3389 + 22 + 53
		t.Fatalf("got %d endpoints, want 3", len(eps))
	}
	// Each endpoint gets a distinct external port, and PublicAddress is ip:port.
	seen := map[string]bool{}
	for _, e := range eps {
		if e.PublicAddress == "" || e.ExternalPort == "" {
			t.Errorf("endpoint missing address/port: %+v", e)
		}
		if seen[e.PublicAddress] {
			t.Errorf("duplicate public address %q", e.PublicAddress)
		}
		seen[e.PublicAddress] = true
	}
}
