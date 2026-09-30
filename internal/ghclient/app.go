// GitHub App authentication: a JWT signed with the App's own private key
// (proves "this request comes from App N," valid a few minutes at a
// time), exchanged for a short-lived installation access token (proves
// "and it's acting for installation M," scoped only to that
// installation's repositories). Nothing here depends on a JWT library --
// GitHub's app JWT is exactly RS256 over
// base64url(header) + "." + base64url(claims), which stdlib crypto/rsa
// signs directly, so this avoids a new dependency for three well-known
// field names.
//
// This replaces the plain OAuth-App-plus-shared-
// webhook-secret model: an installation token lets the server verify a
// repository's real clone URL and metadata independently of anything an
// inbound webhook payload claims, instead of trusting the payload.
package ghclient

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"
)

// ParsePrivateKeyPEM reads a GitHub App's private key exactly as
// downloaded from its settings page (a PKCS#1 "RSA PRIVATE KEY" PEM
// block). GitHub only ever offers that format for App keys, so PKCS#8
// isn't accepted here -- a config error should say so plainly rather
// than accept a key type GitHub itself would never generate.
func ParsePrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in GitHub App private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing GitHub App private key (expected PKCS#1 RSA, as downloaded from the App's settings page): %w", err)
	}
	return key, nil
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

type appJWTClaims struct {
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
}

// GenerateAppJWT signs a short-lived (10 minute, GitHub's own maximum)
// JWT identifying appID, for the app-level endpoints (minting
// installation tokens). now is a parameter rather than time.Now()
// directly so tests can produce deterministic, reproducible tokens.
// iat is backdated 60 seconds, GitHub's own documented recommendation,
// to tolerate clock drift between this host and GitHub's servers.
func GenerateAppJWT(appID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	header := base64URLEncode([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := appJWTClaims{
		IssuedAt:  now.Add(-60 * time.Second).Unix(),
		ExpiresAt: now.Add(9 * time.Minute).Unix(),
		Issuer:    appID,
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := header + "." + base64URLEncode(claimsJSON)

	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("signing app JWT: %w", err)
	}
	return signingInput + "." + base64URLEncode(sig), nil
}

// InstallationToken is one installation access token: token itself, and
// when it stops being valid (GitHub issues these for one hour).
type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateInstallationToken exchanges appJWT (from GenerateAppJWT) for a
// token scoped to exactly installationID's repositories -- the
// authoritative credential internal/api/webhook.go's reconcile uses to
// fetch a repository's real clone URL and metadata, instead of trusting
// whatever a webhook payload claims.
func (c *Client) CreateInstallationToken(ctx context.Context, appJWT string, installationID int64) (*InstallationToken, error) {
	u := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.APIBaseURL, installationID)
	var out InstallationToken
	if err := c.doJSON(ctx, http.MethodPost, u, appJWT, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
