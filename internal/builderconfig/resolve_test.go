package builderconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/builder/incuspool"
	"github.com/globalcptc/laforge/internal/builder/microcloud"
	"github.com/globalcptc/laforge/internal/db"
)

func strPtr(s string) *string { return &s }

func TestResolveFakeReturnsFakeBuilder(t *testing.T) {
	b, err := Resolve(nil, db.BuilderConfig{Name: "test-fake", Kind: "fake"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if _, ok := b.(*fake.Builder); !ok {
		t.Fatalf("Resolve(kind=fake) = %T, want *fake.Builder", b)
	}
}

func TestResolveUnknownKindFails(t *testing.T) {
	if _, err := Resolve(nil, db.BuilderConfig{Name: "test-bogus", Kind: "bogus"}); err == nil {
		t.Fatal("expected an error for an unknown kind")
	}
}

func TestResolveMicrocloudRequiresApiUrl(t *testing.T) {
	_, err := Resolve(nil, db.BuilderConfig{Name: "test-microcloud", Kind: "microcloud"})
	if err == nil {
		t.Fatal("expected an error for a missing incus_api_url")
	}
}

func TestResolveMicrocloudRequiresClientCredentials(t *testing.T) {
	_, err := Resolve(nil, db.BuilderConfig{
		Name: "test-microcloud", Kind: "microcloud", IncusApiUrl: strPtr("https://example.invalid:8443"),
	})
	if err == nil {
		t.Fatal("expected an error for missing client cert/key paths")
	}
}

func TestResolveMicrocloudRequiresServerCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	os.WriteFile(certPath, []byte("not-really-a-cert"), 0o600)
	os.WriteFile(keyPath, []byte("not-really-a-key"), 0o600)

	_, err := Resolve(nil, db.BuilderConfig{
		Name: "test-microcloud", Kind: "microcloud", IncusApiUrl: strPtr("https://example.invalid:8443"),
		IncusClientCertPath: &certPath, IncusClientKeyPath: &keyPath,
	})
	if err == nil {
		t.Fatal("expected an error for a missing incus_server_cert_pem")
	}
}

// TestResolveMicrocloudConstructsARealClient proves the full, real
// assembly path -- a genuine client cert/key pair (agentpki.GenerateCA,
// the same technique internal/builder/incus's own tests use), a real (if
// unreachable) server cert PEM, and real images/sizes JSON -- succeeds
// and produces a real *microcloud.Builder, without needing a live daemon:
// NewClient only parses and pins locally, it doesn't dial anything.
func TestResolveMicrocloudConstructsARealClient(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")

	ca, err := agentpki.GenerateCA("builderconfig-resolve-test")
	if err != nil {
		t.Fatalf("generating test cert: %v", err)
	}
	os.WriteFile(certPath, ca.CertPEM, 0o600)
	os.WriteFile(keyPath, ca.KeyPEM, 0o600)

	images := []byte(`{"ubuntu22": {"alias": "ubuntu22", "vm": true}}`)
	sizes := []byte(`{"small": {"cpu": "1", "memory": "1GiB"}}`)
	timeout := int32(300)

	row := db.BuilderConfig{
		Name: "test-microcloud-real", Kind: "microcloud",
		IncusApiUrl:                  strPtr("https://example.invalid:8443"),
		IncusClientCertPath:          &certPath,
		IncusClientKeyPath:           &keyPath,
		IncusServerCertPem:           strPtr(string(ca.CertPEM)), // any valid PEM cert pins fine for this test's purposes
		IncusOvnUplinkNetwork:        strPtr("UPLINK"),
		IncusStoragePool:             strPtr("remote"),
		IncusOperationTimeoutSeconds: &timeout,
		IncusImages:                  images,
		IncusSizes:                   sizes,
	}

	b, err := Resolve(nil, row)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	ib, ok := b.(*microcloud.Builder)
	if !ok {
		t.Fatalf("Resolve(kind=microcloud) = %T, want *microcloud.Builder", b)
	}
	if ib.Config.OVNUplinkNetwork != "UPLINK" {
		t.Errorf("OVNUplinkNetwork = %q, want UPLINK", ib.Config.OVNUplinkNetwork)
	}
	if ib.Config.StoragePool != "remote" {
		t.Errorf("StoragePool = %q, want remote", ib.Config.StoragePool)
	}
	if img, ok := ib.Config.Images["ubuntu22"]; !ok || img.Alias != "ubuntu22" || !img.VM {
		t.Errorf("Images[ubuntu22] = %+v, ok=%v, want alias=ubuntu22 vm=true", img, ok)
	}
	if size, ok := ib.Config.Sizes["small"]; !ok || size.CPU != "1" || size.Memory != "1GiB" {
		t.Errorf("Sizes[small] = %+v, ok=%v, want cpu=1 memory=1GiB", size, ok)
	}
	if ib.Client.OperationTimeout.Seconds() != 300 {
		t.Errorf("OperationTimeout = %v, want 300s", ib.Client.OperationTimeout)
	}
}

// TestResolveIncusPoolRequiresAtLeastOneHost proves kind "incus" now
// means the pool, not a single endpoint: an empty (or absent)
// incus_hosts array is refused even though every other field is absent
// too -- there's no single incus_api_url fallback anymore.
func TestResolveIncusPoolRequiresAtLeastOneHost(t *testing.T) {
	_, err := Resolve(nil, db.BuilderConfig{Name: "test-incus-pool", Kind: "incus", IncusHosts: []byte("[]")})
	if err == nil {
		t.Fatal("expected an error for an empty incus_hosts array")
	}
}

func TestResolveIncusPoolRequiresPerHostFields(t *testing.T) {
	hosts := []byte(`[{"api_url": "https://example.invalid:8443"}]`) // missing cert/key/server cert
	_, err := Resolve(nil, db.BuilderConfig{Name: "test-incus-pool", Kind: "incus", IncusHosts: hosts})
	if err == nil {
		t.Fatal("expected an error for a host missing required fields")
	}
}

// TestResolveIncusPoolConstructsARealPool proves the real, multi-host
// assembly path: two real (if unreachable) hosts, each with their own
// cert/key/OVN uplink/storage pool, produce a real *incuspool.Pool with
// two real *incus.Builder entries carrying their own, independent Config
// -- not one config shared across both, since two genuinely independent
// hosts can each need their own uplink network name and storage pool
// (see migration 00014's own doc comment).
func TestResolveIncusPoolConstructsARealPool(t *testing.T) {
	dir := t.TempDir()
	ca, err := agentpki.GenerateCA("builderconfig-pool-resolve-test")
	if err != nil {
		t.Fatalf("generating test cert: %v", err)
	}
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	os.WriteFile(certPath, ca.CertPEM, 0o600)
	os.WriteFile(keyPath, ca.KeyPEM, 0o600)
	serverCert := string(ca.CertPEM)

	hostConfigs := []incus.HostConfig{
		{
			APIURL: "https://host-a.invalid:8443", ClientCertPath: certPath, ClientKeyPath: keyPath,
			ServerCertPEM: serverCert, OVNUplinkNetwork: "UPLINK-A", StoragePool: "pool-a",
		},
		{
			APIURL: "https://host-b.invalid:8443", ClientCertPath: certPath, ClientKeyPath: keyPath,
			ServerCertPEM: serverCert, OVNUplinkNetwork: "UPLINK-B", StoragePool: "pool-b",
		},
	}
	hostsJSON, err := json.Marshal(hostConfigs)
	if err != nil {
		t.Fatalf("marshaling host configs: %v", err)
	}

	images := []byte(`{"ubuntu22": {"alias": "ubuntu22"}}`)
	row := db.BuilderConfig{
		Name: "test-incus-pool-real", Kind: "incus",
		IncusHosts:  hostsJSON,
		IncusImages: images, IncusSizes: []byte("{}"),
	}

	b, err := Resolve(nil, row)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	p, ok := b.(*incuspool.Pool)
	if !ok {
		t.Fatalf("Resolve(kind=incus) = %T, want *incuspool.Pool", b)
	}
	if len(p.Hosts) != 2 {
		t.Fatalf("Pool.Hosts = %d, want 2", len(p.Hosts))
	}
	if p.Hosts[0].Config.OVNUplinkNetwork != "UPLINK-A" || p.Hosts[1].Config.OVNUplinkNetwork != "UPLINK-B" {
		t.Errorf("hosts don't carry their own independent OVNUplinkNetwork: %q, %q",
			p.Hosts[0].Config.OVNUplinkNetwork, p.Hosts[1].Config.OVNUplinkNetwork)
	}
	if p.Hosts[0].Config.StoragePool != "pool-a" || p.Hosts[1].Config.StoragePool != "pool-b" {
		t.Errorf("hosts don't carry their own independent StoragePool: %q, %q",
			p.Hosts[0].Config.StoragePool, p.Hosts[1].Config.StoragePool)
	}
	if _, ok := p.Images["ubuntu22"]; !ok {
		t.Errorf("Pool.Images missing the shared ubuntu22 entry: %+v", p.Images)
	}
}
