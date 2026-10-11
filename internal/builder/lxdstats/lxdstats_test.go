package lxdstats

import (
	"context"
	"encoding/json"
	"fmt"
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
		"/1.0":                                          `{"environment":{"server_name":"node1","server_clustered":true}}`,
		"/1.0/cluster/members?recursion=1":              `[{"server_name":"node1"},{"server_name":"node2"}]`,
		"/1.0/cluster/members/node1/state":              `{"sysinfo":{"load_averages":[1.5,1.2,0.9],"total_ram":100,"free_ram":40}}`,
		"/1.0/cluster/members/node2/state":              `{"sysinfo":{"load_averages":[0.5,0.4,0.3],"total_ram":200,"free_ram":150}}`,
		"/1.0/storage-pools/default/state?target=node1": `{"space":{"used":20,"total":40}}`,
		"/1.0/storage-pools/default/state?target=node2": `{"space":{"used":10,"total":40}}`,
	})
	stats, err := Host(context.Background(), get, "default")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if _, ok := metricByKey(stats, "api_latency"); !ok {
		t.Error("missing api_latency metric")
	}
	// node1: mem (100-40)/100 = 60%, cpu 1.5, disk 20/40 = 50%.
	if m, ok := metricFor(stats, "mem", "node1"); !ok || m.Value != 60 {
		t.Errorf("node1 mem = %+v, want 60%%", m)
	}
	if m, ok := metricFor(stats, "cpu_load", "node1"); !ok || m.Value != 1.5 {
		t.Errorf("node1 cpu_load = %+v, want 1.5", m)
	}
	if m, ok := metricFor(stats, "disk", "node1"); !ok || m.Value != 50 {
		t.Errorf("node1 disk = %+v, want 50%%", m)
	}
	// node2: mem (200-150)/200 = 25%, cpu 0.5.
	if m, ok := metricFor(stats, "mem", "node2"); !ok || m.Value != 25 {
		t.Errorf("node2 mem = %+v, want 25%%", m)
	}
	if m, ok := metricFor(stats, "cpu_load", "node2"); !ok || m.Value != 0.5 {
		t.Errorf("node2 cpu_load = %+v, want 0.5", m)
	}
}

func TestHostStandaloneOmitsCPU(t *testing.T) {
	get := fakeGet(map[string]string{
		"/1.0":                             `{"environment":{"server_name":"solo","server_clustered":false}}`,
		"/1.0/resources":                   `{"memory":{"used":25,"total":100}}`,
		"/1.0/storage-pools/default/state": `{"space":{"used":10,"total":100}}`,
	})
	stats, err := Host(context.Background(), get, "default")
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
	stats, err := Host(context.Background(), get, "default")
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
	if _, err := Host(context.Background(), fakeGet(map[string]string{}), "default"); err == nil {
		t.Error("want an error when the API probe fails")
	}
}
