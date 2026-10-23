// laforge-orchestrator runs internal/orchestrator.Reconcile in a loop for
// every non-terminal build. "Orchestrators are stateless replicas, so a
// crash mid-build just means another picks up" -- this binary holds no
// state of its own beyond what's in Postgres, so more than one can run at
// once, and killing one at any point is safe: the next reconcile pass
// (from this process restarting, or another replica already running)
// picks up exactly where the desired-vs-observed diff says to.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/envfile"
	"github.com/globalcptc/laforge/internal/orchestrator"
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
	// see internal/envfile) so this and every other laforge-* service
	// can share one .env/environment, while an explicit -db-url still
	// overrides it -- flags win over env, same precedence every other
	// flag package gives an explicit argument over its own default.
	dbURL := flag.String("db-url", os.Getenv("DATABASE_URL"), "postgres connection string (required, or set DATABASE_URL)")
	// -repo is the fallback every build resolves against when no checkout
	// cache is configured (see checkoutCacheDir below) -- the original,
	// still-supported single-repository mode this flag has always meant.
	repoRoot := flag.String("repo", ".", "content checkout path to resolve builds against (fallback when no checkout cache is configured)")
	checkoutCacheDir := flag.String("checkout-cache-dir", filepath.Join(os.TempDir(), "laforge-checkouts"), "where fetched checkouts are cached, one per {repository, commit}, when GitHub credentials are configured (GITHUB_APP_ID/GITHUB_APP_PRIVATE_KEY_PATH or GITHUB_SERVICE_TOKEN) -- see internal/checkout")
	checkoutCacheRetention := flag.Duration("checkout-cache-retention", 24*time.Hour, "how long an unused cached checkout is kept before internal/checkout.Evict removes it; 0 disables cleanup")
	interval := flag.Duration("interval", 2*time.Second, "how often to reconcile every non-terminal build")
	reconcileConcurrency := flag.Int("reconcile-concurrency", envIntDefault("RECONCILE_CONCURRENCY", 8), "how many builds to reconcile in PARALLEL each pass. Reconciles are per-build independent, so a bounded worker pool keeps one slow build (a cold checkout, a hanging hoster API) from blocking the rest. Default 8; 1 restores the old serial behavior.")
	reconcileBuildTimeout := flag.Duration("reconcile-build-timeout", envDurationDefault("RECONCILE_BUILD_TIMEOUT", 2*time.Minute), "max time one build's reconcile may take (checkout + hoster calls) before its context is cancelled, so a hanging builder fails fast and the next pass retries instead of holding a worker. 0 disables.")
	plannedInterval := flag.Duration("planned-interval", envDurationDefault("PLANNED_RECONCILE_INTERVAL", 60*time.Second), "how often to reconcile a PLANNED build. A planned build touches no hoster and its content is immutable, so its pass only re-derives teams/fingerprints and the builder-compatibility verdict; once a minute tracks a builder config change without the checkout + content load every -interval that let 150 stale planned builds starve a live one. Live builds (deploying/building/tearing_down) reconcile every -interval and take priority for the -reconcile-concurrency slots; a planned build that is deployed moves to the live cadence within one -interval.")
	heartbeatRetention := flag.Duration("heartbeat-retention", 72*time.Hour, "how long to keep agent_heartbeat rows (an append-only log with no other retention policy -- see migrations/00007's own doc comment); 0 disables cleanup")
	scheduleInterval := flag.Duration("schedule-interval", 5*time.Second, "how often to check for due scheduled_task rows (schedule: entries and ad-hoc scheduled tasks)")
	accessInterval := flag.Duration("access-interval", 15*time.Second, "how often to enforce each live build's authored access: windows (open/close teams on schedule)")
	builderMetricsInterval := flag.Duration("builder-metrics-interval", 120*time.Second, "how often to sample each builder's backend health (CPU/mem/disk/net/API-latency/quotas) for the builder page; 0 disables it. 120s keeps hoster/cluster load low -- see docs/plans/builder-monitoring.md")
	checkinLinux := flag.Duration("agent-checkin-deadline-linux", envDurationDefault("AGENT_CHECKIN_DEADLINE_LINUX", orchestrator.DefaultCheckinWatchdog.Linux), "how long after deploy a Linux host's agent may take to send its first heartbeat before the watchdog flags it (event + needs-attention, console log captured); 0 disables")
	checkinWindows := flag.Duration("agent-checkin-deadline-windows", envDurationDefault("AGENT_CHECKIN_DEADLINE_WINDOWS", orchestrator.DefaultCheckinWatchdog.Windows), "the same deadline for a Windows host, whose first boot (sysprep, cloudbase-init) is far slower; 0 disables")
	checkinReboot := flag.Bool("agent-checkin-reboot", envBoolDefault("AGENT_CHECKIN_REBOOT", false), "force-reboot a host once when its agent misses the check-in deadline")
	once := flag.Bool("once", false, "reconcile every non-terminal build exactly once, then exit (for scripting/tests)")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("-db-url (or DATABASE_URL) is required")
	}

	ctx := context.Background()
	// Per-build reconcile lanes run one reconcile goroutine per active build, so
	// the pool must hold more than the default max(4, NumCPU): a lane briefly
	// holds a connection for its queries (and some hoster/access calls), and many
	// builds reconcile at once. Configurable; Postgres max_connections is the
	// ceiling, and lanes never hold a connection across the slow parts of a deploy
	// (those are the runner's tasks).
	q, pool, err := db.OpenWithMaxConns(ctx, *dbURL, int32(envIntDefault("ORCHESTRATOR_MAX_CONNS", 32)))
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	// resolveContent is the real fix, made
	// concrete for this process: with GitHub credentials configured, a
	// build resolves its OWN repository/commit through a real checkout
	// cache instead of every build sharing one fixed -repo path
	// regardless of which repository it actually belongs to. Without
	// credentials (a single-repo dev/compose setup, same as before this
	// existed), this degrades gracefully to the original fixed-path
	// behavior -- see checkout.FromEnv's own doc comment.
	cache, err := checkout.FromEnv(q, *checkoutCacheDir)
	if err != nil {
		log.Fatalf("configuring checkout cache: %v", err)
	}
	resolveContent := orchestrator.FixedRepoRoot(*repoRoot)
	if cache != nil {
		resolveContent = cache.ForBuild
		log.Printf("checkout cache enabled at %s -- builds resolve their own repository/commit", *checkoutCacheDir)
	}

	// A reconcile pass is serial over every non-terminal build, and each build
	// does real work inline -- a git checkout plus hoster API calls (network and
	// external access). A successful pass otherwise logs nothing, so a slow one
	// (a cold checkout, or a builder whose API is hanging/retrying) looks like
	// the orchestrator is idle when it is in fact blocked. These thresholds make
	// that time visible without drowning a healthy, sub-second pass in noise:
	// log a per-build phase breakdown only when a single build's pass is slow,
	// and a one-line pass summary only when the whole pass is slow.
	const (
		slowBuildThreshold = 3 * time.Second
		slowCycleThreshold = 10 * time.Second
	)
	// reconcileOne is the whole per-build pass, factored out of the loop so a
	// bounded worker pool can run several builds at once (see reconcileAll).
	// Each build gets its OWN context deadline: the pass does a git checkout and
	// several hoster API calls, and a single hanging/throttling builder must not
	// be able to hold a worker -- and, before the pool, the entire loop -- for
	// minutes. On a timeout the orchestrator calls return a context error, which
	// is logged and recorded like any other reconcile failure, and the next pass
	// simply retries. Reconciles are per-build independent (distinct build_ids,
	// distinct team/object rows) and the DB pool, the checkout cache, and log are
	// all concurrency-safe, so running them in parallel needs no other locking.
	// pctx is the caller's context: the process context for reconcileAll, or
	// the lane's own context for a lane pass (cancelled when the lane is
	// reaped, so a torn-down build's in-flight pass stops too).
	reconcileOne := func(pctx context.Context, b db.Build) {
		buildStart := time.Now()
		bctx := pctx
		if *reconcileBuildTimeout > 0 {
			var cancel context.CancelFunc
			bctx, cancel = context.WithTimeout(pctx, *reconcileBuildTimeout)
			defer cancel()
		}
		lastMark := buildStart
		var phases []string
		// While a build is still PREPARING -- it hasn't completed its first
		// resolve, so resolved_revision_id doesn't yet match its content revision
		// -- surface each pass phase's duration on the build timeline, so the
		// otherwise-silent startup gap shows where the time actually goes
		// (checkout vs content load vs topology resolve vs the access passes).
		// Gated to the preparing window so steady-state 2s passes don't spam it;
		// once the first resolve lands (build.resolved sets resolved_revision_id)
		// this stops. Each `mark` boundary becomes a build.phase event.
		preparing := !b.ResolvedRevisionID.Valid || b.ResolvedRevisionID.Bytes != b.ContentRevisionID.Bytes
		mark := func(label string) {
			now := time.Now()
			d := now.Sub(lastMark).Round(time.Millisecond)
			phases = append(phases, fmt.Sprintf("%s=%s", label, d))
			if preparing {
				q.CreateEvent(bctx, db.CreateEventParams{
					BuildID: b.ID, Kind: "build.phase",
					Message: fmt.Sprintf("%s took %s", label, d),
					Payload: []byte("{}"),
				})
			}
			lastMark = now
		}
		// Teardown deliberately
		// never touches content at all -- a build has to be
		// destroyable even once its repository access, branch, or
		// checkout is no longer resolvable, unlike Reconcile below,
		// which requires it.
		if b.Status == "tearing_down" {
			if err := orchestrator.Teardown(bctx, pool, b.ID); err != nil {
				log.Printf("tearing down build %s: %v", b.ID, err)
			}
			if d := time.Since(buildStart); d >= slowBuildThreshold {
				log.Printf("reconcile build %s (tearing_down): slow pass %s", b.ID, d.Round(time.Millisecond))
			}
			return
		}
		// Advance object lifecycle (agent check-in, steps, validators)
		// then the build phase before touching content -- like teardown,
		// neither depends on the checkout being resolvable, so a build
		// whose revision has since moved past the branch tip still settles
		// (running -> building -> finished/failed) instead of erroring here
		// forever. AdvanceBuildStatus returns settled=true only for a
		// TERMINAL build (finished/failed): a build still "building" keeps
		// reconciling so drift/fingerprint changes are caught live.
		if err := orchestrator.AdvanceObjectLifecycle(bctx, pool, b.ID); err != nil {
			log.Printf("advancing object lifecycle for build %s: %v", b.ID, err)
		}
		if settled, err := orchestrator.AdvanceBuildStatus(bctx, pool, b.ID); err != nil {
			log.Printf("advancing build status for build %s: %v", b.ID, err)
		} else if settled {
			return
		}
		mark("lifecycle")
		dir, err := resolveContent(bctx, b.ID)
		if err != nil {
			log.Printf("resolving content for build %s: %v", b.ID, err)
			// Surface the checkout failure ON THE BUILD, not just in the
			// orchestrator's stdout: a slow/failed/timed-out content fetch is the
			// common "the build just sits there with nothing in the logs" case,
			// and returning silently here (as this did) left the build with no
			// reason at all. Recorded on a detached context so a timed-out bctx
			// can still persist it, change-gated so it isn't rewritten every 2s;
			// a later successful pass clears it via the normal reconcile-error
			// path below.
			cerr := "content checkout failed: " + err.Error()
			if b.ReconcileError == nil || *b.ReconcileError != cerr {
				recCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if e := q.SetBuildReconcileError(recCtx, db.SetBuildReconcileErrorParams{ID: b.ID, ReconcileError: &cerr}); e != nil {
					log.Printf("recording checkout error for build %s: %v", b.ID, e)
				}
				cancel()
			}
			return
		}
		mark("checkout")
		recErr := ""
		// While preparing, split the reconcile phase into its internal steps
		// (content parse vs topology resolve + render) onto the timeline too.
		var onPhase func(string, time.Duration)
		if preparing {
			onPhase = func(label string, d time.Duration) {
				q.CreateEvent(bctx, db.CreateEventParams{BuildID: b.ID, Kind: "build.phase", Message: fmt.Sprintf("%s took %s", label, d.Round(time.Millisecond)), Payload: []byte("{}")})
			}
		}
		if err := orchestrator.Reconcile(bctx, pool, dir, b.ID, onPhase); err != nil {
			log.Printf("reconcile build %s: %v", b.ID, err)
			recErr = err.Error()
		}
		mark("reconcile")
		// Record the reconcile outcome on the build so the UI can show WHY a
		// build is stuck -- a build/builder incompatibility, say, which
		// isn't a content error and so can't be caught by `laforge check`.
		// Write only on a change to avoid an UPDATE every tick.
		prev := ""
		if b.ReconcileError != nil {
			prev = *b.ReconcileError
		}
		if recErr != prev {
			var val *string
			if recErr != "" {
				val = &recErr
			}
			// A detached context: when the pass timed out, bctx is already
			// dead, and the one thing that must still be written is WHY.
			recCtx, cancelRec := context.WithTimeout(context.Background(), 5*time.Second)
			if e := q.SetBuildReconcileError(recCtx, db.SetBuildReconcileErrorParams{ID: b.ID, ReconcileError: val}); e != nil {
				log.Printf("recording reconcile error for build %s: %v", b.ID, e)
			}
			if recErr != "" {
				msg := "reconcile pass failed: " + recErr
				if errors.Is(bctx.Err(), context.DeadlineExceeded) {
					msg = fmt.Sprintf("reconcile pass gave up after %s (RECONCILE_BUILD_TIMEOUT) in phase %q: %s -- the hoster calls in this phase are logged above; hosts in later teams were not reconciled this pass", reconcileBuildTimeout.Round(time.Second), lastPhase(phases), recErr)
				}
				q.CreateEvent(recCtx, db.CreateEventParams{BuildID: b.ID, Kind: "reconcile.error", Message: msg, Payload: []byte("{}")})
			} else {
				q.CreateEvent(recCtx, db.CreateEventParams{BuildID: b.ID, Kind: "reconcile.recovered", Message: "reconcile passes are completing again (was: " + prev + ")", Payload: []byte("{}")})
			}
			cancelRec()
		}
		// Enforce visible_from once a team's networks are up (peering + ACLs
		// via the builder). Deliberately after Reconcile, which drives the
		// networks to deployed in the first place.
		if err := orchestrator.ReconcileNetworkAccess(bctx, pool, b.ID); err != nil {
			log.Printf("reconcile network access for build %s: %v", b.ID, err)
		}
		mark("netaccess")
		// Realize content `public:` ports once a team's hosts are up (a
		// public IP or a port-NAT on a shared uplink IP, per builder).
		if err := orchestrator.ReconcileExternalAccess(bctx, pool, dir, b.ID); err != nil {
			log.Printf("reconcile external access for build %s: %v", b.ID, err)
		}
		mark("extaccess")
		// Name the slow build and which phase ate the time, so a stalled pass
		// points straight at the cause (a cold checkout, a hanging hoster API)
		// instead of looking like an idle process.
		if d := time.Since(buildStart); d >= slowBuildThreshold {
			log.Printf("reconcile build %s (%s): slow pass %s [%s]", b.ID, b.Status, d.Round(time.Millisecond), strings.Join(phases, " "))
		}
	}

	reconcileAll := func() {
		cycleStart := time.Now()
		builds, err := q.ListBuildsByStatus(ctx, []string{"planned", "deploying", "building", "tearing_down"})
		if err != nil {
			log.Printf("listing builds: %v", err)
			return
		}
		// Reconcile builds through a bounded worker pool instead of serially, so
		// one slow build no longer blocks every build behind it (the head-of-line
		// stall that made a new build sit for minutes). A buffered channel caps
		// the number in flight; each build still gets its own deadline inside
		// reconcileOne.
		limit := *reconcileConcurrency
		if limit < 1 {
			limit = 1
		}
		sem := make(chan struct{}, limit)
		var wg sync.WaitGroup
		for _, b := range builds {
			b := b
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				reconcileOne(ctx, b)
			}()
		}
		wg.Wait()
		// One line per slow pass (not per 2s tick) so the real reconcile cadence
		// -- the gap between a build's successive passes -- is visible when it
		// stretches far past the -interval the ticker nominally asks for.
		if d := time.Since(cycleStart); d >= slowCycleThreshold {
			log.Printf("reconcile pass: %d builds (%d in parallel) in %s", len(builds), limit, d.Round(time.Millisecond))
		}
	}

	// Per-build reconcile lanes: one goroutine per non-terminal build, each
	// reconciling ITS build on its own cadence. This replaces the shared-pass
	// model (reconcileAll) for steady-state running: there, a pass drained every
	// build before the next pass could start AND the build list was snapshotted at
	// pass start, so a brand-new build was not even discovered -- let alone have
	// its first deploy_network task created -- until the current pass (bounded by
	// the slowest build's timeout) finished. With lanes, a slow build only delays
	// its OWN next tick; a new build gets a lane within one supervisor poll and
	// reconciles immediately. reconcileAll is still used for -once.
	//
	// The lanes are BOUNDED (internal/orchestrator/lanes.go): every pass takes
	// one of -reconcile-concurrency slots from a priority semaphore -- live
	// builds (deploying/building/tearing_down) first, planned builds only on
	// the slots live ones leave free, and never more than half of them -- and a
	// planned build runs a pass only every -planned-interval. Unbounded, 153
	// stale planned builds each doing a checkout + content load every 2s put
	// the orchestrator above 300% CPU on a 2-core box and the one live build's
	// pass could no longer finish inside RECONCILE_BUILD_TIMEOUT: it restarted
	// from team 1 every pass. A lane also exits on its own as soon as it loads
	// its build in a terminal status, instead of reconciling a torn-down build
	// until the supervisor's next listing happens to omit it.
	lanes := orchestrator.NewLanes(orchestrator.LaneConfig{
		Concurrency: *reconcileConcurrency, Interval: *interval, PlannedInterval: *plannedInterval,
	}, q.GetBuild, reconcileOne)

	superviseReconcileLanes := func() {
		builds, err := q.ListBuildsByStatus(ctx, []string{"planned", "deploying", "building", "tearing_down"})
		if err != nil {
			log.Printf("listing builds: %v", err)
			return
		}
		lanes.Supervise(ctx, builds)
	}

	// Stale planned builds -- superseded by a newer planned build of the same
	// configured build, or orphaned by a removed configured build -- go
	// terminal ('discarded') so they stop holding a lane. The API keeps this
	// true as builds are created/removed; this sweep heals a backlog that
	// predates it and backstops those paths. Startup + the slow housekeeping
	// tick: a miss costs one planned-cadence pass, nothing more.
	discardStalePlannedBuilds := func(ctx context.Context) {
		n, err := orchestrator.DiscardStalePlannedBuilds(ctx, pool)
		if err != nil {
			log.Printf("discarding stale planned builds: %v", err)
			return
		}
		if n > 0 {
			log.Printf("discarded %d stale planned build(s) (superseded, or their configured build was removed)", n)
		}
	}

	// Checkout cache retention: same "time-bounded retention policy"
	// style as agent_heartbeat's own cleanupHeartbeats below -- an
	// unused cached checkout costs disk indefinitely otherwise. A no-op
	// on every tick when no cache is configured.
	cleanupCheckouts := func(ctx context.Context) {
		if cache == nil || *checkoutCacheRetention <= 0 {
			return
		}
		n, err := cache.Evict(time.Now().Add(-*checkoutCacheRetention))
		if err != nil {
			log.Printf("cleaning up checkout cache: %v", err)
			return
		}
		if n > 0 {
			log.Printf("cleaned up %d cached checkout(s) unused since before %s", n, *checkoutCacheRetention)
		}
	}

	// Heartbeat retention: "a time-bounded retention policy" -- the
	// simpler of the two options migrations/00007's own doc comment
	// named for agent_heartbeat's unbounded growth (the other, folding
	// into "purge artifacts," means folding into a build lifecycle
	// action that isn't built yet). Runs
	// on its own, much longer interval than reconcileAll's -interval:
	// deleting rows older than a multi-day retention window has no
	// reason to run every couple of seconds. Lives in the orchestrator
	// because it's the one service in this system with an existing real
	// background loop, not because heartbeat retention is conceptually
	// an orchestration concern -- there's no dedicated housekeeping
	// service, and standing one up for a single DELETE query would be
	// its own kind of overbuilt.
	cleanupHeartbeats := func(ctx context.Context) {
		if *heartbeatRetention <= 0 {
			return
		}
		cutoff := pgtype.Timestamptz{Time: time.Now().Add(-*heartbeatRetention), Valid: true}
		n, err := q.DeleteAgentHeartbeatsOlderThan(ctx, cutoff)
		if err != nil {
			log.Printf("cleaning up agent_heartbeat: %v", err)
			return
		}
		if n > 0 {
			log.Printf("cleaned up %d agent_heartbeat row(s) older than %s", n, *heartbeatRetention)
		}
	}

	// Repeating-event retention: the orchestrator's "still waiting" journal
	// entries say what a stuck object is waiting on; after a day the old ones
	// say only that it was. Drop them so a long-stuck build's Logs page stays
	// loadable. Everything else in the journal is kept.
	cleanupRepeatingEvents := func(ctx context.Context) {
		n, err := q.PruneRepeatingEvents(ctx, db.Interval(24*time.Hour))
		if err != nil {
			log.Printf("pruning repeating events: %v", err)
			return
		}
		if n > 0 {
			log.Printf("pruned %d repeating 'waiting' event(s) older than a day", n)
		}
	}

	// Session retention: expired session rows are otherwise never
	// deleted anywhere -- the exact same unbounded-growth shape already
	// fixed for agent_heartbeat, just missed for session. No retention window to
	// configure -- expires_at is already the boundary, set once at
	// CreateSession time -- so this always runs, same interval as
	// heartbeat cleanup.
	cleanupSessions := func(ctx context.Context) {
		n, err := q.DeleteExpiredSessions(ctx)
		if err != nil {
			log.Printf("cleaning up expired sessions: %v", err)
			return
		}
		if n > 0 {
			log.Printf("cleaned up %d expired session(s)", n)
		}
	}

	// Scheduled tasks: `schedule:` entries and ad-hoc scheduled tasks
	// -- a due row dispatches
	// through the exact same CreateAgentTask machinery the immediate
	// ad-hoc endpoint already uses, so this needs no new agent-side
	// protocol. Its own interval, independent of -interval: firing a
	// schedule entry a few seconds late is fine, and coupling it to
	// reconcileAll's own cadence would make one flag control two
	// unrelated things.
	dispatchScheduledTasks := func(ctx context.Context) {
		if err := orchestrator.DispatchDueScheduledTasks(ctx, pool, resolveContent, 100); err != nil {
			log.Printf("dispatching scheduled tasks: %v", err)
		}
	}

	// Instance power state: "is the instance actually up at the hoster,"
	// refreshed from Builder.Inspect on its own slow interval and tracked
	// apart from the deploy status and the agent's health. Needs no content
	// -- like teardown/AdvanceDeployStatus, it's a pure hoster query -- so a
	// build whose revision no longer resolves still gets its liveness
	// tracked. Only builds that can have live instances are polled.
	checkinWatchdog := orchestrator.DefaultCheckinWatchdog
	checkinWatchdog.Linux, checkinWatchdog.Windows, checkinWatchdog.Reboot = *checkinLinux, *checkinWindows, *checkinReboot
	pollInstanceStates := func(ctx context.Context) {
		builds, err := q.ListBuildsByStatus(ctx, []string{"deploying", "building", "finished", "failed"})
		if err != nil {
			log.Printf("listing builds for instance-state poll: %v", err)
			return
		}
		// Refresh live power states for every build in one pass that does
		// Inspect ONCE per builder, not once per build -- many builds share a
		// builder, and a builder's Inspect returns all of its resources, so the
		// old per-build loop re-scanned the same hoster N times each cycle.
		ids := make([]pgtype.UUID, len(builds))
		for i, b := range builds {
			ids[i] = b.ID
		}
		orchestrator.PollInstanceStatesForBuilds(ctx, pool, ids)
		for _, b := range builds {
			// Backstop the object/build lifecycle advance on this slow cadence
			// too: once a build leaves reconcileAll's list (it stops watching
			// planned/deploying/building once terminal), this is the loop that
			// still moves a lingering running host to building/finished from a
			// late agent check-in.
			if err := orchestrator.AdvanceObjectLifecycle(ctx, pool, b.ID); err != nil {
				log.Printf("advancing object lifecycle for build %s: %v", b.ID, err)
			}
			if _, err := orchestrator.AdvanceBuildStatus(ctx, pool, b.ID); err != nil {
				log.Printf("advancing build status for build %s: %v", b.ID, err)
			}
			// Flag hosts whose agent never checked in. On this ~30s cadence, not
			// every reconcile: deadlines are measured in minutes.
			if err := orchestrator.CheckAgentCheckins(ctx, pool, b.ID, checkinWatchdog); err != nil {
				log.Printf("check-in watchdog for build %s: %v", b.ID, err)
			}
		}
	}

	// collectBuilderMetrics samples every builder's own backend health on its own
	// slow cadence (one gather per builder, not per build). cleanupBuilderMetrics
	// ages the samples out on the same retention window as agent_heartbeat.
	collectBuilderMetrics := func(ctx context.Context) {
		if *builderMetricsInterval <= 0 {
			return
		}
		// Bounded: a probe of an unreachable hoster (a decommissioned test
		// cluster) otherwise waits the full operation timeout, minutes, and
		// holds the metrics loop with it.
		mctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := orchestrator.CollectBuilderMetrics(mctx, pool); err != nil {
			log.Printf("collecting builder metrics: %v", err)
		}
	}
	cleanupBuilderMetrics := func(ctx context.Context) {
		if *heartbeatRetention <= 0 {
			return
		}
		n, err := orchestrator.CleanupBuilderMetricSamples(ctx, pool, time.Now().Add(-*heartbeatRetention))
		if err != nil {
			log.Printf("cleaning up builder_metric_sample: %v", err)
			return
		}
		if n > 0 {
			log.Printf("cleaned up %d builder_metric_sample row(s) older than %s", n, *heartbeatRetention)
		}
	}

	// Access enforcement: turn each live build's authored `access:` windows
	// into real open/close actions (the windows were shown and anchored
	// against, but never enforced). Pure
	// Postgres (windows come from the persisted environment row), so it runs on
	// its own cadence for every build that can have live teams -- crucially
	// including `finished`, the competition state reconcileAll stops watching,
	// which is exactly when windows open and close. A near-future window edge
	// takes effect within one -access-interval.
	reconcileAccess := func(ctx context.Context) {
		builds, err := q.ListBuildsByStatus(ctx, []string{"deploying", "building", "finished"})
		if err != nil {
			log.Printf("listing builds for access enforcement: %v", err)
			return
		}
		now := time.Now().UTC()
		for _, b := range builds {
			if err := orchestrator.ReconcileAccess(ctx, pool, b.ID, now); err != nil {
				log.Printf("enforcing access for build %s: %v", b.ID, err)
			}
		}
	}

	// Per-builder docker base image builds: pick up any queued build and run
	// it (in its own goroutine -- a build takes minutes). Its own short
	// cadence, independent of reconcile: a build is queued when a builder is
	// added or the operator clicks Rebuild, and should start promptly.
	dispatchImageBuilds := func(ctx context.Context) {
		if err := orchestrator.RunPendingImageBuilds(ctx, pool); err != nil {
			log.Printf("dispatching image builds: %v", err)
		}
	}

	if *once {
		reconcileAll()
	} else {
		// No separate startup pass: a lane reconciles its build immediately
		// when started, so starting the lanes IS the startup pass -- bounded
		// by the same slot budget, instead of one unbounded reconcileAll over
		// every planned build before the ticker even begins.
		discardStalePlannedBuilds(ctx)
		superviseReconcileLanes()
	}
	// Every periodic job runs off the main loop, bounded, and a tick is skipped
	// while the previous run of that job is still going: one slow hoster (an
	// Inspect against a loaded cluster, a probe of a decommissioned one) used
	// to hold this single goroutine, and with it every other job's tick.
	jobs := &tickJobs{}
	jobs.run(ctx, "heartbeat-cleanup", 5*time.Minute, cleanupHeartbeats)
	jobs.run(ctx, "session-cleanup", 5*time.Minute, cleanupSessions)
	jobs.run(ctx, "event-cleanup", 5*time.Minute, cleanupRepeatingEvents)
	jobs.run(ctx, "schedules", 2*time.Minute, dispatchScheduledTasks)
	jobs.run(ctx, "access", 2*time.Minute, reconcileAccess)
	jobs.run(ctx, "instance-states", 3*time.Minute, pollInstanceStates)
	jobs.run(ctx, "image-builds", 0, dispatchImageBuilds) // spawns long-running builds: never bounded
	jobs.run(ctx, "builder-metrics", time.Minute, collectBuilderMetrics)
	jobs.run(ctx, "checkout-cleanup", 5*time.Minute, cleanupCheckouts)
	if *once {
		return
	}
	t := time.NewTicker(*interval)
	defer t.Stop()
	const heartbeatCleanupInterval = 10 * time.Minute
	heartbeatTicker := time.NewTicker(heartbeatCleanupInterval)
	defer heartbeatTicker.Stop()
	scheduleTicker := time.NewTicker(*scheduleInterval)
	defer scheduleTicker.Stop()
	accessTicker := time.NewTicker(*accessInterval)
	defer accessTicker.Stop()
	// Instance-state poll on its own ~30s cadence -- slow enough not to hammer
	// the hoster's API every reconcile, fresh enough to catch a host going
	// down within the same window the agent's own "late/missing" thresholds use.
	const instanceStatePollInterval = 30 * time.Second
	instanceStateTicker := time.NewTicker(instanceStatePollInterval)
	defer instanceStateTicker.Stop()
	// Same own-much-longer-interval reasoning as heartbeatTicker: an
	// unused checkout aging out has no reason to be checked every
	// reconcile pass.
	checkoutCleanupTicker := time.NewTicker(heartbeatCleanupInterval)
	defer checkoutCleanupTicker.Stop()
	// Image builds are queued on demand and rare; a brisk check makes a
	// newly-added builder start building its base image promptly.
	const imageBuildPollInterval = 10 * time.Second
	imageBuildTicker := time.NewTicker(imageBuildPollInterval)
	defer imageBuildTicker.Stop()
	// Builder health sampling on its own slow cadence (default 120s) so it never
	// rides the hoster API every reconcile. A zero interval disables it; use a
	// harmless 1h placeholder ticker in that case (the case body no-ops).
	builderMetricsEvery := *builderMetricsInterval
	if builderMetricsEvery <= 0 {
		builderMetricsEvery = time.Hour
	}
	builderMetricsTicker := time.NewTicker(builderMetricsEvery)
	defer builderMetricsTicker.Stop()
	for {
		select {
		case <-t.C:
			// Keep each active build's reconcile lane running (start new builds'
			// lanes, reap terminal ones). The lanes reconcile on their own cadence;
			// this just supervises their lifecycle, so one slow build never blocks
			// another's reconcile or a new build's first task.
			superviseReconcileLanes()
		case <-checkoutCleanupTicker.C:
			jobs.run(ctx, "checkout-cleanup", 5*time.Minute, cleanupCheckouts)
		case <-heartbeatTicker.C:
			jobs.run(ctx, "heartbeat-cleanup", 5*time.Minute, cleanupHeartbeats)
			jobs.run(ctx, "session-cleanup", 5*time.Minute, cleanupSessions)
			jobs.run(ctx, "event-cleanup", 5*time.Minute, cleanupRepeatingEvents)
			jobs.run(ctx, "metrics-cleanup", 5*time.Minute, cleanupBuilderMetrics)
			jobs.run(ctx, "stale-planned", 2*time.Minute, discardStalePlannedBuilds)
		case <-scheduleTicker.C:
			jobs.run(ctx, "schedules", 2*time.Minute, dispatchScheduledTasks)
		case <-accessTicker.C:
			jobs.run(ctx, "access", 2*time.Minute, reconcileAccess)
		case <-instanceStateTicker.C:
			jobs.run(ctx, "instance-states", 3*time.Minute, pollInstanceStates)
		case <-imageBuildTicker.C:
			jobs.run(ctx, "image-builds", 0, dispatchImageBuilds)
		case <-builderMetricsTicker.C:
			jobs.run(ctx, "builder-metrics", time.Minute, collectBuilderMetrics)
		}
	}
}

