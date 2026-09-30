package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// ResolveBuilderFromDB is the real, database-backed BuilderResolver --
// the "what about the MicroCloud builder" gap,
// closed: task's own build -> configured_build -> builder_config (by
// name) -> internal/builderconfig.Resolve, so a build genuinely deploys
// against whatever cluster its own configured_build named, instead of
// every task in this process sharing one fixed builder regardless of
// which one it actually asked for. cmd/laforge-runner sets
// r.Builders = r.ResolveBuilderFromDB; every test that constructs a
// Runner directly and just sets Builder keeps working completely
// unchanged, since Builders stays nil for them (see Runner's own doc
// comment on both fields).
//
// Falls back to r.Builder when the build has no configured_build at all
// (db.Build.ConfiguredBuildID invalid -- an ad-hoc/manually-created
// build, e.g. every test's own newTestBuild-style helper), the same
// builder every build used before builder_config existed.
func (r *Runner) ResolveBuilderFromDB(ctx context.Context, task db.Task) (builder.Builder, error) {
	q := db.New(r.Pool)
	build, err := q.GetBuild(ctx, task.BuildID)
	if err != nil {
		return nil, fmt.Errorf("loading build: %w", err)
	}
	if !build.ConfiguredBuildID.Valid {
		return r.Builder, nil
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return nil, fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no such builder config %q (configured_build %s names it, but no builder_config row exists with that name)", cb.BuilderConfigName, cb.ID)
		}
		return nil, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	return builderconfig.Resolve(r.Pool, row)
}
