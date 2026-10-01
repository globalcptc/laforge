-- Split a host/container's infrastructure deploy from its step execution.
-- The box (image/instance) now deploys ahead of time, regardless of
-- depends_on, but its authored `steps:` must not run until every dependency
-- has fully FINISHED configuring (a domain controller must be a working DC,
-- not merely a booted Windows box). The orchestrator materializes an object's
-- steps only once its dependencies are finished; this column records WHEN that
-- happened, so the lifecycle advancer can tell "no steps because none were
-- authored" (finish immediately) apart from "no steps because they haven't
-- been materialized yet" (keep building, still waiting on a dependency). NULL
-- until materialized; reset to NULL on a redeploy so the rebuilt instance
-- re-materializes.

-- +goose Up
ALTER TABLE deployed_object ADD COLUMN steps_materialized_at timestamptz;

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN steps_materialized_at;
