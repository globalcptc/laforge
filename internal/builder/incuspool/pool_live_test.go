package incuspool

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/incus"
)

// livePoolOfOne is the degenerate but real case this session's own
// infrastructure can actually prove: a "pool" of exactly one real host,
// via the same LAFORGE_INCUS_TEST_URL/CLIENT_CERT/CLIENT_KEY env vars
// internal/builder/incus's own live tests use (see incus.DialFromEnv).
// True multi-host routing (two distinct real daemons) is proven by
// TestHostForTeamIsDeterministicModuloAssignment in pool_test.go without
// needing a live daemon at all -- hostForTeam's own logic doesn't care
// whether a *incus.Builder is reachable, only how many there are -- so
// this live test's job is narrower and different: proving the Pool
// actually dispatches a real Deploy/Destroy call through to a real host,
// not just picking the right index.
func livePoolOfOne(t *testing.T) *Pool {
	t.Helper()
	client, ok, err := incus.DialFromEnv(context.Background())
	if err != nil {
		t.Fatalf("DialFromEnv: %v", err)
	}
	if !ok {
		t.Skip("no live Incus daemon configured (set LAFORGE_INCUS_TEST_URL/CLIENT_CERT/CLIENT_KEY -- see incus.DialFromEnv)")
	}
	host := incus.New(client, incus.Config{OVNUplinkNetwork: "UPLINK"})
	return New([]*incus.Builder{host}, nil)
}

// TestPoolDispatchesARealDeployAndDestroyThroughItsOneHost is the real,
// live proof that incuspool.Pool is not just routing logic in isolation:
// a real DeployNetwork call for team "1" actually reaches the real host
// and creates a real OVN network, and DestroyNetwork actually removes it
// -- both going through Pool's own hostForTeam dispatch, not calling the
// underlying incus.Builder directly.
func TestPoolDispatchesARealDeployAndDestroyThroughItsOneHost(t *testing.T) {
	p := livePoolOfOne(t)
	ctx := context.Background()

	// A long, deterministic-style name -- the same shape internal/runner's
	// own deterministicExternalName produces -- deliberately: proves the
	// pool's dispatch works with realistic names too. incus.Builder itself
	// shortens anything over Incus's own 15-character network name limit
	// (see its own shortName doc comment) and returns the shortened name
	// as the real ref, so this test asserts on whatever ref comes back,
	// not the literal input.
	name := fmt.Sprintf("lf-pool-live-team1-network-%d", time.Now().UnixNano())
	ref, err := p.DeployNetwork(ctx, builder.NetworkSpec{
		ExternalName: name, Team: "1", CIDR: "10.251.251.0/24",
	})
	if err != nil {
		var apiErr *incus.APIError
		if errors.As(err, &apiErr) && apiErr.OVNUnavailable() {
			t.Skipf("OVN isn't available on this daemon: %v", err)
		}
		t.Fatalf("Pool.DeployNetwork: %v", err)
	}

	resources, err := p.Inspect(ctx)
	if err != nil {
		t.Fatalf("Pool.Inspect: %v", err)
	}
	found := false
	for _, r := range resources {
		if r.ExternalRef == ref && r.Kind == "network" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Pool.Inspect did not list the network %s (ref returned by DeployNetwork) it should have found: %+v", ref, resources)
	}

	if err := p.DestroyNetwork(ctx, "1", ref); err != nil {
		t.Fatalf("Pool.DestroyNetwork: %v", err)
	}
	// Safe to call again -- same "ensure" idempotency every Destroy* call
	// in this codebase requires.
	if err := p.DestroyNetwork(ctx, "1", ref); err != nil {
		t.Fatalf("Pool.DestroyNetwork (already destroyed, must be idempotent): %v", err)
	}
}
