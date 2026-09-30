package orchestrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

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
	stateByRef := make(map[string]string, len(resources))
	for _, r := range resources {
		stateByRef[r.ExternalRef] = r.State
	}

	q := db.New(pool)
	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading deployed objects: %w", err)
	}
	for _, o := range objs {
		if o.Kind != "host" && o.Kind != "container" {
			continue // networks/DNS have no power state
		}
		if o.Status == "destroyed" || o.Status == "destroying" {
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
