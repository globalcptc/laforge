package gateway

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// The HTTP sink ships enqueued records as newline-delimited JSON to the
// configured endpoint, with configured headers and static labels applied.
func TestHTTPSinkPostsNDJSON(t *testing.T) {
	var mu sync.Mutex
	var got []LogRecord
	var gotAuth string
	done := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/x-ndjson" {
			t.Errorf("content-type = %q, want application/x-ndjson", ct)
		}
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		sc := bufio.NewScanner(r.Body)
		for sc.Scan() {
			var rec LogRecord
			if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
				t.Errorf("line is not valid JSON: %v (%q)", err, sc.Text())
				continue
			}
			got = append(got, rec)
		}
		n := len(got)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		if n >= 2 {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	}))
	defer srv.Close()

	sink := NewHTTPSink(srv.URL,
		map[string]string{"Authorization": "Splunk tok"},
		map[string]string{"env": "cptc-test"})
	sink.Enqueue(LogRecord{ObjectID: "o1", Team: 3, Object: "scoreboard", Stream: "stdout", Line: "hello"})
	sink.Enqueue(LogRecord{ObjectID: "o1", Team: 3, Object: "scoreboard", Stream: "stderr", Line: "oops"})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sink did not deliver both records in time")
	}
	sink.Close()

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "Splunk tok" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Splunk tok")
	}
	if len(got) < 2 {
		t.Fatalf("got %d records, want >= 2", len(got))
	}
	for _, rec := range got {
		if rec.Labels["env"] != "cptc-test" {
			t.Errorf("record missing static label env=cptc-test: %+v", rec)
		}
		if rec.Object != "scoreboard" {
			t.Errorf("record object = %q, want scoreboard", rec.Object)
		}
	}
}

// A down backend must not block Enqueue (which runs on the agent-facing
// connection) -- records are dropped, not back-pressured. Flooding past the
// buffer cap returns promptly rather than hanging.
func TestHTTPSinkEnqueueNeverBlocks(t *testing.T) {
	// Point at an unroutable address so posts fail; Enqueue must still return.
	sink := NewHTTPSink("http://127.0.0.1:1/never", nil, nil)
	defer sink.Close()
	doneCh := make(chan struct{})
	go func() {
		for i := 0; i < logSinkQueueCap*2; i++ {
			sink.Enqueue(LogRecord{ObjectID: "o1", Stream: "stdout", Line: "x"})
		}
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Enqueue blocked under a full buffer / failing backend")
	}
}

// TestFallbackTarget covers the container_logs -> gateway-fallback mapping: the
// splunk driver maps to the HEC raw endpoint with the token as an auth header,
// a missing url is rejected, and any non-splunk driver is unsupported.
func TestFallbackTarget(t *testing.T) {
	url, headers, ok := fallbackTarget("splunk", map[string]string{
		"splunk-url":   "https://splunk.example:8088/",
		"splunk-token": "tok-123",
		"splunk-index": "cptc",
	})
	if !ok {
		t.Fatal("splunk driver should be supported")
	}
	if url != "https://splunk.example:8088/services/collector/raw" {
		t.Fatalf("url = %q, want the HEC raw endpoint (trailing slash trimmed)", url)
	}
	if headers["Authorization"] != "Splunk tok-123" {
		t.Fatalf("Authorization = %q, want \"Splunk tok-123\"", headers["Authorization"])
	}

	if _, _, ok := fallbackTarget("splunk", map[string]string{"splunk-token": "x"}); ok {
		t.Error("splunk without splunk-url must be unsupported")
	}
	if _, _, ok := fallbackTarget("fluentd", map[string]string{"fluentd-address": "x:24224"}); ok {
		t.Error("a non-splunk driver must be unsupported on the gateway fallback")
	}
}
