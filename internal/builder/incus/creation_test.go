package incus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

func creationResponse(w http.ResponseWriter, metadata any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "sync", "metadata": metadata})
}

func creationOperations(name string) map[string]any {
	return map[string]any{"running": []any{map[string]any{
		"id": "create", "description": "Creating instance", "status_code": 103,
		"resources": map[string][]string{"instances": {"/1.0/instances/" + name}},
	}}}
}

// Several completed long-poll windows must not restart the deploy, replace its
// token, or start the guest while the original image conversion is still busy.
func TestDeployWaitsBeyondCreationPollingWindow(t *testing.T) {
	for _, kind := range []string{"vm", "windows", "oci"} {
		t.Run(kind, func(t *testing.T) {
			creates, waits, starts, imports := 0, 0, 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("project") != "competition" {
					t.Errorf("lost project on %s", r.URL)
				}
				switch {
				case r.URL.Path == "/1.0/operations":
					creationResponse(w, map[string]any{})
				case strings.Contains(r.URL.Path, "/volumes/custom"):
					if creates != 0 {
						t.Error("replaced the config drive after submitting creation")
					}
					if r.Method == http.MethodPost {
						imports++
					}
					configDriveResponse(w, 200, "")
				case r.Method == http.MethodPost && r.URL.Path == "/1.0/instances":
					creates++
					_ = json.NewEncoder(w).Encode(map[string]any{"type": "async", "operation": "/1.0/operations/create?project=competition&target=node1"})
				case r.URL.Path == "/1.0/operations/create/wait":
					if r.URL.Query().Get("target") != "node1" || r.URL.Query().Get("timeout") != "1" {
						t.Errorf("lost wait parameters: %s", r.URL)
					}
					waits++
					op := operationResult{Status: "Running", StatusCode: 103}
					if waits == 3 {
						op = operationResult{Status: "Success", StatusCode: 200}
					}
					creationResponse(w, op)
				case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/state"):
					starts++
					if waits != 3 {
						t.Error("started an instance before creation finished")
					}
					configDriveResponse(w, 200, "")
				default:
					t.Errorf("unexpected mutation/read: %s %s", r.Method, r.URL)
					configDriveResponse(w, 500, "unexpected request")
				}
			}))
			defer srv.Close()
			b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), Project: "competition", OperationTimeout: time.Second}, Config{
				Images: map[string]ImageRef{"linux": {VM: true, Alias: "linux"}, "windows": {VM: true, Alias: "win"}}, Sizes: map[string]SizeSpec{"small": {}},
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			if kind != "oci" {
				// The builder derives the Windows config-drive-ISO path from the OS
				// now, not a spec flag; agent delivery is signalled by AgentDownloadURL.
				os := "linux"
				if kind == "windows" {
					os = "windows"
				}
				_, err = b.DeployHost(ctx, builder.HostSpec{ExternalName: "box", OS: os, Size: "small", AgentDownloadURL: "http://api/agent-binary/box?token=t"})
			} else {
				_, err = b.DeployContainer(ctx, builder.ContainerSpec{ExternalName: "box", Image: "alpine", Size: "small"})
			}
			if err != nil || creates != 1 || waits != 3 || starts != 1 {
				t.Fatalf("creates=%d waits=%d starts=%d err=%v", creates, waits, starts, err)
			}
			if kind == "windows" && imports != 1 {
				t.Fatalf("imports=%d, want one unchanged config drive", imports)
			}
		})
	}
}

func TestCreationWaitFailureAndCancellation(t *testing.T) {
	for _, outcome := range []string{"failure", "cancel", "deadline"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if outcome == "deadline" {
				var deadlineCancel context.CancelFunc
				ctx, deadlineCancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer deadlineCancel()
			}
			creates := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					creates++
					_ = json.NewEncoder(w).Encode(map[string]any{"type": "async", "operation": "/1.0/operations/create"})
					return
				}
				if outcome == "failure" {
					creationResponse(w, operationResult{Status: "Failure", StatusCode: 400, Err: "disk full"})
				} else {
					creationResponse(w, operationResult{Status: "Running", StatusCode: 103})
					if outcome == "cancel" {
						cancel()
					}
				}
			}))
			defer srv.Close()
			client := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			err := client.createInstance(ctx, "box", map[string]string{"name": "box"}, instanceOwner{})
			if creates != 1 || err == nil {
				t.Fatalf("creates=%d err=%v", creates, err)
			}
			if outcome == "failure" && !strings.Contains(err.Error(), "disk full") {
				t.Fatalf("lost creation failure: %v", err)
			}
			if outcome == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
			if outcome == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("lost deadline: %v", err)
			}
		})
	}
}

