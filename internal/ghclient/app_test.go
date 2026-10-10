package ghclient

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test RSA key: %v", err)
	}
	return key
}

// TestGenerateAppJWTHasValidSignatureAndClaims verifies the actual
// cryptography, not just "it produced three dot-separated parts": decode
// the real claims back out and verify the signature against the key's
// own public half with rsa.VerifyPKCS1v15, the same check GitHub's
// server performs on the other end.
func TestGenerateAppJWTHasValidSignatureAndClaims(t *testing.T) {
	key := testRSAKey(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tok, err := GenerateAppJWT("123456", key, now)
	if err != nil {
		t.Fatalf("GenerateAppJWT: %v", err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d dot-separated parts, want 3", len(parts))
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decoding header: %v", err)
	}
	if string(headerJSON) != `{"alg":"RS256","typ":"JWT"}` {
		t.Fatalf("header = %s, want RS256/JWT", headerJSON)
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decoding claims: %v", err)
	}
	var claims appJWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatalf("unmarshaling claims: %v", err)
	}
	if claims.Issuer != "123456" {
		t.Fatalf("iss = %q, want 123456", claims.Issuer)
	}
	wantIat := now.Add(-60 * time.Second).Unix()
	if claims.IssuedAt != wantIat {
		t.Fatalf("iat = %d, want %d (now - 60s clock-drift buffer)", claims.IssuedAt, wantIat)
	}
	if d := claims.ExpiresAt - claims.IssuedAt; d <= 0 || d > 10*60 {
		t.Fatalf("exp - iat = %ds, want (0, 600] (GitHub's own 10-minute maximum)", d)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decoding signature: %v", err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("signature does not verify against the signing key's own public half: %v", err)
	}

	// A DIFFERENT key's public half must NOT verify this signature --
	// proves the test above isn't accidentally vacuous (e.g. a
	// verification call that always returns nil).
	otherKey := testRSAKey(t)
	if err := rsa.VerifyPKCS1v15(&otherKey.PublicKey, crypto.SHA256, digest[:], sig); err == nil {
		t.Fatal("signature verified against an unrelated key's public half -- verification is not actually checking anything")
	}
}

func TestCreateInstallationToken(t *testing.T) {
	const jwt = "fake-app-jwt"
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/app/installations/42/access_tokens"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.Method, http.MethodPost; got != want {
			t.Errorf("method = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer "+jwt; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"token": "ghs_realistic-installation-token", "expires_at": "2026-01-01T13:00:00Z",
		})
	}), nil)

	tok, err := c.CreateInstallationToken(context.Background(), jwt, 42)
	if err != nil {
		t.Fatalf("CreateInstallationToken: %v", err)
	}
	if tok.Token != "ghs_realistic-installation-token" {
		t.Fatalf("Token = %q, want the real value from the response body", tok.Token)
	}
	want := time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)
	if !tok.ExpiresAt.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", tok.ExpiresAt, want)
	}
}

// TestInstallationTokenCachesUntilNearExpiry is the regression test for the
// rate-limit incident: minting a fresh installation token on every call
// tripped GitHub's secondary rate limit on the access_tokens endpoint.
// InstallationToken must reuse a still-valid token (no second network
// call) and only re-mint when the cached one is within the expiry margin.
func TestInstallationTokenCachesUntilNearExpiry(t *testing.T) {
	key := testRSAKey(t)
	var mints int
	// expiresAt is controlled per response so the test can drive the
	// cache from "comfortably valid" to "inside the expiry margin".
	expiresAt := time.Now().Add(time.Hour)
	c := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mints++
		json.NewEncoder(w).Encode(map[string]string{
			"token":      "ghs_token",
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		})
	}), nil)

	// First call mints.
	if _, err := c.InstallationToken(context.Background(), "app-id", key, 42); err != nil {
		t.Fatalf("InstallationToken (first): %v", err)
	}
	// Several more calls while the token is comfortably valid must all be
	// served from cache -- still exactly one mint.
	for i := 0; i < 5; i++ {
		if _, err := c.InstallationToken(context.Background(), "app-id", key, 42); err != nil {
			t.Fatalf("InstallationToken (cached): %v", err)
		}
	}
	if mints != 1 {
		t.Fatalf("minted %d times, want 1 (the rest must be cache hits)", mints)
	}

	// A different installation is a separate cache entry -- it mints.
	if _, err := c.InstallationToken(context.Background(), "app-id", key, 99); err != nil {
		t.Fatalf("InstallationToken (other installation): %v", err)
	}
	if mints != 2 {
		t.Fatalf("minted %d times, want 2 (one per installation)", mints)
	}

	// A token that comes back already inside the expiry margin must never
	// be treated as a usable cache entry: every call for installation 7
	// re-mints, because the token it just got is too close to expiry to
	// reuse. (This is what guarantees correctness near the boundary -- it
	// trades an extra mint for never handing out a token about to lapse.)
	expiresAt = time.Now().Add(installationTokenExpiryMargin / 2)
	if _, err := c.InstallationToken(context.Background(), "app-id", key, 7); err != nil {
		t.Fatalf("InstallationToken (near-expiry mint): %v", err)
	}
	if mints != 3 {
		t.Fatalf("minted %d times, want 3", mints)
	}
	if _, err := c.InstallationToken(context.Background(), "app-id", key, 7); err != nil {
		t.Fatalf("InstallationToken (near-expiry re-mint): %v", err)
	}
	if mints != 4 {
		t.Fatalf("minted %d times, want 4 (a near-expiry token is never cached as usable)", mints)
	}
}
