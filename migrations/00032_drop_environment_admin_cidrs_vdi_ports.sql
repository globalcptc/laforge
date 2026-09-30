-- Drop the environment `admin_cidrs` and `vdi_ports` columns. Both were
-- authored, validated and stored but read by nothing (S5 write-only fields):
-- `vdi_ports` targeted a "shared VDI network" concept that no longer exists in
-- 3.0 (reachability is now per-network `visible_from` ACLs), and `admin_cidrs`
-- ("admin access regardless of the access schedule") can't be expressed by the
-- current access model, which removes a team's NIC entirely on close. Both are
-- removed from the schema, loader and ingest; content tables are repopulated
-- from content on every ingest, so no data preservation is needed.

-- +goose Up

ALTER TABLE environment DROP COLUMN admin_cidrs;
ALTER TABLE environment DROP COLUMN vdi_ports;

-- +goose Down

ALTER TABLE environment ADD COLUMN admin_cidrs jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE environment ADD COLUMN vdi_ports jsonb NOT NULL DEFAULT '[]'::jsonb;
