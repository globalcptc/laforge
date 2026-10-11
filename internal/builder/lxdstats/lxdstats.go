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
	addNetwork(ctx, getRaw, node, &stats)
	return stats, nil
}

// addNetwork reports total network throughput from the daemon's own
// /1.0/metrics (OpenMetrics), summing every instance's receive+transmit byte
// counters -- real traffic the LXD daemon itself measures on the host-side
// interfaces, not anything asked of the guests' agents. Reported as a COUNTER so
// the API turns the cumulative bytes into a per-second rate across samples.
// Best-effort: an unreachable/empty metrics endpoint warns and omits the chart.
func addNetwork(ctx context.Context, getRaw RawGetter, node string, stats *builder.Stats) {
	if getRaw == nil {
		return
	}
	raw, err := getRaw(ctx, "/1.0/metrics")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "network metrics: "+err.Error())
		return
	}
	total, ok := sumNetworkBytes(raw)
	if !ok {
		return // endpoint returned no network counters -- nothing to chart
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "net", Label: "Network", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: total,
	})
}

// sumNetworkBytes totals the lxd_network_{receive,transmit}_bytes_total counters
// across every instance in an OpenMetrics text body, excluding loopback. Returns
// ok=false when no such counter line is present. Parses by hand (the body is tiny
// and the format simple) to avoid a Prometheus-parser dependency.
func sumNetworkBytes(raw []byte) (total float64, ok bool) {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "lxd_network_receive_bytes_total") &&
			!strings.HasPrefix(line, "lxd_network_transmit_bytes_total") {
			continue
		}
		if strings.Contains(line, `device="lo"`) {
			continue // loopback isn't real network traffic
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

// clusterMetrics reports memory, CPU load and disk for EVERY cluster member, from
// each member's state (sysinfo carries total/free RAM and the load averages).
func clusterMetrics(ctx context.Context, get Getter, pool string, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/cluster/members?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "cluster members: "+err.Error())
		return
	}
	var members []struct {
		ServerName string `json:"server_name"`
	}
	if json.Unmarshal(raw, &members) != nil {
		return
	}
	for _, m := range members {
		node := m.ServerName
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
			} `json:"sysinfo"`
		}
		if json.Unmarshal(stateRaw, &st) != nil {
			continue
		}
		if st.SysInfo.TotalRAM > 0 {
			used := st.SysInfo.TotalRAM - st.SysInfo.FreeRAM
			stats.Metrics = append(stats.Metrics, memMetric(node, used, st.SysInfo.TotalRAM))
		}
		if len(st.SysInfo.LoadAverages) > 0 {
			stats.Metrics = append(stats.Metrics, builder.Metric{
				Key: "cpu_load", Label: "CPU load (1m)", Node: node,
				Kind: builder.MetricTimeseries, Unit: "load", Value: st.SysInfo.LoadAverages[0],
			})
		}
		// Per-member storage pool usage (targeted at that member).
		addDisk(ctx, get, pool, node, node, stats)
	}
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
