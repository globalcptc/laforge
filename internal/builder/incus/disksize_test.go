package incus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// routeClient serves canned metadata per path (wrapped in the LXD sync envelope).
func routeClient(t *testing.T, routes map[string]interface{}) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta, ok := routes[r.URL.Path]
		if !ok {
			meta = map[string]interface{}{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "status_code": 200, "metadata": meta})
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: 5}
}

func TestParseSizeBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		bad  bool
	}{
		{"64GiB", 64 << 30, false},
		{"60GB", 60_000_000_000, false},
		{"68719476736", 68719476736, false},
		{"", 0, true},
		{"12ZB", 0, true},
	}
	for _, c := range cases {
		got, err := parseSizeBytes(c.in)
		if c.bad {
			if err == nil {
				t.Errorf("parseSizeBytes(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseSizeBytes(%q) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
}

func TestIsShrinkError(t *testing.T) {
	if !isShrinkError(errors.New(`code=500: Create instance from copy: Block volumes cannot be shrunk: Cannot be shrunk`)) {
		t.Error("real shrink error not detected")
	}
	if isShrinkError(errors.New("some other failure")) || isShrinkError(nil) {
		t.Error("false positive")
	}
}

func TestSourceDiskMinBytes(t *testing.T) {
	snap := ImageRef{Source: SourceSnapshot, Instance: "win2016-base", Snapshot: "sysprepped", SourceProject: "Templates", VM: true}

	t.Run("explicit root device size", func(t *testing.T) {
		client := routeClient(t, map[string]interface{}{
			"/1.0/instances/win2016-base/snapshots/sysprepped": map[string]interface{}{
				"expanded_devices": map[string]interface{}{"root": map[string]string{"type": "disk", "pool": "remote", "size": "64GiB"}},
			},
		})
		b := New(client, Config{Images: map[string]ImageRef{"win": snap}, StoragePool: "remote"})
		min, src, ok, err := b.SourceDiskMinBytes(context.Background(), "win")
		if err != nil || !ok || min != 64<<30 {
			t.Fatalf("SourceDiskMinBytes = (%d,%q,%v,%v), want 64 GiB", min, src, ok, err)
		}
	})

	t.Run("falls back to the volume's configured size", func(t *testing.T) {
		img := ImageRef{Source: SourceSnapshot, Instance: "win2019", Snapshot: "s", SourceProject: "Templates", VM: true}
		client := routeClient(t, map[string]interface{}{
			"/1.0/instances/win2019/snapshots/s": map[string]interface{}{
				"expanded_devices": map[string]interface{}{"root": map[string]string{"type": "disk", "pool": "remote"}},
			},
			"/1.0/storage-pools/remote/volumes/virtual-machine/win2019/snapshots/s": map[string]interface{}{
				"config": map[string]string{"size": "100GiB"},
			},
		})
		b := New(client, Config{Images: map[string]ImageRef{"win": img}, StoragePool: "remote"})
		min, _, ok, err := b.SourceDiskMinBytes(context.Background(), "win")
		if err != nil || !ok || min != 100<<30 {
			t.Fatalf("SourceDiskMinBytes = (%d,%v,%v), want 100 GiB", min, ok, err)
		}
	})

	t.Run("image source has no shrink constraint", func(t *testing.T) {
		b := New(routeClient(t, nil), Config{Images: map[string]ImageRef{"win": {Alias: "ubuntu"}}})
		if _, _, ok, err := b.SourceDiskMinBytes(context.Background(), "win"); ok || err != nil {
			t.Errorf("image source should return ok=false, got ok=%v err=%v", ok, err)
		}
	})
}

func TestInstanceNameFitsSocketPath(t *testing.T) {
	const project = "CPTC-Comp-Environment"
	const ext = "build-f8417f6d-team-1-triplecorkscrew"

	socketPath := func(name string) string { return incusLogsDir + project + "_" + name + qemuMonitorSuffix }

	t.Run("long project + placement stays within the socket limit", func(t *testing.T) {
		name := instanceName("lf-cptc12-forrest-t1-triplecorkscrew", ext, project)
		if p := socketPath(name); len(p) > unixSocketPathMax {
			t.Fatalf("socket path %d bytes (name %q), over %d", len(p), name, unixSocketPathMax)
		}
	})

	t.Run("boundary: name at the budget hits exactly the limit", func(t *testing.T) {
		atLimit := strings.Repeat("a", maxInstanceNameLen(project))
		if p := len(socketPath(atLimit)); p != unixSocketPathMax {
			t.Fatalf("name at maxInstanceNameLen yields %d bytes, want %d", p, unixSocketPathMax)
		}
	})

	t.Run("default project is not shrunk below the old 48-base cap", func(t *testing.T) {
		if got := maxInstanceNameLen("default"); got != 63-len("-cidata") {
			t.Fatalf("maxInstanceNameLen(default) = %d, want %d", got, 63-len("-cidata"))
		}
	})

	t.Run("project too long is a permanent error", func(t *testing.T) {
		if err := validateInstanceNameBudget("default"); err != nil {
			t.Fatalf("default should validate: %v", err)
		}
		err := validateInstanceNameBudget(strings.Repeat("x", 95))
		if err == nil || !errors.Is(err, builder.ErrPermanentConfig) {
			t.Fatalf("want ErrPermanentConfig for a 95-char project, got %v", err)
		}
	})
}
