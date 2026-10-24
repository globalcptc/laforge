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

// DeleteInstallation uninstalls the App from installationID's account -- the
// same as an owner clicking Uninstall on GitHub. GitHub then sends the
// `installation` "deleted" webhook. A 404 (*APIError) means it is already gone.
func (c *Client) DeleteInstallation(ctx context.Context, appJWT string, installationID int64) error {
	u := fmt.Sprintf("%s/app/installations/%d", c.APIBaseURL, installationID)
	return c.doJSON(ctx, http.MethodDelete, u, appJWT, nil, nil)
}

// CreateInstallationToken exchanges appJWT (from GenerateAppJWT) for a
// token scoped to exactly installationID's repositories -- the
// authoritative credential internal/api/webhook.go's reconcile uses to
// fetch a repository's real clone URL and metadata, instead of trusting
// whatever a webhook payload claims.
//
// This mints a fresh token on every call. Prefer InstallationToken,
// which caches: minting is rate-limited by GitHub (a secondary limit on
// POST /app/installations/{id}/access_tokens) and the hot paths that
// need a token -- the orchestrator's few-second reconcile and
// scheduled-task dispatch loops -- would otherwise trip it within
// seconds once more than a couple of builds exist.
func (c *Client) CreateInstallationToken(ctx context.Context, appJWT string, installationID int64) (*InstallationToken, error) {
	u := fmt.Sprintf("%s/app/installations/%d/access_tokens", c.APIBaseURL, installationID)
	var out InstallationToken
	if err := c.doJSON(ctx, http.MethodPost, u, appJWT, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Installation is one GitHub App installation as GET /app/installations
// reports it: which account installed the App and the current numeric id.
type Installation struct {
	ID      int64 `json:"id"`
	Account struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"account"`
	RepositorySelection string `json:"repository_selection"`
}

// ListInstallations lists every installation of this App, authenticated
// with an app JWT. The authoritative way to rediscover installations
// after a missed `installation` webhook -- GitHub sends the one-time
// `created` event once and never resends it, so an endpoint that wasn't
// reachable at install time would otherwise never learn the installation
// exists. Used by the admin resync. Requests up to 100 in a single page
// (far more than any real deployment has); pagination beyond that is a
// deliberate non-goal.
func (c *Client) ListInstallations(ctx context.Context, appJWT string) ([]Installation, error) {
	u := fmt.Sprintf("%s/app/installations?per_page=100", c.APIBaseURL)
	var out []Installation
	if err := c.doJSON(ctx, http.MethodGet, u, appJWT, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// cachedInstallationToken is one installation's memoized access token;
// see InstallationToken.
type cachedInstallationToken struct {
	token     string
	expiresAt time.Time
}

// installationTokenExpiryMargin treats a token as spent this long before
// its real expiry, so one handed out here never lapses mid-request from
// clock drift or the gap between minting it and GitHub seeing it used.
const installationTokenExpiryMargin = 5 * time.Minute

// InstallationToken returns an installation access token for
// installationID, reusing a previously minted one while it is still
// comfortably valid and only exchanging a fresh app JWT for a new token
// when the cache is empty or the cached one is near expiry. GitHub
// issues these valid for an hour, so this collapses the orchestrator's
// per-pass token needs (2s reconcile, 5s scheduled-task dispatch, per
// build) down to at most one real mint per installation per ~hour --
// which is what keeps those loops from tripping GitHub's secondary rate
// limit on the mint endpoint. Every token-needing path should call this
// rather than GenerateAppJWT + CreateInstallationToken directly.
//
// Concurrent callers for the same installation serialize on the mint
// (the lock is held across the network call) so a burst of first-time
// misses produces one mint, not one per caller.
func (c *Client) InstallationToken(ctx context.Context, appID string, key *rsa.PrivateKey, installationID int64) (*InstallationToken, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if t, ok := c.tokens[installationID]; ok && time.Now().Before(t.expiresAt.Add(-installationTokenExpiryMargin)) {
		return &InstallationToken{Token: t.token, ExpiresAt: t.expiresAt}, nil
	}
	jwt, err := GenerateAppJWT(appID, key, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing app jwt: %w", err)
	}
	tok, err := c.CreateInstallationToken(ctx, jwt, installationID)
	if err != nil {
		return nil, err
	}
	if c.tokens == nil {
		c.tokens = make(map[int64]cachedInstallationToken)
	}
	c.tokens[installationID] = cachedInstallationToken{token: tok.Token, expiresAt: tok.ExpiresAt}
	return tok, nil
}
