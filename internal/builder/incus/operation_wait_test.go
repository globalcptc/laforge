package incus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitForOperationRequiresCompletion(t *testing.T) {
	for _, test := range []struct {
		status string
		code   int
	}{
		{"Pending", 105}, {"Running", 103}, {"Cancelling", 104},
		{"Success", 200}, {"Failure", 400}, {"Cancelled", 401},
	} {
		t.Run(test.status, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type": "sync", "metadata": operationResult{Status: test.status, StatusCode: test.code},
				})
			}))
			defer srv.Close()
			c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client(), OperationTimeout: time.Second}
			_, err := c.waitForOperation(context.Background(), "/1.0/operations/test")
			if test.code < 200 && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("unfinished operation returned %v, want deadline exceeded", err)
			}
			if test.code == 200 && err != nil {
				t.Fatalf("successful operation: %v", err)
			}
			if test.code >= 400 {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.ErrorCode != test.code {
					t.Fatalf("failed operation lost its error: %v", err)
				}
			}
		})
	}
}

type failedOperationWaitTransport struct{}

func (failedOperationWaitTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

func TestWaitForOperationRetainsPathOnTransportFailure(t *testing.T) {
	c := &Client{BaseURL: "https://incus.invalid", HTTPClient: &http.Client{Transport: failedOperationWaitTransport{}}, OperationTimeout: time.Second}
	_, err := c.waitForOperation(context.Background(), "/1.0/operations/delete")
	var pending *operationPendingError
	if !errors.As(err, &pending) || pending.Path != "/1.0/operations/delete" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lost the interrupted operation: %v", err)
	}
}
