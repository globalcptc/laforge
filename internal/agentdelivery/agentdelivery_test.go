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

// TestDeliveredUserData locks the read-side selector to the same choices the
// builders make: Linux -> cloud-init, Windows -> cloudbase-init everywhere
// except AWS (EC2Launch). It must not drift from each builder's own
// agentUserData, since it's what the UI shows as "what this host received".
func TestDeliveredUserData(t *testing.T) {
	const url = "https://example/agent"
	cases := []struct {
		os, kind   string
		wantFormat DeliveryFormat
		wantMarker string // a substring unique to the chosen wrapper
	}{
		{"ubuntu22", "incus", FormatCloudInit, "#cloud-config"},
		{"ubuntu22", "aws", FormatCloudInit, "#cloud-config"},
		{"win2019", "incus", FormatCloudbase, "#ps1_sysnative"},
		{"windows-server-2022", "microcloud", FormatCloudbase, "#ps1_sysnative"},
		{"win2019", "openstack", FormatCloudbase, "#ps1_sysnative"},
		{"win2019", "aws", FormatEC2Launch, "<powershell>"},
		{"win2019", "", FormatCloudbase, "#ps1_sysnative"}, // unknown kind -> cloudbase default
	}
	for _, c := range cases {
		data, format := DeliveredUserData(c.os, c.kind, url, "")
		if format != c.wantFormat {
			t.Errorf("DeliveredUserData(%q,%q) format = %q, want %q", c.os, c.kind, format, c.wantFormat)
		}
		if !strings.Contains(data, c.wantMarker) {
			t.Errorf("DeliveredUserData(%q,%q) missing marker %q", c.os, c.kind, c.wantMarker)
		}
		if !strings.Contains(data, url) {
			t.Errorf("DeliveredUserData(%q,%q) did not embed the download URL", c.os, c.kind)
		}
	}
}

func TestTempAdminPassword(t *testing.T) {
	// Deterministic for the same (secret, id); different per host; meets Windows
	// complexity (upper, lower, digit, special) and a sane length.
	a := TempAdminPassword("build-secret", "obj-1")
	if a != TempAdminPassword("build-secret", "obj-1") {
		t.Error("not deterministic for the same inputs")
	}
	if a == TempAdminPassword("build-secret", "obj-2") {
		t.Error("two hosts got the same temp password")
	}
	if a == TempAdminPassword("other-secret", "obj-1") {
		t.Error("a different build secret should give a different password")
	}
	hasUpper := strings.ContainsAny(a, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasLower := strings.ContainsAny(a, "abcdefghijklmnopqrstuvwxyz")
	hasDigit := strings.ContainsAny(a, "0123456789")
	hasSpecial := strings.ContainsAny(a, "!@#$%^&*")
	if !(hasUpper && hasLower && hasDigit && hasSpecial) {
		t.Errorf("temp password %q fails Windows complexity (need upper/lower/digit/special)", a)
	}
	if len(a) < 12 {
		t.Errorf("temp password %q is too short", a)
	}
	// It must appear verbatim in the rendered user-data (so the box actually sets
	// it) for both OS forms.
	if !strings.Contains(LinuxUserData("https://x/agent", a), a) {
		t.Error("Linux user-data did not carry the temp password")
	}
	if !strings.Contains(WindowsCloudbaseUserData("https://x/agent", a), a) {
		t.Error("Windows user-data did not carry the temp password")
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
	if !strings.Contains(del.DownloadURL, "agent-binary/"+objID+"?token=tok-abc") {
		t.Fatalf("delivery missing the download URL: %s", del.DownloadURL)
	}
	// The builder renders the format; the renderers must embed the URL and the
	// right install mechanism per platform.
	if ud := LinuxUserData(del.DownloadURL, ""); !strings.Contains(ud, del.DownloadURL) || !strings.Contains(ud, "systemd") {
		t.Fatalf("linux user-data should install a systemd service from the URL: %s", ud)
	}
	if ud := WindowsCloudbaseUserData(del.DownloadURL, ""); !strings.HasPrefix(ud, "#ps1_sysnative") || !strings.Contains(ud, "sc.exe create LaForgeAgent") {
		t.Fatalf("windows cloudbase user-data malformed: %s", ud)
	}
	if ud := WindowsEC2LaunchUserData(del.DownloadURL, ""); !strings.HasPrefix(ud, "<powershell>") || !strings.Contains(ud, "sc.exe create LaForgeAgent") {
		t.Fatalf("windows EC2Launch user-data malformed: %s", ud)
	}
	// The agent must be installed as a real service (SCM owns boot-start and
	// crash-restart): auto-start, failure/restart actions, and started now -- with
	// a hardened scheduled task as a mutually-exclusive fallback if it doesn't run.
	ud := WindowsEC2LaunchUserData(del.DownloadURL, "")
	for _, want := range []string{
		"sc.exe create LaForgeAgent", "start= auto", "obj= LocalSystem",
		"sc.exe failure LaForgeAgent", "actions= restart/5000", "sc.exe start LaForgeAgent",
		// Fallback path, gated on the service not running.
		"-ne 'Running'", "schtasks /create /tn LaForgeAgent /xml", "<BootTrigger>", "<ExecutionTimeLimit>PT0S</ExecutionTimeLimit>",
	} {
		if !strings.Contains(ud, want) {
			t.Errorf("windows install missing %q", want)
		}
	}
	// A future-skewed clock must not gate the fallback task either.
	if strings.Contains(ud, "<StartBoundary>") {
		t.Errorf("fallback boot trigger must have no StartBoundary")
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
