package microcloud

import (
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
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
)

// TrustToken is a certificate-add token, decoded -- the same shape from
// Incus (`incus config trust add <name>`) and from LXD, which a MicroCloud
// runs (`lxc config trust add --name <name>`): base64 JSON carrying the server's
// certificate fingerprint (hex SHA-256 of the DER), every address the
// server listens on, and a one-time secret. The fingerprint is what makes
// enrollment verified rather than trust-on-first-use -- a server
// presenting any other certificate is refused.
type TrustToken struct {
	ClientName  string    `json:"client_name"`
	Fingerprint string    `json:"fingerprint"`
	Addresses   []string  `json:"addresses"`
	Secret      string    `json:"secret"`
	ExpiresAt   time.Time `json:"expires_at"`
	// Type is LXD-only (API extension access_management_tls): set when the
	// token came from `lxc auth identity create tls/<name>` rather than
	// `lxc config trust add`, and then it has to be redeemed through the
	// identities API instead of /1.0/certificates.
	Type string `json:"type,omitempty"`
}

const notATokenMessage = "that doesn't look like a trust token -- copy the whole token printed by `lxc config trust add --name laforge` (MicroCloud or LXD) or `incus config trust add laforge` (Incus)"

// ParseTrustToken decodes a pasted token, tolerating surrounding
// whitespace and the line wrapping a terminal copy often adds.
func ParseTrustToken(raw string) (TrustToken, error) {
	cleaned := strings.Join(strings.Fields(raw), "")
	if cleaned == "" {
		return TrustToken{}, errors.New("the trust token is empty")
	}
	data, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		if data, err = base64.RawURLEncoding.DecodeString(cleaned); err != nil {
			return TrustToken{}, errors.New(notATokenMessage)
		}
	}
	var t TrustToken
	if err := json.Unmarshal(data, &t); err != nil {
		return TrustToken{}, errors.New(notATokenMessage)
	}
	if t.Fingerprint == "" || t.Secret == "" || len(t.Addresses) == 0 {
		return TrustToken{}, errors.New("the trust token is missing its fingerprint, secret, or addresses")
	}
	if !t.ExpiresAt.IsZero() && t.ExpiresAt.Before(time.Now()) {
		return TrustToken{}, fmt.Errorf("the trust token expired at %s -- generate a new one", t.ExpiresAt.Format(time.RFC3339))
	}
	return t, nil
}

// Enrollment is what a successful Enroll produces: everything needed to
// talk to this server from now on, none of it typed by a person.
type Enrollment struct {
	APIURL            string
	ServerName        string
	ServerFingerprint string
	ServerCertPEM     []byte
	ClientCertPEM     []byte
	ClientKeyPEM      []byte
}

// ErrFingerprintMismatch means the server answering at a token's address
// presented a certificate other than the one the token vouches for.
var ErrFingerprintMismatch = errors.New("the server presented a certificate that doesn't match the trust token's fingerprint -- refusing to connect")

