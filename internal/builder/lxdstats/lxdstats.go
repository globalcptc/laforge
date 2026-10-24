// Package lxdstats samples an LXD/Incus host's own backend health from its REST
// API -- shared by the incus and microcloud builders (both speak the same API)
// so the gathering lives in one place without coupling the two builders to each
// other. It reports memory and storage-pool utilization, host CPU load, host
// NIC throughput, guest CPU/IO-wait/disk/network throughput (from the daemon's
// own /1.0/metrics), in-flight instance operations, and the API response time
// as builder.Metrics. See docs/plans/builder-monitoring.md.
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
			ServerName string `json:"server_name"`
		} `json:"environment"`
	}
	_ = json.Unmarshal(raw, &env)
	node := env.Environment.ServerName
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "api_latency", Label: "API response time", Node: node,
		Kind: builder.MetricTimeseries, Unit: "ms", Value: latency,
	})

	// Prefer the per-member sysinfo (load averages, RAM, swap, process count) from
	// the cluster member-state endpoint. We ASK for it rather than gating on the
	// server_clustered flag: an Incus host run as a single-node "cluster" serves
	// member-state and its full sysinfo even though it isn't a multi-node cluster,
	// so this is how a standalone-looking Incus host still yields the same charts
	// MicroCloud does. Only when that endpoint genuinely isn't there do we fall
	// back to /1.0/resources (host memory alone).
	members, ok := memberStateMetrics(ctx, get, pool, node, &stats)
	if !ok {
		singleHostMetrics(ctx, get, pool, node, &stats)
	}
	// A host's own NICs are read through the connected daemon, so they are only
	// attributable when it is the only member (a standalone host or a cluster of
	// one) -- a multi-member cluster would report one member's NICs as its own.
	if members <= 1 {
		addHostNIC(ctx, get, node, &stats)
	}
	addMetricsStats(ctx, getRaw, node, &stats)
	addOperations(ctx, get, node, &stats)
	addWarnings(ctx, get, node, &stats)
	addStoragePoolHealth(ctx, get, pool, &stats)
	return stats, nil
}

// metricNames is name under both daemons' OpenMetrics prefixes: LXD exports
// lxd_*, Incus exports the same series as incus_*. Matching only lxd_* is why an
// Incus pool published no CPU, network or disk-I/O charts at all.
func metricNames(names ...string) []string {
	out := make([]string, 0, 2*len(names))
	for _, n := range names {
		out = append(out, "lxd_"+n, "incus_"+n)
	}
	return out
}

// addHostNIC reports the host's own uplink traffic -- every unmanaged physical
// NIC or bond, from /1.0/networks/<nic>/state counters -- as receive and
// transmit counters (the API rates them). Unlike the per-instance "net" series
// this includes the host's own traffic: image downloads, agent binaries, and
// every guest's internet traffic leaving the box, so a saturated uplink during
// a build shows up here. Best-effort.
func addHostNIC(ctx context.Context, get Getter, node string, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/networks?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "host networks: "+err.Error())
		return
	}
	var nets []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Managed bool   `json:"managed"`
	}
	if json.Unmarshal(raw, &nets) != nil {
		return
	}
	var rx, tx float64
	found := false
	for _, n := range nets {
		if n.Managed || (n.Type != "physical" && n.Type != "bond") {
			continue
		}
		stateRaw, err := get(ctx, "/1.0/networks/"+url.PathEscape(n.Name)+"/state")
		if err != nil {
			continue
		}
		var st struct {
			Counters struct {
				BytesReceived int64 `json:"bytes_received"`
				BytesSent     int64 `json:"bytes_sent"`
			} `json:"counters"`
		}
		if json.Unmarshal(stateRaw, &st) != nil {
			continue
		}
		rx += float64(st.Counters.BytesReceived)
		tx += float64(st.Counters.BytesSent)
		found = true
	}
	if !found {
		return
	}
	stats.Metrics = append(stats.Metrics,
		builder.Metric{Key: "host_net_rx", Label: "Host NIC receive", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: rx},
		builder.Metric{Key: "host_net_tx", Label: "Host NIC transmit", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: tx},
	)
}

