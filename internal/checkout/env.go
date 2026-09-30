package checkout

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/ghclient"
)

// FromEnv builds a Cache from the same GITHUB_APP_ID/
// GITHUB_APP_PRIVATE_KEY_PATH/GITHUB_SERVICE_TOKEN environment variables
// cmd/laforge-api already reads (see its own main.go), so every
// laforge-* process configures GitHub credentials the same way instead
// of each background process (cmd/laforge-orchestrator,
// cmd/laforge-runner) reimplementing the same App-key parsing.
//
// Returns (nil, nil) -- not an error -- when neither an App nor a
// service token is configured: every real caller treats a nil *Cache as
// "fall back to the old fixed -repo/REPO_ROOT path," the same graceful
// degradation cmd/laforge-api's own credential handling already uses
// for a repository with no installation.
func FromEnv(q *db.Queries, baseDir string) (*Cache, error) {
	appID := os.Getenv("GITHUB_APP_ID")
	appPrivateKeyPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	serviceToken := os.Getenv("GITHUB_SERVICE_TOKEN")
	if (appID == "") != (appPrivateKeyPath == "") {
		return nil, fmt.Errorf("GITHUB_APP_ID and GITHUB_APP_PRIVATE_KEY_PATH must be set together (or both left unset)")
	}
	if appID == "" && serviceToken == "" {
		return nil, nil
	}

	var appPrivateKey *rsa.PrivateKey
	if appPrivateKeyPath != "" {
		pemBytes, err := os.ReadFile(appPrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("reading GITHUB_APP_PRIVATE_KEY_PATH: %w", err)
		}
		key, err := ghclient.ParsePrivateKeyPEM(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing GitHub App private key: %w", err)
		}
		appPrivateKey = key
	}

	gh := ghclient.New()
	if v := os.Getenv("GITHUB_API_BASE_URL"); v != "" {
		gh.APIBaseURL = v // only ever set in tests/dev against a mock server
	}

	return &Cache{
		BaseDir: baseDir, Q: q, GH: gh,
		AppID: appID, AppPrivateKey: appPrivateKey, ServiceToken: serviceToken,
	}, nil
}
