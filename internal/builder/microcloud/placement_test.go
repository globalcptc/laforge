package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

const GiB = uint64(1 << 30)

type placementFixture struct {
	mu       sync.Mutex
	members  []placementMember
	stats    map[string]memberSysinfo
	project  map[string]string
	failures map[string]int
	reads    map[string]int
	targets  []string
	exists   bool
}

func newPlacementFixture(t *testing.T) (*placementFixture, *Client) {
	t.Helper()
	f := &placementFixture{
		members: []placementMember{
			{Name: "micro-a", Status: "Online", Architecture: "x86_64"},
			{Name: "micro-b", Status: "Online", Architecture: "x86_64"},
		},
		stats: map[string]memberSysinfo{
			"micro-a": {LoadAverages: []float64{0, 0, 0}, LogicalCPUs: 8, TotalRAM: 16 * GiB, FreeRAM: 16 * GiB},
			"micro-b": {LoadAverages: []float64{0, 0, 0}, LogicalCPUs: 8, TotalRAM: 16 * GiB, FreeRAM: 16 * GiB},
		},
		reads: make(map[string]int), failures: make(map[string]int),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Query().Get("project") != "competition" {
			t.Errorf("lost project scope: %s", r.URL)
		}
		f.reads[r.URL.Path]++
		fail := func(code int) {
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": code, "error": "fixture failure"})
		}
		if code := f.failures[r.URL.Path]; code != 0 {
			fail(code)
			return
		}
		var metadata interface{}
		switch {
		case r.URL.Path == "/1.0/cluster/members":
			if r.URL.Query().Get("recursion") != "1" {
				t.Error("missing recursion")
			}
			metadata = f.members
		case strings.HasPrefix(r.URL.Path, "/1.0/cluster/members/"):
			name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/1.0/cluster/members/"), "/state")
			metadata = map[string]interface{}{"sysinfo": f.stats[name]}
		case r.URL.Path == "/1.0/projects/competition":
			metadata = map[string]interface{}{"config": f.project}
		case r.URL.Path == "/1.0/instances" && r.Method == "POST":
			f.targets = append(f.targets, r.URL.Query().Get("target"))
			f.exists = true
			metadata = map[string]string{}
		case strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			if !f.exists {
				fail(404)
				return
			}
			metadata = map[string]string{"status": "Running", "location": "micro-b"}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			fail(500)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), Project: "competition"}
	t.Cleanup(func() { placements.Delete(client.BaseURL + "\x00" + client.Project) })
	return f, client
}

func TestHostStatsPlacement(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*placementFixture)
		want  string
	}{
		{"normalized CPU load", func(f *placementFixture) {
			a, b := f.stats["micro-a"], f.stats["micro-b"]
			a.LogicalCPUs, a.LoadAverages = 8, []float64{4, 4, 4}
			b.LogicalCPUs, b.LoadAverages = 32, []float64{8, 8, 8}
			f.stats["micro-a"], f.stats["micro-b"] = a, b
		}, "micro-b"},
		{"sustained load", func(f *placementFixture) {
			a := f.stats["micro-a"]
			a.LoadAverages = []float64{0, 7, 0}
			f.stats["micro-a"] = a
		}, "micro-b"},
		{"RAM pressure", func(f *placementFixture) {
			a := f.stats["micro-a"]
			a.FreeRAM = 4 * GiB
			f.stats["micro-a"] = a
		}, "micro-b"},
		{"offline member", func(f *placementFixture) { f.members[0].Status = "Offline" }, "micro-b"},
		{"manual member", func(f *placementFixture) { f.members[0].Config = map[string]string{"scheduler.instance": "manual"} }, "micro-b"},
		{"group-only member", func(f *placementFixture) { f.members[0].Config = map[string]string{"scheduler.instance": "group"} }, "micro-b"},
		{"malformed stats", func(f *placementFixture) { f.stats["micro-a"] = memberSysinfo{} }, "micro-b"},
		{"unreachable stats", func(f *placementFixture) { f.failures["/1.0/cluster/members/micro-a/state"] = 503 }, "micro-b"},
		{"project groups", func(f *placementFixture) {
			f.project = map[string]string{"restricted": "true", "restricted.cluster.target": "allow", "restricted.cluster.groups": " lab , other "}
			f.members[0].Groups = []string{"reserved"}
			f.members[1].Groups = []string{"lab"}
		}, "micro-b"},
		{"deterministic tie", func(f *placementFixture) {}, "micro-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, client := newPlacementFixture(t)
			tc.setup(f)
			b := New(client, Config{Images: map[string]ImageRef{"linux": {VM: true, Fingerprint: "image"}}, Sizes: map[string]SizeSpec{"small": {CPU: "2", Memory: "4GiB"}}})
			if _, err := b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "placement", OS: "linux", Size: "small"}); err != nil {
				t.Fatal(err)
			}
			if len(f.targets) != 1 || f.targets[0] != tc.want {
				t.Fatalf("targets = %v, want %s", f.targets, tc.want)
			}
			if tc.name == "offline member" || tc.name == "manual member" || tc.name == "group-only member" || tc.name == "project groups" {
				if f.reads["/1.0/cluster/members/micro-a/state"] != 0 {
					t.Fatal("sampled an ineligible member")
				}
			}
		})
	}
}

