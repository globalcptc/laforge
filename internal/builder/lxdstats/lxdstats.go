// Package lxdstats samples an LXD/Incus host's own backend health from its REST
// API -- shared by the incus and microcloud builders (both speak the same API)
// so the gathering lives in one place without coupling the two builders to each
// other. It reports memory and storage-pool utilization, CPU load (clustered
// hosts only), network throughput (from the daemon's own /1.0/metrics), and the
// API response time as builder.Metrics. See docs/plans/builder-monitoring.md.
package lxdstats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// Getter is the one JSON API call Host needs: a GET returning the response's
// already-unwrapped metadata (what both clients' get() return). A method value
// like client.get satisfies it.
type Getter func(ctx context.Context, path string) (json.RawMessage, error)

// RawGetter is a GET that returns the untouched response body, for /1.0/metrics
// (OpenMetrics text, not the JSON envelope). A method value like client.getRaw
// satisfies it.
type RawGetter func(ctx context.Context, path string) ([]byte, error)

// Host samples an LXD/Incus backend's capacity utilization + API latency. A
// standalone host reports its own memory/disk; a CLUSTER (MicroCloud, a
// clustered Incus) reports every member's memory, CPU load and disk, each tagged
// with its server name as the Node so the page reads per-server (and can
// total/average). Best-effort per metric: a reading that can't be fetched is
// dropped (with a warning), never failing the sample -- the self-describing
// contract just shows fewer charts. The one hard error is being unable to reach
// the API at all (the GET /1.0 probe).
func Host(ctx context.Context, get Getter, getRaw RawGetter, pool string) (builder.Stats, error) {
	var stats builder.Stats

	// API response time: time the cheap GET /1.0, which also tells us the server
	// name (the node label) and whether it's clustered.
	start := time.Now()
	raw, err := get(ctx, "/1.0")
	latency := float64(time.Since(start).Milliseconds())
	if err != nil {
		return stats, fmt.Errorf("probing /1.0: %w", err)
	}
	var env struct {
		Environment struct {
			ServerName      string `json:"server_name"`
			ServerClustered bool   `json:"server_clustered"`
		} `json:"environment"`
	}
	_ = json.Unmarshal(raw, &env)
	node := env.Environment.ServerName
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "api_latency", Label: "API response time", Node: node,
		Kind: builder.MetricTimeseries, Unit: "ms", Value: latency,
	})

	if env.Environment.ServerClustered {
		clusterMetrics(ctx, get, pool, &stats)
	} else {
		singleHostMetrics(ctx, get, pool, node, &stats)
	}
	addMetricsStats(ctx, getRaw, node, &stats)
	addWarnings(ctx, get, &stats)
	addStoragePoolHealth(ctx, get, &stats)
	return stats, nil
}

// addWarnings reports the count of unresolved LXD warnings as a trend line and
// surfaces each one as a stats warning, so an operator sees the daemon's own
// health notices (a degraded pool, a failed task) on the builder page. Works on
// a standalone host and a cluster alike. Best-effort.
func addWarnings(ctx context.Context, get Getter, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/warnings?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "warnings: "+err.Error())
		return
	}
	var warnings []struct {
		Status      string `json:"status"`
		Severity    string `json:"severity"`
		Location    string `json:"location"`
		LastMessage string `json:"last_message"`
	}
	if json.Unmarshal(raw, &warnings) != nil {
		return
	}
	unresolved := 0
	for _, w := range warnings {
		if strings.EqualFold(w.Status, "resolved") {
			continue
		}
		unresolved++
		if unresolved <= 10 { // don't flood the UI if a cluster is very unhappy
			loc := w.Location
			if loc == "" {
				loc = "cluster"
			}
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("LXD warning (%s/%s): %s", w.Severity, loc, w.LastMessage))
		}
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "lxd_warnings", Label: "Unresolved warnings", Kind: builder.MetricTimeseries, Unit: "count", Value: float64(unresolved),
	})
}

// addStoragePoolHealth surfaces any storage pool not in the "Created" state as a
// warning -- a pool stuck Pending/Errored means deploys to it will fail. Config
// status only (it doesn't probe Ceph/OVN dataplane health). Best-effort.
func addStoragePoolHealth(ctx context.Context, get Getter, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/storage-pools?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "storage pools: "+err.Error())
		return
	}
	var pools []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if json.Unmarshal(raw, &pools) != nil {
		return
	}
	for _, p := range pools {
		if p.Status != "" && p.Status != "Created" {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("storage pool %s is %s, not Created", p.Name, p.Status))
		}
	}
}

