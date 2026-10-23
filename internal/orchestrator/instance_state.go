package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// builderConfigNameForBuild returns the builder_config name a build deploys to,
// for grouping builds by builder (so a shared Inspect is done once per builder,
// not once per build).
func builderConfigNameForBuild(ctx context.Context, q *db.Queries, buildID pgtype.UUID) (string, error) {
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return "", fmt.Errorf("loading build: %w", err)
	}
	if !build.ConfiguredBuildID.Valid {
		return "", errors.New("this build has no configured_build, so there's no builder to resolve")
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return "", fmt.Errorf("loading configured build: %w", err)
	}
	return cb.BuilderConfigName, nil
}

// resolveBuilderForBuild resolves a build's own builder through the same
// build -> configured_build -> builder_config chain DetectDrift and
// PowerObjects use. Returns a clear error for a build with no configured
// build (nothing to resolve a builder against).
func resolveBuilderForBuild(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) (builder.Builder, error) {
	q := db.New(pool)
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("loading build: %w", err)
	}
	if !build.ConfiguredBuildID.Valid {
		return nil, errors.New("this build has no configured_build, so there's no builder to resolve")
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return nil, fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		return nil, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	bldr, err := builderconfig.Resolve(pool, row)
	if err != nil {
		return nil, fmt.Errorf("resolving builder %q: %w", cb.BuilderConfigName, err)
	}
	return bldr, nil
}

// PollInstanceStates refreshes every deployed host/container's live power
// state (running/stopped/other/missing) from the builder, so "is the
// instance up" is answered by the hoster itself -- tracked independently of
// the deploy status (which settles at finished/failed) and of the LaForge
// agent's own health (which can be down while the instance is fine, or
// vice versa). An object whose ref the hoster no longer lists is marked
// "missing": the VM is gone even if its last heartbeat hasn't aged out yet.
// Networks/DNS have no power state and are skipped, as are objects not yet
// (or no longer) meant to exist.
func PollInstanceStates(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) error {
	bldr, err := resolveBuilderForBuild(ctx, pool, buildID)
	if err != nil {
		return err
	}
	resources, err := bldr.Inspect(ctx)
	if err != nil {
		return fmt.Errorf("inspecting hoster: %w", err)
	}
	return applyInstanceStates(ctx, pool, buildID, inspectStateMap(resources))
}

// PollInstanceStatesForBuilds refreshes power states for many builds while doing
// Inspect ONCE per builder, not once per build: a builder's Inspect returns all
// of its managed resources (across every build on it), so for N builds sharing a
// builder it's one hoster scan, not N. Best-effort per build/builder -- a resolve
// or Inspect failure is logged and that builder's builds are skipped this pass.
//
// Builders are inspected in parallel, each within inspectTimeout: one slow or
// unreachable hoster (a decommissioned test cluster still configured) must
// not hold up every other builder's power states -- nor the orchestrator's
// main loop, which this runs on.
func PollInstanceStatesForBuilds(ctx context.Context, pool *pgxpool.Pool, buildIDs []pgtype.UUID) {
	q := db.New(pool)
	type cached struct {
		stateByRef map[string]string
		ok         bool
	}
	// Which builder each build is on, and one representative build per
	// builder to resolve it through.
	builderOf := map[pgtype.UUID]string{}
	firstBuild := map[string]pgtype.UUID{}
	for _, id := range buildIDs {
		name, err := builderConfigNameForBuild(ctx, q, id)
		if err != nil {
			continue // no builder to resolve (e.g. a planned build with no configured_build)
		}
		builderOf[id] = name
		if _, seen := firstBuild[name]; !seen {
			firstBuild[name] = id
		}
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	byBuilder := map[string]cached{}
	for name, id := range firstBuild {
		wg.Add(1)
		go func(name string, id pgtype.UUID) {
			defer wg.Done()
			ictx, cancel := context.WithTimeout(ctx, inspectTimeout)
			defer cancel()
			started := time.Now()
			c := inspectBuilder(ictx, pool, id, name)
			if d := time.Since(started); d > 10*time.Second {
				log.Printf("orchestrator: inspecting builder %q for power states took %s (ok=%v)", name, d.Round(time.Millisecond), c.ok)
			}
			mu.Lock()
			byBuilder[name] = c
			mu.Unlock()
		}(name, id)
	}
	wg.Wait()
	for _, id := range buildIDs {
		c, ok := byBuilder[builderOf[id]]
		if !ok || !c.ok {
			continue
		}
		if err := applyInstanceStates(ctx, pool, id, c.stateByRef); err != nil {
			log.Printf("recording power states for build %s: %v", id.String(), err)
		}
	}
}

// inspectTimeout bounds one builder's Inspect in the power-state poll.
const inspectTimeout = 60 * time.Second

func inspectBuilder(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, name string) (c struct {
	stateByRef map[string]string
	ok         bool
}) {
	bldr, err := resolveBuilderForBuild(ctx, pool, buildID)
	if err != nil {
		log.Printf("resolving builder %q: %v", name, err)
		return c
	}
	resources, err := bldr.Inspect(ctx)
	if err != nil {
		log.Printf("inspecting builder %q: %v", name, err)
		return c
	}
	c.stateByRef = inspectStateMap(resources)
	c.ok = true
	return c
}

func inspectStateMap(resources []builder.Resource) map[string]string {
	m := make(map[string]string, len(resources))
	for _, r := range resources {
		m[r.ExternalRef] = r.State
	}
	return m
}

// applyInstanceStates records each host/container's power state from a builder's
// already-fetched resource map (ref -> state). An object whose ref the hoster no
// longer lists is "missing".
func applyInstanceStates(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, stateByRef map[string]string) error {
	q := db.New(pool)
	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading deployed objects: %w", err)
	}
	for _, o := range objs {
		if o.Kind != "host" && o.Kind != "container" {
			continue // networks/DNS have no power state
		}
		if o.Status == "destroyed" || o.Status == "destroying" || o.Status == "destroy_queued" {
			continue // intentionally gone -- not a liveness concern
		}
		if o.ExternalRef == nil || *o.ExternalRef == "" {
			continue // never got far enough at the hoster to have a ref
		}
		state, present := stateByRef[*o.ExternalRef]
		switch {
		case !present:
			state = "missing" // the hoster no longer has it
		case state == "":
			state = builder.PowerStateOther // exists but the builder didn't report a state
		}
		if err := q.SetDeployedObjectPowerState(ctx, db.SetDeployedObjectPowerStateParams{ID: o.ID, PowerState: state}); err != nil {
			return fmt.Errorf("recording power state for %s: %w", o.ID.String(), err)
		}
	}
	return nil
}
