// Builder backend-health collection: on an interval, poll each configured
// builder's self-reported Stats (CPU/memory/disk/network/API-latency time-series
// and quota gauges) and store them for the builder page's charts. One gather per
// builder per cycle; a builder that doesn't implement builder.MetricProvider
// (OpenStack, the fake builder) is simply skipped. See
// docs/plans/builder-monitoring.md.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// CollectBuilderMetrics samples every configured builder that can report its
// stack's health and stores the readings. Best-effort per builder: a resolve or
// sample failure is logged and skipped, never fatal -- one unreachable hoster
// must not stop the rest. All metrics from one cycle share a timestamp so the
// charts line up.
func CollectBuilderMetrics(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	configs, err := q.ListBuilderConfigs(ctx)
	if err != nil {
		return fmt.Errorf("listing builder configs: %w", err)
	}
	sampledAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	for _, row := range configs {
		bldr, err := builderconfig.Resolve(pool, row)
		if err != nil {
			log.Printf("builder metrics: resolving %q: %v", row.Name, err)
			continue
		}
		mp, ok := bldr.(builder.MetricProvider)
		if !ok {
			continue // this builder doesn't publish stats
		}
		stats, err := mp.Stats(ctx)
		if err != nil {
			log.Printf("builder metrics: sampling %q: %v", row.Name, err)
			continue
		}
		for _, w := range stats.Warnings {
			log.Printf("builder metrics: %q: %s", row.Name, w)
		}
		for _, m := range stats.Metrics {
			if err := q.InsertBuilderMetricSample(ctx, db.InsertBuilderMetricSampleParams{
				BuilderConfigID: row.ID,
				Key:             m.Key,
				Label:           m.Label,
				Node:            m.Node,
				Kind:            m.Kind,
				Unit:            m.Unit,
				Value:           m.Value,
				Max:             m.Max,
				SampledAt:       sampledAt,
			}); err != nil {
				log.Printf("builder metrics: storing %q/%s: %v", row.Name, m.Key, err)
			}
		}
	}
	return nil
}

// CleanupBuilderMetricSamples drops samples older than cutoff -- the same
// time-bounded retention agent_heartbeat uses, so the table doesn't grow without
// bound. Returns how many rows were removed.
func CleanupBuilderMetricSamples(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	return db.New(pool).DeleteBuilderMetricSamplesOlderThan(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}
