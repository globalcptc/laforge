-- name: UpsertAgentSession :one
-- First heartbeat creates the row; every later one just bumps
-- last_heartbeat_at (and cert_fingerprint, in case a host was ever
-- re-provisioned with a new identity -- shouldn't happen mid-build, but
-- recording the fingerprint that's actually presenting is more honest
-- than assuming it never changes).
INSERT INTO agent_session (deployed_object_id, cert_fingerprint)
VALUES ($1, $2)
ON CONFLICT (deployed_object_id) DO UPDATE
    SET cert_fingerprint = EXCLUDED.cert_fingerprint, last_heartbeat_at = now()
RETURNING *;

-- name: GetAgentSessionByDeployedObject :one
SELECT * FROM agent_session WHERE deployed_object_id = $1;

-- name: ListAgentSessions :many
SELECT * FROM agent_session ORDER BY last_heartbeat_at DESC;

-- name: CreateAgentHeartbeat :exec
-- The append-only counterpart to UpsertAgentSession -- called alongside
-- it, never instead of it, on every real heartbeat the gateway receives
-- (see internal/gateway/server.go's handleHeartbeat and
-- migrations/00007's own doc comment for why both exist). :exec, not
-- :one -- no RETURNING, deliberately: Postgres requires SELECT
-- privilege on a table for ANY RETURNING clause to work, even on an
-- INSERT-only grant (found live: laforge_gateway's real INSERT-only
-- grant made exactly this query fail with "permission denied," not a
-- theoretical concern) -- so this query only ever needs the INSERT
-- privilege migrations/00007 actually grants, matching its own "no
-- SELECT" design intent instead of contradicting it.
INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr, next_poll_ms)
VALUES ($1, $2, $3, $4);

-- name: DeleteAgentHeartbeatsOlderThan :execrows
-- The real answer to migrations/00007's own "deliberately unaddressed:
-- retention" note -- a time-bounded retention policy, since "folding
-- into purge artifacts" (that migration's other named option) means
-- folding into a build lifecycle action that doesn't exist yet
-- (not yet implemented). Deliberately NOT scoped to one build: this
-- runs on a wall-clock age cutoff across every build, the same way a log
-- rotation policy would, so it needs no per-build bookkeeping and keeps
-- working correctly for a build that's still deployed (a live event's
-- own heartbeats past the cutoff are exactly as disposable as a torn-down
-- one's -- "live troubleshooting" per ListAgentHeartbeatsByObject's own
-- comment only ever needs recent history, never the full lifetime of a
-- multi-day event). :execrows so a caller can log how many it actually
-- removed.
DELETE FROM agent_heartbeat WHERE created_at < $1;

-- name: ListAgentHeartbeatsByObject :many
-- One object's own heartbeat history, newest first -- "live
-- troubleshooting" (per-object log panel)
-- and any future rules-checking query (e.g. spotting a remote_addr
-- change) both read from here. $2 caps how far back, since this table
-- has no retention policy yet and could in principle be very large.
SELECT * FROM agent_heartbeat WHERE deployed_object_id = $1 ORDER BY created_at DESC LIMIT $2;

-- name: ListAgentHeartbeatsByBuild :many
-- The build-wide time series behind the dashboard's real "agent
-- check-ins over time" chart -- joins through deployed_object -> team,
-- same reasoning as ListAgentSessionsByBuild: this table carries no
-- denormalized build_id of its own.
SELECT agent_heartbeat.* FROM agent_heartbeat
JOIN deployed_object ON deployed_object.id = agent_heartbeat.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1
ORDER BY agent_heartbeat.created_at;

-- name: ListAgentSessionsByBuild :many
-- Agent health for a whole build in one query -- "agent healthy / late /
-- missing" (the environment
-- dashboard's health band) needs last_heartbeat_at per deployed_object,
-- and there's no direct build_id on agent_session to filter by, only a
-- path through deployed_object -> team -> build.
SELECT agent_session.* FROM agent_session
JOIN deployed_object ON deployed_object.id = agent_session.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1;

-- name: CreateAgentTask :one
INSERT INTO agent_task (deployed_object_id, step_index, command, payload, ignore_errors)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (deployed_object_id, step_index) DO NOTHING
RETURNING *;