// addOperations reports how many instance operations (create, copy, start,
// ...) the daemon is running right now -- the deploy contention itself, read
// straight from the daemon: twenty creates queued on one host read as twenty
// here, and a per-host deploy cap shows as a flat line at the cap. Best-effort.
func addOperations(ctx context.Context, get Getter, node string, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/operations?recursion=1")
	if err != nil {
		return
	}
	var groups map[string][]struct {
		Class      string              `json:"class"`
		StatusCode int                 `json:"status_code"`
		Resources  map[string][]string `json:"resources"`
	}
	if json.Unmarshal(raw, &groups) != nil {
		return
	}
	running := 0
	for _, ops := range groups {
		for _, op := range ops {
			// 103 Running / 105 Pending; only operations on instances count.
			if (op.StatusCode == 103 || op.StatusCode == 105) && op.Class == "task" && len(op.Resources["instances"]) > 0 {
				running++
			}
		}
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "instance_ops", Label: "Instance operations running", Node: node, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(running),
	})
}

// addWarnings reports the count of unresolved LXD warnings as a trend line and
// surfaces each one as a stats warning, so an operator sees the daemon's own
// health notices (a degraded pool, a failed task) on the builder page. Works on
// a standalone host and a cluster alike. Best-effort.
func addWarnings(ctx context.Context, get Getter, node string, stats *builder.Stats) {
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
		Key: "lxd_warnings", Label: "Unresolved warnings", Node: node, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(unresolved),
	})
}

// addStoragePoolHealth surfaces any storage pool not in the "Created" state as a
// warning -- a pool stuck Pending/Errored means deploys to it will fail. Config
// status only (it doesn't probe Ceph/OVN dataplane health). Best-effort.
//
// It also flags the deploy pool when its driver makes every instance a FULL copy
// of its image or snapshot rather than a copy-on-write clone: "dir" always, and
// "lvm" without a thin pool. That is the difference between a create taking
// seconds and every concurrent create competing to write a whole disk image.
func addStoragePoolHealth(ctx context.Context, get Getter, pool string, stats *builder.Stats) {
	raw, err := get(ctx, "/1.0/storage-pools?recursion=1")
	if err != nil {
		stats.Warnings = append(stats.Warnings, "storage pools: "+err.Error())
		return
	}
	var pools []struct {
		Name   string            `json:"name"`
		Status string            `json:"status"`
		Driver string            `json:"driver"`
		Config map[string]string `json:"config"`
	}
	if json.Unmarshal(raw, &pools) != nil {
		return
	}
	for _, p := range pools {
		if p.Status != "" && p.Status != "Created" {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("storage pool %s is %s, not Created", p.Name, p.Status))
		}
		if p.Name == pool && fullCopyDriver(p.Driver, p.Config) {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("storage pool %s uses the %s driver without copy-on-write: every instance is a full copy of its image, so concurrent deploys contend for disk -- prefer zfs, btrfs or lvm with a thin pool", p.Name, p.Driver))
		}
	}
}

// fullCopyDriver reports whether a storage pool driver creates instances by
// fully copying their source rather than cloning it.
func fullCopyDriver(driver string, config map[string]string) bool {
	switch driver {
	case "dir":
		return true
	case "lvm":
		return config["lvm.use_thinpool"] == "false"
	}
	return false
}

