package microcloud

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type deadlineTransport func(*http.Request) (*http.Response, error)

func (f deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClusterReadsAndWritesHonorConfiguredBudgetAndCancellation(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, parentBudget := range []time.Duration{0, time.Second} {
			ctx, cancel := context.WithCancel(context.Background())
			want := 2 * time.Minute
			if parentBudget > 0 {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), parentBudget)
				want = parentBudget
			}
			client := &Client{BaseURL: "https://cluster.example", OperationTimeout: 2 * time.Minute}
			client.HTTPClient = &http.Client{Transport: deadlineTransport(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				remaining := time.Until(deadline)
				if !ok || remaining > want || remaining < want-500*time.Millisecond {
					t.Errorf("%s parent=%s: request budget %s, want %s", method, parentBudget, remaining, want)
				}
				cancel()
				if r.Context().Err() != context.Canceled {
					t.Error("caller cancellation did not reach the request")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"type":"sync","metadata":{}}`))}, nil
			})}
			_, err := client.do(ctx, method, "/1.0/networks/lab", nil)
			cancel()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
