// laforge-migrate runs the real migrations/*.sql against a real
// Postgres -- the docker-compose "migrate" service's own image (see
// Dockerfile's migrate stage), so `docker compose up` needs nothing
// installed on the host beyond Docker itself.
//
// Deliberately its own tiny binary using goose as a *library*
// (github.com/pressly/goose/v3), not `go install`ing goose's own CLI
// (github.com/pressly/goose/v3/cmd/goose): the CLI blank-imports a
// driver for every database goose supports (MySQL, ClickHouse, libsql,
// YDB, MSSQL, SQLite, ...) so it can dispatch on a `-dir X <dialect>`
// flag, which drags in Azure SDK, gRPC, and OpenTelemetry transitively
// -- a real, measured difference (`go install .../cmd/goose` vs. `go get
// github.com/pressly/goose/v3`) found while actually building this
// image, not a guess. This binary only ever needs Postgres, so it only
// imports pgx's own database/sql adapter.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: laforge-migrate <postgres-dsn> <up|down|status|...> [args...]")
	}
	dsn, command, args := os.Args[1], os.Args[2], os.Args[3:]

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations" // matches running this from the repo root directly
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("setting dialect: %v", err)
	}
	if err := goose.RunContext(context.Background(), command, db, dir, args...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}
