package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/db"
)

// TestBuilderConfigCRUDEndToEnd is the "what about
// the MicroCloud builder" gap, closed at the admin-API layer: create a
// real "fake"-kind builder config through the real HTTP endpoints, list
// it, fetch it by name, update it, delete it -- instance-admin gated the
// same way installations.go's own approve flow is, not per-repository
// levelAdmin, since a builder config is shared infrastructure.
func TestBuilderConfigCRUDEndToEnd(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin"}
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = 'test-fake'")
	})

	adminClient := signedInClient(t, env, "builder-config-admin")
	nonAdminClient := signedInClient(t, env, "not-an-admin")

	body, _ := json.Marshal(builderConfigRequest{Kind: "fake"})

	// A signed-in user who isn't an instance admin can't create one.
	resp, err := nonAdminClient.Post(env.httpURL+"/builder-configs/test-fake", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create as non-admin: status = %d, want 403", resp.StatusCode)
	}

	// An instance admin can.
	resp, err = adminClient.Post(env.httpURL+"/builder-configs/test-fake", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var created db.BuilderConfig
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", resp.StatusCode)
	}
	if created.Name != "test-fake" || created.Kind != "fake" {
		t.Fatalf("created = %+v, want name=test-fake kind=fake", created)
	}

	// It shows up in the list.
	resp, err = adminClient.Get(env.httpURL + "/builder-configs")
	if err != nil {
		t.Fatal(err)
	}
	var list []db.BuilderConfig
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	found := false
	for _, bc := range list {
		if bc.Name == "test-fake" {
			found = true
		}
	}
	if !found {
		t.Fatalf("list = %+v, want test-fake present", list)
	}

	// Fetch by name.
	resp, err = adminClient.Get(env.httpURL + "/builder-configs/test-fake")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: status = %d, want 200", resp.StatusCode)
	}

	// Update it -- change kind's own config, still "fake" here.
	updateBody, _ := json.Marshal(builderConfigRequest{Kind: "fake"})
	req, _ := http.NewRequest(http.MethodPut, env.httpURL+"/builder-configs/test-fake", bytes.NewReader(updateBody))
	resp, err = adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: status = %d, want 200", resp.StatusCode)
	}

	// Delete it.
	req, _ = http.NewRequest(http.MethodDelete, env.httpURL+"/builder-configs/test-fake", nil)
	resp, err = adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want 204", resp.StatusCode)
	}
	resp, err = adminClient.Get(env.httpURL + "/builder-configs/test-fake")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: status = %d, want 404", resp.StatusCode)
	}
}

// TestBuilderConfigCreateRejectsInvalidKind and
// TestBuilderConfigCreateRejectsBrokenMicrocloudConfig prove the real,
// immediate validation builderConfigRequest.validate runs (via
// internal/builderconfig.Resolve) -- a broken config is refused at
// registration time, not only discovered once a real deploy tries to
// use it.
func TestBuilderConfigCreateRejectsInvalidKind(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin2"}
	adminClient := signedInClient(t, env, "builder-config-admin2")

	body, _ := json.Marshal(map[string]string{"kind": "not-a-real-kind"})
	resp, err := adminClient.Post(env.httpURL+"/builder-configs/test-bad-kind", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create with invalid kind: status = %d, want 400", resp.StatusCode)
	}
}

func TestBuilderConfigCreateRejectsBrokenMicrocloudConfig(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin3"}
	adminClient := signedInClient(t, env, "builder-config-admin3")

	// kind=microcloud with no incus_api_url at all -- must fail validation.
	body, _ := json.Marshal(builderConfigRequest{Kind: "microcloud"})
	resp, err := adminClient.Post(env.httpURL+"/builder-configs/test-broken-microcloud", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create with no incus_api_url: status = %d, want 400", resp.StatusCode)
	}
}

// TestBuilderConfigCreateRejectsIncusPoolWithNoHosts proves the new
// kind="incus" pool path's own required field: incus_hosts must list at
// least one host, since there's no single incus_api_url fallback anymore
// for this kind (see migration 00014's own doc comment).
func TestBuilderConfigCreateRejectsIncusPoolWithNoHosts(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin5"}
	adminClient := signedInClient(t, env, "builder-config-admin5")

	body, _ := json.Marshal(builderConfigRequest{Kind: "incus"})
	resp, err := adminClient.Post(env.httpURL+"/builder-configs/test-empty-incus-pool", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create with no incus_hosts: status = %d, want 400", resp.StatusCode)
	}
}

