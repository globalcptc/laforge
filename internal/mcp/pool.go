package mcp

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// Role is the Postgres role the MCP server must connect as (migrations/00081).
const Role = "laforge_mcp"

// maxConns keeps the MCP server's share of the database small: a few
// connections, whatever agents ask for.
const maxConns = 3

// OpenPool opens the MCP server's own pool and refuses one that isn't
// connected as laforge_mcp. A pool on the main, read-write role would quietly
// throw away the read-only guarantee, so it's a startup error, not a warning.
func OpenPool(ctx context.Context, connString string) (*db.Queries, *pgxpool.Pool, error) {
	q, pool, err := db.OpenWithMaxConns(ctx, connString, maxConns)
	if err != nil {
		return nil, nil, err
	}
	var user string
	if err := pool.QueryRow(ctx, "SELECT current_user").Scan(&user); err != nil {
		pool.Close()
		return nil, nil, err
	}
	if user != Role {
		pool.Close()
		return nil, nil, fmt.Errorf("MCP_DATABASE_URL connects as %q; it must connect as %q, the read-only role", user, Role)
	}
	return q, pool, nil
}
