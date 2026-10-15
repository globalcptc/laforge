-- +goose Up
-- Per-Incus-host cap on concurrent instance deploys (create/copy, config drive,
-- first start) for a kind "incus" pool. max_concurrency caps the whole builder;
-- this caps each pool member, where the contention actually is -- one team's
-- ~20 VMs created at once on one host made each deploy crawl. Deploys over the
-- cap queue in the runner and are admitted highest critical-path priority first
-- (hosts others depends_on). 0 = unbounded (the prior behavior).
ALTER TABLE builder_config ADD COLUMN incus_host_max_concurrency integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE builder_config DROP COLUMN incus_host_max_concurrency;
