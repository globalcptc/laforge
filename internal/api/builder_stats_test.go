package api

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

func TestRollupResources(t *testing.T) {
	counts := []db.CountDeployedObjectsByBuilderRow{
		{Kind: "host", Status: "running", N: 3},
		{Kind: "host", Status: "deploying", N: 1},
		{Kind: "container", Status: "running", N: 2},
		{Kind: "network", Status: "running", N: 4},
	}
	out := rollupResources(counts)
	byKind := map[string]usageResource{}
	for _, r := range out {
		byKind[r.Kind] = r
	}
	if h := byKind["host"]; h.Running != 3 || h.Total != 4 {
		t.Errorf("host = %+v, want running 3 total 4", h)
	}
	if c := byKind["container"]; c.Running != 2 || c.Total != 2 {
		t.Errorf("container = %+v, want running 2 total 2", c)
	}
}

func ts(sec int) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Unix(int64(sec), 0), Valid: true}
}

func f(v float64) *float64 { return &v }

// TestGroupBuilderStatsCounterRate checks that a monotonic counter series is
// turned into per-second rates between consecutive samples.
func TestGroupBuilderStatsCounterRate(t *testing.T) {
	samples := []db.ListBuilderMetricSamplesRow{
		{Key: "net", Label: "Network", Node: "n1", Kind: builder.MetricCounter, Unit: "bytes/s", Value: 1000, SampledAt: ts(100)},
		{Key: "net", Label: "Network", Node: "n1", Kind: builder.MetricCounter, Unit: "bytes/s", Value: 1600, SampledAt: ts(160)},
		{Key: "net", Label: "Network", Node: "n1", Kind: builder.MetricCounter, Unit: "bytes/s", Value: 1300, SampledAt: ts(220)}, // reset -> 0
	}
	resp := groupBuilderStats(samples)
	if len(resp.Series) != 1 {
		t.Fatalf("series = %d, want 1", len(resp.Series))
	}
	pts := resp.Series[0].Points
	// Two intervals: (1600-1000)/60 = 10 bytes/s; (1300-1600) clamped to 0.
	if len(pts) != 2 {
		t.Fatalf("points = %d, want 2", len(pts))
	}
	if pts[0].V != 10 {
		t.Errorf("rate[0] = %v, want 10", pts[0].V)
	}
	if pts[1].V != 0 {
		t.Errorf("rate[1] = %v, want 0 (counter reset clamped)", pts[1].V)
	}
}

// TestGroupBuilderStats checks the fold: time-series accumulate points per
// (key,node) in order; gauges keep the latest (samples are ascending).
func TestGroupBuilderStats(t *testing.T) {
	samples := []db.ListBuilderMetricSamplesRow{
		{Key: "cpu_load", Label: "CPU load (1m)", Node: "node1", Kind: builder.MetricTimeseries, Unit: "load", Value: 1.0, SampledAt: ts(100)},
		{Key: "cpu_load", Label: "CPU load (1m)", Node: "node2", Kind: builder.MetricTimeseries, Unit: "load", Value: 0.5, SampledAt: ts(100)},
		{Key: "cpu_load", Label: "CPU load (1m)", Node: "node1", Kind: builder.MetricTimeseries, Unit: "load", Value: 2.0, SampledAt: ts(220)},
		{Key: "quota:vpcs", Label: "VPCs", Node: "", Kind: builder.MetricGauge, Unit: "count", Value: 3, Max: f(5), SampledAt: ts(100)},
		{Key: "quota:vpcs", Label: "VPCs", Node: "", Kind: builder.MetricGauge, Unit: "count", Value: 4, Max: f(5), SampledAt: ts(220)},
	}
	resp := groupBuilderStats(samples)

	// Two cpu_load series (node1, node2); node1 has two points in order.
	if len(resp.Series) != 2 {
		t.Fatalf("series = %d, want 2", len(resp.Series))
	}
	var node1 *metricSeries
	for i := range resp.Series {
		if resp.Series[i].Node == "node1" {
			node1 = &resp.Series[i]
		}
	}
	if node1 == nil || len(node1.Points) != 2 || node1.Points[0].V != 1.0 || node1.Points[1].V != 2.0 {
		t.Fatalf("node1 series = %+v, want points [1.0, 2.0] in order", node1)
	}

	// One VPC gauge, latest value (4) wins.
	if len(resp.Gauges) != 1 {
		t.Fatalf("gauges = %d, want 1", len(resp.Gauges))
	}
	if resp.Gauges[0].Value != 4 || resp.Gauges[0].Max == nil || *resp.Gauges[0].Max != 5 {
		t.Errorf("gauge = %+v, want value 4 max 5", resp.Gauges[0])
	}
}
