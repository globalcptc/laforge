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

func testAllocatedNetwork(name, project, address string) allocatedNetwork {
	return allocatedNetwork{Name: name, Project: project, Type: "ovn", Managed: true, Status: "Created", Config: map[string]string{
		"network": "UPLINK", "ipv4.address": "10.0.1.1/24", "ipv4.nat": "true", "ipv6.address": "none", "volatile.network.ipv4.address": address,
	}}
}

func networkReply(w http.ResponseWriter, metadata any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": metadata})
}

func networkFailure(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": code, "error": message})
}

func TestDeployNetworkValidatesAllocationOnEverySuccessPath(t *testing.T) {
	spec := builder.NetworkSpec{ExternalName: "allocation-test", CIDR: "10.0.1.0/24"}
	name := networkName(spec.DisplayName, spec.ExternalName)
	for _, mode := range []string{"create", "adopt", "already-exists"} {
		for _, duplicate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/duplicate=%v", mode, duplicate), func(t *testing.T) {
				ours := testAllocatedNetwork(name, "competition", "10.10.44.35")
				// The same name in another project identifies a distinct network.
				other := testAllocatedNetwork(name, "unrelated", "10.10.44.36")
				if duplicate {
					other.Config["volatile.network.ipv4.address"] = "10.10.44.35"
				}
				exists := mode == "adopt"
				posts := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch {
					case r.Method == "POST":
						posts++
						var posted allocatedNetwork
						if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
							t.Error(err)
						}
						if posted.Config["network"] != "UPLINK" || posted.Config["ipv4.address"] != "10.0.1.1/24" {
							t.Errorf("bad create: %+v", posted)
						}
						exists = true
						if mode == "already-exists" {
							networkFailure(w, 409, "already exists")
							return
						}
						networkReply(w, nil)
					case r.URL.Path == "/1.0/networks/"+name:
						if r.URL.Query().Get("project") != "competition" {
							t.Error("exact GET lost project")
						}
						if !exists {
							networkFailure(w, 404, "not found")
							return
						}
						networkReply(w, ours)
					case r.URL.Path == "/1.0/networks":
						if r.URL.Query().Get("project") != "" || r.URL.Query().Get("all-projects") != "true" || r.URL.Query().Get("recursion") != "1" {
							t.Errorf("bad allocation query: %s", r.URL)
						}
						networkReply(w, []allocatedNetwork{ours, other})
					case r.URL.Path == "/1.0/projects/competition":
						networkReply(w, map[string]any{"config": map[string]string{"features.networks": "true"}})
					default:
						t.Errorf("unexpected request: %s", r.URL)
						networkFailure(w, 500, "unexpected")
					}
				}))
				defer srv.Close()
				b := New(&Client{BaseURL: srv.URL, Project: "competition", HTTPClient: srv.Client()}, Config{OVNUplinkNetwork: "UPLINK"})
				ref, err := b.DeployNetwork(context.Background(), spec)
				if ref != name {
					t.Fatalf("lost cleanup reference: %q", ref)
				}
				if duplicate {
					if !errors.Is(err, builder.ErrPermanentConfig) {
						t.Fatalf("duplicate accepted or retried: %v", err)
					}
					for _, detail := range []string{"competition/" + name, "unrelated/" + name, "UPLINK", "10.10.44.35"} {
						if !strings.Contains(err.Error(), detail) {
							t.Errorf("missing diagnostic %q: %v", detail, err)
						}
					}
					// Retrying the created object must still fail instead of blindly adopting it.
					if _, err := b.DeployNetwork(context.Background(), spec); !errors.Is(err, builder.ErrPermanentConfig) {
						t.Fatalf("retry adopted duplicate: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				wantPosts := 1
				if mode == "adopt" {
					wantPosts = 0
				}
				if posts != wantPosts {
					t.Errorf("posts=%d, want %d", posts, wantPosts)
				}
			})
		}
	}
}

