package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTempCert writes a self-signed cert PEM expiring at notAfter and returns
// its path -- enough for readCertInfo, which only reads NotAfter/CN.
func writeTempCert(t *testing.T, cn string, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadCertInfo(t *testing.T) {
	// Unconfigured path: reported as not-configured, no error.
	if ci := readCertInfo(""); ci.Configured {
		t.Errorf("empty path should be unconfigured, got %+v", ci)
	}

	// Within the 3-month window -> expiring soon, not expired.
	soon := readCertInfo(writeTempCert(t, "laforge-ca", time.Now().Add(30*24*time.Hour)))
	if !soon.Configured || soon.Expired || !soon.ExpiringSoon {
		t.Errorf("30-day cert should be expiring-soon and not expired, got %+v", soon)
	}
	if soon.DaysRemaining < 28 || soon.DaysRemaining > 31 {
		t.Errorf("30-day cert days_remaining = %d, want ~30", soon.DaysRemaining)
	}
	if soon.Subject != "laforge-ca" {
		t.Errorf("subject = %q, want laforge-ca", soon.Subject)
	}

	// Already expired.
	expired := readCertInfo(writeTempCert(t, "laforge-ca", time.Now().Add(-time.Hour)))
	if !expired.Expired || expired.ExpiringSoon {
		t.Errorf("past cert should be expired and not expiring-soon, got %+v", expired)
	}

	// Comfortably in the future -> neither.
	ok := readCertInfo(writeTempCert(t, "laforge-ca", time.Now().Add(400*24*time.Hour)))
	if ok.Expired || ok.ExpiringSoon {
		t.Errorf("400-day cert should be neither expired nor expiring-soon, got %+v", ok)
	}

	// A non-PEM file surfaces a clear error rather than a silent zero value.
	bad := filepath.Join(t.TempDir(), "notcert.pem")
	os.WriteFile(bad, []byte("not a cert"), 0o600)
	if ci := readCertInfo(bad); ci.Error == "" {
		t.Errorf("non-PEM file should report an error, got %+v", ci)
	}
}