// addMetricsStats reads the daemon's OWN OpenMetrics endpoint (/1.0/metrics) once
// and derives host-level trend lines from the per-instance counters it exposes --
// crucially, these are available even on a STANDALONE (non-clustered) host that
// has no cluster member-state: total network throughput and total guest CPU
// usage. Both are real measurements the daemon makes (not the guest agent), and
// both are COUNTERs the API turns into per-second rates (bytes/s) / cores. This
// is how a standalone Incus-pool host still gets a CPU and a Network chart even
// though its load-average/swap/process sysinfo isn't exposed without clustering.
// Best-effort: an unreachable endpoint warns and omits these charts.
func addMetricsStats(ctx context.Context, getRaw RawGetter, node string, stats *builder.Stats) {
	if getRaw == nil {
		return
	}
	raw, err := getRaw(ctx, "/1.0/metrics")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "metrics endpoint: "+err.Error())
		return
	}
	text := string(raw)
	if v, ok := sumOpenMetric(text, []string{"lxd_network_receive_bytes_total", "lxd_network_transmit_bytes_total"}, []string{`device="lo"`}); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "net", Label: "Network", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: v,
		})
	}
	// Busy CPU-seconds across every instance (all modes except idle/iowait); rated
	// by the API this reads as cores-in-use -- a CPU chart even where there's no
	// host load average (a standalone host).
	if v, ok := sumOpenMetric(text, []string{"lxd_cpu_seconds_total"}, []string{`mode="idle"`, `mode="iowait"`}); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "cpu", Label: "CPU (guest cores)", Node: node, Kind: builder.MetricCounter, Unit: "cores", Value: v,
		})
	}
}

// sumOpenMetric totals the value of every OpenMetrics sample line whose name
// starts with any of prefixes and whose label set contains none of skip. Returns
// ok=false when no matching line is present. Hand-parsed (the body is small and
// the format simple) to avoid a Prometheus-parser dependency.
func sumOpenMetric(text string, prefixes, skip []string) (total float64, ok bool) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matched := false
		for _, p := range prefixes {
			if strings.HasPrefix(line, p) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		skipLine := false
		for _, s := range skip {
			if strings.Contains(line, s) {
				skipLine = true
				break
			}
		}
		if skipLine {
			continue
		}
		// The value follows the label set `{...}`; an optional trailing timestamp
		// is dropped by taking only the first token after it.
		rest := line
		if i := strings.LastIndex(line, "}"); i >= 0 {
			rest = line[i+1:]
		}
		rest = strings.TrimSpace(rest)
		if sp := strings.IndexAny(rest, " \t"); sp >= 0 {
			rest = rest[:sp]
		}
		v, err := strconv.ParseFloat(rest, 64)
		if err != nil {
			continue
		}
		total += v
		ok = true
	}
	return total, ok
}

// singleHostMetrics reports the one connected host's memory (/1.0/resources, the
// most accurate figure) and configured storage pool. A standalone host has no
// cluster-member state, so there's no CPU load to report.
func singleHostMetrics(ctx context.Context, get Getter, pool, node string, stats *builder.Stats) {
	if raw, err := get(ctx, "/1.0/resources"); err != nil {
		stats.Warnings = append(stats.Warnings, "memory: "+err.Error())
	} else {
		var r struct {
			Memory struct {
				Used  uint64 `json:"used"`
				Total uint64 `json:"total"`
			} `json:"memory"`
		}
		if json.Unmarshal(raw, &r) == nil && r.Memory.Total > 0 {
			stats.Metrics = append(stats.Metrics, memMetric(node, r.Memory.Used, r.Memory.Total))
		}
	}
	addDisk(ctx, get, pool, node, "", stats)
}

