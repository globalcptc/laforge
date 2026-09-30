package microcloud

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// DialFromEnv connects to a real, already-provisioned Incus/MicroCloud
// cluster for integration testing, given LAFORGE_INCUS_TEST_URL (e.g.
// "https://cluster-member.example:8443") plus LAFORGE_INCUS_TEST_CLIENT_CERT and
// LAFORGE_INCUS_TEST_CLIENT_KEY (PEM files for a client certificate
// that cluster already trusts -- `lxc config trust add <cert>` /
// `incus config trust add <cert>` on any cluster member). This is the
// real answer to the "needs
// infrastructure this environment doesn't have" gap -- once that
// infrastructure exists somewhere reachable (a real MicroCloud cluster
// on any network path), this is how a test
// run points at it instead of the local single-node Docker daemon
// liveClient/liveIncusClient fall back to.
//
// ok is false with a nil error when none of these env vars are set at
// all -- the normal case, and the caller's own cue to fall back to its
// local-daemon path (or skip, if that isn't available either) rather
// than treating "not configured" as a failure. A non-nil error means the
// env vars were set but something about using them failed (missing a
// paired var, unreadable file, unreachable host, untrusted cert) --
// worth failing the test on, not silently skipping, since it means
// someone meant to run against a real cluster and it didn't work.
func DialFromEnv(ctx context.Context) (client *Client, ok bool, err error) {
	rawURL := os.Getenv("LAFORGE_INCUS_TEST_URL")
	if rawURL == "" {
		return nil, false, nil
	}
	certPath := os.Getenv("LAFORGE_INCUS_TEST_CLIENT_CERT")
	keyPath := os.Getenv("LAFORGE_INCUS_TEST_CLIENT_KEY")
	if certPath == "" || keyPath == "" {
		return nil, false, fmt.Errorf("LAFORGE_INCUS_TEST_URL is set but LAFORGE_INCUS_TEST_CLIENT_CERT/LAFORGE_INCUS_TEST_CLIENT_KEY is not")
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, false, fmt.Errorf("reading LAFORGE_INCUS_TEST_CLIENT_CERT: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, false, fmt.Errorf("reading LAFORGE_INCUS_TEST_CLIENT_KEY: %w", err)
	}
	host := strings.TrimPrefix(strings.TrimPrefix(rawURL, "https://"), "http://")
	serverCertPEM, err := FetchServerCertificateInsecure(ctx, host)
	if err != nil {
		return nil, false, fmt.Errorf("fetching %s's server certificate: %w", host, err)
	}
	c, err := NewClient(rawURL, certPEM, keyPEM, serverCertPEM, "")
	if err != nil {
		return nil, false, fmt.Errorf("constructing client: %w", err)
	}
	return c, true, nil
}
