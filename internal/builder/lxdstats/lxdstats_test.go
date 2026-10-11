package lxdstats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// fakeGet serves canned LXD metadata per path; a path mapped to "" errors.
func fakeGet(responses map[string]string) Getter {
	return func(_ context.Context, path string) (json.RawMessage, error) {
		body, ok := responses[path]
		if !ok || body == "" {
			return nil, fmt.Errorf("no response for %s", path)
		}
		return json.RawMessage(body), nil
	}
}

func metricByKey(stats builder.Stats, key string) (builder.Metric, bool) {
	for _, m := range stats.Metrics {
		if m.Key == key {
			return m, true
		}
	}
	return builder.Metric{}, false
}

// metricFor finds a metric by (key, node) -- a cluster has one per member.
func metricFor(stats builder.Stats, key, node string) (builder.Metric, bool) {
	for _, m := range stats.Metrics {
		if m.Key == key && m.Node == node {
			return m, true
		}
	}
	return builder.Metric{}, false
}

func TestHostClustered(t *testing.T) {
	// A 2-member cluster: metrics come from each member's state (sysinfo RAM +
	// load averages) and a per-member targeted storage-pool query.
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"node1","server_clustered":true}}`,
		"/1.0/cluster/members?recursion=1": `[{"server_name":"node1","status":"Online"},{"server_name":"node2","status":"Online"}]`,
		"/1.0/cluster/members/node1/state": `{"sysinfo":{"load_averages":[1.5,1.2,0.9],"total_ram":100,"free_ram":40,"total_swap":10,"free_swap":7,"processes":321}}`,
		"/1.0/cluster/members/node2/state": `{"sysinfo":{"load_averages":[0.5,0.4,0.3],"total_ram":200,"free_ram":150,"total_swap":0,"free_swap":0,"processes":100}}`,
		"/1.0/storage-pools/default/state?target=node1": `{"space":{"used":20,"total":40}}`,
		"/1.0/storage-pools/default/state?target=node2": `{"space":{"used":10,"total":40}}`,
		"/1.0/warnings?recursion=1":                     `[{"status":"new","severity":"low","location":"node1","last_message":"a thing"},{"status":"resolved"}]`,
		"/1.0/storage-pools?recursion=1":                `[{"name":"default","status":"Created"},{"name":"broken","status":"Errored"}]`,
	})
	stats, err := Host(context.Background(), get, nil, "default")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if _, ok := metricByKey(stats, "api_latency"); !ok {
		t.Error("missing api_latency metric")
	}
	// node1: mem 60%, load 1.5/1.2/0.9, disk 50%, swap (10-7)/10=30%, 321 procs.
	if m, ok := metricFor(stats, "mem", "node1"); !ok || m.Value != 60 {
		t.Errorf("node1 mem = %+v, want 60%%", m)
	}
	if m, ok := metricFor(stats, "cpu_load", "node1"); !ok || m.Value != 1.5 {
		t.Errorf("node1 cpu_load = %+v, want 1.5", m)
	}
	if m, ok := metricFor(stats, "cpu_load_5m", "node1"); !ok || m.Value != 1.2 {
		t.Errorf("node1 cpu_load_5m = %+v, want 1.2", m)
	}
	if m, ok := metricFor(stats, "cpu_load_15m", "node1"); !ok || m.Value != 0.9 {
		t.Errorf("node1 cpu_load_15m = %+v, want 0.9", m)
	}
	if m, ok := metricFor(stats, "swap", "node1"); !ok || m.Value != 30 {
		t.Errorf("node1 swap = %+v, want 30%%", m)
	}
	if m, ok := metricFor(stats, "processes", "node1"); !ok || m.Value != 321 {
		t.Errorf("node1 processes = %+v, want 321", m)
	}
	if m, ok := metricFor(stats, "disk", "node1"); !ok || m.Value != 50 {
		t.Errorf("node1 disk = %+v, want 50%%", m)
	}
	// node2 has no swap configured -> no swap metric.
	if _, ok := metricFor(stats, "swap", "node2"); ok {
		t.Error("node2 should report no swap metric (total_swap=0)")
	}
	// Cluster-wide: both members online; one unresolved warning surfaced + counted.
	if m, ok := metricByKey(stats, "members_online"); !ok || m.Value != 2 {
		t.Errorf("members_online = %+v, want 2", m)
	}
	if m, ok := metricByKey(stats, "lxd_warnings"); !ok || m.Value != 1 {
		t.Errorf("lxd_warnings = %+v, want 1 unresolved", m)
	}
	if !hasWarningContaining(stats, "broken") {
		t.Error("want a warning about the non-Created 'broken' storage pool")
	}
	if !hasWarningContaining(stats, "a thing") {
		t.Error("want the unresolved LXD warning message surfaced")
	}
}

func hasWarningContaining(stats builder.Stats, sub string) bool {
	for _, w := range stats.Warnings {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}

func TestHostStandaloneOmitsCPU(t *testing.T) {
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"solo","server_clustered":false}}`,
		"/1.0/resources":                   `{"memory":{"used":25,"total":100}}`,
		"/1.0/storage-pools/default/state": `{"space":{"used":10,"total":100}}`,
	})
	stats, err := Host(context.Background(), get, nil, "default")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if _, ok := metricByKey(stats, "cpu_load"); ok {
		t.Error("standalone host should not report cpu_load")
	}
	if m, ok := metricByKey(stats, "mem"); !ok || m.Value != 25 {
		t.Errorf("mem = %+v, want 25%%", m)
	}
}