func TestPlacementConcurrentBuildersReserveCapacity(t *testing.T) {
	f, client := newPlacementFixture(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	counts := make(map[string]int)
	var releases []func()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Separate Builders and Clients like separate runner tasks.
			copyClient := *client
			member, release, err := New(&copyClient, Config{}).selectInstanceTarget(context.Background(), fmt.Sprintf("box-%d", i), SizeSpec{CPU: "2", Memory: "4GiB"})
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			counts[member]++
			releases = append(releases, release)
			mu.Unlock()
		}()
	}
	wg.Wait()
	for _, release := range releases {
		release()
		release()
	}
	if counts["micro-a"] != 4 || counts["micro-b"] != 4 {
		t.Fatalf("unbalanced reservations: %v", counts)
	}
	if f.reads["/1.0/cluster/members"] != 1 || f.reads["/1.0/cluster/members/micro-a/state"] != 1 {
		t.Fatalf("duplicate metric refreshes: %v", f.reads)
	}
	// A completed boot still consumes capacity until host stats catch up.
	_, _, err := New(client, Config{}).selectInstanceTarget(context.Background(), "overflow", SizeSpec{CPU: "2", Memory: "4GiB"})
	if !errors.Is(err, errPlacementCapacity) {
		t.Fatalf("overcommitted RAM: %v", err)
	}
	value, _ := placements.Load(client.BaseURL + "\x00" + client.Project)
	state := value.(*placementState)
	state.mu.Lock()
	for _, r := range state.reservations {
		r.expires = time.Now().Add(-time.Second)
	}
	state.mu.Unlock()
	_, release, err := New(client, Config{}).selectInstanceTarget(context.Background(), "after-cooldown", SizeSpec{CPU: "2", Memory: "4GiB"})
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestPlacementFallbackAndNoCapacity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		setup         func(*placementFixture)
		capacityError bool
	}{
		{"cluster API forbidden", func(f *placementFixture) { f.failures["/1.0/cluster/members"] = 403 }, false},
		{"cluster API unsupported", func(f *placementFixture) { f.failures["/1.0/cluster/members"] = 404 }, false},
		{"mixed architecture", func(f *placementFixture) { f.members[1].Architecture = "aarch64" }, false},
		{"restricted target", func(f *placementFixture) { f.project = map[string]string{"restricted": "true"} }, false},
		{"all offline", func(f *placementFixture) {
			for i := range f.members {
				f.members[i].Status = "Offline"
			}
		}, true},
		{"empty members", func(f *placementFixture) { f.members = nil }, true},
		{"no RAM", func(f *placementFixture) {
			for name, stats := range f.stats {
				stats.FreeRAM = GiB
				f.stats[name] = stats
			}
		}, true},
		{"no usable stats", func(f *placementFixture) { f.stats = nil }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, client := newPlacementFixture(t)
			tc.setup(f)
			member, release, err := New(client, Config{}).selectInstanceTarget(context.Background(), "box", SizeSpec{CPU: "2", Memory: "4GiB"})
			defer release()
			if member != "" || errors.Is(err, errPlacementCapacity) != tc.capacityError || (err != nil && !tc.capacityError) {
				t.Fatalf("member=%q err=%v", member, err)
			}
		})
	}
}

func TestPlacementPreservesExistingInstanceAndReadErrors(t *testing.T) {
	for _, code := range []int{0, 403, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			f, client := newPlacementFixture(t)
			f.exists = true
			f.failures["/1.0/instances/box"] = code
			member, release, err := New(client, Config{}).selectInstanceTarget(context.Background(), "box", SizeSpec{CPU: "2", Memory: "4GiB"})
			defer release()
			if (err != nil) != (code != 0) || member != "" || f.reads["/1.0/cluster/members"] != 0 {
				t.Fatalf("member=%q err=%v reads=%v", member, err, f.reads)
			}
		})
	}
}

func TestPlacementWaitCanBeCanceled(t *testing.T) {
	state := &placementState{refreshing: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := state.refresh(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestPlacementResourceUnits(t *testing.T) {
	for _, tc := range []struct {
		cpu, memory string
		want        uint64
	}{
		{"2", "4GiB", 4 * GiB}, {"1", "1.5GiB", 3 * GiB / 2}, {"2", "4GB", 4000000000}, {"1", "1048576", 1048576},
		{"0-3", "4GiB", 0}, {"2", "50%", 0}, {"2", "NaN", 0}, {"2", "+Inf", 0}, {"2", "-1GiB", 0}, {"2", "18446744073709551616", 0}, {"", "", 0},
	} {
		_, memory, ok := placementResources(SizeSpec{CPU: tc.cpu, Memory: tc.memory})
		if memory != tc.want || ok != (tc.want != 0) {
			t.Errorf("%s/%s: bytes=%d ok=%t", tc.cpu, tc.memory, memory, ok)
		}
	}
}
