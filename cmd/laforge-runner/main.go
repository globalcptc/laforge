// laforge-runner is the runner: "a runner takes a task with FOR
// UPDATE SKIP LOCKED, gets a lease, and heartbeats." Holds nothing
// durable -- every fact it needs comes from Postgres or the checkout on
// every task, so it's always safe to kill.
//
// Two crash-injection flags exist purely for internal/chaos and manual
// testing, matching a proven lease-reclaim pattern:
// -crash-after-lease crashes with zero work done, -crash-after-builder-call
// crashes after the (idempotent) builder call succeeds but before the
// result is recorded, proving the "partial success converges, doesn't
// duplicate" half of the resilience design, not just "a dead lease gets
// reclaimed."
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/envfile"
	"github.com/globalcptc/laforge/internal/runner"
)

func main() {
	envFile := os.Getenv("LAFORGE_ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	if err := envfile.Load(envFile); err != nil {
		log.Fatalf("reading .env: %v", err)
	}

	// -db-url defaults from DATABASE_URL (set directly, or via .env --
	// see internal/envfile), same reasoning as cmd/laforge-orchestrator.
	dbURL := flag.String("db-url", os.Getenv("DATABASE_URL"), "postgres connection string (required, or set DATABASE_URL)")
	repoRoot := flag.String("repo", ".", "content checkout path to resolve tasks against (fallback when no checkout cache is configured)")
	checkoutCacheDir := flag.String("checkout-cache-dir", filepath.Join(os.TempDir(), "laforge-checkouts"), "where fetched checkouts are cached, one per {repository, commit}, when GitHub credentials are configured (GITHUB_APP_ID/GITHUB_APP_PRIVATE_KEY_PATH or GITHUB_SERVICE_TOKEN) -- see internal/checkout")
	id := flag.String("id", "", "runner identity (required, must be unique among concurrently running runners)")
	leaseSeconds := flag.Int("lease-seconds", 30, "lease duration in seconds")
	heartbeatSeconds := flag.Int("heartbeat-seconds", 5, "heartbeat interval in seconds")
	maxTasks := flag.Int("max-tasks", 0, "stop after completing this many tasks (0 = run until the queue is empty)")
	workDelayMS := flag.Int("work-delay-ms", 0, "fake builder: simulated hoster API latency per call, in milliseconds")
	pollInterval := flag.Duration("poll-interval", 200*time.Millisecond, "how often to check for work when the queue is empty")
	concurrency := flag.Int("concurrency", envIntDefault("RUNNER_CONCURRENCY", 1), "how many tasks to lease and execute in PARALLEL. Each worker is an independent lease->execute loop; FOR UPDATE SKIP LOCKED makes concurrent leasing safe, so N workers deploy N hosts at once instead of one at a time. Default 1 (serial). The chaos/crash flags assume 1.")
	crashAfterLease := flag.Int("crash-after-lease", -1, "os.Exit(9) immediately after leasing the Nth task this process leases (1-indexed; -1 = never), zero work done")
	crashAfterBuilderCall := flag.Int("crash-after-builder-call", -1, "os.Exit(9) immediately after the Nth builder call this process makes succeeds, before recording the result (1-indexed; -1 = never)")
	buildID := flag.String("build-id", "", "restrict leasing to this one build's tasks (test isolation only -- see runner.Runner.BuildID; a real deployment never sets this)")
	flag.Parse()

	if *dbURL == "" || *id == "" {
		log.Fatal("-db-url (or DATABASE_URL) and -id are required")
	}

	workers := *concurrency
	if workers < 1 {
		workers = 1
	}

	ctx := context.Background()
	// Size the pool to the worker count (+ headroom for the per-task heartbeat
	// loops and result-recording queries): each worker only holds a connection
	// briefly (the multi-minute deploy is an HTTP call to the hoster, no DB
	// connection held), but N workers querying at once must not starve the
	// default max(4, NumCPU) pool. Postgres's own max_connections is the ceiling.
	q, pool, err := db.OpenWithMaxConns(ctx, *dbURL, int32(workers+8))
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	// See cmd/laforge-orchestrator/main.go's identical wiring and
	// checkout.FromEnv's own doc comment: with GitHub credentials
	// configured, each task resolves its own repository/commit through a
	// real checkout cache instead of every task sharing one fixed -repo
	// path. Without credentials, this is nil and Runner falls back to
	// RepoRoot exactly as before this existed.
	cache, err := checkout.FromEnv(q, *checkoutCacheDir)
	if err != nil {
		log.Fatalf("configuring checkout cache: %v", err)
	}

	// Agent auto-delivery: when the gateway's public address, the api's
	// public URL (both as reachable from the deployed hosts), the gateway
	// CA, and a directory of platform base binaries are all configured,
	// the runner delivers an agent to every host it deploys. Unset means
	// "deploy without agents," exactly as before -- see agentdelivery.
	delivery, err := agentdelivery.Load(
		os.Getenv("GATEWAY_PUBLIC_ADDR"), os.Getenv("API_PUBLIC_URL"),
		os.Getenv("GATEWAY_CA_CERT"), os.Getenv("GATEWAY_CA_KEY"), os.Getenv("AGENT_BASE_DIR"),
	)
	if err != nil {
		log.Fatalf("configuring agent delivery: %v", err)
	}
	if delivery.Enabled() {
		log.Printf("[%s] agent auto-delivery enabled (gateway %s, api %s)", *id, os.Getenv("GATEWAY_PUBLIC_ADDR"), os.Getenv("API_PUBLIC_URL"))
	}

	b := fake.New(pool)
	b.WorkDelay = time.Duration(*workDelayMS) * time.Millisecond

	r := &runner.Runner{
		// Builder (the fake one, still) is the fallback for a build with
		// no configured_build at all -- see Runner.ResolveBuilderFromDB's
		// own doc comment. Every build that DOES have one resolves its
		// real builder_config by name instead (the
		// "what about the MicroCloud builder" gap).
		Pool: pool, Builder: b, RepoRoot: *repoRoot, Checkouts: cache, ID: *id, Delivery: delivery,
		LeaseDuration:     time.Duration(*leaseSeconds) * time.Second,
		HeartbeatInterval: time.Duration(*heartbeatSeconds) * time.Second,
	}
	r.Builders = r.ResolveBuilderFromDB
	if *buildID != "" {
		var bid pgtype.UUID
		if err := bid.Scan(*buildID); err != nil {
			log.Fatalf("invalid -build-id %q: %v", *buildID, err)
		}
		r.BuildID = &bid
	}

	var leaseCount int64
	builderCallCount := 0
	if *crashAfterBuilderCall >= 0 {
		// Chaos/crash flags assume a single worker (concurrency 1); the hook's
		// counter is not synchronized because nothing sets these in a real,
		// multi-worker deployment.
		r.AfterBuilderCall = func() {
			builderCallCount++
			if builderCallCount == *crashAfterBuilderCall {
				log.Printf("[%s] BUILDER-CALL-DONE %d -- crashing now (hard exit, no cleanup) as instructed", *id, builderCallCount)
				os.Stdout.Sync()
				os.Exit(9)
			}
		}
	}

	if workers > 1 {
		log.Printf("[%s] running %d workers (parallel task execution)", *id, workers)
	}

	var completed int64
	done := make(chan struct{}) // closed when max-tasks is reached, to stop every worker

	worker := func(workerNum int) {
		for {
			select {
			case <-done:
				return
			default:
			}
			if *maxTasks > 0 && atomic.LoadInt64(&completed) >= int64(*maxTasks) {
				return
			}

			task, err := r.LeaseOne(ctx)
			if err != nil {
				// A lease error is a transient DB problem, not a reason to kill
				// every other in-flight worker -- log, back off, retry.
				log.Printf("[%s] lease error: %v", *id, err)
				time.Sleep(*pollInterval)
				continue
			}
			if task == nil {
				time.Sleep(*pollInterval)
				continue
			}
			n := atomic.AddInt64(&leaseCount, 1)
			log.Printf("LEASED %s kind=%s attempt=%d", task.ID, task.Kind, task.Attempts)
			os.Stdout.Sync()

			if *crashAfterLease >= 0 && n == int64(*crashAfterLease) {
				log.Printf("[%s] crashing now (hard exit, no cleanup) right after leasing task %s, as instructed", *id, task.ID)
				os.Stdout.Sync()
				os.Exit(9)
			}

			if err := r.ExecuteAndRecord(ctx, *task); err != nil {
				log.Printf("[%s] task %s error: %v", *id, task.ID, err)
				continue
			}
			total := atomic.AddInt64(&completed, 1)
			log.Printf("COMPLETED %s (total %d)", task.ID, total)
			os.Stdout.Sync()
			if *maxTasks > 0 && total >= int64(*maxTasks) {
				log.Printf("[%s] reached -max-tasks=%d, exiting cleanly", *id, *maxTasks)
				closeOnce(done)
				return
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			worker(n)
		}(i)
	}
	wg.Wait()
}

// closeOnce closes ch if it isn't already closed -- several workers may reach
// max-tasks at once.
func closeOnce(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}

// envIntDefault reads an integer environment variable, falling back to def when
// unset or unparseable -- lets the runner's concurrency be set from the compose
// env (RUNNER_CONCURRENCY) without a flag.
func envIntDefault(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
