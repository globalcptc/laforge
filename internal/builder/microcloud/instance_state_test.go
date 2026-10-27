package microcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// TestInstanceStateReadsOneInstance: the power-state poll's direct check of an
// instance its listing left out. A present instance reports its state; only a
// 404 means it doesn't exist; any other failure is an error, never "gone".
func TestInstanceStateReadsOneInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1.0/instances/lf-running":
			json.NewEncoder(w).Encode(map[string]any{"type": "sync", "status_code": 200, "metadata": map[string]any{"name": "lf-running", "status": "Running"}})
		case "/1.0/instances/lf-stopped":
			json.NewEncoder(w).Encode(map[string]any{"type": "sync", "status_code": 200, "metadata": map[string]any{"name": "lf-stopped", "status": "Stopped"}})
		case "/1.0/instances/lf-gone":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": 404, "error": "Instance not found"})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": 500, "error": "member timed out"})
		}
	}))
	t.Cleanup(srv.Close)
	var stater builder.InstanceStater = New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}, Config{})

	for _, c := range []struct {
		ref, state    string
		exists, isErr bool
	}{
		{"lf-running", builder.PowerStateRunning, true, false},
		{"lf-stopped", builder.PowerStateStopped, true, false},
		{"lf-gone", "", false, false},
		{"lf-busy", "", false, true},
	} {
		state, exists, err := stater.InstanceState(context.Background(), c.ref)
		if state != c.state || exists != c.exists || (err != nil) != c.isErr {
			t.Errorf("%s: state %q exists %v err %v; want %q %v error=%v", c.ref, state, exists, err, c.state, c.exists, c.isErr)
		}
	}
}
