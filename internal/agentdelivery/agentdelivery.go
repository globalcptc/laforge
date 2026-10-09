// Package agentdelivery builds what a freshly-created host needs to bring
// its agent up on its own: agents aren't baked into images, they're fetched
// at first boot. For one deployed object this produces (1) a per-host agent
// binary -- a platform base binary with a per-host client certificate
// (CN = the object's id, the identity the gateway keys agents by) patched
// in -- and (2) the cloud-init user-data that downloads it by one-time token
// from the api and runs it as a service. The binary carries its own
// identity, so the download is authenticated only by the token; nothing
// secret rides in the clear beyond that host's own credentials.
//
// This is hoster-agnostic on purpose: the runner calls Build, stores the
// binary, and hands the user-data to whichever builder is deploying. A
// builder's only job is to set the instance's cloud-init user-data.
package agentdelivery

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/agentfactory"
)

// Platform is one target the base binaries are built for.
type Platform string

const (
	Linux   Platform = "x86_64-unknown-linux-musl"
	Windows Platform = "x86_64-pc-windows-gnu"
)

// PlatformFor maps a content OS name to a target. The rule is deliberately
// coarse -- anything that looks like Windows is Windows, everything else is
// x86_64 Linux -- because that's the real split the agent is built for, and
// a builder's images are all x86_64 here.
func PlatformFor(os string) Platform {
	if strings.Contains(strings.ToLower(os), "win") {
		return Windows
	}
	return Linux
}

// Config is what the runner needs configured before it can deliver agents
// at all. When any of it is unset, the runner skips delivery and a build
// still deploys (just without agents) -- the same degraded behavior every
// other optional capability here has.
type Config struct {
	// GatewayAddr is the host:port an agent dials, reachable from the
	// deployed hosts (not the gateway's own listen address).
	GatewayAddr string
	// APIBaseURL is where a booting host downloads its binary, reachable
	// from the deployed hosts (e.g. http://100.x.y.z:8080).
	APIBaseURL string
	// BaseDir holds the platform base binaries, named agent-<platform>
	// (and agent-<platform>.exe for Windows).
	BaseDir string

	caCert *x509.Certificate
	caKey  crypto.Signer
	caPEM  []byte
}

// Enabled reports whether delivery is configured.
func (c *Config) Enabled() bool { return c != nil && c.caCert != nil }

// Load resolves a Config from the gateway CA PEM pair (the SAME CA the
// gateway trusts, so agent certs verify) and the base-binary directory.
// Returns a disabled Config, not an error, when nothing is set -- delivery
// is optional.
func Load(gatewayAddr, apiBaseURL, caCertPath, caKeyPath, baseDir string) (*Config, error) {
	if gatewayAddr == "" || apiBaseURL == "" || caCertPath == "" || caKeyPath == "" || baseDir == "" {
		return &Config{}, nil
	}
	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("reading gateway CA cert: %w", err)
	}
	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading gateway CA key: %w", err)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("gateway CA cert is not PEM")
	}
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing gateway CA cert: %w", err)
	}
	caKey, err := parsePrivateKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parsing gateway CA key: %w", err)
	}
	return &Config{
		GatewayAddr: gatewayAddr, APIBaseURL: strings.TrimRight(apiBaseURL, "/"), BaseDir: baseDir,
		caCert: caCert, caKey: caKey, caPEM: certPEM,
	}, nil
}

// parsePrivateKey accepts whatever key type the CA actually uses (the dev
// CA is RSA in PKCS#8; a generated one is EC) -- a signer, not a concrete
// type, is all leaf issuance needs.
func parsePrivateKey(keyPEM []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if signer, ok := k.(crypto.Signer); ok {
			return signer, nil
		}
		return nil, fmt.Errorf("PKCS#8 key is not a signer")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf("unrecognized private key format")
}

// Delivery is one host's ready-to-store result.
type Delivery struct {
	Platform Platform
	Binary   []byte
	UserData string
	// DownloadURL is the one-time-token URL a deployed object fetches its patched
	// binary from ({APIBaseURL}/agent-binary/{id}?token=…). UserData embeds it for
	// a VM; a container platform that can't take a pushed binary (Fargate, Zun)
	// uses it directly to pull the agent at start.
	DownloadURL string
}

// Build produces the per-host binary and its cloud-init user-data.
// objectID is the deployed object's id (the agent cert's CommonName and
// the download path); token is the one-time capability the user-data uses
// to fetch the binary; os is the content OS name, only used to pick a
// platform; debug is the environment's agent-debug flag, baked into the binary
// so it can't be flipped on a captured box (off = the agent is silent locally).
func (c *Config) Build(objectID, token, os string, debug bool) (Delivery, error) {
	platform := PlatformFor(os)
	base, err := c.baseBinary(platform)
	if err != nil {
		return Delivery{}, err
	}
	// Seal the self-hash before patching identity, so the delivered binary
	// carries a valid self-hash even if the base wasn't produced by the
	// factory's `build` seal pass. A no-op when the base is already sealed
	// (or has no self-hash statics), and it must run BEFORE PatchBinary so
	// the per-host identity region is masked out of the sealed hash.
	base, err = agentfactory.SelfHashSeal(base)
	if err != nil {
		return Delivery{}, fmt.Errorf("sealing agent self-hash: %w", err)
	}
	certPEM, keyPEM, err := c.issueLeaf(objectID)
	if err != nil {
		return Delivery{}, fmt.Errorf("issuing agent certificate: %w", err)
	}
	patched, err := agentfactory.PatchBinary(base, c.GatewayAddr, c.caPEM, certPEM, keyPEM, debug)
	if err != nil {
		return Delivery{}, fmt.Errorf("patching agent identity: %w", err)
	}
	url := fmt.Sprintf("%s/agent-binary/%s?token=%s", c.APIBaseURL, objectID, token)
	userData := linuxUserData(url)
	if platform == Windows {
		userData = windowsUserData(url)
	}
	return Delivery{Platform: platform, Binary: patched, UserData: userData, DownloadURL: url}, nil
}

func (c *Config) baseBinary(p Platform) ([]byte, error) {
	name := "agent-" + string(p)
	if p == Windows {
		name += ".exe"
	}
	path := filepath.Join(c.BaseDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading base agent binary %s: %w", path, err)
	}
	return data, nil
}

func (c *Config) issueLeaf(commonName string) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.caCert, &key.PublicKey, c.caKey)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}
