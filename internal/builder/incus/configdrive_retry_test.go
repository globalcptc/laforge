package incus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

func configDriveResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status >= 400 {
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error_code": status, "error": message})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": map[string]any{}})
}

func TestImportISOStopsOnCleanupFailure(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			imports := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					configDriveResponse(w, status, "cannot remove previous config drive")
					return
				}
				imports++
				configDriveResponse(w, http.StatusOK, "")
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			err := client.importISOVolume(context.Background(), "default", "cidata-test", []byte("fresh ISO"))
			if err == nil || !strings.Contains(err.Error(), "cannot remove previous config drive") {
				t.Fatalf("cleanup error was lost: %v", err)
			}
			if imports != 0 {
				t.Fatalf("uploaded %d ISOs despite failed cleanup", imports)
			}
		})
	}
}

func TestImportISOAfterMissingOrRemovedVolume(t *testing.T) {
	for _, deleteStatus := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(fmt.Sprint(deleteStatus), func(t *testing.T) {
			imports := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("project") != "team1" {
					t.Errorf("request escaped its project: %s", r.URL)
				}
				if r.Method == http.MethodDelete {
					configDriveResponse(w, deleteStatus, "Storage volume not found")
					return
				}
				imports++
				body, _ := io.ReadAll(r.Body)
				if string(body) != "fresh ISO" || r.Header.Get("X-Incus-name") != "cidata-test" || r.Header.Get("X-Incus-type") != "iso" {
					t.Error("incorrect ISO upload")
				}
				configDriveResponse(w, http.StatusOK, "")
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL, Project: "team1", HTTPClient: srv.Client(), OperationTimeout: time.Second}
			if err := client.importISOVolume(context.Background(), "default", "cidata-test", []byte("fresh ISO")); err != nil {
				t.Fatal(err)
			}
			if imports != 1 {
				t.Fatalf("imports = %d, want 1", imports)
			}
		})
	}
}

// An earlier deploy left a stopped VM with its config drive still attached.
// A retry must finish cleanup before uploading the new one-time agent token.
func TestDeployConfigDriveRetry(t *testing.T) {
	for _, failure := range []string{"", "stop", "instance deletion", "volume deletion", "pending deletion"} {
		t.Run("failure="+failure, func(t *testing.T) {
			instanceExists, volumeExists := true, true
			imports, creates := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/1.0/operations":
					configDriveResponse(w, http.StatusOK, "")
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/volumes/custom/"):
					if instanceExists {
						configDriveResponse(w, http.StatusBadRequest, "The storage volume is still in use")
					} else if failure == "volume deletion" {
						configDriveResponse(w, http.StatusInternalServerError, "volume deletion failed: storage offline")
					} else {
						volumeExists = false
						configDriveResponse(w, http.StatusOK, "")
					}
				case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/volumes/custom"):
					imports++
					if volumeExists {
						configDriveResponse(w, http.StatusBadRequest, "Cannot create volume, already exists on target storage")
					} else {
						volumeExists = true
						configDriveResponse(w, http.StatusOK, "")
					}
				case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/state"):
					var state struct{ Action string }
					_ = json.NewDecoder(r.Body).Decode(&state)
					if state.Action == "stop" {
						if failure == "stop" {
							configDriveResponse(w, http.StatusBadRequest, "stop failed: instance is busy running an operation")
						} else {
							configDriveResponse(w, http.StatusBadRequest, "The instance is already stopped")
						}
					} else {
						configDriveResponse(w, http.StatusOK, "")
					}
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/instances/"):
					if failure == "instance deletion" {
						configDriveResponse(w, http.StatusBadRequest, "instance deletion failed: instance is protected")
					} else if failure == "pending deletion" {
						_ = json.NewEncoder(w).Encode(map[string]any{"type": "async", "operation": "/1.0/operations/delete"})
					} else {
						instanceExists = false
						configDriveResponse(w, http.StatusOK, "")
					}
				case r.Method == http.MethodPost && r.URL.Path == "/1.0/instances":
					creates++
					if !volumeExists || instanceExists {
						t.Error("created a VM before config-drive replacement finished")
					}
					instanceExists = true
					configDriveResponse(w, http.StatusOK, "")
				case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/network-acls/"):
					configDriveResponse(w, http.StatusNotFound, "ACL not found")
				case r.Method == http.MethodGet && r.URL.Path == "/1.0/operations/delete/wait":
					_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": operationResult{Status: "Running", StatusCode: 103}})
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					configDriveResponse(w, http.StatusInternalServerError, "unexpected request")
				}
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			b := New(client, Config{Images: map[string]ImageRef{"windows": {Alias: "win2019", VM: true}}, Sizes: map[string]SizeSpec{"small": {}}})
			ref, err := b.DeployHost(context.Background(), builder.HostSpec{
				ExternalName: "test-dc", OS: "windows", Size: "small",
				AgentDownloadURL: "http://api/agent-binary/test-dc?token=t",
			})
			if ref == "" {
				t.Fatal("failed deploy lost its cleanup reference")
			}
			if failure != "" {
				if failure == "pending deletion" {
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("continued while instance deletion was still running: %v", err)
					}
				} else if err == nil || !strings.Contains(err.Error(), failure+" failed") {
					t.Fatalf("lost %s error: %v", failure, err)
				}
				if imports != 0 || creates != 0 {
					t.Fatalf("continued after failed cleanup: %d imports, %d instance creates", imports, creates)
				}
			} else if err != nil || imports != 1 || creates != 1 {
				t.Fatalf("retry: err=%v, imports=%d, creates=%d", err, imports, creates)
			}
		})
	}
}

