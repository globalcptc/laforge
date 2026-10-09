package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// fakeIncusServer wraps a real HTTP server answering exactly the sync
// envelope shape client.go's own do() decodes ({"type":"sync",
// "metadata": ...}) -- the same discipline this package's own doc
// comments insist on (verified against a live daemon, not assumed from
// docs), applied here to a plain unit test instead of a live cluster:
// discover.go's own JSON handling doesn't need a real Incus daemon to
// prove correct, only a real HTTP round trip through Client.do.
func fakeIncusServer(t *testing.T, routes map[string]interface{}) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		raw, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "sync", "status": "Success", "status_code": 200,
			"metadata": json.RawMessage(raw),
		})
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: defaultTestOperationTimeout}
}

const defaultTestOperationTimeout = 5

func TestListStoragePools(t *testing.T) {
	// Names from one cheap listing, each pool's details separately (never a
	// recursive listing), and a pool whose details fail is still listed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("recursion") != "" {
			t.Errorf("listing used recursion: %s", r.URL)
		}
		var metadata interface{}
		switch r.URL.Path {
		case "/1.0/storage-pools":
			metadata = []string{"/1.0/storage-pools/default", "/1.0/storage-pools/remote", "/1.0/storage-pools/busy"}
		case "/1.0/storage-pools/default":
			metadata = StoragePoolInfo{Name: "default", Driver: "zfs", Status: "Created"}
		case "/1.0/storage-pools/remote":
			metadata = StoragePoolInfo{Name: "remote", Driver: "ceph", Status: "Created"}
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 500, "error": "timed out"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}
	pools, err := client.ListStoragePools(context.Background())
	if err != nil {
		t.Fatalf("ListStoragePools: %v", err)
	}
	if len(pools) != 3 || pools[1].Name != "remote" || pools[1].Driver != "ceph" || pools[2].Name != "busy" || pools[2].Driver != "" {
		t.Fatalf("pools = %+v, want default/zfs, remote/ceph, and busy by name only", pools)
	}
}

func TestListNetworks(t *testing.T) {
	// A shared cluster: LaForge's own team networks, many of other tenants',
	// and an uplink listed last. Details are read for a bounded number,
	// likely uplinks first; never all of them, never recursively.
	var mu sync.Mutex
	var detailed []string
	urls := []string{"/1.0/networks/lf-t1lan-abc", "/1.0/networks/lf-t2lan-def"}
	for i := 0; i < 60; i++ {
		urls = append(urls, fmt.Sprintf("/1.0/networks/club-%02d", i))
	}
	urls = append(urls, "/1.0/networks/UPLINK?project=default")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("recursion") != "" {
			t.Errorf("listing used recursion: %s", r.URL)
		}
		var metadata interface{}
		switch {
		case r.URL.Path == "/1.0/networks":
			metadata = urls
		case strings.HasPrefix(r.URL.Path, "/1.0/networks/"):
			name := strings.TrimPrefix(r.URL.Path, "/1.0/networks/")
			mu.Lock()
			detailed = append(detailed, name)
			mu.Unlock()
			typ := "bridge"
			if name == "UPLINK" {
				typ = "physical"
			}
			metadata = NetworkInfo{Name: name, Type: typ, Managed: true, Status: "Created"}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}

	networks, names, err := client.ListNetworks(context.Background())
	if err != nil {
		t.Fatalf("ListNetworks: %v", err)
	}
	if len(names) != 61 || names[0] != "UPLINK" {
		t.Errorf("names = %d (first %q), want the 61 non-LaForge networks, sorted", len(names), names[0])
	}
	if len(detailed) > maxNetworkDetails {
		t.Errorf("read %d networks in full, want at most %d", len(detailed), maxNetworkDetails)
	}
	for _, d := range detailed {
		if strings.HasPrefix(d, "lf-") {
			t.Errorf("read LaForge's own team network %s in full", d)
		}
	}
	var sawUplink bool
	for _, n := range networks {
		if n.Name == "UPLINK" && n.Type == "physical" {
			sawUplink = true
		}
	}
	if !sawUplink {
		t.Errorf("the uplink (listed last) wasn't read in full: %v", detailed)
	}
}

func TestListImagesFlattensAliases(t *testing.T) {
	client := fakeIncusServer(t, map[string]interface{}{
		"/1.0/images": []map[string]interface{}{
			{
				"fingerprint":  "abc123",
				"architecture": "x86_64",
				"type":         "virtual-machine",
				"properties":   map[string]string{"os": "Windows Server", "release": "2022"},
				"aliases":      []map[string]string{{"name": "windows-server-2022"}, {"name": "win2022"}},
			},
			{
				"fingerprint":  "def456",
				"architecture": "x86_64",
				"type":         "container",
				"properties":   map[string]string{"os": "Ubuntu", "release": "22.04"},
				"aliases":      []map[string]string{{"name": "ubuntu/22.04"}},
			},
		},
	})
	images, err := client.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}
	if len(images[0].Aliases) != 2 || images[0].Aliases[0] != "windows-server-2022" {
		t.Fatalf("images[0].Aliases = %v, want [windows-server-2022 win2022]", images[0].Aliases)
	}
	if images[0].Type != "virtual-machine" || images[0].Properties.OS != "Windows Server" {
		t.Fatalf("images[0] = %+v, want a Windows VM image", images[0])
	}
	if images[1].Type != "container" {
		t.Fatalf("images[1].Type = %q, want container", images[1].Type)
	}
}

// An image with no alias (common for images cached from a remote) must
// serialize its aliases as [], not null -- found by clicking through the
// builder workflow, where the UI indexes the list.
func TestListImagesUnaliasedImageSerializesEmptyAliases(t *testing.T) {
	client := fakeIncusServer(t, map[string]interface{}{
		"/1.0/images": []map[string]interface{}{{"fingerprint": "abc", "type": "container"}},
	})
	images, err := client.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	out, _ := json.Marshal(images[0])
	if !strings.Contains(string(out), `"aliases":[]`) {
		t.Fatalf("serialized = %s, want \"aliases\":[]", out)
	}
}

// A fingerprint-pinned image is deployed by fingerprint -- the exact same
// image on every host -- not by alias.
func TestDeployHostUsesImageFingerprint(t *testing.T) {
	var source map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/1.0/instances" {
			var body struct {
				Source map[string]string `json:"source"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			source = body.Source
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": map[string]interface{}{}})
	}))
	t.Cleanup(srv.Close)
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}, Config{
		Images: map[string]ImageRef{"win2019": {Fingerprint: "f74402e46da8", Alias: "win2019-core", VM: true}},
		Sizes:  map[string]SizeSpec{"small": {CPU: "1", Memory: "1GiB"}},
	})
	if _, err := b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "x", Team: "1", OS: "win2019", Size: "small"}); err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	if source["fingerprint"] != "f74402e46da8" || source["alias"] != "" {
		t.Fatalf("source = %v, want the fingerprint and no alias", source)
	}
}