// envIntDefault reads an integer environment variable, falling back to def when
// unset or unparseable -- lets reconcile concurrency be set from the compose env
// (RECONCILE_CONCURRENCY) without a flag, the same way the runner reads
// RUNNER_CONCURRENCY.
func envIntDefault(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// envBoolDefault reads a boolean environment variable (strconv.ParseBool
// syntax), falling back to def when it is unset or unparseable.
func envBoolDefault(name string, def bool) bool {
	if v := os.Getenv(name); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// envDurationDefault reads a Go duration environment variable (e.g. "90s",
// "2m"), falling back to def when unset or unparseable -- the duration twin of
// envIntDefault, so RECONCILE_BUILD_TIMEOUT can be set from the compose env.
func envDurationDefault(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// lastPhase names the phase a pass was in when it stopped, from the marks
// collected so far ("reconcile=1m59s" -> "reconcile").
func lastPhase(phases []string) string {
	if len(phases) == 0 {
		return "start"
	}
	name, _, _ := strings.Cut(phases[len(phases)-1], "=")
	return name
}

// tickJobs runs the orchestrator's periodic jobs off the main loop. Each job
// runs in its own goroutine under a bounded context (0 = unbounded, for a
// dispatcher that only spawns work), at most one instance at a time: a tick
// that arrives while the job is still running is skipped and logged, with
// how long the job has been going, so a job that has stopped making progress
// is visible in the log instead of silently stalling every other job.
type tickJobs struct {
	mu      sync.Mutex
	running map[string]time.Time
}

func (j *tickJobs) run(ctx context.Context, name string, timeout time.Duration, fn func(ctx context.Context)) {
	j.mu.Lock()
	if j.running == nil {
		j.running = map[string]time.Time{}
	}
	if since, busy := j.running[name]; busy {
		j.mu.Unlock()
		log.Printf("job %q: skipping this tick; the previous run has been going for %s", name, time.Since(since).Round(time.Second))
		return
	}
	j.running[name] = time.Now()
	j.mu.Unlock()
	go func() {
		started := time.Now()
		jctx := ctx
		if timeout > 0 {
			var cancel context.CancelFunc
			jctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		defer func() {
			j.mu.Lock()
			delete(j.running, name)
			j.mu.Unlock()
			if d := time.Since(started); d > 30*time.Second {
				log.Printf("job %q: took %s%s", name, d.Round(time.Second), map[bool]string{true: " (hit its " + timeout.String() + " limit)", false: ""}[timeout > 0 && d >= timeout])
			}
		}()
		fn(jctx)
	}()
}