// addMetricsStats reads the daemon's OWN OpenMetrics endpoint (/1.0/metrics) once
// and derives host-level trend lines from the per-instance counters it exposes.
// These are the charts BOTH builders share, because /1.0/metrics is served the
// same way on a STANDALONE host and a cluster member -- so the Incus pool and
// MicroCloud show the identical set (CPU, network, disk I/O, process count),
// rather than MicroCloud alone getting the cluster-member-state sysinfo. Network,
// CPU and disk I/O are COUNTERs the API turns into per-second rates; process
// count is a point-in-time gauge. Best-effort: an unreachable endpoint warns and
// omits these charts.
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
	if v, ok := sumOpenMetric(text, metricNames("network_receive_bytes_total", "network_transmit_bytes_total"), []string{`device="lo"`}); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "net", Label: "Network", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: v,
		})
	}
	// Busy CPU-seconds across every instance (all modes except idle/iowait); rated
	// by the API this reads as cores-in-use -- a CPU chart even where there's no
	// host load average (a standalone host).
	if v, ok := sumOpenMetric(text, metricNames("cpu_seconds_total"), []string{`mode="idle"`, `mode="iowait"`}); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "cpu", Label: "CPU (guest cores)", Node: node, Kind: builder.MetricCounter, Unit: "cores", Value: v,
		})
	}
	// Guests' CPU time stalled on I/O, as cores: storage contention seen from
	// inside the instances that report it (VMs with the guest agent). The host's
	// own iowait isn't exposed by the API.
	if v, ok := sumOpenMetricWith(text, metricNames("cpu_seconds_total"), `mode="iowait"`); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "cpu_iowait", Label: "CPU I/O wait (guest cores)", Node: node, Kind: builder.MetricCounter, Unit: "cores", Value: v,
		})
	}
	// Disk throughput: read + written bytes across every instance's disks.
	if v, ok := sumOpenMetric(text, metricNames("disk_read_bytes_total", "disk_written_bytes_total"), nil); ok {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "disk_io", Label: "Disk I/O", Node: node, Kind: builder.MetricCounter, Unit: "bytes/s", Value: v,
		})
	}
	// NB: process count comes from the member-state sysinfo (host processes), not
	// from /1.0/metrics, so it isn't duplicated here.
}

// sumOpenMetricWith totals the lines matching prefixes that DO carry label.
func sumOpenMetricWith(text string, prefixes []string, label string) (total float64, ok bool) {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, label) {
			kept = append(kept, line)
		}
	}
	return sumOpenMetric(strings.Join(kept, "\n"), prefixes, nil)
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
// most accurate figure), its CPU load average and process count (the same
// resources response's "load" section, which Incus serves with the
// resources_load API extension -- a standalone host has no cluster-member
// sysinfo, so this is its only load figure; LXD omits it and the chart is
// simply absent), and its configured storage pool.
func singleHostMetrics(ctx context.Context, get Getter, pool, node string, stats *builder.Stats) {
	if raw, err := get(ctx, "/1.0/resources"); err != nil {
		stats.Warnings = append(stats.Warnings, "memory: "+err.Error())
	} else {
		var r struct {
			Memory struct {
				Used  uint64 `json:"used"`
				Total uint64 `json:"total"`
			} `json:"memory"`
			// Incus 7.5 serves these keys untagged ("Average1Min", "Processes"),
			// verified live against r1-scaleway.
			Load *struct {
				Processes   int64   `json:"Processes"`
				Average1Min float64 `json:"Average1Min"`
			} `json:"load"`
		}
		if json.Unmarshal(raw, &r) == nil {
			if r.Memory.Total > 0 {
				stats.Metrics = append(stats.Metrics, memMetric(node, r.Memory.Used, r.Memory.Total))
			}
			if r.Load != nil {
				stats.Metrics = append(stats.Metrics, builder.Metric{
					Key: "cpu_load", Label: "CPU load (1m)", Node: node, Kind: builder.MetricTimeseries, Unit: "load", Value: r.Load.Average1Min,
				})
				if r.Load.Processes > 0 {
					stats.Metrics = append(stats.Metrics, builder.Metric{
						Key: "processes", Label: "Processes", Node: node, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(r.Load.Processes),
					})
				}
			}
		}
	}
	addDisk(ctx, get, pool, node, "", stats)
}

