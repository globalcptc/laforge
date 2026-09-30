-- Replace the boolean `vdi_visible` on a network with `visible_from`, a JSON
-- array of the other network names allowed to reach it (an ACL allowlist the
-- builder enforces on the OVN routing host). The old boolean could only express
-- "the VDI network reaches this"; the array lets any network's reachability be
-- defined explicitly (`visible_from: [vdi, client]`). Content tables are
-- repopulated from content on every ingest, so no data preservation is needed.

-- +goose Up

ALTER TABLE network ADD COLUMN visible_from jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE network DROP COLUMN vdi_visible;

-- +goose Down

ALTER TABLE network ADD COLUMN vdi_visible boolean NOT NULL DEFAULT false;
ALTER TABLE network DROP COLUMN visible_from;
