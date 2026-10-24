`schema.sql` in this directory is **not hand-written and not a source of
truth** — it's a `pg_dump --schema-only` of `laforge_dev` after the real
`migrations/*.sql` were applied with goose, regenerate it the same way
after any migration change:

```
pg_dump --schema-only -h /tmp -p 5432 -d laforge_dev --no-owner --no-privileges \
  | grep -vE '^\\|^SET |^SELECT pg_catalog|^$' > internal/db/schema.sql
```

It exists only so sqlc has plain `CREATE TABLE` statements to read.
Pointing sqlc directly at `migrations/*.sql` doesn't work here: goose
migration files carry both the `-- +goose Up` and `-- +goose Down` sections
in one file, and since `-- +goose Down` is just a SQL comment, sqlc parses
straight through it and executes the `DROP TABLE` statements right after
the `CREATE TABLE` statements that precede them — it has no goose-aware
mode that stops at the boundary. Dumping the real, already-migrated
database sidesteps the whole problem and has the added benefit of being
schema sqlc reads verified against what's actually running, not assumed
from reading the migration files by eye.
