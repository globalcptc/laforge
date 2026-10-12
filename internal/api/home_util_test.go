package api

import (
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

func TestBuilderUtilization(t *testing.T) {
	t.Run("AWS: the most-constrained quota drives it", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "quota:vpcs", Label: "VPCs (region)", Kind: builder.MetricGauge, Unit: "count", Value: 2, Max: f(5), SampledAt: ts(100)},   // 40%
			{Key: "quota:eips", Label: "Elastic IPs (region)", Kind: builder.MetricGauge, Unit: "count", Value: 4, Max: f(5), SampledAt: ts(100)}, // 80%
			{Key: "api_latency", Label: "API response time", Kind: builder.MetricTimeseries, Unit: "ms", Value: 999, SampledAt: ts(100)},         // ignored (no ceiling)
		}
		u := builderUtilization(samples)
		if u == nil || u.Pct != 80 {
			t.Fatalf("utilization = %+v, want 80%% (EIPs)", u)
		}
		if u.Driver != "Elastic IPs (region)" {
			t.Errorf("driver = %q, want the EIP quota", u.Driver)
		}
	})

	t.Run("LXD: the busiest node's memory/disk drives it; latest wins", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "mem", Label: "Memory", Node: "node1", Kind: builder.MetricTimeseries, Unit: "%", Value: 30, SampledAt: ts(100)},
			{Key: "disk", Label: "Disk (pool)", Node: "node2", Kind: builder.MetricTimeseries, Unit: "%", Value: 88, SampledAt: ts(100)},
			{Key: "mem", Label: "Memory", Node: "node1", Kind: builder.MetricTimeseries, Unit: "%", Value: 92, SampledAt: ts(220)}, // newer, wins over 30
			{Key: "cpu_load", Label: "CPU load (1m)", Node: "node1", Kind: builder.MetricTimeseries, Unit: "load", Value: 7, SampledAt: ts(220)}, // ignored (not %)
		}
		u := builderUtilization(samples)
		if u == nil || u.Pct != 92 {
			t.Fatalf("utilization = %+v, want 92%% (node1 memory)", u)
		}
		if u.Driver != "Memory on node1" {
			t.Errorf("driver = %q, want the node1 memory label", u.Driver)
		}
	})

	t.Run("nothing capacity-bearing -> nil (falls back to machine count)", func(t *testing.T) {
		samples := []db.ListBuilderMetricSamplesRow{
			{Key: "api_latency", Kind: builder.MetricTimeseries, Unit: "ms", Value: 12, SampledAt: ts(100)},
			{Key: "net", Kind: builder.MetricCounter, Unit: "bytes/s", Value: 1000, SampledAt: ts(100)},
		}
		if u := builderUtilization(samples); u != nil {
			t.Errorf("utilization = %+v, want nil", u)
		}
	})
}
