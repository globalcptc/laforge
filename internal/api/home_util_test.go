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

	t.Run("LXD: each pool/node percent-full is its own signal; latest wins", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "mem", Label: "Memory", Node: "node1", Kind: builder.MetricTimeseries, Unit: "%", Value: 30, SampledAt: ts(100)},
			{Key: "disk", Label: "Disk (pool)", Node: "node2", Kind: builder.MetricTimeseries, Unit: "%", Value: 88, SampledAt: ts(100)},
			{Key: "mem", Label: "Memory", Node: "node1", Kind: builder.MetricTimeseries, Unit: "%", Value: 92, SampledAt: ts(220)},             // newer, wins over 30
			{Key: "cpu_load", Label: "CPU load (1m)", Node: "node1", Kind: builder.MetricTimeseries, Unit: "load", Value: 7, SampledAt: ts(220)}, // ignored (not %)
		}
		got := builderCapacity(samples)
		if len(got) != 2 {
			t.Fatalf("capacity has %d signals, want 2 (cpu_load ignored): %+v", len(got), got)
		}
		if got[0].Key != "mem" || got[0].Node != "node1" || got[0].Pct != 92 || got[0].Max != nil {
			t.Errorf("worst signal = %+v, want node1 memory 92%% (percent-full, no max)", got[0])
		}
		if got[1].Key != "disk" || got[1].Node != "node2" || got[1].Pct != 88 {
			t.Errorf("second signal = %+v, want node2 disk 88%%", got[1])
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
