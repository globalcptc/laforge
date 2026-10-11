package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// Builder backend-health stats for the builder page: the stored samples
// (internal/orchestrator.CollectBuilderMetrics) grouped into time-series (a
// point list per metric per node, for the charts) and gauges (the latest
// value/limit per metric, for quota usage bars). Readable by any authenticated
// user, like the other builder-config GETs. See docs/plans/builder-monitoring.md.

type metricPoint struct {
	T time.Time `json:"t"`
	V float64   `json:"v"`
}

type metricSeries struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Node   string        `json:"node"`
	Unit   string        `json:"unit"`
	Points []metricPoint `json:"points"`
}

type metricGauge struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Node  string   `json:"node"`
	Unit  string   `json:"unit"`
	Value float64  `json:"value"`
	Max   *float64 `json:"max,omitempty"`
}

type builderStatsResponse struct {
	Series []metricSeries `json:"series"`
	Gauges []metricGauge  `json:"gauges"`
	// Warnings explain metrics a live read couldn't produce (e.g. a quota the
	// credential isn't allowed to read), so "no quotas showing" isn't silent.
	// Only populated on the live path -- stored samples carry no warnings.
	Warnings []string `json:"warnings,omitempty"`
}

func (s *Server) handleGetBuilderConfigStats(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authAndAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	row, err := s.Queries.GetBuilderConfigByName(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	window := 6 * time.Hour
	if q := r.URL.Query().Get("window"); q != "" {
		if d, perr := time.ParseDuration(q); perr == nil && d > 0 {
			window = d
		}
	}
	samples, err := s.Queries.ListBuilderMetricSamples(r.Context(), db.ListBuilderMetricSamplesParams{
		BuilderConfigID: row.ID,
		SampledAt:       pgtype.Timestamptz{Time: time.Now().Add(-window), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := groupBuilderStats(samples)
	// No stored history yet (a freshly-added builder, or one looked at before the
	// first 120s collection) -- seed the blocks from one live reading so a
	// builder's health charts always appear instead of a bare "no metrics yet".
	// Only on the empty case, so steady state still reads the collected history
	// and never hits the hoster on every page load.
	if len(resp.Series) == 0 && len(resp.Gauges) == 0 {
		if live, ok := s.liveBuilderStats(r.Context(), row); ok {
			resp = live
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// liveBuilderStats does a one-shot Stats() against the builder and shapes it like
// the stored-sample response: each time-series metric as a single-point series
// (so its chart renders at its current value), each gauge as a gauge. Returns
// ok=false when the builder publishes no stats (OpenStack, fake) or can't be
// reached -- the caller then falls back to the empty state.
func (s *Server) liveBuilderStats(ctx context.Context, row db.BuilderConfig) (builderStatsResponse, bool) {
	resp := builderStatsResponse{Series: []metricSeries{}, Gauges: []metricGauge{}}
	bldr, err := builderconfig.Resolve(s.Pool, row)
	if err != nil {
		return resp, false
	}
	mp, ok := bldr.(builder.MetricProvider)
	if !ok {
		return resp, false
	}
	stats, err := mp.Stats(ctx)
	if err != nil {
		return resp, false
	}
	now := time.Now()
	for _, m := range stats.Metrics {
		if m.Kind == builder.MetricGauge {
			max := m.Max
			resp.Gauges = append(resp.Gauges, metricGauge{Key: m.Key, Label: m.Label, Node: m.Node, Unit: m.Unit, Value: m.Value, Max: max})
			continue
		}
		// A counter's rate needs two samples; a single live reading has none yet,
		// so show the block at 0 rather than plotting the raw cumulative total.
		v := m.Value
		if m.Kind == builder.MetricCounter {
			v = 0
		}
		resp.Series = append(resp.Series, metricSeries{
			Key: m.Key, Label: m.Label, Node: m.Node, Unit: m.Unit,
			Points: []metricPoint{{T: now, V: v}},
		})
	}
	resp.Warnings = stats.Warnings
	// Warnings alone (e.g. every quota read denied) are still worth returning so
	// the page can say why nothing but api_latency showed up.
	return resp, len(resp.Series) > 0 || len(resp.Gauges) > 0 || len(resp.Warnings) > 0
}

// usageResource is a per-kind rollup (hosts/containers/networks): how many are
// up vs total deployed on this builder's active builds.
type usageResource struct {
	Kind    string `json:"kind"`
	Running int64  `json:"running"`
	Total   int64  `json:"total"`
}

type builderUsageResponse struct {
	ActiveBuilds []db.ListActiveBuildsByBuilderRow    `json:"active_builds"`
	Resources    []usageResource                      `json:"resources"`
	Public       []db.ListPublicEndpointsByBuilderRow `json:"public"`
}

// handleGetBuilderConfigUsage is the builder page's status panel + public list:
// the active builds on this builder, their resource counts per kind, and the
// public endpoints they expose. Readable by any authenticated user.
func (s *Server) handleGetBuilderConfigUsage(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authAndAdmin(r.Context(), r); err != nil {
		writeAuthError(w, err)
		return
	}
	name := r.PathValue("name")
	if _, err := s.Queries.GetBuilderConfigByName(r.Context(), name); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	builds, err := s.Queries.ListActiveBuildsByBuilder(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	counts, err := s.Queries.CountDeployedObjectsByBuilder(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	public, err := s.Queries.ListPublicEndpointsByBuilder(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, builderUsageResponse{
		ActiveBuilds: builds,
		Resources:    rollupResources(counts),
		Public:       public,
	})
}

// rollupResources folds the (kind,status,count) rows into per-kind running/total,
// in a stable kind order.
func rollupResources(counts []db.CountDeployedObjectsByBuilderRow) []usageResource {
	idx := map[string]int{}
	out := []usageResource{}
	for _, c := range counts {
		i, ok := idx[c.Kind]
		if !ok {
			i = len(out)
			idx[c.Kind] = i
			out = append(out, usageResource{Kind: c.Kind})
		}
		out[i].Total += c.N
		if c.Status == "running" {
			out[i].Running += c.N
		}
	}
	return out
}

// groupBuilderStats folds the flat, time-ordered sample rows into series (one
// per key+node, points in order) and gauges (latest per key+node -- samples are
// ascending, so the last write wins). Insertion order is preserved so the UI
// renders a stable layout.
func groupBuilderStats(samples []db.ListBuilderMetricSamplesRow) builderStatsResponse {
	type gk struct{ key, node string }
	resp := builderStatsResponse{Series: []metricSeries{}, Gauges: []metricGauge{}}
	seriesIdx := map[gk]int{}
	gaugeIdx := map[gk]int{}
	counter := map[gk]bool{} // series built from monotonic counters -> rate at the end
	for _, s := range samples {
		k := gk{s.Key, s.Node}
		if s.Kind == builder.MetricGauge {
			g := metricGauge{Key: s.Key, Label: s.Label, Node: s.Node, Unit: s.Unit, Value: s.Value, Max: s.Max}
			if i, ok := gaugeIdx[k]; ok {
				resp.Gauges[i] = g // ascending order -> last is latest
			} else {
				gaugeIdx[k] = len(resp.Gauges)
				resp.Gauges = append(resp.Gauges, g)
			}
			continue
		}
		if s.Kind == builder.MetricCounter {
			counter[k] = true
		}
		i, ok := seriesIdx[k]
		if !ok {
			i = len(resp.Series)
			seriesIdx[k] = i
			resp.Series = append(resp.Series, metricSeries{Key: s.Key, Label: s.Label, Node: s.Node, Unit: s.Unit, Points: []metricPoint{}})
		}
		resp.Series[i].Points = append(resp.Series[i].Points, metricPoint{T: s.SampledAt.Time, V: s.Value})
	}
	// A counter is stored raw; the chart wants a per-second rate, so replace each
	// counter series' cumulative points with deltas over their own interval.
	for k := range counter {
		i := seriesIdx[k]
		resp.Series[i].Points = ratePoints(resp.Series[i].Points)
	}
	return resp
}

// ratePoints turns a monotonic counter's cumulative points into per-second rates
// between consecutive samples. Fewer than two points yields none (no interval to
// rate over yet); a negative delta (the counter reset, e.g. a daemon restart) is
// clamped to 0 rather than drawn as a huge spike.
func ratePoints(pts []metricPoint) []metricPoint {
	if len(pts) < 2 {
		return []metricPoint{}
	}
	out := make([]metricPoint, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		dt := pts[i].T.Sub(pts[i-1].T).Seconds()
		if dt <= 0 {
			continue
		}
		d := pts[i].V - pts[i-1].V
		if d < 0 {
			d = 0
		}
		out = append(out, metricPoint{T: pts[i].T, V: d / dt})
	}
	return out
}
