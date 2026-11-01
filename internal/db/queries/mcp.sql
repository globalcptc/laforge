-- name: McpGetAPIKeyByHash :one
-- Queries for the read-only MCP server (internal/mcp), which runs them as the
-- laforge_mcp role (migrations/00081). That role is granted only the columns
-- named here, so every query in this file lists its columns explicitly -- a
-- `*` fails with "permission denied" -- and adding a column to a tool means
-- adding it to the migration's grants too.
SELECT k.id, k.account_id, k.name, k.prefix, k.scope, k.repository_ids, k.expires_at, k.revoked_at,
       a.github_login
FROM api_key k
JOIN account a ON a.id = k.account_id
WHERE k.token_hash = $1;

-- name: McpTouchAPIKey :exec
-- At most one write per key per minute, not one per call.
UPDATE api_key SET last_used_at = now()
WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < now() - interval '1 minute');

-- name: McpInsertAudit :exec
INSERT INTO mcp_audit (api_key_id, account_id, tool, args, build_id, status, error, duration_ms, result_bytes, remote_addr, user_agent, detail)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: McpListRepositories :many
SELECT id, github_owner, github_repo, created_at
FROM repository
WHERE id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY github_owner, github_repo;

-- name: McpGetBuildRepository :one
-- Which repository a build belongs to: the access check every build tool runs first.
SELECT cr.repository_id
FROM build b
JOIN content_revision cr ON cr.id = b.content_revision_id
WHERE b.id = $1;

-- name: McpListBuildsByRepository :many
SELECT b.id, b.status, b.environment_name, b.created_at, b.auto_built, b.reconcile_error,
       cr.commit_sha, cr.ref AS branch, cr.commit_message,
       cb.builder_config_name,
       (SELECT count(*) FROM team t WHERE t.build_id = b.id) AS team_count
FROM build b
JOIN content_revision cr ON cr.id = b.content_revision_id
LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE cr.repository_id = sqlc.arg(repository_id)
  AND (sqlc.narg(status)::text IS NULL OR b.status = sqlc.narg(status)::text)
ORDER BY b.created_at DESC
LIMIT sqlc.arg(max_rows);

-- name: McpGetBuild :one
SELECT b.id, b.status, b.environment_name, b.created_at, b.auto_built, b.reconcile_error,
       b.content_revision_id, b.configured_build_id,
       cr.repository_id, cr.commit_sha, cr.ref AS branch, cr.commit_message, cr.committed_at,
       r.github_owner, r.github_repo,
       cb.builder_config_name, cb.competition_started,
       bc.kind AS builder_kind
FROM build b
JOIN content_revision cr ON cr.id = b.content_revision_id
JOIN repository r ON r.id = cr.repository_id
LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
LEFT JOIN builder_config bc ON bc.name = cb.builder_config_name
WHERE b.id = $1;

-- name: McpGetBuildEnvironment :one
SELECT e.name, e.path, e.description, e.teams, e.start_at, e.stop_at, e.access
FROM environment e
WHERE e.content_revision_id = $1 AND e.name = $2;

-- name: McpListTeamsByBuild :many
SELECT id, team_number, access_state, access_override_state, access_override_until
FROM team
WHERE build_id = $1
ORDER BY team_number;

-- name: McpListObjectsByBuild :many
-- Every deployed object in a build with its team number and the agent's last
-- check-in, for the object listing and the build summary.
SELECT o.id, o.kind, o.object_name, o.as_name, o.network_name, o.status, o.power_state, o.power_state_checked_at,
       o.instance_location, o.public_address, o.external_ref, o.last_error, o.failure_kind, o.failed_at,
       o.blocked_on, o.deployed_at, o.updated_at, o.agent_platform, o.agent_checkin_overdue_at,
       t.team_number,
       s.last_heartbeat_at, s.boot_at, s.agent_started_at
FROM deployed_object o
JOIN team t ON t.id = o.team_id
LEFT JOIN agent_session s ON s.deployed_object_id = o.id
WHERE t.build_id = $1
ORDER BY t.team_number, COALESCE(o.as_name, o.object_name);

-- name: McpFindBuildsByIDPrefix :many
-- Lets an agent name a build by the short id people paste ("9531b5b4"). Two
-- rows back means the prefix is ambiguous.
SELECT id FROM build
WHERE id::text LIKE sqlc.arg(prefix)::text || '%'
ORDER BY created_at DESC
LIMIT 2;

-- name: McpFindObjectInBuild :many
-- An object named by id, id prefix, or name (optionally within one team).
SELECT o.id, o.kind, o.object_name, o.as_name, t.team_number
FROM deployed_object o
JOIN team t ON t.id = o.team_id
WHERE t.build_id = sqlc.arg(build_id)
  AND (sqlc.arg(team)::int = 0 OR t.team_number = sqlc.arg(team)::int)
  AND (o.id::text LIKE lower(sqlc.arg(ref)::text) || '%'
       OR lower(o.object_name) = lower(sqlc.arg(ref)::text)
       OR lower(COALESCE(o.as_name, '')) = lower(sqlc.arg(ref)::text))
ORDER BY t.team_number
LIMIT 50;

