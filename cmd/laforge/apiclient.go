package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// apiClient is deliberately tiny: the CLI's additions are thin
// wrappers over internal/api's HTTP endpoints, decoding straight into the
// same db.Repository/db.ConfiguredBuild types the server itself returns
// (sqlc already gives them proper json tags, and pgtype.UUID round-trips
// through JSON cleanly) rather than maintaining a second, parallel set of
// client-side structs.
type apiClient struct {
	baseURL string
	token   string
}

func newAPIClient() *apiClient {
	base := os.Getenv("LAFORGE_API_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	tok, _ := readStoredToken()
	return &apiClient{baseURL: base, token: tok}
}

// apiError is a decoded {"error": "..."} body from internal/api's own
// writeError, so a 403 from the server reads as "your github token does
// not have push access to this repository," not a bare "HTTP 403".
type apiError struct {
	StatusCode int
	Message    string
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("laforge-api returned HTTP %d", e.StatusCode)
}

func (c *apiClient) do(method, path string, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("reaching laforge-api at %s: %w (is it running? see LAFORGE_API_URL)", c.baseURL, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var decoded struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &decoded)
		return &apiError{StatusCode: resp.StatusCode, Message: decoded.Error}
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding response from %s: %w", path, err)
		}
	}
	return nil
}

// --- token storage ---
//
// A plain file under ~/.laforge, 0600, holding nothing but the raw GitHub
// token `laforge login` obtained. No keychain integration yet -- matches
// the scope of everything else GitHub-auth-related: real, working, and
// honestly not yet at production hardening.

func laforgeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".laforge"), nil
}

func tokenPath() (string, error) {
	dir, err := laforgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token"), nil
}

func readStoredToken() (string, error) {
	path, err := tokenPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

func writeStoredToken(tok string) error {
	dir, err := laforgeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := tokenPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(tok), 0o600)
}
