-- Link a fired scheduled task to the agent_task run(s) it spawns, so the
-- Schedule page can show, per team, exactly which run a scheduled task produced
-- and surface that run's status + console output. A scheduled task fires through
-- CreateAdHocAgentTask (immediate `run` leaves it NULL; scheduled dispatch sets
-- it). ON DELETE SET NULL keeps a run's history even if the scheduled task row
-- is later removed.

-- +goose Up
ALTER TABLE agent_task ADD COLUMN scheduled_task_id uuid REFERENCES scheduled_task(id) ON DELETE SET NULL;
CREATE INDEX agent_task_scheduled_task_idx ON agent_task (scheduled_task_id) WHERE scheduled_task_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS agent_task_scheduled_task_idx;
ALTER TABLE agent_task DROP COLUMN scheduled_task_id;
