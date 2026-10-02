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
	"strings"
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
	// Optional container-log forwarding. LOG_SINK_URL is the HTTP endpoint the
	// gateway POSTs NDJSON to (a generic collector -- Vector/Fluent Bit in front
	// of Splunk/Loki/Elastic, or anything that accepts newline-JSON). Unset =
	// disabled. LOG_SINK_HEADERS carries any auth ("Authorization: Splunk <tok>")
	// as comma-separated "Name: value" pairs; LOG_SINK_LABELS adds static
	// key=value fields to every record (e.g. env=cptc2026).
	logSinkURL := os.Getenv("LOG_SINK_URL")
	logSinkHeaders := parseKV(os.Getenv("LOG_SINK_HEADERS"), ":")
	logSinkLabels := parseKV(os.Getenv("LOG_SINK_LABELS"), "=")

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
	if logSinkURL != "" {
		sink := gateway.NewHTTPSink(logSinkURL, logSinkHeaders, logSinkLabels)
		defer sink.Close()
		srv.LogSink = sink
		log.Printf("laforge-gateway forwarding container logs to %s", logSinkURL)
	}
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

// parseKV splits a comma-separated "key<sep>value" list into a map, trimming
// surrounding whitespace. Empty or malformed entries are skipped. Used for the
// optional log-sink headers (sep ":") and static labels (sep "=").
func parseKV(s, sep string) map[string]string {
	if s == "" {
		return nil
	}
	m := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		i := strings.Index(pair, sep)
		if i < 0 {
			continue
		}
		k := strings.TrimSpace(pair[:i])
		v := strings.TrimSpace(pair[i+1:])
		if k != "" {
			m[k] = v
		}
	}
	return m
}
