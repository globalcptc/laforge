package api

import (
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

func TestBuilderCapacity(t *testing.T) {
	t.Run("AWS: every quota is decoded, worst-first, with real used/limit", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "quota:vpcs", Label: "VPCs (region)", Kind: builder.MetricGauge, Unit: "count", Value: 2, Max: f(5), SampledAt: ts(100)},   // 40%
			{Key: "quota:eips", Label: "Elastic IPs (region)", Kind: builder.MetricGauge, Unit: "count", Value: 4, Max: f(5), SampledAt: ts(100)}, // 80%
			{Key: "api_latency", Label: "API response time", Kind: builder.MetricTimeseries, Unit: "ms", Value: 999, SampledAt: ts(100)},         // ignored (no ceiling)
		}
		got := builderCapacity(samples)
		if len(got) != 2 {
			t.Fatalf("capacity has %d signals, want 2 (latency ignored): %+v", len(got), got)
		}
		if got[0].Key != "quota:eips" || got[0].Pct != 80 || got[0].Value != 4 || got[0].Max == nil || *got[0].Max != 5 {
			t.Errorf("worst signal = %+v, want EIPs 4/5 (80%%)", got[0])
		}
		if got[1].Key != "quota:vpcs" || got[1].Pct != 40 {
			t.Errorf("second signal = %+v, want VPCs 40%%", got[1])
		}
	})

	t.Run("LXD: per-node percent-full collapses to one aggregate row per metric (cluster peak)", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			// Memory on three nodes -- the aggregate must be the PEAK (75), not an
			// average, so a near-full node isn't hidden. node3's newer sample wins.
			{Key: "mem", Label: "Memory", Node: "node1", Kind: builder.MetricTimeseries, Unit: "%", Value: 30, SampledAt: ts(100)},
			{Key: "mem", Label: "Memory", Node: "node2", Kind: builder.MetricTimeseries, Unit: "%", Value: 75, SampledAt: ts(100)},
			{Key: "mem", Label: "Memory", Node: "node3", Kind: builder.MetricTimeseries, Unit: "%", Value: 10, SampledAt: ts(100)},
			{Key: "mem", Label: "Memory", Node: "node3", Kind: builder.MetricTimeseries, Unit: "%", Value: 40, SampledAt: ts(220)}, // newer, wins over 10
			{Key: "disk", Label: "Disk (pool)", Node: "node2", Kind: builder.MetricTimeseries, Unit: "%", Value: 88, SampledAt: ts(100)},
			{Key: "cpu_load", Label: "CPU load (1m)", Node: "node1", Kind: builder.MetricTimeseries, Unit: "load", Value: 7, SampledAt: ts(220)}, // ignored (not %)
		}
		got := builderCapacity(samples)
		if len(got) != 2 {
			t.Fatalf("capacity has %d rows, want 2 aggregates (mem, disk; cpu_load ignored): %+v", len(got), got)
		}
		// disk 88 outranks mem's peak 75.
		if got[0].Key != "disk" || got[0].Pct != 88 {
			t.Errorf("worst row = %+v, want disk 88%%", got[0])
		}
		if got[1].Key != "mem" || got[1].Pct != 75 || got[1].Node != "" {
			t.Errorf("second row = %+v, want aggregate memory 75%% (cluster peak, no node)", got[1])
		}
	})

	t.Run("nothing capacity-bearing -> empty", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "api_latency", Kind: builder.MetricTimeseries, Unit: "ms", Value: 12, SampledAt: ts(100)},
			{Key: "net", Kind: builder.MetricCounter, Unit: "bytes/s", Value: 1000, SampledAt: ts(100)},
		}
		if got := builderCapacity(samples); len(got) != 0 {
			t.Errorf("capacity = %+v, want empty", got)
		}
	})
}
