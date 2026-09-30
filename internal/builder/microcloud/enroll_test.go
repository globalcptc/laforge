package microcloud

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeEnrollServer behaves like the parts of a real Incus or LXD daemon
// enrollment touches: it asks for (but doesn't require) a client
// certificate, reports auth "trusted" on GET /1.0 only for certificates it
// has enrolled, and enrolls the presented certificate only when the token
// arrives the way that kind of server expects it: POST /1.0/certificates
// as trust_token (explicit_trust_token servers) or as password (older
// LXD), or POST /1.0/auth/identities/tls for an LXD identity token.
type fakeEnrollServer struct {
	srv     *httptest.Server
	token   string
	mu      sync.Mutex
	trusted map[string]bool
	// legacyLXD: no explicit_trust_token extension, token expected as password.
	legacyLXD bool
}

func newFakeEnrollServer(t *testing.T) *fakeEnrollServer {
	return newFakeEnrollServerWithToken(t, "")
}

// newFakeEnrollServerWithToken issues a token of the given LXD type ("" for
// a plain certificate-add token).
func newFakeEnrollServerWithToken(t *testing.T, tokenType string) *fakeEnrollServer {
	t.Helper()
	f := &fakeEnrollServer{trusted: map[string]bool{}}
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(f.handle))
	// Every httptest TLS server shares Go's one built-in test certificate,
	// which would make two fake servers indistinguishable by fingerprint --
	// give each its own, like two real Incus daemons.
	certPEM, keyPEM, err := generateClientCredential("fake-incus")
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	f.srv.TLS = &tls.Config{ClientAuth: tls.RequestClientCert, Certificates: []tls.Certificate{cert}}
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)

	sum := sha256.Sum256(f.srv.Certificate().Raw)
	payload, _ := json.Marshal(TrustToken{
		ClientName:  "laforge",
		Fingerprint: hex.EncodeToString(sum[:]),
		// An unreachable address first, the way a real token lists every
		// interface (docker bridges and all) before the reachable one.
		Addresses: []string{"127.0.0.1:1", strings.TrimPrefix(f.srv.URL, "https://")},
		Secret:    "s3cret",
		Type:      tokenType,
	})
	f.token = base64.StdEncoding.EncodeToString(payload)
	return f
}

func clientFingerprint(r *http.Request) string {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return ""
	}
	sum := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	return hex.EncodeToString(sum[:])
}

func (f *fakeEnrollServer) handle(w http.ResponseWriter, r *http.Request) {
	reply := func(metadata interface{}) {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status": "Success", "status_code": 200, "metadata": metadata})
	}
	fp := clientFingerprint(r)
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/1.0":
		f.mu.Lock()
		auth := "untrusted"
		if f.trusted[fp] {
			auth = "trusted"
		}
		f.mu.Unlock()
		extensions := []string{"certificate_token", "explicit_trust_token"}
		if f.legacyLXD {
			extensions = []string{"certificate_token"}
		}
		reply(map[string]interface{}{"auth": auth, "api_extensions": extensions, "environment": map[string]string{"server_name": "fake-incus"}})
	case r.Method == http.MethodPost && (r.URL.Path == "/1.0/certificates" || r.URL.Path == "/1.0/auth/identities/tls"):
		var body struct {
			TrustToken string `json:"trust_token"`
			Password   string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		presented := body.TrustToken
		if f.legacyLXD {
			presented = body.Password
		}
		var tok TrustToken
		raw, _ := base64.StdEncoding.DecodeString(f.token)
		json.Unmarshal(raw, &tok)
		wantPath := "/1.0/certificates"
		if tok.Type != "" {
			wantPath = "/1.0/auth/identities/tls"
		}
		if r.URL.Path != wantPath || presented != f.token || fp == "" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 403, "error": "not authorized"})
			return
		}
		f.mu.Lock()
		f.trusted[fp] = true
		f.mu.Unlock()
		reply(map[string]interface{}{})
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "not found"})
	}
}