func TestDeployNetworkRejectsUnverifiableReadback(t *testing.T) {
	spec := builder.NetworkSpec{ExternalName: "bad-readback", CIDR: "10.0.1.0/24"}
	name := networkName("", spec.ExternalName)
	for _, failure := range []string{"missing", "pending", "errored", "empty-status", "unmanaged", "wrong-type", "wrong-uplink", "wrong-subnet", "no-nat", "no-address", "invalid-address", "ipv6-address", "no-project", "list-error", "malformed-list", "missing-from-list", "changed-list", "hidden-config"} {
		t.Run(failure, func(t *testing.T) {
			ours := testAllocatedNetwork(name, "default", "10.10.44.35")
			switch failure {
			case "pending":
				ours.Status = "Pending"
			case "errored":
				ours.Status = "Errored"
			case "empty-status":
				ours.Status = ""
			case "unmanaged":
				ours.Managed = false
			case "wrong-type":
				ours.Type = "bridge"
			case "wrong-uplink":
				ours.Config["network"] = "OTHER"
			case "wrong-subnet":
				ours.Config["ipv4.address"] = "10.0.2.1/24"
			case "no-nat":
				ours.Config["ipv4.nat"] = "false"
			case "no-address":
				delete(ours.Config, "volatile.network.ipv4.address")
			case "invalid-address":
				ours.Config["volatile.network.ipv4.address"] = "auto"
			case "ipv6-address":
				ours.Config["volatile.network.ipv4.address"] = "2001:db8::1"
			case "no-project":
				ours.Project = ""
			}
			posted := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "POST":
					posted = true
					networkReply(w, nil)
				case r.URL.Path == "/1.0/networks/"+name:
					if !posted || failure == "missing" {
						networkFailure(w, 404, "not found")
						return
					}
					networkReply(w, ours)
				case r.URL.Path == "/1.0/networks":
					switch failure {
					case "list-error":
						networkFailure(w, 503, "unavailable")
					case "malformed-list":
						networkReply(w, map[string]string{"bad": "shape"})
					case "missing-from-list":
						networkReply(w, []allocatedNetwork{})
					case "changed-list":
						changed := testAllocatedNetwork(name, "default", "10.10.44.99")
						networkReply(w, []allocatedNetwork{changed})
					case "hidden-config":
						networkReply(w, []allocatedNetwork{ours, {Name: "hidden", Project: "other", Type: "ovn"}})
					default:
						networkReply(w, []allocatedNetwork{ours})
					}
				default:
					networkFailure(w, 500, "unexpected")
				}
			}))
			defer srv.Close()
			b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{OVNUplinkNetwork: "UPLINK"})
			if ref, err := b.DeployNetwork(context.Background(), spec); err == nil || ref != name {
				t.Fatalf("ref=%q err=%v", ref, err)
			}
		})
	}
}

func TestNetworkAllocationScopesUplinkAndEffectiveProject(t *testing.T) {
	ours := testAllocatedNetwork("inherited", "competition", "10.10.44.35")
	inherited := testAllocatedNetwork("inherited", "default", "10.10.44.35")
	differentUplink := testAllocatedNetwork("unrelated", "other", "10.10.44.35")
	differentUplink.Config["network"] = "ISOLATED"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1.0/projects/competition" {
			networkReply(w, map[string]any{"config": map[string]string{"features.networks": "false"}})
			return
		}
		networkReply(w, []allocatedNetwork{inherited, differentUplink})
	}))
	defer srv.Close()
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{OVNUplinkNetwork: "UPLINK"})
	if err := b.validateNetworkAllocation(context.Background(), ours.Name, "10.0.1.1/24", ours); err != nil {
		t.Fatal(err)
	}
}

// Model the member-local allocator: overlapping creates see the same completed
// network count and choose the same address. Each task resolves a fresh builder.
func testConcurrentNetworkDeploys(t *testing.T, lockers []NetworkLocker) {
	t.Helper()
	var mu sync.Mutex
	networks := map[string]allocatedNetwork{}
	active, maximum := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var n allocatedNetwork
			if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			active++
			if active > maximum {
				maximum = active
			}
			address := fmt.Sprintf("10.10.44.%d", 35+len(networks))
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			n = testAllocatedNetwork(n.Name, "default", address)
			mu.Lock()
			networks[n.Name] = n
			active--
			mu.Unlock()
			networkReply(w, nil)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/1.0/networks" {
			all := make([]allocatedNetwork, 0, len(networks))
			for _, n := range networks {
				all = append(all, n)
			}
			networkReply(w, all)
		} else if n, ok := networks[strings.TrimPrefix(r.URL.Path, "/1.0/networks/")]; ok {
			networkReply(w, n)
		} else {
			networkFailure(w, 404, "not found")
		}
	}))
	defer srv.Close()
	const workers = 8
	errs := make(chan error, workers)
	start := make(chan struct{})
	for i := range workers {
		go func() {
			<-start
			c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), NetworkLock: lockers[i%len(lockers)]}
			b := New(c, Config{OVNUplinkNetwork: "UPLINK"})
			_, err := b.DeployNetwork(context.Background(), builder.NetworkSpec{ExternalName: fmt.Sprintf("network-%d", i), CIDR: "10.0.1.0/24"})
			errs <- err
		}()
	}
	close(start)
	for range workers {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
	if maximum != 1 {
		t.Errorf("overlapping creates=%d", maximum)
	}
	if len(networks) != workers {
		t.Errorf("networks=%d", len(networks))
	}
}

func TestConcurrentNetworkDeploysAcrossBuilders(t *testing.T) {
	testConcurrentNetworkDeploys(t, []NetworkLocker{nil})
}

func TestDeployNetworkDoesNotCreateAfterFailedExistenceRead(t *testing.T) {
	for _, status := range []int{403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			posts := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					posts++
				}
				// A message mentioning not found is not authoritative evidence of absence.
				networkFailure(w, status, "cluster database record not found")
			}))
			defer srv.Close()
			b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{OVNUplinkNetwork: "UPLINK"})
			if _, err := b.DeployNetwork(context.Background(), builder.NetworkSpec{ExternalName: "read-error", CIDR: "10.0.1.0/24"}); err == nil {
				t.Fatal("failed read accepted")
			}
			if posts != 0 {
				t.Fatalf("created after failed read: %d posts", posts)
			}
		})
	}
}
