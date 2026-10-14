package microcloud

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/jackc/pgx/v5/pgxpool"
)

func networkTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LAFORGE_TEST_DATABASE_URL to exercise cross-runner networking locks")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestSQLNetworkLockAcrossIndependentPools(t *testing.T) {
	// Independent pools/lockers model separately resolved builders in different
	// runner processes. No process-local mutex participates in this test.
	testConcurrentNetworkDeploys(t, []NetworkLocker{
		SQLNetworkLocker{Pool: networkTestPool(t)}, SQLNetworkLocker{Pool: networkTestPool(t)},
	})
}

func TestNetworkLockCancellationAndRelease(t *testing.T) {
	for _, kind := range []string{"local", "sql"} {
		t.Run(kind, func(t *testing.T) {
			var first, second NetworkLocker = make(localNetworkLocker, 1), nil
			second = first
			if kind == "sql" {
				first = SQLNetworkLocker{Pool: networkTestPool(t)}
				second = SQLNetworkLocker{Pool: networkTestPool(t)}
			}
			ctx, cancel := context.WithCancel(context.Background())
			release, err := first.Lock(ctx)
			if err != nil {
				t.Fatal(err)
			}
			// Cancellation must not make cleanup use an already canceled context.
			cancel()
			waiter, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer stop()
			if unlock, err := second.Lock(waiter); !errors.Is(err, context.DeadlineExceeded) {
				if unlock != nil {
					unlock()
				}
				release()
				t.Fatalf("waiter did not cancel: %v", err)
			}
			release()
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			release, err = second.Lock(ctx)
			if err != nil {
				t.Fatalf("lock leaked: %v", err)
			}
			release()
		})
	}
}

func TestNetworkingMutationsShareLock(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		networkReply(w, nil)
	}))
	defer srv.Close()
	holder := &Client{}
	_, release, err := holder.lockNetworking(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	// Separate Client, default process lock. Cleanup ACLs and instance ports use
	// the same lock as network creation, including targeted cluster requests.
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	for _, route := range []struct{ method, path string }{
		{"POST", "/1.0/networks"}, {"DELETE", "/1.0/networks/test"},
		{"PATCH", "/1.0/networks/test"}, {"POST", "/1.0/networks/test/peers"},
		{"POST", "/1.0/networks/test/forwards"}, {"POST", "/1.0/network-acls"},
		{"PUT", "/1.0/network-acls/test"}, {"DELETE", "/1.0/network-acls/test"},
		{"POST", "/1.0/instances?target=member-a"}, {"PATCH", "/1.0/instances/test"},
		{"DELETE", "/1.0/instances/test"}, {"PUT", "/1.0/instances/test/state"},
	} {
		t.Run(route.method+route.path, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			if _, err := c.do(ctx, route.method, route.path, nil); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("mutation bypassed lock: %v", err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("%d requests bypassed held lock", requests.Load())
	}
	for _, route := range []struct{ method, path string }{
		{"GET", "/1.0/networks"}, {"POST", "/1.0/instances/test/exec"},
		{"POST", "/1.0/storage-pools/remote/volumes/custom"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := c.do(ctx, route.method, route.path, nil)
		cancel()
		if err != nil {
			t.Errorf("unrelated operation blocked: %v", err)
		}
	}
}

func TestNetworkingLockHeldUntilAsyncOperationCompletes(t *testing.T) {
	waiting, finish := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1.0/operations/create/wait" {
			close(waiting)
			<-finish
			networkReply(w, map[string]any{"status": "Success", "status_code": 200})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"async","operation":"/1.0/operations/create","metadata":{}}`))
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	done := make(chan error, 1)
	go func() { _, err := c.post(context.Background(), "/1.0/instances", nil); done <- err }()
	select {
	case <-waiting:
	case err := <-done:
		t.Fatalf("mutation failed before async wait: %v", err)
	case <-time.After(5 * time.Second):
		close(finish)
		t.Fatal("mutation never reached async wait")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, release, err := (&Client{}).lockNetworking(ctx)
	if release != nil {
		release()
	}
	close(finish)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("lock released before async completion: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestNetworkLockCoversPostCreateReadback(t *testing.T) {
	reading, finish := make(chan struct{}), make(chan struct{})
	posted := false
	spec := builder.NetworkSpec{ExternalName: "locked-readback", CIDR: "10.0.1.0/24"}
	n := testAllocatedNetwork(networkName("", spec.ExternalName), "default", "10.10.44.35")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			posted = true
			networkReply(w, nil)
		case r.URL.Path == "/1.0/networks":
			close(reading)
			<-finish
			networkReply(w, []allocatedNetwork{n})
		case posted:
			networkReply(w, n)
		default:
			networkFailure(w, 404, "not found")
		}
	}))
	defer srv.Close()
	b := New(&Client{BaseURL: srv.URL, HTTPClient: srv.Client()}, Config{OVNUplinkNetwork: "UPLINK"})
	done := make(chan error, 1)
	go func() { _, err := b.DeployNetwork(context.Background(), spec); done <- err }()
	select {
	case <-reading:
	case err := <-done:
		t.Fatalf("deployment failed before read-back: %v", err)
	case <-time.After(5 * time.Second):
		close(finish)
		t.Fatal("deployment never reached read-back")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, release, err := (&Client{}).lockNetworking(ctx)
	if release != nil {
		release()
	}
	close(finish)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("lock released before allocation check: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
