-- +goose Up
-- A MicroCloud builder's host blocklist: cluster member names LaForge must NOT
-- place instances on (a member under maintenance, GPU-reserved, flaky, ...). When
-- non-empty, the builder explicitly targets an allowed member for each instance
-- instead of letting LXD auto-schedule (which could land on a blocked member).
-- Empty = no restriction, LXD schedules as before. A JSON array of member names.
ALTER TABLE builder_config ADD COLUMN microcloud_host_blocklist jsonb NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE builder_config DROP COLUMN microcloud_host_blocklist;
