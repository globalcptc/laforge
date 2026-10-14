-- +goose Up
-- MicroCloud placement: stop LaForge from choosing/blocklisting cluster members
-- and instead target an LXD cluster group by name, letting LXD/MicroCloud
-- schedule within it. The host blocklist (00058) and the member-selection it
-- drove are removed; a single cluster-group name (default "default", the whole
-- cluster) replaces them.
ALTER TABLE builder_config DROP COLUMN microcloud_host_blocklist;
ALTER TABLE builder_config ADD COLUMN microcloud_cluster_group text NOT NULL DEFAULT 'default';

-- +goose Down
ALTER TABLE builder_config DROP COLUMN microcloud_cluster_group;
ALTER TABLE builder_config ADD COLUMN microcloud_host_blocklist jsonb NOT NULL DEFAULT '[]'::jsonb;
