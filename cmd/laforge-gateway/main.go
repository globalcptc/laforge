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
	"net"
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
	// Container-log forwarding is driven by each environment's container_logs
	// (native driver where the builder supports it; gateway fallback for
	// Incus/Zun). The gateway builds those fallback sinks lazily and closes them
	// on shutdown; there is no global LOG_SINK_* env anymore.
	defer srv.CloseSinks()
	ln, err := srv.Listen(listenAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
	log.Printf("laforge-gateway listening on %s (mTLS)", listenAddr)

	// The internal relay listener for interactive shells runs alongside the
	// agent listener on its own goroutine, on a fixed internal port. It is
	// PLAINTEXT on purpose: it never leaves the internal (docker) network and is
	// never funnel-exposed -- the same trust model as every service's Postgres
	// connection. A failure here logs but never takes down the agent protocol,
	// the gateway's real job.
	const relayListenAddr = ":8445"
	relayLn, err := net.Listen("tcp", relayListenAddr)
	if err != nil {
		log.Fatalf("relay listen %s: %v", relayListenAddr, err)
	}
	log.Printf("laforge-gateway shell relay listening on %s (internal, plaintext)", relayListenAddr)
	go func() {
		if err := srv.ServeRelay(ctx, relayLn); err != nil && ctx.Err() == nil {
			log.Printf("laforge-gateway shell relay stopped: %v", err)
		}
	}()

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
