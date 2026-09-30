package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeIncus is just enough of a real Incus daemon for the builder
// workflow: token enrollment (a presented client certificate becomes
// trusted only on POST /1.0/certificates with the right token) and
// discovery (storage pools, networks, images), all refused to an
// untrusted client the way a real daemon refuses them.
type fakeIncus struct {
	srv     *httptest.Server
	token   string
	mu      sync.Mutex
	trusted map[string]bool
	images  []map[string]interface{}
}

func newFakeIncus(t *testing.T) *fakeIncus {
	t.Helper()
	f := &fakeIncus{trusted: map[string]bool{}, images: []map[string]interface{}{{
		"fingerprint": "abc", "type": "virtual-machine", "architecture": "x86_64",
		"properties": map[string]string{"os": "Windows"}, "aliases": []map[string]string{{"name": "win2019-core"}},
	}}}
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(f.handle))
	f.srv.TLS = &tls.Config{ClientAuth: tls.RequestClientCert, Certificates: []tls.Certificate{selfSignedServerCert(t)}}
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)
	sum := sha256.Sum256(f.srv.Certificate().Raw)
	payload, _ := json.Marshal(map[string]interface{}{
		"client_name": "laforge",
		"fingerprint": hex.EncodeToString(sum[:]),
		"addresses":   []string{strings.TrimPrefix(f.srv.URL, "https://")},
		"secret":      "s3cret",
	})
	f.token = base64.StdEncoding.EncodeToString(payload)
	return f
}

func selfSignedServerCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "fake-incus"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func (f *fakeIncus) handle(w http.ResponseWriter, r *http.Request) {
	reply := func(metadata interface{}) {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status": "Success", "status_code": 200, "metadata": metadata})
	}
	deny := func() {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 403, "error": "not authorized"})
	}
	fp := ""
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		sum := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
		fp = hex.EncodeToString(sum[:])
	}
	f.mu.Lock()
	trusted := f.trusted[fp]
	f.mu.Unlock()

	switch {
	case r.URL.Path == "/1.0" && r.Method == http.MethodGet:
		auth := "untrusted"
		if trusted {
			auth = "trusted"
		}
		reply(map[string]interface{}{"auth": auth, "api_extensions": []string{"explicit_trust_token"}, "environment": map[string]string{"server_name": "fake-incus"}})
	case r.URL.Path == "/1.0/certificates" && r.Method == http.MethodPost:
		var body struct {
			TrustToken string `json:"trust_token"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.TrustToken != f.token || fp == "" {
			deny()
			return
		}
		f.mu.Lock()
		f.trusted[fp] = true
		f.mu.Unlock()
		reply(map[string]interface{}{})
	case !trusted:
		deny()
	case r.URL.Path == "/1.0/storage-pools":
		reply([]map[string]string{{"name": "default", "driver": "zfs"}, {"name": "fast", "driver": "lvm"}})
	case r.URL.Path == "/1.0/networks":
		reply([]map[string]interface{}{{"name": "UPLINK", "type": "physical", "managed": true}, {"name": "incusbr0", "type": "bridge", "managed": true}})
	case r.URL.Path == "/1.0/images":
		reply(f.images)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func connectBuilder(t *testing.T, client *http.Client, env *apiTestEnv, token string) (int, []byte) {
	t.Helper()
	body, _ := json.Marshal(connectBuilderRequest{Token: token})
	resp, err := client.Post(env.httpURL+"/builder-connections", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// TestConnectBuilderEnrollsAndDiscovers is the workflow's real first step:
// one pasted token produces a real, trusted connection and the server's
// real pools/networks/images -- and the private key stays server-side.
func TestConnectBuilderEnrollsAndDiscovers(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-connect-admin"}
	admin := signedInClient(t, env, "connect-admin")
	fake := newFakeIncus(t)

	status, raw := connectBuilder(t, admin, env, fake.token)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, body %s", status, raw)
	}
	if strings.Contains(string(raw), "PRIVATE KEY") {
		t.Fatal("the connect response carries private key material -- it must stay server-side")
	}
	var view connectionView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM builder_credential WHERE id = $1", view.Credential.ID)
	})
	if view.Credential.ServerName != "fake-incus" || view.Credential.ApiUrl != fake.srv.URL {
		t.Fatalf("credential = %+v", view.Credential)
	}
	if len(view.Discovery.StoragePools) != 2 || len(view.Discovery.Networks) != 2 || len(view.Discovery.Images) != 1 {
		t.Fatalf("discovery = %+v", view.Discovery)
	}

	// Re-reading an existing connection works from the stored credential.
	resp, err := admin.Get(env.httpURL + "/builder-connections/" + view.Credential.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET connection: status = %d", resp.StatusCode)
	}
	var again connectionView
	json.NewDecoder(resp.Body).Decode(&again)
	if len(again.Discovery.Images) != 1 || again.Discovery.Images[0].Aliases[0] != "win2019-core" {
		t.Fatalf("re-discovery = %+v", again.Discovery)
	}

	// A builder config can reference the enrolled credential directly, for
	// both kinds -- validation resolves it through the database.
	for _, tc := range []struct {
		name string
		body map[string]interface{}
	}{
		{"conn-test-mc", map[string]interface{}{"kind": "microcloud", "incus_credential_id": view.Credential.ID.String(), "incus_storage_pool": "default", "incus_ovn_uplink_network": "UPLINK"}},
		{"conn-test-pool", map[string]interface{}{"kind": "incus", "incus_hosts": []map[string]interface{}{{"credential_id": view.Credential.ID.String(), "storage_pool": "default", "ovn_uplink_network": "UPLINK"}}}},
	} {
		b, _ := json.Marshal(tc.body)
		resp, err := admin.Post(env.httpURL+"/builder-configs/"+tc.name, "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		msg, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		name := tc.name
		t.Cleanup(func() { env.server.Pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = $1", name) })
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create %s: status = %d: %s", tc.name, resp.StatusCode, msg)
		}
	}

	// JSON columns must come back as real JSON, not base64 strings -- a
	// []byte-typed jsonb column is what the UI was silently receiving.
	listResp, err := admin.Get(env.httpURL + "/builder-configs")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var list []map[string]json.RawMessage
	json.NewDecoder(listResp.Body).Decode(&list)
	for _, row := range list {
		var name string
		json.Unmarshal(row["name"], &name)
		if name != "conn-test-pool" {
			continue
		}
		if !strings.HasPrefix(string(row["incus_hosts"]), "[") || !strings.HasPrefix(string(row["incus_images"]), "{") {
			t.Fatalf("incus_hosts = %s, incus_images = %s -- want a JSON array and object, not base64 strings", row["incus_hosts"], row["incus_images"])
		}
		return
	}
	t.Fatal("conn-test-pool missing from the builder config list")
}

func TestConnectBuilderRejectsGarbageToken(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-connect-admin2"}
	admin := signedInClient(t, env, "connect-admin2")
	if status, raw := connectBuilder(t, admin, env, "definitely not a token"); status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", status, raw)
	}
}

func TestConnectBuilderRequiresInstanceAdmin(t *testing.T) {
	env := setupAPITest(t)
	nonAdmin := signedInClient(t, env, "not-an-admin")
	fake := newFakeIncus(t)
	if status, _ := connectBuilder(t, nonAdmin, env, fake.token); status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func connectAndGetID(t *testing.T, client *http.Client, env *apiTestEnv, f *fakeIncus) string {
	t.Helper()
	status, raw := connectBuilder(t, client, env, f.token)
	if status != http.StatusCreated {
		t.Fatalf("connect: status = %d: %s", status, raw)
	}
	var view connectionView
	json.Unmarshal(raw, &view)
	t.Cleanup(func() {
		env.server.Pool.Exec(context.Background(), "DELETE FROM builder_credential WHERE id = $1", view.Credential.ID)
	})
	return view.Credential.ID.String()
}

// TestBuilderImagesMustExistOnEveryHost: saving a builder asks each host's
// own API whether it has every image the builder offers -- by fingerprint --
// and refuses when any host is missing one, naming the image and host.
func TestBuilderImagesMustExistOnEveryHost(t *testing.T) {
	env := setupAPITest(t)
	env.server.AdminLogins = []string{"exchanged-images-admin"}
	admin := signedInClient(t, env, "images-admin")
	stocked := newFakeIncus(t)
	bare := newFakeIncus(t)
	bare.images = []map[string]interface{}{}
	stockedID := connectAndGetID(t, admin, env, stocked)
	bareID := connectAndGetID(t, admin, env, bare)

	create := func(name string, hostIDs ...string) (int, string) {
		hosts := []map[string]interface{}{}
		for _, id := range hostIDs {
			hosts = append(hosts, map[string]interface{}{"credential_id": id, "storage_pool": "default", "ovn_uplink_network": "UPLINK"})
		}
		b, _ := json.Marshal(map[string]interface{}{
			"kind": "incus", "incus_hosts": hosts,
			"incus_images": map[string]interface{}{"win2019": map[string]interface{}{"fingerprint": "abc", "alias": "win2019-core", "vm": true}},
		})
		resp, err := admin.Post(env.httpURL+"/builder-configs/"+name, "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		msg, _ := io.ReadAll(resp.Body)
		t.Cleanup(func() { env.server.Pool.Exec(context.Background(), "DELETE FROM builder_config WHERE name = $1", name) })
		return resp.StatusCode, string(msg)
	}

	if status, msg := create("images-ok", stockedID); status != http.StatusCreated {
		t.Fatalf("image present on the only host: status = %d: %s", status, msg)
	}
	status, msg := create("images-gap", stockedID, bareID)
	if status != http.StatusBadRequest || !strings.Contains(msg, "win2019") || !strings.Contains(msg, "host 2") {
		t.Fatalf("image missing on host 2: status = %d: %s -- want a 400 naming the image and the host", status, msg)
	}
}
