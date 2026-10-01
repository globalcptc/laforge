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
	"flag"
	"log"
	"os"
	"path/filepath"
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
	heartbeatRetention := flag.Duration("heartbeat-retention", 72*time.Hour, "how long to keep agent_heartbeat rows (an append-only log with no other retention policy -- see migrations/00007's own doc comment); 0 disables cleanup")
	scheduleInterval := flag.Duration("schedule-interval", 5*time.Second, "how often to check for due scheduled_task rows (schedule: entries and ad-hoc scheduled tasks)")
	accessInterval := flag.Duration("access-interval", 15*time.Second, "how often to enforce each live build's authored access: windows (open/close teams on schedule)")
	once := flag.Bool("once", false, "reconcile every non-terminal build exactly once, then exit (for scripting/tests)")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("-db-url (or DATABASE_URL) is required")
	}

	ctx := context.Background()
	q, pool, err := db.Open(ctx, *dbURL)
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

	reconcileAll := func() {
		builds, err := q.ListBuildsByStatus(ctx, []string{"planned", "deploying", "building", "tearing_down"})
		if err != nil {
			log.Printf("listing builds: %v", err)
			return
		}
		for _, b := range builds {
			// Teardown deliberately
			// never touches content at all -- a build has to be
			// destroyable even once its repository access, branch, or
			// checkout is no longer resolvable, unlike Reconcile below,
			// which requires it.
			if b.Status == "tearing_down" {
				if err := orchestrator.Teardown(ctx, pool, b.ID); err != nil {
					log.Printf("tearing down build %s: %v", b.ID, err)
				}
				continue
			}
			// Advance object lifecycle (agent check-in, steps, validators)
			// then the build phase before touching content -- like teardown,
			// neither depends on the checkout being resolvable, so a build
			// whose revision has since moved past the branch tip still settles
			// (running -> building -> finished/failed) instead of erroring here
			// forever. AdvanceBuildStatus returns settled=true only for a
			// TERMINAL build (finished/failed): a build still "building" keeps
			// reconciling so drift/fingerprint changes are caught live.
			if err := orchestrator.AdvanceObjectLifecycle(ctx, pool, b.ID); err != nil {
				log.Printf("advancing object lifecycle for build %s: %v", b.ID, err)
			}
			if settled, err := orchestrator.AdvanceBuildStatus(ctx, pool, b.ID); err != nil {
				log.Printf("advancing build status for build %s: %v", b.ID, err)
			} else if settled {
				continue
			}
			dir, err := resolveContent(ctx, b.ID)
			if err != nil {
				log.Printf("resolving content for build %s: %v", b.ID, err)
				continue
			}
			recErr := ""
			if err := orchestrator.Reconcile(ctx, pool, dir, b.ID); err != nil {
				log.Printf("reconcile build %s: %v", b.ID, err)
				recErr = err.Error()
			}
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
				if e := q.SetBuildReconcileError(ctx, db.SetBuildReconcileErrorParams{ID: b.ID, ReconcileError: val}); e != nil {
					log.Printf("recording reconcile error for build %s: %v", b.ID, e)
				}
			}
			// Enforce visible_from once a team's networks are up (peering + ACLs
			// via the builder). Deliberately after Reconcile, which drives the
			// networks to deployed in the first place.
			if err := orchestrator.ReconcileNetworkAccess(ctx, pool, b.ID); err != nil {
				log.Printf("reconcile network access for build %s: %v", b.ID, err)
			}
			// Realize content `public:` ports once a team's hosts are up (a
			// public IP or a port-NAT on a shared uplink IP, per builder).
			if err := orchestrator.ReconcileExternalAccess(ctx, pool, dir, b.ID); err != nil {
				log.Printf("reconcile external access for build %s: %v", b.ID, err)
			}
		}
	}

	// Checkout cache retention: same "time-bounded retention policy"
	// style as agent_heartbeat's own cleanupHeartbeats below -- an
	// unused cached checkout costs disk indefinitely otherwise. A no-op
	// on every tick when no cache is configured.
	cleanupCheckouts := func() {
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
	cleanupHeartbeats := func() {
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

	// Session retention: expired session rows are otherwise never
	// deleted anywhere -- the exact same unbounded-growth shape already
	// fixed for agent_heartbeat, just missed for session. No retention window to
	// configure -- expires_at is already the boundary, set once at
	// CreateSession time -- so this always runs, same interval as
	// heartbeat cleanup.
	cleanupSessions := func() {
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
	dispatchScheduledTasks := func() {
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
	pollInstanceStates := func() {
		builds, err := q.ListBuildsByStatus(ctx, []string{"deploying", "building", "finished", "failed"})
		if err != nil {
			log.Printf("listing builds for instance-state poll: %v", err)
			return
		}
		for _, b := range builds {
			if err := orchestrator.PollInstanceStates(ctx, pool, b.ID); err != nil {
				log.Printf("polling instance states for build %s: %v", b.ID, err)
			}
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
	reconcileAccess := func() {
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
	dispatchImageBuilds := func() {
		if err := orchestrator.RunPendingImageBuilds(ctx, pool); err != nil {
			log.Printf("dispatching image builds: %v", err)
		}
	}

	reconcileAll()
	cleanupHeartbeats()
	cleanupSessions()
	dispatchScheduledTasks()
	reconcileAccess()
	pollInstanceStates()
	dispatchImageBuilds()
	cleanupCheckouts()
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
	for {
		select {
		case <-t.C:
			reconcileAll()
		case <-checkoutCleanupTicker.C:
			cleanupCheckouts()
		case <-heartbeatTicker.C:
			cleanupHeartbeats()
			cleanupSessions()
		case <-scheduleTicker.C:
			dispatchScheduledTasks()
		case <-accessTicker.C:
			reconcileAccess()
		case <-instanceStateTicker.C:
			pollInstanceStates()
		case <-imageBuildTicker.C:
			dispatchImageBuilds()
		}
	}
}
