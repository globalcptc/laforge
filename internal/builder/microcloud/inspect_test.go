package microcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Inspect reads only LaForge's own instances (lf-…), one by one -- never every
// tenant's in one recursive listing -- and tolerates one vanishing mid-read.
func TestInspectReadsOnlyLaForgeInstances(t *testing.T) {
	var mu sync.Mutex
	var read []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("recursion") != "" {
			t.Errorf("listing used recursion: %s", r.URL)
		}
		var metadata interface{}
		switch {
		case r.URL.Path == "/1.0/instances":
			metadata = []string{"/1.0/instances/student-vm-1", "/1.0/instances/club-web", "/1.0/instances/lf-t1-web01-abc123", "/1.0/instances/lf-t2-db01-def456", "/1.0/instances/lf-gone-000000"}
		case r.URL.Path == "/1.0/networks":
			metadata = []string{"/1.0/networks/lf-t1lan", "/1.0/networks/club-net"}
		case r.URL.Path == "/1.0/instances/lf-gone-000000":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "Instance not found"})
			return
		case strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			name := strings.TrimPrefix(r.URL.Path, "/1.0/instances/")
			mu.Lock()
			read = append(read, name)
			mu.Unlock()
			metadata = map[string]interface{}{"name": name, "type": "container", "status": "Running", "config": map[string]string{"user.laforge_team": "1"}}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}, Config{})
	res, err := b.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	for _, name := range read {
		if !strings.HasPrefix(name, "lf-") {
			t.Errorf("read another tenant's instance %s", name)
		}
	}
	var instances, networks int
	for _, r := range res {
		if r.Kind == "network" {
			networks++
		} else {
			instances++
		}
	}
	if instances != 2 || networks != 1 {
		t.Errorf("resources = %+v, want LaForge's 2 instances and 1 network", res)
	}
}

// In a project of LaForge's own, one listing reads every instance in it (they
// are all LaForge's) instead of one request each.
func TestInspectInOwnProjectUsesOneListing(t *testing.T) {
	var perInstance int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") != "laforge" {
			t.Errorf("request outside the project: %s", r.URL)
		}
		var metadata interface{}
		switch {
		case r.URL.Path == "/1.0/instances" && r.URL.Query().Get("recursion") == "1":
			metadata = []map[string]interface{}{
				{"name": "lf-t1-web01-abc123", "type": "container", "status": "Running"},
				{"name": "lf-t2-db01-def456", "type": "virtual-machine", "status": "Stopped"},
			}
		case r.URL.Path == "/1.0/networks":
			metadata = []string{}
		case strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			perInstance++
			metadata = map[string]interface{}{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5, Project: "laforge"}, Config{})
	res, err := b.Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(res) != 2 || perInstance != 0 {
		t.Errorf("resources = %d, per-instance reads = %d; want 2 from one listing", len(res), perInstance)
	}
}
