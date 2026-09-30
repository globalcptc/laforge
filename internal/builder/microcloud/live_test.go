package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// This file is the real MicroCloud smoke test: the LXD builder split was
// code-complete but, until this ran, never exercised against a live
// MicroCloud/LXD cluster ("MicroCloud smoke-test
// against real hardware"). Every test here SKIPS (does not fail) unless
// LAFORGE_INCUS_TEST_URL / _CLIENT_CERT / _CLIENT_KEY point at a real cluster
// member -- so `go test ./...` never depends on one -- and drives the public
// builder.Builder surface end to end (discover, network, host, nested-Docker
// container, access), cleaning up every resource it creates even on failure.
//
// Verified live against a 4-node MicroCloud cluster (LXD + MicroCeph + OVN).

// liveBuilder dials the cluster from the env, or skips. It returns a Builder
// wired to a Config with the real cluster's pool/uplink names and its
// already-present laforge-docker-base image (a CONTAINER image, so host deploys
// don't wait on a remote image pull).
func liveBuilder(t *testing.T) *Builder {
	t.Helper()
	c, ok, err := DialFromEnv(context.Background())
	if err != nil {
		t.Fatalf("DialFromEnv: %v", err)
	}
	if !ok {
		t.Skip("no MicroCloud configured -- set LAFORGE_INCUS_TEST_URL / LAFORGE_INCUS_TEST_CLIENT_CERT / LAFORGE_INCUS_TEST_CLIENT_KEY to a real cluster member")
	}
	return New(c, liveConfig(t))
}

func liveConfig(t *testing.T) Config {
	t.Helper()
	// dockerBaseFingerprint is discovered live rather than hardcoded, so the
	// test survives a base-image rebuild. Falls back to "" (container test then
	// skips) if the cluster has no laforge-docker-base.
	fp := findImageFingerprint(t, "laforge-docker-base")
	return Config{
		Images: map[string]ImageRef{
			// The base image doubles as a fast host image here: it's already on
			// the cluster, so DeployHost needs no remote pull.
			"base": {Fingerprint: fp},
		},
		Sizes:                 map[string]SizeSpec{"small": {CPU: "1", Memory: "512MiB"}},
		OVNUplinkNetwork:      "UPLINK",
		StoragePool:           "remote",
		DockerBaseFingerprint: fp,
	}
}

func findImageFingerprint(t *testing.T, alias string) string {
	t.Helper()
	c, ok, err := DialFromEnv(context.Background())
	if err != nil || !ok {
		return ""
	}
	imgs, err := c.ListImages(context.Background())
	if err != nil {
		return ""
	}
	for _, img := range imgs {
		for _, a := range img.Aliases {
			if a == alias {
				return img.Fingerprint
			}
		}
	}
	return ""
}

// uniqueName gives a per-run, deterministic-per-call external name; the builder
// hashes it into the real LXD resource name.
func uniqueName(t *testing.T, kind string) string {
	t.Helper()
	return fmt.Sprintf("smoke-%s-%s-%d", kind, sanitizeCompact(t.Name()), time.Now().UnixNano())
}

// testCIDR picks a /24 well clear of the cluster's existing networks
// (10.0.1.0/24, 10.93.57.0/24) to avoid OVN uplink route collisions.
func testCIDR() string {
	return fmt.Sprintf("10.%d.%d.0/24", 200+rand.Intn(40), 1+rand.Intn(240))
}

