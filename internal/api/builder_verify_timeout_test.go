package api

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/db"
)

// Exercise the save-time path against a slow TLS remote, including its real
// client deadlines. No database or live cluster is required.
func TestVerifyImagesSlowRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("exercises the former 30-second deadline")
	}
	for _, tc := range []struct {
		name, kind                string
		snapshotDelay, imageDelay time.Duration
		timeout                   int32
		parentTimeout             time.Duration
		wantDeadline              bool
	}{
		{name: "microcloud snapshot", kind: "microcloud", snapshotDelay: 31 * time.Second},
		{name: "microcloud images", kind: "microcloud", imageDelay: 31 * time.Second},
		{name: "incus cumulative checks", kind: "incus", snapshotDelay: 16 * time.Second, imageDelay: 16 * time.Second},
		{name: "configured budget", kind: "microcloud", snapshotDelay: 31 * time.Second, timeout: 30, wantDeadline: true},
		{name: "caller deadline", kind: "microcloud", snapshotDelay: time.Second, parentTimeout: 100 * time.Millisecond, wantDeadline: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var delay time.Duration
				var metadata any
				switch r.URL.Path {
				case "/1.0/instances/template/snapshots/golden":
					if r.URL.Query().Get("project") != "templates" {
						t.Error("snapshot lost source project")
					}
					delay, metadata = tc.snapshotDelay, map[string]any{}
				case "/1.0/images":
					delay, metadata = tc.imageDelay, []map[string]any{{"fingerprint": "abc", "aliases": []map[string]string{{"name": "ubuntu"}}}}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
					return
				}
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-r.Context().Done():
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": metadata})
			}))
			defer remote.Close()
			ca, err := agentpki.GenerateCA("verify-timeout-test")
			if err != nil {
				t.Fatal(err)
			}
			certPath, keyPath := filepath.Join(t.TempDir(), "client.crt"), filepath.Join(t.TempDir(), "client.key")
			if err := os.WriteFile(certPath, ca.CertPEM, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(keyPath, ca.KeyPEM, 0600); err != nil {
				t.Fatal(err)
			}
			pin := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: remote.Certificate().Raw}))
			params := db.CreateBuilderConfigParams{
				Kind: tc.kind, IncusApiUrl: db.StrPtr(remote.URL), IncusClientCertPath: db.StrPtr(certPath),
				IncusClientKeyPath: db.StrPtr(keyPath), IncusServerCertPem: db.StrPtr(pin),
				IncusImages: []byte(`{"snapshot":{"source":"snapshot","instance":"template","snapshot":"golden","source_project":"templates"},"image":{"fingerprint":"abc"},"alias":{"alias":"ubuntu"}}`),
			}
			if tc.timeout > 0 {
				params.IncusOperationTimeoutSeconds = &tc.timeout
			}
			if tc.kind == "incus" {
				params.IncusHosts, err = json.Marshal([]incus.HostConfig{{APIURL: remote.URL, ClientCertPath: certPath, ClientKeyPath: keyPath, ServerCertPEM: pin, OperationTimeoutSeconds: 60}})
				if err != nil {
					t.Fatal(err)
				}
			}
			ctx := context.Background()
			if tc.parentTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.parentTimeout)
				defer cancel()
			}
			req := httptest.NewRequest(http.MethodPut, "/builder-configs/test", nil).WithContext(ctx)
			err = (&Server{}).verifyImages(req, params)
			if tc.wantDeadline {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("got %v, want deadline exceeded", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
