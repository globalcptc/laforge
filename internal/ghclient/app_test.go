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