-- name: NextStepIndexForHost :one
-- Ad-hoc tasks (internal/api's tasks.go) append after whatever steps a
-- host already has -- authored deploy-time steps if any exist, or
-- previous ad-hoc commands -- rather than picking an index that could
-- collide. COALESCE handles "no rows yet" (a host with no authored
-- steps at all getting its first ad-hoc command).
SELECT COALESCE(MAX(step_index), -1) + 1 FROM agent_task WHERE deployed_object_id = $1;

-- name: NextAgentTaskForHost :one
-- get-task: the lowest step_index for this host that isn't done or ignored
-- yet (an ignore_errors step that failed terminally is 'ignored' -- terminal,
-- non-blocking, so the sequence steps past it), but
-- ONLY leased if that exact row is actually available -- this has to be
-- two conditions on the SAME row, not "skip it and try the next one."
-- An earlier version of this query filtered pending-or-expired *before*
-- picking the lowest step_index, which meant a step currently leased (an
-- agent genuinely working on it) was invisible to the candidate set
-- entirely, and the query happily leased the NEXT step instead -- a real
-- bug, caught by a real second get-task call while step 0 was still
-- leased returning step 1. The inner subquery here finds the lowest
-- not-done step_index unconditionally; the outer UPDATE then only
-- commits if THAT SPECIFIC row is pending or its lease expired, matching
-- 0 rows (NextAgentTaskForHost returns pgx.ErrNoRows, "nothing to do
-- right now") rather than reaching past it -- exactly "an agent must
-- finish step N before starting N+1."
--
-- No FOR UPDATE SKIP LOCKED here, unlike the task table:
-- there's only ever one live candidate row per host (the lowest
-- not-done step_index), not a shared pool of many interchangeable rows
-- to pick among, so plain UPDATE's own row-level locking on the matched
-- id already serializes two concurrent callers correctly -- the second
-- one re-evaluates the WHERE clause after the first's lock releases and
-- sees status is no longer pending/expired.
UPDATE agent_task
SET status = 'leased', lease_expires_at = now() + $2::interval,
    attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT t2.id FROM agent_task t2
    WHERE t2.deployed_object_id = $1
      AND t2.status NOT IN ('done', 'ignored')
    ORDER BY t2.step_index
    LIMIT 1
)
AND (status = 'pending' OR (status = 'leased' AND lease_expires_at < now()))
RETURNING *;

-- name: CompleteAgentTask :one
UPDATE agent_task SET status = 'done', output = $2, last_error = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: FailAgentTask :one
UPDATE agent_task SET status = $2, last_error = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListAgentTasksByHost :many
SELECT * FROM agent_task WHERE deployed_object_id = $1 ORDER BY step_index;

-- name: GetAgentTask :one
SELECT * FROM agent_task WHERE id = $1;

-- name: ListFailedStepObjectIDsByBuild :many
-- Every deployed object in this build that has at least one failed step
-- (a failed agent_task) -- the "failed steps" health signal for the
-- Overview's per-team state chart, kept distinct from a provisioning
-- (infra) failure, which is deployed_object.status = 'failed' itself.
SELECT DISTINCT agent_task.deployed_object_id
FROM agent_task
JOIN deployed_object ON deployed_object.id = agent_task.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1 AND agent_task.status = 'failed';

-- name: SummarizeAgentTasksByObjectForBuild :many
-- Per deployed object in this build, how many authored steps (agent_task
-- rows) it has and how many are still open (pending/leased) or failed --
-- the raw material the object-lifecycle poll uses to decide building ->
-- finished / build_failed. Objects with no steps simply don't appear here
-- (an outer join isn't needed: the poll treats a missing entry as zero
-- steps, i.e. nothing left to build).
SELECT deployed_object.id AS deployed_object_id,
       count(*)::bigint AS total,
       count(*) FILTER (WHERE agent_task.status IN ('pending', 'leased'))::bigint AS open,
       count(*) FILTER (WHERE agent_task.status = 'failed')::bigint AS failed
FROM agent_task
JOIN deployed_object ON deployed_object.id = agent_task.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1
GROUP BY deployed_object.id;

