-- Revalidation (migration 00077): re-run a host's validators and report.

-- name: ListValidateTasksForObject :many
-- The host's own validate checks, as its build materialized them (already
-- rendered) -- what a revalidation copies and runs again. Authored rows only:
-- not ad-hoc commands, not an earlier revalidation's copies.
SELECT step_index, payload, ignore_errors FROM agent_task
WHERE deployed_object_id = $1 AND command = 'validate' AND NOT ad_hoc AND validation_run_id IS NULL
ORDER BY step_index;

-- name: HasOpenValidationRunForObject :one
SELECT EXISTS (SELECT 1 FROM validation_run WHERE deployed_object_id = $1 AND status = 'running');

-- name: CreateValidationRun :one
INSERT INTO validation_run (report_id, build_id, deployed_object_id, created_by_account_id, status_before)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateValidationRunTask :one
-- One copied validate check. ad_hoc puts it in the always-eligible lane (it
-- runs even past a failed step); max_attempts 1 because a revalidation reports
-- what the check says now -- retrying until it passes would hide the answer.
INSERT INTO agent_task (deployed_object_id, step_index, command, payload, ignore_errors, ad_hoc, max_attempts, validation_run_id)
VALUES ($1, $2, 'validate', $3, $4, true, 1, $5)
RETURNING *;

-- name: ListOpenValidationRunsByBuild :many
SELECT * FROM validation_run WHERE build_id = $1 AND status = 'running' ORDER BY created_at;

-- name: SummarizeValidationRun :one
-- Where a run stands: how many copied checks are left to run, and whether any
-- that count failed. A check on a step marked ignore_errors is reported but
-- never fails the run, the same rule that keeps it from marking a build
-- invalid. A task that errored without reporting a check result (the agent
-- couldn't run it at all) is a failure too.
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE t.status IN ('pending', 'leased', 'blocked'))::bigint AS open,
       count(*) FILTER (WHERE t.status = 'failed' OR (NOT t.ignore_errors AND EXISTS (
           SELECT 1 FROM validator_result vr WHERE vr.agent_task_id = t.id AND NOT vr.passed)))::bigint AS failed
FROM agent_task t WHERE t.validation_run_id = $1;

-- name: FinishValidationRun :execrows
UPDATE validation_run SET status = sqlc.arg(status), status_after = sqlc.narg(status_after), finished_at = now()
WHERE id = sqlc.arg(id) AND status = 'running';

-- name: DeleteOpenValidationRunTasks :exec
-- A timed-out run's checks that never ran: dropped so they don't run later,
-- long after anyone is looking at the report.
DELETE FROM agent_task WHERE validation_run_id = $1 AND status IN ('pending', 'leased', 'blocked');

-- name: SetDeployedObjectInvalidFromFinished :execrows
-- A revalidation found a failing check on a finished host.
UPDATE deployed_object SET status = 'invalid', last_error = $2, updated_at = now()
WHERE id = $1 AND status = 'finished';

-- name: SetDeployedObjectFinishedFromInvalid :execrows
-- A revalidation found every check passing on an invalid host.
UPDATE deployed_object SET status = 'finished', last_error = NULL, updated_at = now()
WHERE id = $1 AND status = 'invalid';

-- name: ListValidationRunsByReport :many
SELECT vr.*, COALESCE(o.as_name, o.object_name)::text AS object_label, o.kind AS object_kind, t.team_number
FROM validation_run vr
JOIN deployed_object o ON o.id = vr.deployed_object_id
JOIN team t ON t.id = o.team_id
WHERE vr.build_id = $1 AND vr.report_id = $2
ORDER BY t.team_number, object_label;

-- name: ListValidationRunsByObject :many
SELECT * FROM validation_run WHERE deployed_object_id = $1 ORDER BY created_at DESC LIMIT 20;

-- name: ListValidationReportsByBuild :many
-- One row per report (a click of Revalidate), newest first.
SELECT report_id,
       min(created_at)::timestamptz AS created_at,
       max(finished_at)::timestamptz AS finished_at,
       count(*)::bigint AS hosts,
       count(*) FILTER (WHERE status = 'running')::bigint AS running,
       count(*) FILTER (WHERE status = 'passed')::bigint AS passed,
       count(*) FILTER (WHERE status = 'failed')::bigint AS failed,
       count(*) FILTER (WHERE status IN ('canceled', 'timed_out'))::bigint AS incomplete
FROM validation_run WHERE build_id = $1
GROUP BY report_id
ORDER BY min(created_at) DESC
LIMIT 50;

-- name: ListValidationRunTasks :many
-- A run's copied checks, each with the result rows its agent reported.
SELECT id, step_index, status, ignore_errors, last_error, payload, started_at, finished_at
FROM agent_task WHERE validation_run_id = $1 ORDER BY step_index;
