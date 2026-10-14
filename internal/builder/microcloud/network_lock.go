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

// SQLNetworkLocker coordinates all runners using the same database. The lock
// deliberately spans configurations, projects, uplinks and API URLs: different
// endpoints or credentials can reach the same cluster, and there is no stable
// cluster identity in the builder configuration.
type SQLNetworkLocker struct {
	Pool *pgxpool.Pool
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
		err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('microcloud-networking', 0))`).Scan(&locked)
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

func isNetworkingMutation(method, path string) bool {
	if method == http.MethodGet {
		return false
	}
	path, _, _ = strings.Cut(path, "?")
	for _, collection := range []string{"/1.0/networks", "/1.0/network-acls"} {
		if path == collection || strings.HasPrefix(path, collection+"/") {
			return true
		}
	}
	// Instance creation, deletion, device updates and power changes also add or
	// remove OVN ports. Exec, files, snapshots and storage operations do not.
	if path == "/1.0/instances" {
		return true
	}
	if name, ok := strings.CutPrefix(path, "/1.0/instances/"); ok {
		return !strings.Contains(name, "/") || strings.HasSuffix(name, "/state")
	}
	return false
}
