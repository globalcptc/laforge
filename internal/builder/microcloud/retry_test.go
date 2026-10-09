package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestCreateRetriesWaitForOriginalOperation(t *testing.T) {
	creates, waits, reads := 0, 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1.0/instances":
			creates++
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "async", "operation": "/1.0/operations/copy-1?project=competition"})
		case "/1.0/operations/copy-1/wait":
			if r.URL.Query().Get("project") != "competition" || r.URL.Query().Get("timeout") != "1" {
				t.Errorf("operation query: %s", r.URL.RawQuery)
			}
			waits++
			if waits == 2 {
				w.WriteHeader(503)
				json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 503, "error": "database unavailable"})
				return
			}
			status := 100
			if waits == 3 {
				status = 200
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": map[string]interface{}{"status_code": status}})
		default:
			reads++
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}, Config{})
	var delays []time.Duration
	b.retryWait = func(_ context.Context, d time.Duration) error { delays = append(delays, d); return nil }
	if err := b.createInstance(context.Background(), "box", map[string]string{"name": "box"}); err != nil {
		t.Fatal(err)
	}
	if creates != 1 || waits != 3 || reads != 0 || !reflect.DeepEqual(delays, []time.Duration{15 * time.Second, 30 * time.Second}) {
		t.Fatalf("creates=%d waits=%d reads=%d delays=%v", creates, waits, reads, delays)
	}
}

func TestCreateRetryChecksExactInstanceBeforeResubmitting(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "completed-despite-error"}[exists], func(t *testing.T) {
			creates, reads := 0, 0
			var bodies []map[string]interface{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && r.URL.Path == "/1.0/instances" {
					creates++
					var body map[string]interface{}
					json.NewDecoder(r.Body).Decode(&body)
					bodies = append(bodies, body)
					if creates == 1 {
						w.WriteHeader(500)
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 500, "error": "Failed to begin transaction: context deadline exceeded"})
						return
					}
				} else if r.Method == "GET" && r.URL.Path == "/1.0/instances/box" {
					reads++
					// A transient exact-read failure must not permit another POST.
					if reads == 1 {
						w.WriteHeader(503)
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 503, "error": "unavailable"})
						return
					}
					if !exists {
						w.WriteHeader(404)
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "not found"})
						return
					}
				} else {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
				json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": map[string]string{"name": "box"}})
			}))
			defer srv.Close()
			b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{})
			b.retryWait = func(context.Context, time.Duration) error { return nil }
			body := map[string]interface{}{"name": "box", "config": map[string]string{"cloud-init.user-data": "same token", publicNICKey: "same IP"}}
			if err := b.createInstance(context.Background(), "box", body); err != nil {
				t.Fatal(err)
			}
			want := 2
			if exists {
				want = 1
			}
			if creates != want || reads != 2 {
				t.Fatalf("creates=%d reads=%d", creates, reads)
			}
			if len(bodies) == 2 && !reflect.DeepEqual(bodies[0], bodies[1]) {
				t.Fatal("retry changed creation metadata")
			}
		})
	}
}

func TestInstanceRetriesBoundedAndCancelable(t *testing.T) {
	b := New(&Client{}, Config{})
	var delays []time.Duration
	b.retryWait = func(_ context.Context, d time.Duration) error { delays = append(delays, d); return nil }
	attempts := 0
	err := b.retryInstanceStep(context.Background(), "box", "test", func() error {
		attempts++
		return &APIError{HTTPStatus: 503, Message: "unavailable"}
	})
	if err == nil || attempts != 6 || !reflect.DeepEqual(delays, instanceRetryDelays[:]) {
		t.Fatalf("attempts=%d delays=%v err=%v", attempts, delays, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.retryWait = nil
	attempts = 0
	err = b.retryInstanceStep(ctx, "box", "test", func() error {
		attempts++
		cancel()
		return &APIError{HTTPStatus: 503}
	})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("cancel: %d %v", attempts, err)
	}
	if err := waitInstanceRetry(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestInstanceRetryErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		code  int
		msg   string
		retry bool
	}{
		{400, "invalid device option", false}, {403, "permission denied", false},
		{404, "image not found", false}, {500, "Source image size exceeds specified volume size", false},
		{500, "Failed to begin transaction: context deadline exceeded", true},
		{400, "Instance is busy running a stop operation", true},
		{500, "Peer cluster member micro-06 is down", true}, {503, "unavailable", true},
	} {
		if got := transientInstanceError(&APIError{HTTPStatus: 200, ErrorCode: tc.code, Message: tc.msg}); got != tc.retry {
			t.Errorf("%s: retry=%t", tc.msg, got)
		}
	}
	b := New(&Client{}, Config{})
	b.retryWait = func(context.Context, time.Duration) error { t.Fatal("retried invalid config"); return nil }
	if err := b.retryInstanceStep(context.Background(), "box", "test", func() error { return &APIError{HTTPStatus: 400, Message: "invalid device"} }); err == nil {
		t.Fatal("accepted invalid config")
	}
}

func TestStartVerifiesPowerStateAndRetriesStoppedGuest(t *testing.T) {
	starts, reads := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.0/instances/box/state" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		metadata := map[string]string{}
		if r.Method == "PUT" {
			starts++
		} else if r.Method == "GET" {
			reads++
			metadata["status"] = "Stopped"
			if starts == 2 {
				metadata["status"] = "Running"
			}
		} else {
			t.Errorf("unexpected method %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": metadata})
	}))
	defer srv.Close()
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{})
	var delays []time.Duration
	b.retryWait = func(_ context.Context, d time.Duration) error { delays = append(delays, d); return nil }
	if err := b.startInstance(context.Background(), "box"); err != nil {
		t.Fatal(err)
	}
	if starts != 2 || reads != 2 || !reflect.DeepEqual(delays, []time.Duration{15 * time.Second}) {
		t.Fatalf("starts=%d reads=%d delays=%v", starts, reads, delays)
	}
}

func TestStartAlreadyRunningStillRequiresStateRead(t *testing.T) {
	for _, async := range []bool{false, true} {
		for _, allowed := range []bool{false, true} {
			t.Run(fmt.Sprintf("async=%t/allowed=%t", async, allowed), func(t *testing.T) {
				reads := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "PUT" && async {
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "async", "operation": "/1.0/operations/start"})
						return
					}
					if r.Method == "PUT" || r.URL.Path == "/1.0/operations/start/wait" {
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 500, "error": "The instance is already running"})
						return
					}
					reads++
					if allowed {
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": map[string]string{"status": "Running"}})
					} else {
						w.WriteHeader(403)
						json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 403, "error": "not authorized"})
					}
				}))
				defer srv.Close()
				b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}, Config{})
				b.retryWait = func(context.Context, time.Duration) error { t.Fatal("unexpected retry"); return nil }
				err := b.startInstance(context.Background(), "box")
				if (err == nil) != allowed || reads != 1 {
					t.Fatalf("state read not enforced: reads=%d err=%v", reads, err)
				}
			})
		}
	}
}