// Enroll connects to the server a trust token describes and makes LaForge a
// trusted client of it:
//
//  1. Probe every address in the token (or just addressOverride, when the
//     server is reachable some other way, e.g. through NAT) in parallel,
//     keeping the first address -- in the token's own order -- whose
//     certificate matches the token's fingerprint.
//  2. Generate a fresh client keypair for LaForge.
//  3. Redeem the token, presenting that client certificate, the way the
//     servers' own clients do (`lxc remote add` / `incus remote add`): a
//     token with a type goes to LXD's identities API; otherwise POST
//     /1.0/certificates, carrying it as trust_token when the server has
//     the explicit_trust_token extension and as password when it doesn't
//     (older LXD).
//  4. Confirm GET /1.0 now reports auth "trusted".
func Enroll(ctx context.Context, rawToken, clientName, addressOverride string) (Enrollment, error) {
	token, err := ParseTrustToken(rawToken)
	if err != nil {
		return Enrollment{}, err
	}
	addresses := token.Addresses
	if addressOverride != "" {
		addresses = []string{normalizeAddress(addressOverride)}
	}

	addr, serverCertPEM, err := firstMatchingAddress(ctx, addresses, token.Fingerprint)
	if err != nil {
		return Enrollment{}, err
	}

	if clientName == "" {
		clientName = token.ClientName
	}
	clientCertPEM, clientKeyPEM, err := generateClientCredential(clientName)
	if err != nil {
		return Enrollment{}, fmt.Errorf("generating LaForge's client certificate: %w", err)
	}

	apiURL := "https://" + addr
	client, err := NewClient(apiURL, clientCertPEM, clientKeyPEM, serverCertPEM, "")
	if err != nil {
		return Enrollment{}, err
	}
	if err := redeemToken(ctx, client, token, strings.Join(strings.Fields(rawToken), "")); err != nil {
		return Enrollment{}, fmt.Errorf("the server rejected the trust token (it may already have been used, or been revoked): %w", err)
	}

	raw, err := client.get(ctx, "/1.0")
	if err != nil {
		return Enrollment{}, fmt.Errorf("confirming trust: %w", err)
	}
	var info struct {
		Auth        string `json:"auth"`
		Environment struct {
			ServerName string `json:"server_name"`
		} `json:"environment"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return Enrollment{}, fmt.Errorf("decoding server info: %w", err)
	}
	if info.Auth != "trusted" {
		return Enrollment{}, fmt.Errorf("the server accepted the token but still reports LaForge as %q", info.Auth)
	}

	return Enrollment{
		APIURL:            apiURL,
		ServerName:        info.Environment.ServerName,
		ServerFingerprint: token.Fingerprint,
		ServerCertPEM:     serverCertPEM,
		ClientCertPEM:     clientCertPEM,
		ClientKeyPEM:      clientKeyPEM,
	}, nil
}

func redeemToken(ctx context.Context, client *Client, token TrustToken, raw string) error {
	if token.Type != "" {
		_, err := client.post(ctx, "/1.0/auth/identities/tls", map[string]string{"trust_token": raw})
		return err
	}
	// Send the token in the explicit "trust_token" field. We can't sniff the
	// server's api_extensions to decide this: GET /1.0 from a not-yet-trusted
	// client omits api_extensions entirely (it returns only {auth: untrusted}),
	// so the old "look for explicit_trust_token" check always missed and fell
	// back to the legacy "password" field -- which modern LXD/Incus reject with
	// 403. A structured trust token (the base64-JSON shape ParseTrustToken
	// accepts) only comes from a server that supports trust_token anyway; the
	// pre-token "password" trust flow never produced these. Fall back to
	// "password" only if trust_token is refused, for a genuinely old LXD.
	if _, err := client.post(ctx, "/1.0/certificates", map[string]string{"type": "client", "trust_token": raw}); err != nil {
		if _, pwErr := client.post(ctx, "/1.0/certificates", map[string]string{"type": "client", "password": raw}); pwErr != nil {
			return err
		}
	}
	return nil
}

func normalizeAddress(a string) string {
	a = strings.TrimSpace(a)
	a = strings.TrimPrefix(strings.TrimPrefix(a, "https://"), "http://")
	a = strings.TrimSuffix(a, "/")
	if _, _, err := net.SplitHostPort(a); err != nil {
		a = net.JoinHostPort(strings.Trim(a, "[]"), "8443")
	}
	return a
}

// probeTimeout bounds each address probe: a token lists every interface
// the server has, and most of them (docker bridges, link-local networks)
// are typically unreachable from wherever LaForge runs.
const probeTimeout = 4 * time.Second

// AddressCheck is what probing one of a token's addresses found: whether
// LaForge could open a TLS connection to it, and whether the certificate there
// is the one the token names.
type AddressCheck struct {
	Address            string `json:"address"`
	Reachable          bool   `json:"reachable"`
	FingerprintMatches bool   `json:"fingerprint_matches"`
	Error              string `json:"error,omitempty"`
	Millis             int64  `json:"millis"`
	certPEM            []byte
}

// TokenAddresses is the list Enroll tries, in order: the token's own
// addresses, or only addressOverride when one is given.
func TokenAddresses(token TrustToken, addressOverride string) []string {
	if strings.TrimSpace(addressOverride) != "" {
		return []string{normalizeAddress(addressOverride)}
	}
	return token.Addresses
}

// CheckAddresses probes every address in parallel exactly as Enroll does, but
// redeems nothing -- the token stays unused. Results are in the input order.
func CheckAddresses(ctx context.Context, addresses []string, fingerprint string) []AddressCheck {
	want := strings.ToLower(strings.ReplaceAll(fingerprint, ":", ""))
	results := make([]AddressCheck, len(addresses))
	var wg sync.WaitGroup
	for i, addr := range addresses {
		wg.Add(1)
		go func(i int, addr string) {
			defer wg.Done()
			start := time.Now()
			certPEM, err := fetchServerCertificate(ctx, addr, probeTimeout)
			r := AddressCheck{Address: addr, Millis: time.Since(start).Milliseconds()}
			if err != nil {
				r.Error = err.Error()
			} else {
				r.Reachable, r.certPEM = true, certPEM
				r.FingerprintMatches = certFingerprintHex(certPEM) == want
				if !r.FingerprintMatches {
					r.Error = "reachable, but its certificate isn't the one the token names"
				}
			}
			results[i] = r
		}(i, addr)
	}
	wg.Wait()
	return results
}

func firstMatchingAddress(ctx context.Context, addresses []string, fingerprint string) (string, []byte, error) {
	results := CheckAddresses(ctx, addresses, fingerprint)
	anyReached := false
	var reasons []string
	for _, r := range results {
		if r.FingerprintMatches {
			return r.Address, r.certPEM, nil
		}
		if r.Reachable {
			anyReached = true
		}
		reasons = append(reasons, r.Address+": "+r.Error)
	}
	if anyReached {
		return "", nil, ErrFingerprintMismatch
	}
	return "", nil, fmt.Errorf("couldn't reach the server at any of the token's addresses (%s) -- if it's reachable some other way, enter that address instead", strings.Join(reasons, "; "))
}

// fetchServerCertificate is FetchServerCertificateInsecure with a real
// per-call deadline -- the caller verifies the result against a
// fingerprint, so skipping chain verification here is safe.
func fetchServerCertificate(ctx context.Context, addr string, timeout time.Duration) ([]byte, error) {
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: timeout},
		Config:    &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // verified against the token's fingerprint by the caller
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("%s presented no certificate", addr)
	}
	return pemEncodeCert(certs[0].Raw), nil
}

// certFingerprintHex is the fingerprint form Incus itself uses in trust
// tokens and `incus config trust list`: lowercase hex SHA-256 of the DER.
func certFingerprintHex(certPEM []byte) string {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return ""
	}
	sum := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(sum[:])
}

// generateClientCredential makes the self-signed client certificate
// LaForge presents to one Incus server. Incus trusts clients by exact
// certificate, not by chain, so self-signed is the normal shape here --
// the same thing `incus remote add` generates for the CLI.
func generateClientCredential(name string) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: name, Organization: []string{"LaForge"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pemEncodeCert(der), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), nil
}