-- name: ListObjectsWithFailedValidatorByBuild :many
-- Every deployed object in this build with at least one failed validator
-- result -- an object whose steps all completed but which does not match
-- what the environment asserts (the 'invalid' state), told apart from a
-- failed step (build_failed) above. A failed check on an 'ignored' task
-- (its step set ignore_errors) is deliberately excluded: the author chose to
-- tolerate that step's failure, so it must not mark the object invalid.
SELECT DISTINCT deployed_object.id
FROM validator_result
JOIN agent_task ON agent_task.id = validator_result.agent_task_id
JOIN deployed_object ON deployed_object.id = agent_task.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1 AND validator_result.passed = false
  AND agent_task.status != 'ignored';

-- name: RecordValidatorResult :exec
-- Not :one/RETURNING *: found live that Postgres
-- requires SELECT on a table's columns for RETURNING to work, even on an
-- INSERT the caller definitely has permission for -- laforge_gateway is
-- deliberately INSERT-only on this table (see migrations/00004's own
-- comment, and TestGatewayRoleCannotReadContent's real proof that it
-- can't read agent_heartbeat/event either), so RETURNING * always failed
-- with "permission denied," 100% of the time, for the only role that
-- ever calls this. Undetected until this had a real caller at all.
INSERT INTO validator_result (agent_task_id, kind, args, passed, message)
VALUES ($1, $2, $3, $4, $5);

-- name: ListValidatorResultsByTask :many
SELECT * FROM validator_result WHERE agent_task_id = $1 ORDER BY created_at;

-- name: CreateAgentArtifact :exec
-- Upsert: an object's deploy task can be retried, and re-delivering just
-- replaces the (regenerable) binary and token.
INSERT INTO agent_artifact (deployed_object_id, platform, token, agent_binary)
VALUES ($1, $2, $3, $4)
ON CONFLICT (deployed_object_id) DO UPDATE SET
    platform = EXCLUDED.platform, token = EXCLUDED.token, agent_binary = EXCLUDED.agent_binary, created_at = now();

-- name: GetAgentArtifactByToken :one
-- What the api serves to a booting instance: the bytes for this one-time
-- token. The object id is returned too, purely so the download can be
-- logged against the host it belongs to.
SELECT deployed_object_id, platform, agent_binary FROM agent_artifact WHERE token = $1;

-- name: DeleteAgentArtifactsCheckedIn :execrows
-- The cleanup sweep: an agent that has a session has downloaded and run,
-- so its (large, regenerable) binary is no longer needed.
DELETE FROM agent_artifact
WHERE deployed_object_id IN (SELECT deployed_object_id FROM agent_session);

-- name: SumAgentBinaryStorageByBuild :one
-- Build -> Artifacts: how many per-host agent binaries this build holds and
-- their total bytes (octet_length of the stored bytea). Regenerable on the
-- next deploy, so these are the primary purge target.
SELECT count(*)::bigint AS count,
       COALESCE(sum(octet_length(agent_binary)), 0)::bigint AS bytes
FROM agent_artifact aa
JOIN deployed_object o ON o.id = aa.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE t.build_id = $1;

-- name: SumStepOutputStorageByBuild :one
-- Total captured step output (agent_task.output) across the build, and how
-- many steps carry any -- the console/script output the Steps tab shows.
SELECT count(*) FILTER (WHERE at.output IS NOT NULL AND at.output <> '')::bigint AS count,
       COALESCE(sum(octet_length(COALESCE(at.output, ''))), 0)::bigint AS bytes
FROM agent_task at
JOIN deployed_object o ON o.id = at.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE t.build_id = $1;

-- name: SumEventStorageByBuild :one
-- The event journal's own footprint: one row per lifecycle/step event, sized
-- by its message plus JSON payload.
SELECT count(*)::bigint AS count,
       COALESCE(sum(octet_length(message) + octet_length(payload::text)), 0)::bigint AS bytes
FROM event WHERE build_id = $1;

-- name: DeleteAgentArtifactsByBuild :execrows
-- Purge every stored agent binary for this build (regenerable on redeploy).
-- Leaves step output and the event journal intact -- those are the record of
-- what happened, not a rebuildable cache.
DELETE FROM agent_artifact aa
USING deployed_object o, team t
WHERE aa.deployed_object_id = o.id AND o.team_id = t.id AND t.build_id = $1;