func TestConfigDriveRecoveryRetries(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		failures  int
		wantCalls int
		wantError bool
		cancel    bool
	}{
		{"busy recovers", 400, 2, 3, false, false},
		{"unavailable recovers", 503, 2, 3, false, false},
		{"retry limit", 400, 10, 3, true, false},
		{"permission denied", 403, 10, 1, true, false},
		{"cancelled", 400, 10, 1, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deletes, imports := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == "/1.0/operations" {
					configDriveResponse(w, http.StatusOK, "")
					return
				}
				if r.Method == http.MethodDelete {
					deletes++
					if deletes <= test.failures {
						message := "permission denied"
						if test.status == http.StatusBadRequest {
							message = "Instance is busy running a stop operation"
						}
						configDriveResponse(w, test.status, message)
						if test.cancel {
							cancel()
						}
						return
					}
				} else if r.Method == http.MethodPost {
					imports++
				} else {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				configDriveResponse(w, http.StatusOK, "")
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			err := New(client, Config{}).importConfigDrive(ctx, "test", []byte("fresh ISO"))
			if (err != nil) != test.wantError || deletes != test.wantCalls {
				t.Fatalf("err=%v, delete calls=%d; want error=%v and %d calls", err, deletes, test.wantError, test.wantCalls)
			}
			if test.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation was lost: %v", err)
			}
			if test.wantCalls == 3 && test.wantError && !strings.Contains(err.Error(), "after 3 attempts") {
				t.Fatalf("retry exhaustion did not report its attempts: %v", err)
			}
			if (test.wantError && imports != 0) || (!test.wantError && imports != 1) {
				t.Fatalf("imports=%d for err=%v", imports, err)
			}
		})
	}
}

func TestConfigDriveRecoveryWaitsForPendingDeletion(t *testing.T) {
	deletes, waits, imports := 0, 0, 0
	pending := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/1.0/operations" {
			configDriveResponse(w, http.StatusOK, "")
			return
		}
		switch r.Method {
		case http.MethodDelete:
			deletes++
			if deletes == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "async", "operation": "/1.0/operations/volume-delete"})
				return
			}
			if pending {
				t.Error("issued another delete before the previous one completed")
			}
			configDriveResponse(w, http.StatusNotFound, "Storage volume not found")
		case http.MethodGet:
			if r.URL.Path != "/1.0/operations/volume-delete/wait" {
				t.Errorf("waited on the wrong operation: %s", r.URL)
			}
			waits++
			result := operationResult{Status: "Running", StatusCode: 103}
			if waits == 3 {
				pending = false
				result = operationResult{Status: "Success", StatusCode: 200}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": result})
		case http.MethodPost:
			imports++
			if pending {
				t.Error("imported ISO before its previous volume was deleted")
			}
			configDriveResponse(w, http.StatusOK, "")
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
	}))
	defer srv.Close()
	client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
	if err := New(client, Config{}).importConfigDrive(context.Background(), "test", []byte("fresh ISO")); err != nil {
		t.Fatal(err)
	}
	if deletes != 2 || waits != 3 || imports != 1 {
		t.Fatalf("deletes=%d waits=%d imports=%d, want 2/3/1", deletes, waits, imports)
	}
}
