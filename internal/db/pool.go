package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open connects to Postgres and returns a ready-to-use *Queries backed by
// a connection pool. *pgxpool.Pool satisfies the sqlc-generated DBTX
// interface directly (Exec/Query/QueryRow), so no adapter is needed.
func Open(ctx context.Context, connString string) (*Queries, *pgxpool.Pool, error) {
	return OpenWithMaxConns(ctx, connString, 0)
}

// OpenWithMaxConns is Open with an explicit pool size, for a process that runs
// many concurrent workers (the runner's -concurrency): pgxpool's default of
// max(4, NumCPU) starves once worker count exceeds it. maxConns <= 0 keeps the
// default (or whatever pool_max_conns the connection string already sets).
func OpenWithMaxConns(ctx context.Context, connString string, maxConns int32) (*Queries, *pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}
	return New(pool), pool, nil
}
