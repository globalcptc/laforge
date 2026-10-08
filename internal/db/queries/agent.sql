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
INSERT INTO agent_heartbeat (deployed_object_id, cert_fingerprint, remote_addr, next_poll_ms, cpu_pct, mem_pct, disk_pct, net_rx_bps, net_tx_bps)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

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
-- Authored deploy steps. ad_hoc defaults false (strict ordered, failure-blocking
-- lane). Operator commands use CreateAdHocAgentTask instead.
INSERT INTO agent_task (deployed_object_id, step_index, command, payload, ignore_errors)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (deployed_object_id, step_index) DO NOTHING
RETURNING *;

-- name: CreateAdHocAgentTask :one
-- Operator-dispatched tasks (immediate `run` and scheduled dispatch). ad_hoc =
-- true puts them in the always-eligible lane so they run even past a failed
-- authored step -- see NextAgentTaskForHost. Same shape as CreateAgentTask
-- otherwise (step_index still from NextStepIndexForHost, so it's unique).
INSERT INTO agent_task (deployed_object_id, step_index, command, payload, ignore_errors, ad_hoc)
VALUES ($1, $2, $3, $4, $5, true)
ON CONFLICT (deployed_object_id, step_index) DO NOTHING
RETURNING *;

-- name: DeleteAgentTasksForObject :exec
-- Clear every materialized step for an object so a redeploy re-materializes
-- them from scratch. materializeSteps skips when ANY agent_task already exists
-- (NextStepIndexForHost != 0), so without this a rebuilt instance would never
-- re-run its steps. validator_result rows reference agent_task ON DELETE
-- CASCADE, so they go with it; the append-only `event` journal (keyed by
-- deployed_object_id, not agent_task_id) is untouched history.
DELETE FROM agent_task WHERE deployed_object_id = $1;

-- name: NextStepIndexForHost :one
-- Ad-hoc tasks (internal/api's tasks.go) append after whatever steps a
-- host already has -- authored deploy-time steps if any exist, or
-- previous ad-hoc commands -- rather than picking an index that could
-- collide. COALESCE handles "no rows yet" (a host with no authored
-- steps at all getting its first ad-hoc command).
SELECT COALESCE(MAX(step_index), -1) + 1 FROM agent_task WHERE deployed_object_id = $1;

-- name: NextAgentTaskForHost :one
-- get-task: which task the host should run next. There are two lanes:
--
--   * Ad-hoc tasks (ad_hoc = true: operator `run` and scheduled dispatch) are
--     ALWAYS eligible and take priority. They must run even when an authored
--     step has terminally failed and stopped the deploy -- that is exactly when
--     an operator needs to run commands on the box to debug why a step failed.
--     A failed authored step no longer blocks them.
--
--   * Authored steps (ad_hoc = false) keep strict ordered semantics: the lowest
--     not-done/ignored step (an ignore_errors step that failed terminally is
--     'ignored' -- terminal, non-blocking, so the sequence steps past it), and
--     the sequence stops on a 'failed' step. This is "finish step N before N+1."
--
-- The inner subquery picks the single highest-priority runnable candidate:
-- any ad-hoc row, or the lowest not-done AUTHORED step, ordered ad-hoc-first
-- then by step_index. The outer UPDATE then only commits if THAT SPECIFIC row
-- is pending or its lease expired -- the candidate and the lease gate are
-- evaluated on the SAME row (an earlier version filtered pending-or-expired
-- before picking the lowest step_index, so a step an agent was genuinely
-- working on went invisible and the NEXT step got leased instead -- a real bug
-- caught by a second get-task while step 0 was still leased returning step 1).
-- A non-leasable candidate matches 0 rows (ErrNoRows, "nothing to do right
-- now"), so the agent finishes its current task before being handed another,
-- and a failed authored step with no ad-hoc work pending correctly yields
-- nothing.
--
-- No FOR UPDATE SKIP LOCKED: there's only ever one live candidate row per host,
-- so plain UPDATE's own row-level locking serializes two concurrent callers.
UPDATE agent_task
SET status = 'leased', lease_expires_at = now() + $2::interval,
    attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT t.id FROM agent_task t
    WHERE t.deployed_object_id = $1
      AND t.status NOT IN ('done', 'ignored')
      AND (
        t.ad_hoc
        OR t.step_index = (
          SELECT MIN(t2.step_index) FROM agent_task t2
          WHERE t2.deployed_object_id = $1
            AND NOT t2.ad_hoc
            AND t2.status NOT IN ('done', 'ignored')
        )
      )
    ORDER BY t.ad_hoc DESC, t.step_index
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