func TestMicroCloudDiscoveryLive(t *testing.T) {
	b := liveBuilder(t)
	ctx := context.Background()

	pools, err := b.Client.ListStoragePools(ctx)
	if err != nil {
		t.Fatalf("ListStoragePools: %v", err)
	}
	var haveCeph bool
	for _, p := range pools {
		if p.Driver == "ceph" {
			haveCeph = true
		}
	}
	if !haveCeph {
		t.Errorf("no ceph storage pool found live; pools=%+v", pools)
	}

	nets, err := b.Client.ListNetworks(ctx)
	if err != nil {
		t.Fatalf("ListNetworks: %v", err)
	}
	var haveUplink bool
	for _, n := range nets {
		if n.Name == "UPLINK" {
			haveUplink = true
		}
	}
	if !haveUplink {
		t.Errorf("OVN uplink %q not found live; networks=%+v", "UPLINK", nets)
	}

	imgs, err := b.Client.ListImages(ctx)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(imgs) == 0 {
		t.Errorf("ListImages returned nothing live -- expected the cluster's own images")
	}
}

func TestMicroCloudNetworkLifecycleLive(t *testing.T) {
	b := liveBuilder(t)
	ctx := context.Background()

	spec := builder.NetworkSpec{
		ExternalName: uniqueName(t, "net"),
		DisplayName:  "t97lab",
		Team:         "97",
		CIDR:         testCIDR(),
	}
	ref, err := b.DeployNetwork(ctx, spec)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), spec.Team, ref) })

	// Adopt: a second deploy of the same spec must not error or duplicate.
	ref2, err := b.DeployNetwork(ctx, spec)
	if err != nil {
		t.Fatalf("DeployNetwork (adopt): %v", err)
	}
	if ref2 != ref {
		t.Fatalf("adopt returned a different ref: %q vs %q", ref2, ref)
	}

	// Inspect must report our network (lf- prefixed, managed).
	if !inspectHas(t, b, ref, "network") {
		t.Errorf("Inspect did not list deployed network %q", ref)
	}

	if err := b.DestroyNetwork(ctx, spec.Team, ref); err != nil {
		t.Fatalf("DestroyNetwork: %v", err)
	}
	// Idempotent: destroying an already-gone network is success.
	if err := b.DestroyNetwork(ctx, spec.Team, ref); err != nil {
		t.Fatalf("DestroyNetwork (second, idempotent): %v", err)
	}
}

func TestMicroCloudHostLifecycleAndAccessLive(t *testing.T) {
	b := liveBuilder(t)
	if b.Config.Images["base"].Fingerprint == "" {
		t.Skip("no laforge-docker-base image on the cluster to use as a fast host image")
	}
	ctx := context.Background()

	net := builder.NetworkSpec{ExternalName: uniqueName(t, "net"), DisplayName: "t97hostnet", Team: "97", CIDR: testCIDR()}
	netRef, err := b.DeployNetwork(ctx, net)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), net.Team, netRef) })

	host := builder.HostSpec{
		ExternalName:       uniqueName(t, "host"),
		DisplayName:        "t97web",
		Team:               "97",
		Network:            net.ExternalName,
		NetworkDisplayName: net.DisplayName,
		OS:                 "base",
		Size:               "small",
	}
	hostRef, err := b.DeployHost(ctx, host)
	// Always attempt teardown, even if deploy half-succeeded (a partial
	// instance still returns its deterministic ref).
	if hostRef != "" {
		t.Cleanup(func() { _ = b.DestroyHost(context.Background(), host.Team, hostRef) })
	}
	if err != nil {
		t.Fatalf("DeployHost: %v", err)
	}

	// It should come up running within a bounded window.
	if !waitInstanceState(t, b, hostRef, builder.PowerStateRunning, 90*time.Second) {
		t.Fatalf("host %q never reached running", hostRef)
	}

	// Adopt: a second identical deploy must not error or make a second instance.
	if ref2, err := b.DeployHost(ctx, host); err != nil || ref2 != hostRef {
		t.Fatalf("DeployHost (adopt) = %q, %v; want %q, nil", ref2, err, hostRef)
	}

	// Access: CloseAccess removes the NIC (drops the interface, killing live
	// connections); OpenAccess restores the exact same attachment.
	if err := b.CloseAccess(ctx, host.Team); err != nil {
		t.Fatalf("CloseAccess: %v", err)
	}
	if working, err := b.HasWorkingEth0(ctx, hostRef); err != nil {
		t.Fatalf("HasWorkingEth0 after close: %v", err)
	} else if working {
		t.Errorf("after CloseAccess, host %q still has a working eth0 (connection not severed)", hostRef)
	}
	if err := b.OpenAccess(ctx, host.Team); err != nil {
		t.Fatalf("OpenAccess: %v", err)
	}
	if working, err := b.HasWorkingEth0(ctx, hostRef); err != nil {
		t.Fatalf("HasWorkingEth0 after open: %v", err)
	} else if !working {
		t.Errorf("after OpenAccess, host %q did not get a working eth0 back", hostRef)
	}

	if err := b.DestroyHost(ctx, host.Team, hostRef); err != nil {
		t.Fatalf("DestroyHost: %v", err)
	}
	if err := b.DestroyHost(ctx, host.Team, hostRef); err != nil {
		t.Fatalf("DestroyHost (second, idempotent): %v", err)
	}
}

