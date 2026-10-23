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
	"fmt"
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
	// Production uses PER-BUILD LANES: one lane per build, each leasing only its
	// own tasks, so no build's work ever sits behind another's in a shared FIFO
	// pool (the cross-build stall that made a new build wait minutes on an
	// unrelated one). The legacy shared worker pool stays only for the
	// test-isolation / crash-injection flags -- chaos always sets -build-id. In
	// lane mode, -concurrency / RUNNER_CONCURRENCY is the PER-BUILD parallelism.
	legacyMode := *buildID != "" || *maxTasks > 0 || *crashAfterLease >= 0

	ctx := context.Background()
	// Pool sizing. A worker/task holds a connection only briefly (the multi-minute
	// deploy is an HTTP call to the hoster, no DB connection held), so a modest
	// pool serves far more concurrent tasks than its size. Legacy: workers+8. Lane
	// mode: one generous pool shared by every build's lane (RUNNER_MAX_CONNS);
	// Postgres's own max_connections is the ceiling.
	poolConns := int32(workers + 8)
	if !legacyMode {
		poolConns = int32(envIntDefault("RUNNER_MAX_CONNS", 50))
	}
	q, pool, err := db.OpenWithMaxConns(ctx, *dbURL, poolConns)
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

	if !legacyMode {
		// PRODUCTION: one worker pool PER CONFIGURED BUILDER. Concurrency is a
		// property of the hardware -- the cluster/account/host-pool -- not of a
		// build, so each builder gets its own pool sized to its own concurrency
		// limit (builder_config.max_concurrency; 0 = unbounded). All builds on a
		// builder share its pool; different builders never contend. The only shared
		// resource is Postgres (RUNNER_MAX_CONNS), and task DB holds are brief.
		// Runs until the process is killed.
		log.Printf("[%s] per-builder worker pools (pool %d conns; per-builder concurrency from builder_config.max_concurrency, 0 = unbounded)", *id, poolConns)
		runBuilderPools(ctx, r, *id, *pollInterval)
		return
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

// runBuilderPools is the production runner loop: a supervisor that keeps one
// worker pool per CONFIGURED BUILDER that has leasable work. Each pool leases and
// executes only its own builder's tasks (across every build using that builder),
// up to that builder's own concurrency limit -- concurrency is a property of the
// hardware (the cluster/account/host-pool), not of a build. Builders never
// contend; builds on the same builder share its pool. New builders are picked up
// on the next poll; a pool exits once its builder's queue is drained and is
// restarted when work reappears. Never returns.
func runBuilderPools(ctx context.Context, r *runner.Runner, id string, pollInterval time.Duration) {
	q := db.New(r.Pool)
	type poolHandle struct{}
	var mu sync.Mutex
	pools := map[string]*poolHandle{}

	for {
		names, err := q.ListBuildersWithLeasableTasks(ctx)
		if err != nil {
			log.Printf("[%s] listing builders with work: %v", id, err)
			time.Sleep(pollInterval)
			continue
		}
		mu.Lock()
		for _, name := range names {
			if _, running := pools[name]; running {
				continue
			}
			h := &poolHandle{}
			pools[name] = h
			builderName := name
			go func() {
				runBuilderPool(ctx, r, id, builderName, pollInterval)
				mu.Lock()
				if pools[builderName] == h { // don't delete a newer pool for the same builder
					delete(pools, builderName)
				}
				mu.Unlock()
			}()
		}
		mu.Unlock()
		time.Sleep(pollInterval)
	}
}

// builderConcurrency is a configured builder's own concurrency limit
// (builder_config.max_concurrency). 0 = unbounded (the default -- e.g. AWS, which
// scales; the hoster and Postgres are the only real limits). A build with no
// configured_build (ad-hoc/fake, builder key "") or an unknown builder is
// unbounded too.
func builderConcurrency(ctx context.Context, q *db.Queries, builderConfigName string) int {
	if builderConfigName == "" {
		return 0
	}
	bc, err := q.GetBuilderConfigByName(ctx, builderConfigName)
	if err != nil {
		return 0
	}
	return int(bc.MaxConcurrency)
}

// runBuilderPool drains one configured builder's task queue, running up to the
// builder's concurrency limit (builder_config.max_concurrency; 0 = unbounded)
// in parallel. It leases only this builder's tasks, so it is fully independent
// of every other builder. Exits when the builder has no leasable tasks left AND
// nothing is in flight; the supervisor restarts it when work reappears. Exits
// promptly on context cancellation (shutdown).
//
// Two things keep the limit honest. A slot is taken BEFORE a task is leased:
// leasing first and then waiting for a slot left a leased task sitting with no
// heartbeat, its lease quietly expiring (and, with a second runner, open to
// being leased again and run twice). And the limit is re-read from the
// database every few seconds rather than once: a limit set or changed while a
// pool is alive -- during a competition, which is when it gets set -- applies
// to the next lease, not after the queue has drained. The limit is per runner
// process: run one laforge-runner per deployment.
func runBuilderPool(ctx context.Context, r *runner.Runner, id, builderConfigName string, pollInterval time.Duration) {
	q := db.New(r.Pool)
	limit := builderConcurrency(ctx, q, builderConfigName)
	limitRead := time.Now()
	log.Printf("[%s] builder %q: worker pool started, concurrency limit %s", id, builderConfigName, describeLimit(limit))
	var inFlight int64
	var wg sync.WaitGroup
	defer wg.Wait() // never leave tasks running past the pool
	lastSaturated := time.Time{}

	for {
		if ctx.Err() != nil {
			return
		}
		if time.Since(limitRead) > 10*time.Second {
			if fresh := builderConcurrency(ctx, q, builderConfigName); fresh != limit {
				log.Printf("[%s] builder %q: concurrency limit changed %s -> %s; applies to the next lease", id, builderConfigName, describeLimit(limit), describeLimit(fresh))
				limit = fresh
			}
			limitRead = time.Now()
		}
		busy := atomic.LoadInt64(&inFlight)
		if limit > 0 && busy >= int64(limit) {
			// Every slot is taken: don't lease anything yet. Say so every 30s
			// while it lasts, with how much is waiting, so a slow builder
			// reads as "at its limit", not as "doing nothing".
			if time.Since(lastSaturated) > 30*time.Second {
				waiting, _ := q.CountLeasableTasksForBuilder(ctx, builderConfigName)
				log.Printf("[%s] builder %q: at its concurrency limit (%d/%d tasks running, %d waiting to be leased)", id, builderConfigName, busy, limit, waiting)
				lastSaturated = time.Now()
			}
			if sleepCtx(ctx, pollInterval) {
				return
			}
			continue
		}
		task, err := r.LeaseOneForBuilder(ctx, builderConfigName)
		if err != nil {
			log.Printf("[%s] builder %q lease error: %v", id, builderConfigName, err)
			if sleepCtx(ctx, pollInterval) {
				return
			}
			continue
		}
		if task != nil {
			running := atomic.AddInt64(&inFlight, 1)
			log.Printf("LEASED %s kind=%s attempt=%d builder=%q slot=%d/%s", task.ID, task.Kind, task.Attempts, builderConfigName, running, describeLimit(limit))
			wg.Add(1)
			t := *task
			go func() {
				defer wg.Done()
				defer atomic.AddInt64(&inFlight, -1)
				if err := r.ExecuteAndRecord(ctx, t); err != nil {
					log.Printf("[%s] task %s error: %v", id, t.ID, err)
				}
			}()
			continue
		}
		// Nothing leased: the builder's queue is drained for now.
		pending, err := q.CountLeasableTasksForBuilder(ctx, builderConfigName)
		if err != nil {
			if sleepCtx(ctx, pollInterval) {
				return
			}
			continue
		}
		if pending == 0 && atomic.LoadInt64(&inFlight) == 0 {
			return // queue drained, nothing in flight -- this builder's pool is idle
		}
		if sleepCtx(ctx, pollInterval) {
			return
		}
	}
}

// sleepCtx sleeps for d, returning true if ctx was cancelled first (so callers
// can exit promptly on shutdown instead of after a full poll interval).
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-t.C:
		return false
	}
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

// describeLimit renders a concurrency limit for the log: "unbounded" for 0.
func describeLimit(limit int) string {
	if limit <= 0 {
		return "unbounded"
	}
	return fmt.Sprint(limit)
}
