package microcloud

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NetworkLocker serializes MicroCloud networking across builder instances.
type NetworkLocker interface {
	Lock(context.Context) (func(), error)
}

// SQLNetworkLocker coordinates all runners using the same database. It
// serializes OVN NETWORK operations (creating/deleting the network itself),
// which the cluster's OVN control plane cannot do concurrently -- NOT instance
// creation or ACLs, which run at the builder's own pool concurrency. The lock is
// keyed PER CLUSTER (Key, the builder_config name), so operations on different
// configured clusters never wait on each other; an empty Key falls back to one
// shared lock.
type SQLNetworkLocker struct {
	Pool *pgxpool.Pool
	Key  string
}

func (l SQLNetworkLocker) Lock(ctx context.Context) (func(), error) {
	if l.Pool == nil {
		return nil, fmt.Errorf("MicroCloud networking lock requires a database")
	}
	for {
		tx, err := l.Pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		release := func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tx.Rollback(cleanup); err != nil {
				slog.Warn("releasing MicroCloud networking lock", "error", err)
			}
		}
		var locked bool
		err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('microcloud-networking:' || $1, 0))`, l.Key).Scan(&locked)
		if err == nil && locked {
			return release, nil
		}
		release()
		if err != nil {
			return nil, err
		}
		// Waiting workers must not exhaust the pool: the lock holder may need
		// another connection (for example, to reserve a public NIC address).
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

type localNetworkLocker chan struct{}

func (l localNetworkLocker) Lock(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case l <- struct{}{}:
		return func() { <-l }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Direct library users without a database still coordinate across clients in
// this process. The production builder resolver always installs the SQL lock.
var processNetworkLock = make(localNetworkLocker, 1)

type networkLockContextKey struct{}

// lockNetworking lets a sequential, multi-request operation retain the lock.
// Nested client mutations inherit it; never use the returned context for
// concurrent work or after calling release.
func (c *Client) lockNetworking(ctx context.Context) (context.Context, func(), error) {
	if ctx.Value(networkLockContextKey{}) != nil {
		return ctx, func() {}, nil
	}
	locker := c.NetworkLock
	if locker == nil {
		locker = processNetworkLock
	}
	release, err := locker.Lock(ctx)
	if err != nil {
		return ctx, nil, fmt.Errorf("waiting for MicroCloud networking lock: %w", err)
	}
	return context.WithValue(ctx, networkLockContextKey{}, true), release, nil
}

// isNetworkingMutation reports whether a request is an OVN NETWORK lifecycle
// operation that the cluster's control plane cannot run concurrently -- creating,
// updating or deleting the network itself (/1.0/networks). That is the ONLY
// MicroCloud operation that must be serialized. Instance creation/deletion and
// ACLs do add/remove OVN ports, but the cluster handles those concurrently;
// serializing them (as an earlier, broader version did) needlessly made every
// host deploy take turns. Those now run at the builder's own pool concurrency.
func isNetworkingMutation(method, path string) bool {
	if method == http.MethodGet {
		return false
	}
	path, _, _ = strings.Cut(path, "?")
	return path == "/1.0/networks" || strings.HasPrefix(path, "/1.0/networks/")
}