func TestMicroCloudNestedDockerContainerLive(t *testing.T) {
	b := liveBuilder(t)
	if b.Config.DockerBaseFingerprint == "" {
		t.Skip("no laforge-docker-base image on the cluster -- nested-Docker container path can't be tested")
	}
	ctx := context.Background()

	net := builder.NetworkSpec{ExternalName: uniqueName(t, "net"), DisplayName: "t97ctrnet", Team: "97", CIDR: testCIDR()}
	netRef, err := b.DeployNetwork(ctx, net)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), net.Team, netRef) })

	ctr := builder.ContainerSpec{
		ExternalName:       uniqueName(t, "ctr"),
		DisplayName:        "t97svc",
		Team:               "97",
		Network:            net.ExternalName,
		NetworkDisplayName: net.DisplayName,
		Image:              "nginx:alpine", // the OCI ref a materialized step would `docker run`
		Size:               "small",
	}
	ref, err := b.DeployContainer(ctx, ctr)
	if ref != "" {
		t.Cleanup(func() { _ = b.DestroyContainer(context.Background(), ctr.Team, ref) })
	}
	if err != nil {
		t.Fatalf("DeployContainer: %v", err)
	}

	// The LXD system container (nesting host for Docker) should be running.
	if !waitInstanceState(t, b, ref, builder.PowerStateRunning, 90*time.Second) {
		t.Fatalf("container host %q never reached running", ref)
	}
	// It must have security.nesting set (the whole point of this path).
	if !instanceConfigTrue(t, b, ref, "security.nesting") {
		t.Errorf("container %q missing security.nesting=true", ref)
	}

	if err := b.DestroyContainer(ctx, ctr.Team, ref); err != nil {
		t.Fatalf("DestroyContainer: %v", err)
	}
}

// --- live helpers (same package, so they can use the raw client) ---

func inspectHas(t *testing.T, b *Builder, ref, kind string) bool {
	t.Helper()
	res, err := b.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	for _, r := range res {
		if r.ExternalRef == ref && r.Kind == kind {
			return true
		}
	}
	return false
}

func waitInstanceState(t *testing.T, b *Builder, ref, want string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		res, err := b.Inspect(context.Background())
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		for _, r := range res {
			if r.ExternalRef == ref && r.State == want {
				return true
			}
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

func rawInstance(t *testing.T, b *Builder, ref string) map[string]any {
	t.Helper()
	raw, err := b.Client.get(context.Background(), "/1.0/instances/"+ref)
	if err != nil {
		t.Fatalf("GET instance %s: %v", ref, err)
	}
	var inst map[string]any
	if err := json.Unmarshal(raw, &inst); err != nil {
		t.Fatalf("decode instance %s: %v", ref, err)
	}
	return inst
}

func instanceConfigTrue(t *testing.T, b *Builder, ref, key string) bool {
	t.Helper()
	inst := rawInstance(t, b, ref)
	config, _ := inst["config"].(map[string]any)
	v, _ := config[key].(string)
	return strings.EqualFold(v, "true")
}
