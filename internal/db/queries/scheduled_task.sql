-- name: CreateContentScheduledTask :one
-- A host/container's own `schedule:` entry, materialized once per
-- deployed_object per schedule_index (see migration 10's partial unique
-- index) -- re-deploying the same commit must not duplicate it, so this
-- is an upsert, not a plain insert.
--
-- The DO UPDATE's own WHERE guards against a real, live issue: Reconcile
-- calls this every time it opens a fresh deploy task, which for a build
-- whose deploy keeps failing (no real runner/builder ever completing it)
-- can mean every ~2s, forever. Unconditionally overwriting next_fire_at
-- on every one of those calls kept resetting an already-correct,
-- in-flight timer before it ever had a chance to become due -- caught
-- live against the real running stack, not found by reading the code.
-- Only when the entry's real content (when_expr/command/payload)
-- actually changed does recomputing timing make sense; an unchanged
-- entry leaves its existing next_fire_at/status alone. When nothing
-- needed updating, this returns zero rows (pgx.ErrNoRows) -- a real,
-- expected "already correct" outcome the caller treats as success, not
-- a failure -- rather than a row that looks unchanged but isn't.
INSERT INTO scheduled_task (build_id, source, deployed_object_id, schedule_index, when_expr, command, payload, anchor, fires_once, next_fire_at)
VALUES ($1, 'content', $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (deployed_object_id, schedule_index) WHERE source = 'content'
DO UPDATE SET when_expr = EXCLUDED.when_expr, command = EXCLUDED.command, payload = EXCLUDED.payload,
    anchor = EXCLUDED.anchor, fires_once = EXCLUDED.fires_once, next_fire_at = EXCLUDED.next_fire_at,
    updated_at = now()
WHERE scheduled_task.when_expr IS DISTINCT FROM EXCLUDED.when_expr
   OR scheduled_task.command IS DISTINCT FROM EXCLUDED.command
   OR scheduled_task.payload IS DISTINCT FROM EXCLUDED.payload
RETURNING *;

-- name: CreateAdHocScheduledTask :one
-- A UI-created scheduled task, targeting a selector re-resolved at fire
-- time (see orchestrator.MatchAdHocTargets) rather than one fixed object.
INSERT INTO scheduled_task (build_id, source, target, when_expr, command, payload, anchor, fires_once, next_fire_at, created_by)
VALUES ($1, 'adhoc', $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListDueScheduledTasks :many
-- The dispatch loop's own query: every pending row whose next_fire_at
-- has passed, oldest first -- what cmd/laforge-orchestrator's ticker
-- calls every tick.
SELECT * FROM scheduled_task
WHERE status = 'pending' AND next_fire_at IS NOT NULL AND next_fire_at <= now()
ORDER BY next_fire_at
LIMIT $1;

-- name: MarkScheduledTaskFired :one
-- fires_once rows: done for good, never fires again.
UPDATE scheduled_task SET status = 'fired', next_fire_at = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RescheduleScheduledTask :one
-- Recurring rows: stays pending, moves to its next real fire time.
UPDATE scheduled_task SET next_fire_at = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CancelScheduledTask :one
UPDATE scheduled_task SET status = 'canceled', updated_at = now()
WHERE id = $1 AND build_id = $2 AND status = 'pending'
RETURNING *;

-- name: ListScheduledTasksByBuild :many
SELECT * FROM scheduled_task WHERE build_id = $1 ORDER BY created_at DESC;

-- name: GetScheduledTask :one
SELECT * FROM scheduled_task WHERE id = $1;
