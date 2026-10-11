// Stats reports the Incus host's own backend health to the builder page via
// builder.MetricProvider, from the Incus REST API we already talk to (nothing is
// asked of the deployed guests). The gathering lives in internal/builder/lxdstats
// so the MicroCloud builder (same LXD API) reuses it. See
// docs/plans/builder-monitoring.md.
package incus

import (
	"context"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/lxdstats"
)

// Compile-time proof the builder publishes stats (the collector probes for this).
var _ builder.MetricProvider = (*Builder)(nil)

// Stats implements builder.MetricProvider for a single Incus host.
func (b *Builder) Stats(ctx context.Context) (builder.Stats, error) {
	return lxdstats.Host(ctx, b.Client.get, b.Config.storagePoolOrDefault())
}
