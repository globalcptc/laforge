package incus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Inspect reports where each instance and network lives: the server's own
// name on a standalone server (location "none"), the cluster member when
// clustered. The server name is read once, not per resource or per poll.
func TestInspectReportsLocation(t *testing.T) {
	var serverReads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var metadata interface{}
		switch {
		case r.URL.Path == "/1.0":
			serverReads.Add(1)
			metadata = map[string]interface{}{"environment": map[string]string{"server_name": "incus-03"}}
		case r.URL.Path == "/1.0/instances":
			metadata = []string{"/1.0/instances/lf-t3-web01-abc123", "/1.0/instances/lf-t3-db01-def456"}
		case r.URL.Path == "/1.0/networks":
			metadata = []string{"/1.0/networks/lf-t3lan"}
		case strings.HasPrefix(r.URL.Path, "/1.0/instances/"):
			name := strings.TrimPrefix(r.URL.Path, "/1.0/instances/")
			loc := "none"
			if strings.Contains(name, "db01") {
				loc = "member-2"
			}
			metadata = map[string]interface{}{"name": name, "type": "container", "status": "Running", "location": loc, "config": map[string]string{}}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": metadata})
	}))
	t.Cleanup(srv.Close)
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}, Config{})
	for i := 0; i < 2; i++ {
		res, err := b.Inspect(context.Background())
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		want := map[string]string{"lf-t3-web01-abc123": "incus-03", "lf-t3-db01-def456": "member-2", "lf-t3lan": "incus-03"}
		for _, r := range res {
			if r.Location != want[r.ExternalRef] {
				t.Errorf("%s: location %q, want %q", r.ExternalRef, r.Location, want[r.ExternalRef])
			}
		}
	}
	if n := serverReads.Load(); n != 1 {
		t.Errorf("read the server name %d times, want once", n)
	}
}