func TestAdoptWaitsForExactInstanceCreation(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "present", true: "disappeared"}[missing], func(t *testing.T) {
			waits, reads := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("project") != "competition" {
					t.Errorf("unscoped request: %s", r.URL)
				}
				switch r.URL.Path {
				case "/1.0/instances":
					configDriveResponse(w, 409, "Instance already exists")
				case "/1.0/operations":
					if r.URL.Query().Get("recursion") != "1" {
						t.Error("missing recursion")
					}
					ops := creationOperations("box")
					for _, resource := range []string{"box-other", "other", "box?project=other"} {
						op := creationOperations(resource)["running"].([]any)[0].(map[string]any)
						op["id"] = "unrelated"
						ops["running"] = append(ops["running"].([]any), op)
					}
					creationResponse(w, ops)
				case "/1.0/operations/create/wait":
					waits++
					creationResponse(w, operationResult{Status: "Success", StatusCode: 200})
				case "/1.0/instances/box":
					reads++
					if waits != 1 {
						t.Error("adopted before creation finished")
					}
					if missing {
						configDriveResponse(w, 404, "Instance not found")
					} else {
						creationResponse(w, map[string]string{"status": "Stopped"})
					}
				default:
					t.Errorf("waited on unrelated operation: %s", r.URL)
					configDriveResponse(w, 500, "unexpected request")
				}
			}))
			defer srv.Close()
			c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), Project: "competition", OperationTimeout: time.Second}
			err := c.createInstance(context.Background(), "box", map[string]string{"name": "box"}, instanceOwner{})
			if (err != nil) != missing || waits != 1 || reads != 1 {
				t.Fatalf("waits=%d reads=%d err=%v", waits, reads, err)
			}
		})
	}
}

func TestCleanupWaitsForCreationBeforeMutating(t *testing.T) {
	for _, action := range []string{"config drive", "destroy"} {
		for _, outcome := range []string{"success", "failure", "cancel", "list denied"} {
			t.Run(action+"/"+outcome, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				finished, mutations := false, 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/1.0/operations":
						if outcome == "list denied" {
							configDriveResponse(w, 403, "not authorized")
						} else {
							creationResponse(w, creationOperations("box"))
						}
					case "/1.0/operations/create/wait":
						switch outcome {
						case "success":
							finished = true
							creationResponse(w, operationResult{Status: "Success", StatusCode: 200})
						case "failure":
							creationResponse(w, operationResult{Status: "Failure", StatusCode: 400, Err: "disk full"})
						case "cancel":
							cancel()
							creationResponse(w, operationResult{Status: "Running", StatusCode: 103})
						}
					default:
						mutations++
						if !finished {
							t.Errorf("%s %s while creating", r.Method, r.URL)
						}
						configDriveResponse(w, 200, "")
					}
				}))
				defer srv.Close()
				b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}, Config{})
				var err error
				if action == "config drive" {
					err = b.importConfigDrive(ctx, "box", []byte("fresh token"))
				} else {
					err = b.destroyInstance(ctx, "box")
				}
				if outcome == "success" {
					if err != nil || mutations == 0 {
						t.Fatalf("cleanup never completed: mutations=%d err=%v", mutations, err)
					}
				} else if err == nil || mutations != 0 {
					t.Fatalf("cleanup ignored unfinished creation: mutations=%d err=%v", mutations, err)
				}
			})
		}
	}
}

// TestAdoptOnlyOwnInstance: an instance found already existing at create time
// is adopted only when it's this deployment's -- by the deployment name it
// records, or, for one created before that was recorded, by its team -- and
// refused permanently otherwise.
func TestAdoptOnlyOwnInstance(t *testing.T) {
	cases := []struct {
		name   string
		config map[string]string
		owner  instanceOwner
		adopt  bool
	}{
		{"same deployment", map[string]string{externalNameKey: "build-a-team-3-host-web", teamConfigKey: "3"}, instanceOwner{externalName: "build-a-team-3-host-web", team: "3"}, true},
		{"another deployment", map[string]string{externalNameKey: "build-b-team-3-host-web", teamConfigKey: "3"}, instanceOwner{externalName: "build-a-team-3-host-web", team: "3"}, false},
		{"legacy, same team", map[string]string{teamConfigKey: "3"}, instanceOwner{externalName: "build-a-team-3-host-web", team: "3"}, true},
		{"legacy, another team", map[string]string{teamConfigKey: "4"}, instanceOwner{externalName: "build-a-team-3-host-web", team: "3"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/1.0/instances":
					configDriveResponse(w, 409, "Instance already exists")
				case "/1.0/operations":
					creationResponse(w, map[string]any{})
				case "/1.0/instances/box":
					creationResponse(w, map[string]any{"status": "Stopped", "config": tc.config})
				default:
					t.Errorf("unexpected %s", r.URL)
				}
			}))
			defer srv.Close()
			c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			err := c.createInstance(context.Background(), "box", map[string]string{"name": "box"}, tc.owner)
			if tc.adopt && err != nil {
				t.Fatalf("own instance not adopted: %v", err)
			}
			if !tc.adopt && (err == nil || !errors.Is(err, builder.ErrPermanentConfig)) {
				t.Fatalf("err = %v, want a permanent refusal", err)
			}
		})
	}
}