// memberStateMetrics reports each member's sysinfo (load averages 1/5/15m, memory,
// swap, process count) and storage-pool usage from the cluster member-state
// endpoint, tagged per server. It returns false when that endpoint isn't served
// at all (a truly non-clustered daemon with no member-state) so the caller falls
// back to /1.0/resources; an Incus host run as a single-node cluster DOES serve
// it, which is how the Incus pool gets the same sysinfo charts MicroCloud does.
// Best-effort per member: a member that can't be read warns and is skipped.
//
// members is how many cluster members were listed (0 when not clustered).
func memberStateMetrics(ctx context.Context, get Getter, pool, connectedNode string, stats *builder.Stats) (members int, ok bool) {
	raw, err := get(ctx, "/1.0/cluster/members?recursion=1")
	if err != nil {
		return 0, false // not clustered at all -> caller uses /1.0/resources
	}
	var list []struct {
		ServerName string `json:"server_name"`
		Status     string `json:"status"`
	}
	if json.Unmarshal(raw, &list) != nil || len(list) == 0 {
		return 0, false
	}
	gotAny := false
	online := 0
	for _, m := range list {
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
		gotAny = true
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
		// 1-minute load average only (the 5m/15m averages were dropped as noise).
		if len(si.LoadAverages) > 0 {
			stats.Metrics = append(stats.Metrics, builder.Metric{
				Key: "cpu_load", Label: "CPU load (1m)", Node: node, Kind: builder.MetricTimeseries, Unit: "load", Value: si.LoadAverages[0],
			})
		}
		if si.Processes > 0 {
			stats.Metrics = append(stats.Metrics, builder.Metric{
				Key: "processes", Label: "Processes", Node: node, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(si.Processes),
			})
		}
		addDisk(ctx, get, pool, node, node, stats)
	}
	if !gotAny {
		return len(list), false // members listed but none served state -> fall back
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "members_online", Label: "Cluster members online", Node: connectedNode, Kind: builder.MetricTimeseries, Unit: "count", Value: float64(online),
	})
	return len(list), true
}

func memMetric(node string, used, total uint64) builder.Metric {
	return builder.Metric{
		Key: "mem", Label: "Memory", Node: node, Kind: builder.MetricTimeseries, Unit: "%",
		Value: float64(used) / float64(total) * 100,
	}
}

// addDisk reports the configured storage pool's used/total as a disk % for node,
// read from the pool's /resources endpoint (its {space,inodes} usage -- NOT
// /state, which storage pools don't have: that path 404s against the real API).
// target, when set, scopes the read to a specific cluster member (?target=). A
// genuinely non-clustered host rejects ?target outright, and some single-node
// setups serve member-state yet still return nothing for it, so when a targeted
// read fails OR reports no capacity we retry once WITHOUT the target rather than
// silently dropping the only disk signal. Best-effort, but a pool that truly
// can't be read warns (visible on the builder page), never vanishes without a
// trace.
func addDisk(ctx context.Context, get Getter, pool, node, target string, stats *builder.Stats) {
	if pool == "" {
		return
	}
	base := "/1.0/storage-pools/" + url.PathEscape(pool) + "/resources"
	used, total, err := poolSpace(ctx, get, scopeToMember(base, target))
	if (err != nil || total == 0) && target != "" {
		if u, t, e := poolSpace(ctx, get, base); e == nil && t > 0 {
			used, total, err = u, t, nil
		}
	}
	if err != nil {
		stats.Warnings = append(stats.Warnings, "disk pool "+pool+" ("+node+"): "+err.Error())
		return
	}
	if total == 0 {
		stats.Warnings = append(stats.Warnings, "disk pool "+pool+" ("+node+"): no capacity reported")
		return
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "disk", Label: "Disk (" + pool + ")", Node: node, Kind: builder.MetricTimeseries, Unit: "%",
		Value: float64(used) / float64(total) * 100,
	})
}

// scopeToMember adds ?target= to a path when a cluster member is given.
func scopeToMember(base, target string) string {
	if target == "" {
		return base
	}
	return base + "?target=" + url.QueryEscape(target)
}

// poolSpace reads a storage pool's used/total bytes from its /resources endpoint.
func poolSpace(ctx context.Context, get Getter, path string) (used, total uint64, err error) {
	raw, err := get(ctx, path)
	if err != nil {
		return 0, 0, err
	}
	var ps struct {
		Space struct {
			Used  uint64 `json:"used"`
			Total uint64 `json:"total"`
		} `json:"space"`
	}
	if err := json.Unmarshal(raw, &ps); err != nil {
		return 0, 0, err
	}
	return ps.Space.Used, ps.Space.Total, nil
}
