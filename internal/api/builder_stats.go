package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/builder"
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
	writeJSON(w, http.StatusOK, groupBuilderStats(samples))
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
		i, ok := seriesIdx[k]
		if !ok {
			i = len(resp.Series)
			seriesIdx[k] = i
			resp.Series = append(resp.Series, metricSeries{Key: s.Key, Label: s.Label, Node: s.Node, Unit: s.Unit, Points: []metricPoint{}})
		}
		resp.Series[i].Points = append(resp.Series[i].Points, metricPoint{T: s.SampledAt.Time, V: s.Value})
	}
	return resp
}
