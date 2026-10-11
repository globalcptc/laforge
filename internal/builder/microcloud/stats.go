// Stats reports the MicroCloud cluster's own backend health to the builder page
// via builder.MetricProvider. MicroCloud is LXD, so the gathering is the shared
// internal/builder/lxdstats.Host (same as the Incus builder) -- memory and
// storage-pool utilization, per-member CPU load (MicroCloud is always clustered,
// so load averages are available), and API latency, each tagged with its server
// name. See docs/plans/builder-monitoring.md.
package microcloud

import (
	"context"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/lxdstats"
)

// Compile-time proof the builder publishes stats (the collector probes for this).
var _ builder.MetricProvider = (*Builder)(nil)

func (b *Builder) Stats(ctx context.Context) (builder.Stats, error) {
	return lxdstats.Host(ctx, b.Client.get, b.Client.getRaw, b.Config.storagePoolOrDefault())
}
