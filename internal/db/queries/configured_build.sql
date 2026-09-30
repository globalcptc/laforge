-- name: CreateConfiguredBuild :one
INSERT INTO configured_build (repository_id, branch, environment_path, builder_config_name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetConfiguredBuild :one
SELECT * FROM configured_build WHERE id = $1;

-- name: ListConfiguredBuildsByRepository :many
SELECT * FROM configured_build WHERE repository_id = $1 ORDER BY created_at;

-- name: ListConfiguredBuildsByRepositoryAndBranch :many
-- Every configured build tracking this branch, whatever repo a webhook
-- push landed on -- the reconciliation step in internal/ingest walks all
-- of these, since more than one environment file can be configured to
-- follow the same branch.
SELECT * FROM configured_build WHERE repository_id = $1 AND branch = $2;

-- name: SetConfiguredBuildAutoDeploy :one
-- Per-branch auto-deploy: on, a new CI-passing commit is built AND
-- deployed; off, it's built (validated, free) and waits for a manual
-- deploy. Auto-build itself is always on -- see the reconciler.
UPDATE configured_build SET auto_deploy_enabled = $2 WHERE id = $1 RETURNING *;

-- name: SetConfiguredBuildCompetitionStarted :one
-- "Marking the competition started locks automatic deploys" -- the flag
-- itself is just this column; the lock behavior lives in the reconciler
-- (internal/ingest), which checks it before ever touching
-- current_content_revision_id.
UPDATE configured_build SET competition_started = $2 WHERE id = $1 RETURNING *;

-- name: SetConfiguredBuildCurrentRevision :one
UPDATE configured_build SET current_content_revision_id = $2 WHERE id = $1 RETURNING *;

-- name: ConfiguredBuildHasLiveBuilds :one
-- True when this configured build has any build that owns or is acting on real
-- infrastructure -- the same "live" set GetLiveDeployingBuildForConfiguredBuild
-- uses (deploying/building/finished), plus tearing_down (a teardown in flight).
-- Removal is refused while this holds so a config is never detached from a
-- running deployment; planned/failed/torn_down/purged builds do not block.
SELECT EXISTS (
    SELECT 1 FROM build
    WHERE configured_build_id = $1
      AND status IN ('deploying', 'building', 'finished', 'tearing_down')
) AS present;

-- name: DeleteConfiguredBuild :exec
-- Removes the configured build. build.configured_build_id is ON DELETE SET NULL,
-- so existing builds are kept as detached history rather than cascade-deleted.
DELETE FROM configured_build WHERE id = $1;
