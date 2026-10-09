package microcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// fakeSnapshotServer answers just enough of the API to deploy a copy and list
// templates, recording the create and update bodies it was sent.
type fakeSnapshotServer struct {
	mu        sync.Mutex
	createReq map[string]interface{}
	putReq    map[string]interface{}
	order     []string
}

func newFakeSnapshotServer(t *testing.T) (*fakeSnapshotServer, *Client) {
	t.Helper()
	f := &fakeSnapshotServer{}
	ok := func(w http.ResponseWriter, metadata interface{}) {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.order = append(f.order, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/1.0/instances":
			json.NewDecoder(r.Body).Decode(&f.createReq)
			ok(w, map[string]interface{}{})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances":
			ok(w, []string{"/1.0/instances/win-tmpl", "/1.0/instances/lf-t1-web01-abc"})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances/win-tmpl":
			ok(w, map[string]interface{}{"name": "win-tmpl", "type": "virtual-machine", "status": "Stopped", "description": "Windows template", "config": map[string]string{"image.os": "Windows", "image.release": "2019"}})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances/win-tmpl/snapshots" && r.URL.Query().Get("recursion") == "":
			ok(w, []string{"/1.0/instances/win-tmpl/snapshots/golden"})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances/win-tmpl/snapshots":
			ok(w, []map[string]string{{"name": "golden", "created_at": "2026-10-01T00:00:00Z"}})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/1.0/instances/lf-") && strings.HasSuffix(r.URL.Path, "/snapshots"):
			t.Errorf("read LaForge's own instance %s while looking for templates", r.URL.Path)
			ok(w, []string{})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances/win-tmpl/snapshots/golden":
			ok(w, map[string]string{"name": "golden"})
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/instances/win-tmpl/snapshots/missing":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "Instance snapshot not found"})
		case r.Method == http.MethodGet:
			// The copy, as created: still carrying the template's own devices
			// and profiles.
			ok(w, map[string]interface{}{
				"config":   map[string]string{"user.laforge_team": "1"},
				"profiles": []string{"default", "template-mgmt"},
				"devices": map[string]interface{}{
					"eth1": map[string]string{"type": "nic", "network": "mgmt"},
					"data": map[string]string{"type": "disk", "pool": "remote", "source": "shared-data", "path": "/data"},
				},
			})
		case r.Method == http.MethodPut && !strings.HasSuffix(r.URL.Path, "/state"):
			json.NewDecoder(r.Body).Decode(&f.putReq)
			ok(w, map[string]interface{}{})
		default:
			ok(w, map[string]interface{}{})
		}
	}))
	t.Cleanup(srv.Close)
	return f, &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}
}

// A snapshot-sourced host is created as a copy of the template's snapshot
// (from its project, without the template's other snapshots), then given
// exactly LaForge's own devices and the default profile before it ever starts.
func TestDeployFromSnapshotCopiesThenReplacesTemplateDevices(t *testing.T) {
	f, client := newFakeSnapshotServer(t)
	b := New(client, Config{
		Images:      map[string]ImageRef{"win2019": {Source: SourceSnapshot, Instance: "win-tmpl", Snapshot: "golden", SourceProject: "templates", VM: true}},
		Sizes:       map[string]SizeSpec{"small": {CPU: "2", Memory: "4GiB"}},
		StoragePool: "remote",
	})
	if _, err := b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "x", Team: "1", OS: "win2019", Size: "small", DiskGB: 60}); err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	src, _ := f.createReq["source"].(map[string]interface{})
	if src["type"] != "copy" || src["source"] != "win-tmpl/golden" || src["instance_only"] != true || src["project"] != "templates" {
		t.Errorf("create source = %v, want a copy of win-tmpl/golden from project templates, instance only", src)
	}
	if f.createReq["type"] != "virtual-machine" {
		t.Errorf("create type = %v, want virtual-machine", f.createReq["type"])
	}
	if f.putReq == nil {
		t.Fatal("the copy's devices were never replaced")
	}
	devices, _ := f.putReq["devices"].(map[string]interface{})
	if _, ok := devices["eth1"]; ok {
		t.Errorf("template NIC eth1 survived the copy: %v", devices)
	}
	if _, ok := devices["data"]; ok {
		t.Errorf("template data volume survived the copy: %v", devices)
	}
	if root, _ := devices["root"].(map[string]interface{}); root["pool"] != "remote" || root["size"] != "60GB" {
		t.Errorf("root device = %v, want on remote, 60GB", devices["root"])
	}
	if profiles, _ := f.putReq["profiles"].([]interface{}); len(profiles) != 1 || profiles[0] != "default" {
		t.Errorf("profiles = %v, want [default]", f.putReq["profiles"])
	}
	put, start := -1, -1
	for i, call := range f.order {
		isState := strings.HasSuffix(call, "/state")
		if strings.HasPrefix(call, "PUT ") && !isState && put < 0 {
			put = i
		}
		if strings.HasPrefix(call, "PUT ") && isState && start < 0 {
			start = i
		}
	}
	if put < 0 || start < 0 || put > start {
		t.Errorf("devices must be replaced before the first start; calls: %v", f.order)
	}
}

// An image-sourced host is created as before: no copy, no device replacement.
func TestDeployFromImageIsUnchanged(t *testing.T) {
	f, client := newFakeSnapshotServer(t)
	b := New(client, Config{
		Images: map[string]ImageRef{"ubuntu": {Fingerprint: "abc"}},
		Sizes:  map[string]SizeSpec{"small": {CPU: "1", Memory: "1GiB"}},
	})
	if _, err := b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "x", Team: "1", OS: "ubuntu", Size: "small", DiskGB: 20}); err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	if src, _ := f.createReq["source"].(map[string]interface{}); src["type"] != "image" || src["fingerprint"] != "abc" {
		t.Errorf("create source = %v, want the image by fingerprint", src)
	}
	if f.putReq != nil {
		t.Errorf("an image-created instance's devices were rewritten: %v", f.putReq)
	}
}

func TestListSnapshotsSkipsLaForgeInstances(t *testing.T) {
	_, client := newFakeSnapshotServer(t)
	snaps, err := client.ListSnapshots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 || snaps[0].Instance != "win-tmpl" || snaps[0].Snapshot != "golden" || snaps[0].Type != "virtual-machine" || snaps[0].OS != "Windows" {
		t.Errorf("snapshots = %+v, want only win-tmpl/golden", snaps)
	}
}

func TestSnapshotExists(t *testing.T) {
	_, client := newFakeSnapshotServer(t)
	if ok, err := client.SnapshotExists(context.Background(), "", "win-tmpl", "golden"); err != nil || !ok {
		t.Errorf("golden: ok=%v err=%v", ok, err)
	}
	if ok, err := client.SnapshotExists(context.Background(), "", "win-tmpl", "missing"); err != nil || ok {
		t.Errorf("missing: ok=%v err=%v", ok, err)
	}
}
