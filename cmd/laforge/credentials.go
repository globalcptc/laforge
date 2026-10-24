package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The CLI's GitHub credential store. The device-flow access token GitHub issues
// expires in ~8h (the App expires user tokens), so storing just the access
// token -- as the legacy ~/.laforge/token file did -- left `laforge` commands
// failing once it lapsed. We now also keep the refresh token and expiries in
// ~/.laforge/credentials.json and refresh the access token on use. The CLI
// can't refresh against GitHub directly (that needs the App's client secret,
// which never leaves the server), so it refreshes THROUGH laforge-api's
// /auth/github/refresh endpoint.

// refreshSkew refreshes a little before the token actually expires so a token
// that's valid when a command starts doesn't lapse mid-request.
const refreshSkew = 2 * time.Minute

type storedCreds struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token,omitempty"`
	AccessExpiresAt  time.Time `json:"access_expires_at,omitempty"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitempty"`
}

func credsPath() (string, error) {
	dir, err := laforgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func loadCreds() (storedCreds, bool) {
	path, err := credsPath()
	if err != nil {
		return storedCreds{}, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return storedCreds{}, false
	}
	var c storedCreds
	if err := json.Unmarshal(b, &c); err != nil || c.AccessToken == "" {
		return storedCreds{}, false
	}
	return c, true
}

// saveCreds writes the full credential set, and mirrors the access token into
// the legacy ~/.laforge/token file so anything still reading that (and
// readStoredToken's fallback) sees the current token.
func saveCreds(c storedCreds) error {
	dir, err := laforgeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := credsPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return writeStoredToken(c.AccessToken)
}

// resolveToken returns the access token to send on a request, refreshing it
// first if it has expired and a refresh token is available. Best-effort: if
// there's no credentials file it falls back to the legacy token file; if a
// refresh fails it returns the stale token and lets the call surface the real
// "sign in again" error from the server.
func resolveToken(baseURL string) string {
	creds, ok := loadCreds()
	if !ok {
		tok, _ := readStoredToken() // legacy single-token file (pre-refresh logins)
		return tok
	}
	if creds.AccessExpiresAt.IsZero() || time.Now().Before(creds.AccessExpiresAt.Add(-refreshSkew)) {
		return creds.AccessToken // no recorded expiry, or still valid
	}
	if creds.RefreshToken != "" && (creds.RefreshExpiresAt.IsZero() || time.Now().Before(creds.RefreshExpiresAt)) {
		if fresh, err := refreshViaAPI(baseURL, creds.RefreshToken); err == nil {
			_ = saveCreds(fresh)
			return fresh.AccessToken
		}
	}
	return creds.AccessToken
}

// refreshViaAPI trades the refresh token for a fresh one through laforge-api
// (which holds the client secret). A bare request, deliberately not newAPIClient,
// to avoid recursing through resolveToken.
func refreshViaAPI(baseURL, refreshToken string) (storedCreds, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/auth/github/refresh", bytes.NewReader(body))
	if err != nil {
		return storedCreds{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return storedCreds{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return storedCreds{}, fmt.Errorf("refresh endpoint returned HTTP %d", resp.StatusCode)
	}
	var out struct {
		AccessToken           string `json:"access_token"`
		RefreshToken          string `json:"refresh_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return storedCreds{}, err
	}
	if out.AccessToken == "" {
		return storedCreds{}, fmt.Errorf("refresh endpoint returned no access token")
	}
	rt := out.RefreshToken
	if rt == "" {
		rt = refreshToken // GitHub normally rotates; keep the old one if it didn't
	}
	return storedCreds{
		AccessToken:      out.AccessToken,
		RefreshToken:     rt,
		AccessExpiresAt:  expiryFromSeconds(out.ExpiresIn),
		RefreshExpiresAt: expiryFromSeconds(out.RefreshTokenExpiresIn),
	}, nil
}

// expiryFromSeconds is a wall-clock expiry n seconds out, or the zero time when
// n<=0 (GitHub omits a lifetime for a token kind that doesn't expire).
func expiryFromSeconds(n int) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(n) * time.Second)
}
