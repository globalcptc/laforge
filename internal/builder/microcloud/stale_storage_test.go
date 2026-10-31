package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// fakeStaleCeph answers a deploy where some instance names collide with an
// image the Ceph pool still holds but LXD no longer tracks.
type fakeStaleCeph struct {
	mu      sync.Mutex
	stale   map[string]bool // names whose create fails with rbd EEXIST
	known   map[string]bool // names LXD has an instance under
	creates []string
	deletes []string
}

func newFakeStaleCeph(t *testing.T, failWith string) (*fakeStaleCeph, *Client) {
	t.Helper()
	f := &fakeStaleCeph{stale: map[string]bool{}, known: map[string]bool{}}
	reply := func(w http.ResponseWriter, md interface{}) {
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": md})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/1.0/instances":
			var body struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			f.creates = append(f.creates, body.Name)
			if f.stale[body.Name] || (failWith != "" && len(f.creates) == 1) {
				msg := "Create instance from copy: Failed to run: rbd --id admin --cluster ceph clone pool/virtual-machine_p_" + body.Name + ": exit status 17 (librbd: error creating child: (17) File exists rbd: clone error: (17) File exists)"
				if failWith != "" {
					msg = failWith
				}
				w.WriteHeader(500)
				json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 500, "error": msg})
				return
			}
			f.known[body.Name] = true
			reply(w, map[string]interface{}{})
		case r.Method == http.MethodDelete:
			f.deletes = append(f.deletes, r.URL.Path)
			reply(w, map[string]interface{}{})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/1.0/instances/"), "/state")
			if !f.known[name] {
				w.WriteHeader(404)
				json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "Instance not found"})
				return
			}
			if strings.HasSuffix(r.URL.Path, "/state") {
				reply(w, map[string]string{"status": "Running"})
				return
			}
			reply(w, map[string]interface{}{"name": name, "config": map[string]string{}, "devices": map[string]interface{}{}})
		default:
			reply(w, map[string]interface{}{})
		}
	}))
	t.Cleanup(srv.Close)
	return f, &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}
}

func staleTestBuilder(c *Client) *Builder {
	return New(c, Config{
		Images: map[string]ImageRef{"ubuntu": {Fingerprint: "abc"}},
		Sizes:  map[string]SizeSpec{"small": {CPU: "1", Memory: "1GiB"}},
	})
}

var staleSpec = builder.HostSpec{ExternalName: "build-1-team-1-host-bff-ws-02", DisplayName: "bff-ws-02", Team: "1", OS: "ubuntu", Size: "small", DiskGB: 20}

// A host whose name collides with a stale Ceph image deploys under a fresh
// variant of the name, and reports that name so the runner records it.
func TestDeployMovesToFreshNameOnStaleCephImage(t *testing.T) {
	f, c := newFakeStaleCeph(t, "")
	base := instanceName(staleSpec.DisplayName, staleSpec.ExternalName, "")
	f.stale[base] = true
	name, err := staleTestBuilder(c).DeployHost(context.Background(), staleSpec)
	if err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	if name == base || !strings.HasPrefix(name, "lf-bff-ws-02-") {
		t.Fatalf("deployed as %q, want a fresh variant of %q with the same readable base", name, base)
	}
	if len(f.creates) != 2 || f.creates[0] != base || f.creates[1] != name {
		t.Fatalf("creates = %v, want [%s %s]", f.creates, base, name)
	}
	cleared := false
	for _, d := range f.deletes {
		if strings.Contains(d, "cidata-"+base) {
			cleared = true
		}
	}
	if !cleared {
		t.Errorf("the abandoned name's config-drive volume wasn't cleared; deletes = %v", f.deletes)
	}
	// Deterministic: the same collision picks the same fresh name again.
	f2, c2 := newFakeStaleCeph(t, "")
	f2.stale[base] = true
	if again, _ := staleTestBuilder(c2).DeployHost(context.Background(), staleSpec); again != name {
		t.Errorf("second deploy picked %q, want the same fresh name %q", again, name)
	}
}

// When every fresh name collides too, the deploy gives up after a bounded
// number of names and says what to clean up.
func TestDeployGivesUpAfterBoundedFreshNames(t *testing.T) {
	f, c := newFakeStaleCeph(t, "")
	b := staleTestBuilder(c)
	f.stale[instanceName(staleSpec.DisplayName, staleSpec.ExternalName, "")] = true
	for i := 1; i <= maxNameFallbacks; i++ {
		ctx := context.WithValue(context.Background(), nameSaltKey{}, "r"+string(rune('0'+i)))
		f.stale[deployName(ctx, staleSpec.DisplayName, staleSpec.ExternalName, "")] = true
	}
	_, err := b.DeployHost(context.Background(), staleSpec)
	if err == nil || !strings.Contains(err.Error(), "rbd rm") {
		t.Fatalf("err = %v, want a failure naming the Ceph cleanup", err)
	}
	if len(f.creates) != 1+maxNameFallbacks {
		t.Fatalf("tried %d names, want %d", len(f.creates), 1+maxNameFallbacks)
	}
}

// Other create failures, and a collision under a name LXD does have an
// instance for, keep the deterministic name.
func TestDeployKeepsNameForOtherFailures(t *testing.T) {
	f, c := newFakeStaleCeph(t, "Failed creating instance: not enough memory")
	if _, err := staleTestBuilder(c).DeployHost(context.Background(), staleSpec); err == nil {
		t.Fatal("want the create error")
	}
	if len(f.creates) != 1 {
		t.Fatalf("creates = %v, want one attempt under the deterministic name", f.creates)
	}

	collision := errors.New("rbd clone: (17) File exists")
	if !isStaleStorageCollision(collision) || !isStaleStorageCollision(errors.New(`Cannot create volume "cidata-x", volume already exists on storage pool "p"`)) {
		t.Fatal("collision errors not recognized")
	}
	f2, c2 := newFakeStaleCeph(t, "")
	b := staleTestBuilder(c2)
	base := instanceName(staleSpec.DisplayName, staleSpec.ExternalName, "")
	f2.known[base] = true
	name, err := b.withFreshNameOnStaleStorage(context.Background(), staleSpec.DisplayName, staleSpec.ExternalName, func(ctx context.Context) (string, error) {
		return deployName(ctx, staleSpec.DisplayName, staleSpec.ExternalName, ""), collision
	})
	if name != base || err == nil {
		t.Fatalf("name %q err %v, want the deterministic name kept (LXD has that instance)", name, err)
	}
}