func TestHostDegradesOnPartialFailure(t *testing.T) {
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"n","server_clustered":false}}`,
		"/1.0/storage-pools/default/state": `{"space":{"used":30,"total":60}}`,
	})
	stats, err := Host(context.Background(), get, nil, "default")
	if err != nil {
		t.Fatalf("Host should not hard-fail on a sub-metric error: %v", err)
	}
	if _, ok := metricByKey(stats, "mem"); ok {
		t.Error("mem should be omitted when /1.0/resources failed")
	}
	if _, ok := metricByKey(stats, "disk"); !ok {
		t.Error("disk should still be present")
	}
	if len(stats.Warnings) == 0 {
		t.Error("want a warning for the failed memory read")
	}
}

func TestHostHardFailsWhenAPIDown(t *testing.T) {
	if _, err := Host(context.Background(), fakeGet(map[string]string{}), nil, "default"); err == nil {
		t.Error("want an error when the API probe fails")
	}
}

// fakeGetRaw serves a canned /1.0/metrics body.
func fakeGetRaw(body string) RawGetter {
	return func(_ context.Context, _ string) ([]byte, error) { return []byte(body), nil }
}

func TestHostNetworkFromMetrics(t *testing.T) {
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"solo","server_clustered":false}}`,
		"/1.0/resources":                   `{"memory":{"used":25,"total":100}}`,
		"/1.0/storage-pools/default/state": `{"space":{"used":10,"total":100}}`,
	})
	metrics := `# HELP lxd_network_receive_bytes_total received
# TYPE lxd_network_receive_bytes_total counter
lxd_network_receive_bytes_total{device="eth0",name="c1",type="container"} 100
lxd_network_transmit_bytes_total{device="eth0",name="c1",type="container"} 50
lxd_network_receive_bytes_total{device="lo",name="c1",type="container"} 9999
lxd_network_receive_bytes_total{device="eth0",name="v2",type="virtual-machine"} 400
`
	stats, err := Host(context.Background(), get, fakeGetRaw(metrics), "default")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	// 100 + 50 + 400 = 550; loopback (9999) excluded.
	m, ok := metricFor(stats, "net", "solo")
	if !ok {
		t.Fatal("missing net metric")
	}
	if m.Kind != builder.MetricCounter || m.Value != 550 {
		t.Errorf("net = %+v, want counter value 550", m)
	}
}

func TestHostCPUFromMetrics(t *testing.T) {
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"solo","server_clustered":false}}`,
		"/1.0/resources":                   `{"memory":{"used":25,"total":100}}`,
		"/1.0/storage-pools/default/state": `{"space":{"used":10,"total":100}}`,
	})
	// A standalone host (no cluster member-state, so no load average) still gets a
	// CPU chart from the per-instance cpu-seconds counters; idle/iowait excluded.
	metrics := `lxd_cpu_seconds_total{name="a",mode="user"} 30
lxd_cpu_seconds_total{name="a",mode="system"} 12
lxd_cpu_seconds_total{name="a",mode="idle"} 9999
lxd_cpu_seconds_total{name="a",mode="iowait"} 50
lxd_cpu_seconds_total{name="b",mode="user"} 8`
	stats, err := Host(context.Background(), get, fakeGetRaw(metrics), "default")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	m, ok := metricFor(stats, "cpu", "solo")
	if !ok {
		t.Fatal("standalone host should get a CPU metric from /1.0/metrics")
	}
	// 30 + 12 + 8 = 50; idle (9999) and iowait (50) excluded.
	if m.Kind != builder.MetricCounter || m.Value != 50 {
		t.Errorf("cpu = %+v, want counter value 50", m)
	}
}

func TestSumOpenMetric(t *testing.T) {
	// Values after the label set, loopback skipped, a non-matching line ignored,
	// and an optional trailing timestamp tolerated.
	body := `lxd_network_receive_bytes_total{device="eth0",name="a"} 10 1700000000000
lxd_network_transmit_bytes_total{device="eth0",name="a"} 5
lxd_cpu_seconds_total{name="a"} 123
lxd_network_receive_bytes_total{device="lo",name="a"} 77`
	total, ok := sumOpenMetric(body, []string{"lxd_network_receive_bytes_total", "lxd_network_transmit_bytes_total"}, []string{`device="lo"`})
	if !ok || total != 15 {
		t.Errorf("sumOpenMetric(net) = %v, %v; want 15, true", total, ok)
	}
	if _, ok := sumOpenMetric("# no counters here\n", []string{"lxd_network_receive_bytes_total"}, nil); ok {
		t.Error("want ok=false when no matching lines present")
	}
}