// TestBuilderConfigCreateAcceptsARealMicrocloudConfig proves a genuinely
// well-formed "microcloud" kind config -- real client cert/key files, a
// real PEM server cert, real images/sizes -- is accepted and round-trips
// correctly, with no live cluster needed (validate's own Resolve call
// never dials the network for kind=microcloud, and a remote-pulled image
// is exempt from save-time image verification -- see
// TestBuilderImagesMustExistOnEveryHost for the local-image case).
func TestBuilderConfigCreateAcceptsARealMicrocloudConfig(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin4"}
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = 'test-real-microcloud'")
	})
	adminClient := signedInClient(t, env, "builder-config-admin4")

	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	ca, err := agentpki.GenerateCA("builder-config-api-test")
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	os.WriteFile(certPath, ca.CertPEM, 0o600)
	os.WriteFile(keyPath, ca.KeyPEM, 0o600)

	req := builderConfigRequest{
		Kind: "microcloud", IncusApiUrl: "https://example.invalid:8443",
		IncusClientCertPath: certPath, IncusClientKeyPath: keyPath,
		IncusServerCertPem:    string(ca.CertPEM),
		IncusOvnUplinkNetwork: "UPLINK", IncusStoragePool: "remote",
		IncusImages: map[string]incus.ImageRef{"ubuntu22": {Alias: "ubuntu/22.04", Server: "https://images.linuxcontainers.org", Protocol: "simplestreams", VM: true}},
		IncusSizes:  map[string]incus.SizeSpec{"small": {CPU: "1", Memory: "1GiB"}},
	}
	body, _ := json.Marshal(req)
	resp, err := adminClient.Post(env.httpURL+"/builder-configs/test-real-microcloud", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var created db.BuilderConfig
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", resp.StatusCode)
	}
	if created.IncusStoragePool == nil || *created.IncusStoragePool != "remote" {
		t.Fatalf("incus_storage_pool = %v, want remote", created.IncusStoragePool)
	}
}

// TestBuilderConfigCreateAcceptsARealIncusPoolConfig proves the new
// multi-host pool path end to end at the admin-API layer: two real (if
// unreachable) hosts, each with their own credentials and OVN uplink,
// round-trip correctly through incus_hosts.
func TestBuilderConfigCreateAcceptsARealIncusPoolConfig(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-builder-config-admin6"}
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = 'test-real-incus-pool'")
	})
	adminClient := signedInClient(t, env, "builder-config-admin6")

	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	ca, err := agentpki.GenerateCA("builder-config-api-pool-test")
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	os.WriteFile(certPath, ca.CertPEM, 0o600)
	os.WriteFile(keyPath, ca.KeyPEM, 0o600)

	req := builderConfigRequest{
		Kind: "incus",
		IncusHosts: []incus.HostConfig{
			{
				APIURL: "https://host-a.invalid:8443", ClientCertPath: certPath, ClientKeyPath: keyPath,
				ServerCertPEM: string(ca.CertPEM), OVNUplinkNetwork: "UPLINK-A", StoragePool: "default",
			},
			{
				APIURL: "https://host-b.invalid:8443", ClientCertPath: certPath, ClientKeyPath: keyPath,
				ServerCertPEM: string(ca.CertPEM), OVNUplinkNetwork: "UPLINK-B", StoragePool: "default",
			},
		},
		IncusImages: map[string]incus.ImageRef{"ubuntu22": {Alias: "ubuntu/22.04", Server: "https://images.linuxcontainers.org", Protocol: "simplestreams"}},
	}
	body, _ := json.Marshal(req)
	resp, err := adminClient.Post(env.httpURL+"/builder-configs/test-real-incus-pool", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var created db.BuilderConfig
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", resp.StatusCode)
	}
	var hosts []incus.HostConfig
	if err := json.Unmarshal(created.IncusHosts, &hosts); err != nil {
		t.Fatalf("decoding created.IncusHosts: %v", err)
	}
	if len(hosts) != 2 || hosts[0].OVNUplinkNetwork != "UPLINK-A" || hosts[1].OVNUplinkNetwork != "UPLINK-B" {
		t.Fatalf("incus_hosts round-tripped incorrectly: %+v", hosts)
	}
}
