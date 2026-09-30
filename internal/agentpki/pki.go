// Package agentpki is the small certificate authority behind the agent
// protocol's mTLS: "the agent opens a TLS connection to the gateway... and
// presents its client certificate; the gateway presents a certificate the
// agent pinned at build time." One CA issues both the gateway's server
// certificate and every agent's per-host client certificate; the gateway
// trusts exactly that CA and nothing else, and identifies which host is
// connecting from the client certificate's CommonName -- see
// internal/gateway.
//
// This is deliberately not a hand-rolled protocol: everything here is
// crypto/x509 and crypto/tls, unmodified. What this package owns is only
// "mint a CA, mint a leaf cert signed by it" -- ECDSA P-256 throughout,
// matching the agent's own small-binary goals (a P-256 key and signature
// are a fraction of the size of an RSA equivalent, which matters once a
// certificate is embedded in a binary the agent factory patches per
// host).
package agentpki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// CA is a self-signed certificate authority able to issue leaf
// certificates. Nothing about it is persisted by this package -- the
// caller decides what to do with CertPEM/KeyPEM (embed it in a binary,
// write it to a file for a dev server, hold it only in memory for a
// test).
type CA struct {
	cert    *x509.Certificate
	key     *ecdsa.PrivateKey
	CertPEM []byte
	KeyPEM  []byte
}

// GenerateCA creates a new, self-signed CA valid for 10 years -- long
// enough that this code never has to think about CA rotation, short
// enough that "valid forever" was a deliberate choice to avoid, not an
// oversight.
func GenerateCA(commonName string) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return &CA{
		cert:    cert,
		key:     key,
		CertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		KeyPEM:  pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	}, nil
}

// IssueLeaf mints a certificate signed by this CA. serverAuth picks
// ServerAuth (the gateway's own cert, which needs dnsNames/ipAddrs to
// satisfy hostname verification) or ClientAuth (an agent's per-host
// identity, where commonName is the thing that actually matters -- see
// internal/gateway's cert-to-host lookup, which trusts CommonName only
// because the CA signature already proved it wasn't self-asserted).
func (ca *CA) IssueLeaf(commonName string, serverAuth bool, dnsNames []string, ipAddrs []net.IP, validFor time.Duration) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating leaf key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	eku := x509.ExtKeyUsageClientAuth
	if serverAuth {
		eku = x509.ExtKeyUsageServerAuth
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{eku},
		DNSNames:     dnsNames,
		IPAddresses:  ipAddrs,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating leaf certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}

// ServerTLSConfig builds the gateway's listener config: presents
// serverCertPEM/serverKeyPEM, and requires + verifies a client
// certificate against caCertPEM -- real mTLS, both directions checked by
// crypto/tls itself, not by application code after the handshake.
func ServerTLSConfig(caCertPEM, serverCertPEM, serverKeyPEM []byte) (*tls.Config, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("no certificates found in CA PEM")
	}
	cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("loading server keypair: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ClientTLSConfig builds a connecting client's config: presents
// clientCertPEM/clientKeyPEM, and pins the gateway's certificate by
// trusting only caCertPEM as a root -- "a pinned gateway public key/cert
// ... baked into the binary at build time." Used by internal/gateway's
// own tests (a Go client standing in for the agent) and mirrored
// independently in the real Rust agent via rustls.
func ClientTLSConfig(caCertPEM, clientCertPEM, clientKeyPEM []byte, serverName string) (*tls.Config, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("no certificates found in CA PEM")
	}
	cert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("loading client keypair: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS13,
	}, nil
}
