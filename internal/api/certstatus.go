// Agent trust-anchor expiry: the mTLS CA that signs every per-host agent
// certificate, and the gateway's own server certificate, both have a finite
// life. If the CA lapses, every agent connection fails at once -- so the UI
// needs to see the expiry coming and warn well ahead. This endpoint reads the
// cert files (the same ones the gateway/runner already use, mounted read-only)
// and reports their expiry; it returns no key material, only dates.
package api

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"time"
)

// certWarnThresholdDays is "within 3 months of expiration" -- the point at
// which the UI raises a clear banner. ~90 days.
const certWarnThresholdDays = 90

type certInfo struct {
	Configured    bool       `json:"configured"`
	Subject       string     `json:"subject,omitempty"`
	NotAfter      *time.Time `json:"not_after,omitempty"`
	DaysRemaining int        `json:"days_remaining"`
	ExpiringSoon  bool       `json:"expiring_soon"`
	Expired       bool       `json:"expired"`
	Error         string     `json:"error,omitempty"`
}

type certStatusResponse struct {
	WarnThresholdDays int      `json:"warn_threshold_days"`
	CA                certInfo `json:"ca"`
	Server            certInfo `json:"server"`
}

// handleCertStatus reports the agent CA and gateway server certificate expiry.
// Any authenticated caller may read it (session cookie or bearer token) -- it's
// operational metadata, not secret, and every operator benefits from the
// warning. The dates come straight from the cert files; no private key is read.
func (s *Server) handleCertStatus(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authSessionForRequest(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, certStatusResponse{
		WarnThresholdDays: certWarnThresholdDays,
		CA:                readCertInfo(s.CACertPath),
		Server:            readCertInfo(s.ServerCertPath),
	})
}

// readCertInfo parses the first certificate in a PEM file and reports its
// expiry. A missing path is reported as not-configured (not an error); an
// unreadable or unparseable file carries the reason in Error so the UI can say
// what's wrong instead of silently showing nothing.
func readCertInfo(path string) certInfo {
	if path == "" {
		return certInfo{Configured: false}
	}
	info := certInfo{Configured: true}
	raw, err := os.ReadFile(path)
	if err != nil {
		info.Error = "could not read certificate: " + err.Error()
		return info
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		info.Error = "certificate file is not PEM"
		return info
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		info.Error = "could not parse certificate: " + err.Error()
		return info
	}
	notAfter := cert.NotAfter
	info.Subject = cert.Subject.CommonName
	info.NotAfter = &notAfter
	// Whole days remaining, rounded down; negative once expired.
	info.DaysRemaining = int(time.Until(notAfter).Hours() / 24)
	info.Expired = time.Now().After(notAfter)
	info.ExpiringSoon = !info.Expired && info.DaysRemaining <= certWarnThresholdDays
	return info
}
