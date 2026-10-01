// laforge-api is "the api" service: the one service humans, the CLI, and
// GitHub talk to.
// Configuration is environment variables only, on the principle
// "Configuration by environment variables and files, so the same images
// run under Compose, Railway, or anything else" -- no config file, no
// flags beyond nothing. "...and files" is internal/envfile: a real
// deployment (or docker-compose, which has its own native env_file
// handling) sets real env vars and has no .env on disk at all, but local
// dev can drop real values in a gitignored .env once instead of
// re-exporting them every shell session -- see .env.example.
//
// The GitHub-facing config below is a single GitHub App's identity,
// not an OAuth App plus a hand-pasted webhook secret. Every value here is
// per-deployment: nothing in this codebase names a specific App,
// organization, or installation, so another organization self-hosting
// LaForge registers and configures its own.
package main

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/api"
	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/envfile"
	"github.com/globalcptc/laforge/internal/ghclient"
)

func main() {
	if err := envfile.Load(envOr("LAFORGE_ENV_FILE", ".env")); err != nil {
		log.Fatalf("reading .env: %v", err)
	}

	databaseURL := requireEnv("DATABASE_URL")
	// GITHUB_APP_WEBHOOK_SECRET verifies every inbound webhook delivery
	// (push, installation, installation_repositories) actually came from
	// this App -- set once, when the App is registered, never handled by
	// anyone installing it.
	webhookSecret := requireEnv("GITHUB_APP_WEBHOOK_SECRET")
	// GITHUB_SERVICE_TOKEN is optional: it's the fallback resolveCloneURL
	// (internal/api/webhook.go) uses for a repository with no
	// installation behind it, and CI-status gating's own token either
	// way. Without it, CI status gating degrades to judging purely on
	// LaForge's own validation, and a repository with no installation
	// can't be fetched authoritatively at all -- a deliberately graceful
	// degradation, not a hard requirement to start the service.
	serviceToken := os.Getenv("GITHUB_SERVICE_TOKEN")
	addr := envOr("LISTEN_ADDR", ":8080")
	// GITHUB_APP_CLIENT_ID/SECRET are the SAME App's user-to-server OAuth
	// credentials (see internal/api/oauth.go) -- a GitHub App exposes
	// both an app identity (below) and OAuth client credentials under
	// one registration, so this is one App, not two. Unset means
	// /auth/github/login exists but the redirect it builds won't work
	// against a real App.
	oauthClientID := os.Getenv("GITHUB_APP_CLIENT_ID")
	oauthClientSecret := os.Getenv("GITHUB_APP_CLIENT_SECRET")
	publicBaseURL := envOr("PUBLIC_BASE_URL", "http://localhost"+addr)
	uiBaseURL := envOr("UI_BASE_URL", "http://localhost:5173")
	// REPO_ROOT mirrors cmd/laforge-orchestrator's and cmd/laforge-runner's
	// own -repo flag -- see internal/api/server.go's RepoRoot doc comment
	// for the shared single-checkout limitation this carries over.
	repoRoot := envOr("REPO_ROOT", ".")

	// GITHUB_APP_ID/GITHUB_APP_PRIVATE_KEY_PATH together let the server
	// mint its own installation access tokens (internal/ghclient/app.go)
	// -- both optional, same graceful-degradation reasoning as
	// GITHUB_SERVICE_TOKEN: without them, every repository falls back to
	// GITHUB_SERVICE_TOKEN for content-fetch, same as before. Both are
	// required together; one without the other is a
	// config mistake worth failing loudly on rather than silently
	// half-working.
	appID := os.Getenv("GITHUB_APP_ID")
	appPrivateKeyPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	appSlug := os.Getenv("GITHUB_APP_SLUG")
	if (appID == "") != (appPrivateKeyPath == "") {
		log.Fatal("GITHUB_APP_ID and GITHUB_APP_PRIVATE_KEY_PATH must be set together (or both left unset)")
	}
	var appPrivateKey *rsa.PrivateKey
	if appPrivateKeyPath != "" {
		pemBytes, err := os.ReadFile(appPrivateKeyPath)
		if err != nil {
			log.Fatalf("reading GITHUB_APP_PRIVATE_KEY_PATH: %v", err)
		}
		key, err := ghclient.ParsePrivateKeyPEM(pemBytes)
		if err != nil {
			log.Fatalf("parsing GitHub App private key: %v", err)
		}
		appPrivateKey = key
	}

	// LAFORGE_ADMIN_LOGINS is instance-wide admin, for the one action
	// that has no repository yet to check per-repository access
	// against: approving an installed repository into LaForge's own
	// tracking (internal/api/installations.go). Comma-separated GitHub
	// logins; empty means nobody can approve anything yet, which is a
	// safe, honest default over silently allowing everyone.
	var adminLogins []string
	for _, login := range strings.Split(os.Getenv("LAFORGE_ADMIN_LOGINS"), ",") {
		if login = strings.TrimSpace(login); login != "" {
			adminLogins = append(adminLogins, login)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	q, pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	gh := ghclient.New()
	if v := os.Getenv("GITHUB_API_BASE_URL"); v != "" {
		gh.APIBaseURL = v // only ever set in tests/dev against a mock server
	}

	server := api.New(q, pool, gh, webhookSecret, serviceToken)
	server.GitHubClientID = oauthClientID
	server.GitHubClientSecret = oauthClientSecret
	server.PublicBaseURL = publicBaseURL
	server.UIBaseURL = uiBaseURL
	server.RepoRoot = repoRoot
	// LAFORGE_CROSS_DOMAIN_COOKIES: see internal/api.Server.CrossOriginCookies's
	// own doc comment -- only needed when PublicBaseURL and UIBaseURL are
	// genuinely different origins.
	server.CrossOriginCookies = os.Getenv("LAFORGE_CROSS_DOMAIN_COOKIES") == "true"
	server.AppID = appID
	server.AppPrivateKey = appPrivateKey
	server.AppSlug = appSlug
	server.AdminLogins = adminLogins
	// The agent mTLS CA and gateway server cert, read only to report expiry via
	// GET /cert-status (the same files the gateway/runner use; mounted read-only
	// into this service). Optional -- unset means the endpoint reports them as
	// unconfigured rather than failing.
	server.CACertPath = os.Getenv("GATEWAY_CA_CERT")
	server.ServerCertPath = os.Getenv("GATEWAY_SERVER_CERT")
	// Interactive shell relay: the gateway's internal relay address plus the api's
	// own mTLS client identity (CA + api client cert/key), used to bridge a
	// user's terminal to a host agent. All three unset leaves the feature off
	// (the endpoint reports it unavailable). See internal/api/terminal.go.
	server.GatewayRelayAddr = os.Getenv("GATEWAY_RELAY_ADDR")
	server.MaxShellSessions = envInt("MAX_SHELL_SESSIONS", 2)
	if server.GatewayRelayAddr != "" {
		// Server name the api expects on the gateway's relay cert. Defaults to the
		// host part of the relay address; override with GATEWAY_SERVER_NAME when
		// the cert is issued for a different name (e.g. the docker service name
		// must be a SAN on the gateway server cert -- see scripts/gen-certs.sh).
		serverName := envOr("GATEWAY_SERVER_NAME", hostOnly(server.GatewayRelayAddr))
		relayCfg, err := apiRelayTLS(os.Getenv("GATEWAY_CA_CERT"), os.Getenv("API_CLIENT_CERT"), os.Getenv("API_CLIENT_KEY"), serverName)
		if err != nil {
			log.Fatalf("building shell-relay TLS config: %v", err)
		}
		server.RelayTLSConfig = relayCfg
	}
	// Same graceful degradation as everywhere else this exists (see
	// checkout.Cache's own doc comment): with an App or service token
	// configured, handleRenderObject resolves each build's own
	// repository/commit through a real checkout cache
	// instead of the fixed RepoRoot
	// above; without one, Checkouts stays nil and RepoRoot is used
	// directly, exactly as before this existed.
	if appID != "" || serviceToken != "" {
		server.Checkouts = &checkout.Cache{
			BaseDir: envOr("CHECKOUT_CACHE_DIR", filepath.Join(os.TempDir(), "laforge-checkouts")),
			Q:       q, GH: gh, AppID: appID, AppPrivateKey: appPrivateKey, ServiceToken: serviceToken,
		}
	}
	httpServer := &http.Server{Addr: addr, Handler: server}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("laforge-api listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", name)
	}
	return v
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envInt(name string, fallback int) int {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid integer for %s: %q", name, v)
	}
	return n
}

// hostOnly returns the host part of a host:port address (the SNI the api
// presents to the gateway relay), tolerating a bare host with no port.
func hostOnly(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

// apiRelayTLS builds the mTLS client config the api uses to dial the gateway's
// internal shell relay: the shared CA as the trust root, the api's own client
// cert/key for mutual auth, verifying the gateway's cert against serverName.
func apiRelayTLS(caPath, certPath, keyPath, serverName string) (*tls.Config, error) {
	if caPath == "" || certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("GATEWAY_RELAY_ADDR is set but GATEWAY_CA_CERT / API_CLIENT_CERT / API_CLIENT_KEY are not all configured")
	}
	ca, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert %s: %w", caPath, err)
	}
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading API client cert %s: %w", certPath, err)
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading API client key %s: %w", keyPath, err)
	}
	return agentpki.ClientTLSConfig(ca, cert, key, serverName)
}