-- name: McpHeartbeatGaps :many
-- Silences between consecutive check-ins of each agent in a build since a
-- time, longest first.
WITH hb AS (
    SELECT h.deployed_object_id, h.created_at,
           lag(h.created_at) OVER (PARTITION BY h.deployed_object_id ORDER BY h.created_at) AS prev_at
    FROM agent_heartbeat h
    JOIN deployed_object o ON o.id = h.deployed_object_id
    JOIN team t ON t.id = o.team_id
    WHERE t.build_id = sqlc.arg(build_id) AND h.created_at >= sqlc.arg(since)
)
SELECT hb.deployed_object_id, o.object_name, o.as_name, o.instance_location, t.team_number,
       hb.prev_at::timestamptz AS gap_start, hb.created_at AS gap_end,
       extract(epoch FROM hb.created_at - hb.prev_at)::float8 AS gap_seconds
FROM hb
JOIN deployed_object o ON o.id = hb.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE hb.prev_at IS NOT NULL
  AND hb.created_at - hb.prev_at >= make_interval(secs => sqlc.arg(min_gap_seconds)::float8)
ORDER BY gap_seconds DESC
LIMIT sqlc.arg(max_rows);

-- name: McpHeartbeatCounts :many
-- Check-ins per object in the window, for "who's been quiet" next to the gaps.
SELECT h.deployed_object_id, count(*)::bigint AS beats, max(h.created_at)::timestamptz AS last_at
FROM agent_heartbeat h
JOIN deployed_object o ON o.id = h.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE t.build_id = sqlc.arg(build_id) AND h.created_at >= sqlc.arg(since)
GROUP BY h.deployed_object_id;

-- name: McpSearchAgentTasks :many
-- Agent tasks across a build -- authored steps, ad-hoc commands and scheduled
-- runs -- filtered by status, kind and time, newest change first.
SELECT a.id, a.deployed_object_id, o.object_name, o.as_name, o.instance_location, t.team_number,
       a.step_index, a.command, a.payload, a.status, a.attempts, a.max_attempts, a.last_error,
       right(COALESCE(a.output, ''), sqlc.arg(output_tail)::int) AS output_tail,
       length(COALESCE(a.output, ''))::int AS output_bytes,
       a.step_script, a.milestone, a.wait_for, a.ad_hoc, a.scheduled_task_id, a.ignore_errors,
       a.started_at, a.finished_at, a.created_at, a.updated_at
FROM agent_task a
JOIN deployed_object o ON o.id = a.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE t.build_id = sqlc.arg(build_id)
  AND a.validation_run_id IS NULL
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR a.status = ANY(sqlc.arg(statuses)::text[]))
  AND (sqlc.narg(ad_hoc)::bool IS NULL OR a.ad_hoc = sqlc.narg(ad_hoc)::bool)
  AND (sqlc.arg(command)::text = '' OR a.command = sqlc.arg(command)::text)
  AND (sqlc.arg(team)::int = 0 OR t.team_number = sqlc.arg(team)::int)
  AND a.updated_at >= sqlc.arg(since)
ORDER BY a.updated_at DESC
LIMIT sqlc.arg(max_rows);

-- name: McpGetAgentTask :one
SELECT a.id, a.deployed_object_id, o.object_name, o.as_name, t.team_number, t.build_id,
       a.step_index, a.command, a.payload, a.status, a.attempts, a.output, a.last_error,
       a.step_script, a.milestone, a.ad_hoc, a.started_at, a.finished_at, a.updated_at
FROM agent_task a
JOIN deployed_object o ON o.id = a.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE a.id = $1;

-- name: McpListRunnerTasks :many
-- The runner's deploy/destroy/power queue for a build.
SELECT k.id, k.kind, k.status, k.attempts, k.permanent, k.lease_owner, k.lease_expires_at, k.last_error,
       k.created_at, k.updated_at, o.object_name, o.as_name, t.team_number
FROM task k
LEFT JOIN deployed_object o ON o.id = k.deployed_object_id
LEFT JOIN team t ON t.id = o.team_id
WHERE k.build_id = sqlc.arg(build_id)
  AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR k.status = ANY(sqlc.arg(statuses)::text[]))
ORDER BY k.updated_at DESC
LIMIT sqlc.arg(max_rows);

-- name: McpListShellSessions :many
-- Who opened a terminal on which object, and when.
SELECT s.id, s.status, s.client_addr, s.started_at, s.ended_at, a.github_login AS opened_by,
       o.object_name, o.as_name, t.team_number
FROM shell_session s
JOIN deployed_object o ON o.id = s.deployed_object_id
JOIN team t ON t.id = o.team_id
LEFT JOIN account a ON a.id = s.opened_by_account_id
WHERE t.build_id = $1
ORDER BY s.started_at DESC
LIMIT 200;

-- name: McpListContentRevisions :many
-- The commits LaForge has ingested for a repository, with their validation result.
SELECT id, commit_sha, ref, valid, validation_errors, validated_at, commit_message, commit_author, committed_at, created_at
FROM content_revision
WHERE repository_id = sqlc.arg(repository_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(max_rows);

-- name: McpGetConfiguredBuildRepository :one
SELECT repository_id FROM configured_build WHERE id = $1;

-- name: McpGetRevisionRepository :one
SELECT repository_id FROM content_revision WHERE id = $1;

-- name: McpGetImageBuildBuilder :one
SELECT bc.name FROM builder_image_build ib JOIN builder_config bc ON bc.id = ib.builder_config_id WHERE ib.id = $1;
