package microcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// projectRecorder answers every call with an empty sync response and
// records, per method+path, the project query each was made in.
type projectRecorder struct {
	mu       sync.Mutex
	projects map[string]string
	bodies   map[string]map[string]interface{}
	routes   map[string]interface{}
}

func newProjectRecorder(t *testing.T, routes map[string]interface{}) (*projectRecorder, *Client) {
	t.Helper()
	rec := &projectRecorder{projects: map[string]string{}, bodies: map[string]map[string]interface{}{}, routes: routes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		rec.mu.Lock()
		rec.projects[key] = r.URL.Query().Get("project")
		if r.Method == http.MethodPost {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			rec.bodies[key] = body
		}
		rec.mu.Unlock()
		metadata := rec.routes[r.URL.Path]
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	return rec, &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}
}

// Discovery reads images in the chosen project, but storage pools and networks
// (where the uplink lives) from the server as a whole, and lists the projects.
func TestDiscoveryScopesImagesToTheProject(t *testing.T) {
	rec, client := newProjectRecorder(t, map[string]interface{}{
		"/1.0/storage-pools":        []string{"/1.0/storage-pools/remote"},
		"/1.0/storage-pools/remote": map[string]string{"name": "remote", "driver": "ceph"},
		"/1.0/networks":             []string{"/1.0/networks/UPLINK"},
		"/1.0/networks/UPLINK":      map[string]string{"name": "UPLINK", "type": "physical"},
		"/1.0/images":               []map[string]interface{}{{"fingerprint": "abc", "aliases": []map[string]string{{"name": "ubuntu"}}}},
		"/1.0/instances":            []string{},
		"/1.0/projects":             []string{"/1.0/projects/default", "/1.0/projects/laforge", "/1.0/projects/student-0042"},
		"/1.0/projects/default":     map[string]interface{}{"name": "default", "config": map[string]string{"features.networks": "true", "features.images": "true"}},
		"/1.0/projects/laforge":     map[string]interface{}{"name": "laforge", "description": "ours", "config": map[string]string{"features.networks": "true", "features.profiles": "true", "restricted": "true"}},
	})
	got, err := discoverResources(context.Background(), client, "laforge")
	if err != nil {
		t.Fatalf("discoverResources: %v", err)
	}
	for _, p := range []string{"GET /1.0/images", "GET /1.0/instances"} {
		if rec.projects[p] != "laforge" {
			t.Errorf("%s listed in project %q, want laforge", p, rec.projects[p])
		}
	}
	if _, read := rec.projects["GET /1.0/projects/student-0042"]; read {
		t.Error("read a project nobody chose in full")
	}
	for _, p := range []string{"GET /1.0/storage-pools", "GET /1.0/networks", "GET /1.0/projects"} {
		if rec.projects[p] != "" {
			t.Errorf("%s was scoped to project %q, want the server as a whole", p, rec.projects[p])
		}
	}
	if len(got.Projects) != 3 || got.Projects[1].Name != "laforge" || !got.Projects[1].Detailed || !got.Projects[1].Networks || !got.Projects[1].Profiles || !got.Projects[1].Restricted || got.Projects[1].Images {
		t.Errorf("projects = %+v", got.Projects)
	}
	if got.Projects[2].Name != "student-0042" || got.Projects[2].Detailed {
		t.Errorf("an unchosen project should be listed by name only: %+v", got.Projects[2])
	}
	if len(got.Warnings) != 0 {
		t.Errorf("warnings = %v", got.Warnings)
	}
	if len(got.Images) != 1 || len(got.Networks) != 1 || len(got.StoragePools) != 1 {
		t.Errorf("discovery = %+v", got)
	}
}

// In a project other than default every instance is created in that project
// with an explicit root disk on the builder's pool, since the project's
// default profile may have none. In default nothing changes.
func TestDeployInstanceInAProject(t *testing.T) {
	for _, tc := range []struct {
		project  string
		wantRoot bool
	}{{"laforge", true}, {"", false}} {
		rec, client := newProjectRecorder(t, map[string]interface{}{
			"/1.0/instances/" + instanceName("", "x") + "/state": map[string]string{"status": "Running"},
		})
		client.Project = tc.project
		b := New(client, Config{
			Images:      map[string]ImageRef{"ubuntu": {Fingerprint: "abc"}},
			Sizes:       map[string]SizeSpec{"small": {CPU: "1", Memory: "1GiB"}},
			StoragePool: "remote",
		})
		if _, err := b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "x", Team: "1", OS: "ubuntu", Size: "small"}); err != nil {
			t.Fatalf("project %q: DeployHost: %v", tc.project, err)
		}
		if rec.projects["POST /1.0/instances"] != tc.project {
			t.Errorf("instance created in project %q, want %q", rec.projects["POST /1.0/instances"], tc.project)
		}
		devices, _ := rec.bodies["POST /1.0/instances"]["devices"].(map[string]interface{})
		root, hasRoot := devices["root"].(map[string]interface{})
		if hasRoot != tc.wantRoot {
			t.Errorf("project %q: root device present = %v, want %v (devices %v)", tc.project, hasRoot, tc.wantRoot, devices)
		}
		if hasRoot && root["pool"] != "remote" {
			t.Errorf("project %q: root disk on pool %v, want remote", tc.project, root["pool"])
		}
	}
}

// The exec output log is fetched in the client's project too.
func TestRawGetCarriesTheProject(t *testing.T) {
	rec, client := newProjectRecorder(t, nil)
	client.Project = "laforge"
	client.rawGet("/1.0/instances/x/logs/exec-output/out.stdout")
	if got := rec.projects["GET /1.0/instances/x/logs/exec-output/out.stdout"]; got != "laforge" {
		t.Errorf("exec output fetched in project %q, want laforge", got)
	}
}
