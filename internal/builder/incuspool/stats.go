// Stats aggregates the pool's per-host Incus backend health into one reading.
// Each host's metrics already carry its server name as the Node, so the builder
// page shows them per-server (and can total/average across the pool). A host
// that can't be sampled becomes a warning, never a hard failure -- one down host
// shouldn't blank the rest. See docs/plans/builder-monitoring.md.
package incuspool

import (
	"context"
	"fmt"

	"github.com/globalcptc/laforge/internal/builder"
)

// Compile-time proof the pool publishes stats (the collector probes for this).
var _ builder.MetricProvider = (*Pool)(nil)

func (p *Pool) Stats(ctx context.Context) (builder.Stats, error) {
	var out builder.Stats
	for _, h := range p.Hosts {
		s, err := h.Stats(ctx)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("host unreachable: %v", err))
			continue
		}
		out.Metrics = append(out.Metrics, s.Metrics...)
		out.Warnings = append(out.Warnings, s.Warnings...)
	}
	return out, nil
}
