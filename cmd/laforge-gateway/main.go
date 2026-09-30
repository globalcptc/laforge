// laforge-gateway is the agent-gateway: "its own container, its
// own domain, scaled on its own. Speaks only the agent protocol." Real
// mTLS (the CA/cert files are read from disk here, at the process
// boundary -- everything below main() only ever sees an already-built
// *tls.Config, never a file path), connected to Postgres as the
// restricted laforge_gateway role.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/envfile"
	"github.com/globalcptc/laforge/internal/gateway"
)

func main() {
	if err := envfile.Load(envOr("LAFORGE_ENV_FILE", ".env")); err != nil {
		log.Fatalf("reading .env: %v", err)
	}

	dbURL := requireEnv("DATABASE_URL")
	caCertPath := requireEnv("GATEWAY_CA_CERT")
	serverCertPath := requireEnv("GATEWAY_SERVER_CERT")
	serverKeyPath := requireEnv("GATEWAY_SERVER_KEY")
	listenAddr := envOr("LISTEN_ADDR", ":8443")
	leaseSeconds := envInt("LEASE_SECONDS", 60)
	basePollMS := envInt("BASE_POLL_MS", 15000)
	jitterMS := envInt("JITTER_MS", 5000)

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("reading GATEWAY_CA_CERT %s: %v", caCertPath, err)
	}
	serverCert, err := os.ReadFile(serverCertPath)
	if err != nil {
		log.Fatalf("reading GATEWAY_SERVER_CERT %s: %v", serverCertPath, err)
	}
	serverKey, err := os.ReadFile(serverKeyPath)
	if err != nil {
		log.Fatalf("reading GATEWAY_SERVER_KEY %s: %v", serverKeyPath, err)
	}
	tlsCfg, err := agentpki.ServerTLSConfig(caCert, serverCert, serverKey)
	if err != nil {
		log.Fatalf("building TLS config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_, pool, err := db.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	srv := &gateway.Server{
		Pool: pool, TLSConfig: tlsCfg,
		LeaseDuration: time.Duration(leaseSeconds) * time.Second,
		BasePollMS:    basePollMS, JitterMS: jitterMS,
	}
	ln, err := srv.Listen(listenAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
	log.Printf("laforge-gateway listening on %s (mTLS)", listenAddr)
	if err := srv.Serve(ctx, ln); err != nil && ctx.Err() == nil {
		log.Fatalf("serve: %v", err)
	}
	log.Println("laforge-gateway shut down")
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
