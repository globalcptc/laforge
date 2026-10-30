-- +goose Up
-- Why a build_failed host failed, when the failure is one the orchestrator can
-- undo on its own once the cause goes away:
--   agent_lost              its agent stopped heartbeating with steps left
--   agent_never_checked_in  no agent connected long after the instance came up
--   gate                    a wait_for target it waits on failed
--   dependency              a depends_on dependency failed before its steps ran
-- NULL for a failure only an operator can clear (a step failed). failed_at is
-- when it failed: an agent heartbeat after it means the agent is back.
ALTER TABLE deployed_object ADD COLUMN failure_kind text
    CHECK (failure_kind IN ('agent_lost', 'agent_never_checked_in', 'gate', 'dependency'));
ALTER TABLE deployed_object ADD COLUMN failed_at timestamptz;

-- Classify hosts already failed this way, so they recover too.
UPDATE deployed_object SET failed_at = updated_at,
    failure_kind = CASE
        WHEN last_error LIKE 'agent stopped checking in%' THEN 'agent_lost'
        WHEN last_error LIKE 'agent never checked in%' THEN 'agent_never_checked_in'
        WHEN last_error LIKE 'waiting on %, which failed' THEN 'gate'
        WHEN last_error LIKE 'dependency "%" failed to configure' THEN 'dependency'
    END
WHERE status = 'build_failed';

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN failed_at;
ALTER TABLE deployed_object DROP COLUMN failure_kind;
