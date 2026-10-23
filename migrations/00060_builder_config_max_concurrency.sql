-- +goose Up
-- Per-builder cap on how many of its tasks may be leased (deploying) at once, so
-- a hoster with limited capacity (e.g. a MicroCloud cluster) can be turned down
-- independently of the global RUNNER_CONCURRENCY worker pool. 0 = unlimited (the
-- prior behavior). Enforced in the LeaseTask query as a soft cap.
ALTER TABLE builder_config ADD COLUMN max_concurrency integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE builder_config DROP COLUMN max_concurrency;