func TestParseTrustToken(t *testing.T) {
	f := newFakeEnrollServer(t)
	// A terminal copy often wraps the token; that must still parse.
	wrapped := f.token[:20] + "\n  " + f.token[20:] + "\n"
	tok, err := ParseTrustToken(wrapped)
	if err != nil {
		t.Fatalf("ParseTrustToken: %v", err)
	}
	if tok.Secret != "s3cret" || len(tok.Addresses) != 2 {
		t.Fatalf("token = %+v", tok)
	}

	if _, err := ParseTrustToken("not a token"); err == nil {
		t.Fatal("expected an error for garbage input")
	}

	expired, _ := json.Marshal(TrustToken{Fingerprint: "ab", Secret: "x", Addresses: []string{"a:1"}, ExpiresAt: time.Now().Add(-time.Hour)})
	if _, err := ParseTrustToken(base64.StdEncoding.EncodeToString(expired)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected an expiry error, got %v", err)
	}
}

func TestEnrollBecomesTrustedClient(t *testing.T) {
	f := newFakeEnrollServer(t)
	enr, err := Enroll(context.Background(), f.token, "laforge-test", "")
	if err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	if enr.APIURL != f.srv.URL {
		t.Fatalf("APIURL = %q, want %q (the reachable address, skipping the unreachable one listed first)", enr.APIURL, f.srv.URL)
	}
	if enr.ServerName != "fake-incus" {
		t.Fatalf("ServerName = %q", enr.ServerName)
	}

	// The returned credential really is trusted: a fresh client built from
	// it (with the pinned server cert) is reported as trusted.
	client, err := NewClient(enr.APIURL, enr.ClientCertPEM, enr.ClientKeyPEM, enr.ServerCertPEM, "")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	raw, err := client.get(context.Background(), "/1.0")
	if err != nil {
		t.Fatalf("GET /1.0: %v", err)
	}
	var info struct {
		Auth string `json:"auth"`
	}
	json.Unmarshal(raw, &info)
	if info.Auth != "trusted" {
		t.Fatalf("auth = %q, want trusted", info.Auth)
	}
}

func TestEnrollRefusesAServerWithTheWrongCertificate(t *testing.T) {
	f := newFakeEnrollServer(t)
	other := newFakeEnrollServer(t)
	// f's token, but pointed at a different server: the fingerprint check
	// must refuse it rather than trust whatever answers.
	_, err := Enroll(context.Background(), f.token, "laforge-test", strings.TrimPrefix(other.srv.URL, "https://"))
	if !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("err = %v, want ErrFingerprintMismatch", err)
	}
}

func TestEnrollReportsUnreachableServer(t *testing.T) {
	f := newFakeEnrollServer(t)
	_, err := Enroll(context.Background(), f.token, "laforge-test", "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "couldn't reach") {
		t.Fatalf("err = %v, want an unreachable-server error", err)
	}
}

func TestNormalizeAddress(t *testing.T) {
	for in, want := range map[string]string{
		"10.0.0.5":                 "10.0.0.5:8443",
		"https://10.0.0.5:8443/":   "10.0.0.5:8443",
		"incus.example.org:9443":   "incus.example.org:9443",
		"fd7a:115c:a1e0::1":        "[fd7a:115c:a1e0::1]:8443",
		"[fd7a:115c:a1e0::1]:8443": "[fd7a:115c:a1e0::1]:8443",
	} {
		if got := normalizeAddress(in); got != want {
			t.Errorf("normalizeAddress(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestEnrollWithLXDTokens: a MicroCloud runs LXD, whose tokens are redeemed
// the way `lxc remote add` does it -- an identity token (type set) through
// the identities API, and on an LXD without explicit_trust_token, the token
// sent as the password.
func TestEnrollWithLXDTokens(t *testing.T) {
	identity := newFakeEnrollServerWithToken(t, "Client certificate (pending)")
	if _, err := Enroll(context.Background(), identity.token, "laforge-test", ""); err != nil {
		t.Fatalf("identity token: %v", err)
	}

	legacy := newFakeEnrollServer(t)
	legacy.legacyLXD = true
	if _, err := Enroll(context.Background(), legacy.token, "laforge-test", ""); err != nil {
		t.Fatalf("LXD without explicit_trust_token: %v", err)
	}
}
