package agentdelivery

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/agentpki"
)

func TestPlatformFor(t *testing.T) {
	cases := map[string]Platform{
		"windows-server-2022": Windows, "win2019": Windows, "Windows": Windows,
		"ubuntu22": Linux, "debian-bookworm": Linux, "": Linux,
	}
	for os, want := range cases {
		if got := PlatformFor(os); got != want {
			t.Errorf("PlatformFor(%q) = %v, want %v", os, got, want)
		}
	}
}

// TestBuildIssuesAChainedCertAndRealUserData: the per-host cert an agent
// carries must verify against the same CA the gateway trusts (or the
// gateway rejects it), and the user-data must actually point the host at
// its download.
func TestBuildIssuesAChainedCertAndRealUserData(t *testing.T) {
	ca, err := agentpki.GenerateCA("test-gateway-ca")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	caCertPath := filepath.Join(dir, "ca.crt")
	caKeyPath := filepath.Join(dir, "ca.key")
	os.WriteFile(caCertPath, ca.CertPEM, 0o600)
	os.WriteFile(caKeyPath, ca.KeyPEM, 0o600)

	// A real base binary with the identity marker: build one on the fly is
	// heavy, so reuse the checked-in staged one when present, else skip.
	base := filepath.Join("..", "..", ".agent-base", "agent-"+string(Linux))
	if _, err := os.Stat(base); err != nil {
		t.Skip("no staged base agent binary to patch")
	}
	baseDir := t.TempDir()
	data, _ := os.ReadFile(base)
	os.WriteFile(filepath.Join(baseDir, "agent-"+string(Linux)), data, 0o755)

	cfg, err := Load("gw.example:8444", "http://api.example:8080", caCertPath, caKeyPath, baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled() {
		t.Fatal("config should be enabled with everything set")
	}

	objID := "11111111-2222-3333-4444-555555555555"
	del, err := cfg.Build(objID, "tok-abc", "ubuntu22", false)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if del.Platform != Linux || len(del.Binary) == 0 {
		t.Fatalf("delivery = %+v, want a linux binary", del)
	}
	if !strings.Contains(del.UserData, "agent-binary/"+objID+"?token=tok-abc") {
		t.Fatalf("user-data missing the download URL: %s", del.UserData)
	}
	if !strings.Contains(del.UserData, "systemd") {
		t.Fatalf("linux user-data should install a systemd service")
	}

	// The issued cert must verify against the CA, with the object id as CN.
	certPEM, keyPEM, err := cfg.issueLeaf(objID)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(certPEM)
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if leaf.Subject.CommonName != objID {
		t.Fatalf("leaf CN = %q, want %q", leaf.Subject.CommonName, objID)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(ca.CertPEM)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		t.Fatalf("issued leaf doesn't chain to the CA: %v", err)
	}
	if len(keyPEM) == 0 {
		t.Fatal("no key issued")
	}
}