// clusterMetrics reports per-member health for EVERY cluster member from its
// state (sysinfo: load averages, RAM, swap, process count) plus the member's
// storage-pool usage, and a cluster-wide online-member count. Each member's
// readings are tagged with its server name so the page reads per-node.
func clusterMetrics(ctx context.Context, get Getter, pool string, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/cluster/members?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "cluster members: "+err.Error())
		return
	}
	var members []struct {
		ServerName string `json:"server_name"`
		Status     string `json:"status"`
	}
	if json.Unmarshal(raw, &members) != nil {
		return
	}
	online := 0
	for _, m := range members {
		node := m.ServerName
		if strings.EqualFold(m.Status, "online") {
			online++
		} else if m.Status != "" {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("cluster member %s is %s, not online", node, m.Status))
		}
		stateRaw, err := get(ctx, "/1.0/cluster/members/"+url.PathEscape(node)+"/state")
		if err != nil {
			stats.Warnings = append(stats.Warnings, "member "+node+": "+err.Error())
			continue
		}
		var st struct {
			SysInfo struct {
				LoadAverages []float64 `json:"load_averages"`
				TotalRAM     uint64    `json:"total_ram"`
				FreeRAM      uint64    `json:"free_ram"`
				TotalSwap    uint64    `json:"total_swap"`
				FreeSwap     uint64    `json:"free_swap"`
				Processes    int64     `json:"processes"`
			} `json:"sysinfo"`
		}
		if json.Unmarshal(stateRaw, &st) != nil {
			continue
		}
		si := st.SysInfo
		if si.TotalRAM > 0 {
			stats.Metrics = append(stats.Metrics, memMetric(node, si.TotalRAM-si.FreeRAM, si.TotalRAM))
		}
		if si.TotalSwap > 0 {
			stats.Metrics = append(stats.Metrics, builder.Metric{
				Key: "swap", Label: "Swap", Node: node, Kind: builder.MetricTimeseries, Unit: "%",
				Value: float64(si.TotalSwap-si.FreeSwap) / float64(si.TotalSwap) * 100,
			})
		}
		for _, l := range []struct {
			key, label string
			idx        int
		}{{"cpu_load", "CPU load (1m)", 0}, {"cpu_load_5m", "CPU load (5m)", 1}, {"cpu_load_15m", "CPU load (15m)", 2}} {
			if len(si.LoadAverages) > l.idx {
				stats.Metrics = append(stats.Metrics, builder.Metric{
					Key: l.key, Label: l.label, Node: node, Kind: builder.MetricTimeseries, Unit: "load", Value: si.LoadAverages[l.idx],
				})
			}
		}
		if si.Processes > 0 {
			stats.Metrics = append(stats.Metrics, builder.Metric{
				Key: "processes", Label: "Processes", Node: node, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(si.Processes),
			})
		}
		// Per-member storage pool usage (targeted at that member).
		addDisk(ctx, get, pool, node, node, stats)
	}
	// Cluster-wide: how many members are online (a drop below the member count
	// is a real outage, surfaced as both a trend line and a warning above).
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "members_online", Label: "Cluster members online", Kind: builder.MetricTimeseries, Unit: "count", Value: float64(online),
	})
}

func memMetric(node string, used, total uint64) builder.Metric {
	return builder.Metric{
		Key: "mem", Label: "Memory", Node: node, Kind: builder.MetricTimeseries, Unit: "%",
		Value: float64(used) / float64(total) * 100,
	}
}

// addDisk reports the configured storage pool's used/total as a disk % for node.
// target, when set, scopes the pool-state query to a specific cluster member
// (?target=); empty queries the connected host. Best-effort.
func addDisk(ctx context.Context, get Getter, pool, node, target string, stats *builder.Stats) {
	if pool == "" {
		return
	}
	path := "/1.0/storage-pools/" + url.PathEscape(pool) + "/state"
	if target != "" {
		path += "?target=" + url.QueryEscape(target)
	}
	raw, err := get(ctx, path)
	if err != nil {
		stats.Warnings = append(stats.Warnings, "disk pool "+pool+" ("+node+"): "+err.Error())
		return
	}
	var ps struct {
		Space struct {
			Used  uint64 `json:"used"`
			Total uint64 `json:"total"`
		} `json:"space"`
	}
	if json.Unmarshal(raw, &ps) == nil && ps.Space.Total > 0 {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "disk", Label: "Disk (" + pool + ")", Node: node, Kind: builder.MetricTimeseries, Unit: "%",
			Value: float64(ps.Space.Used) / float64(ps.Space.Total) * 100,
		})
	}
}
