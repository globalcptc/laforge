-- Per-builder "docker base image" build job (migration 00024).

-- name: CreateImageBuild :one
-- Queue a new build for a builder. Status starts 'pending'; the orchestrator
-- leases it with StartImageBuild.
INSERT INTO builder_image_build (builder_config_id, kind)
VALUES ($1, $2)
RETURNING *;

-- name: GetImageBuild :one
SELECT * FROM builder_image_build WHERE id = $1;

-- name: ListImageBuildsByBuilder :many
SELECT * FROM builder_image_build
WHERE builder_config_id = $1
ORDER BY created_at DESC
LIMIT 50;

-- name: LatestImageBuildByBuilder :one
SELECT * FROM builder_image_build
WHERE builder_config_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListPendingImageBuilds :many
-- Oldest first, so the orchestrator drains the queue in order.
SELECT * FROM builder_image_build
WHERE status = 'pending'
ORDER BY created_at ASC;

-- name: StartImageBuild :one
-- Lease a pending build: flip it to running. Only succeeds while it's still
-- pending, so two orchestrator replicas can't both run the same build (the
-- loser gets ErrNoRows).
UPDATE builder_image_build
SET status = 'running', started_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: FinishImageBuildSuccess :exec
UPDATE builder_image_build
SET status = 'succeeded', image_fingerprint = $2, error = '', finished_at = now()
WHERE id = $1;

-- name: FinishImageBuildFailed :exec
UPDATE builder_image_build
SET status = 'failed', error = $2, finished_at = now()
WHERE id = $1;

-- name: AppendImageBuildLog :exec
-- One streamed line. seq is computed here (next after the current max for
-- this build), so the caller doesn't track it; a single orchestrator writes a
-- build's log sequentially, and the (build_id, seq) unique index guards races.
INSERT INTO builder_image_build_log (build_id, seq, line)
VALUES ($1, (SELECT COALESCE(MAX(seq), 0) + 1 FROM builder_image_build_log WHERE build_id = $1), $2);

-- name: ListImageBuildLog :many
SELECT seq, line, created_at FROM builder_image_build_log
WHERE build_id = $1
ORDER BY seq ASC;

-- name: ListImageBuildLogSince :many
-- Everything after seq $2 -- the live console's tail.
SELECT seq, line, created_at FROM builder_image_build_log
WHERE build_id = $1 AND seq > $2
ORDER BY seq ASC;

-- name: GetBuilderConfigByID :one
SELECT * FROM builder_config WHERE id = $1;

-- name: SetBuilderConfigDockerBaseFingerprint :exec
UPDATE builder_config
SET docker_base_fingerprint = $2, updated_at = now()
WHERE id = $1;
