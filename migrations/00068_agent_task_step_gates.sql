-- Step-level dependency gates (`wait_for:` / `milestone:` on a step). depends_on
-- holds ALL of a host's steps until a dependency finishes; a step-level gate
-- holds one step and everything after it until another host (the same team's
-- copy) reaches a named point. Both ends live on agent_task:
--
--   * milestone / step_script mark the target side: every command an authored
--     step expanded into carries the step's `milestone:` and, for a `script:`
--     step, the script name. A target point is reached once all of its rows on
--     every copy of the target are done or ignored.
--   * wait_for marks the waiting side: the FIRST command of a gated step
--     carries the step's wait_for entries (JSON list of {host, milestone, step}).
--     The step's rows start 'blocked'; the orchestrator releases blocked rows
--     below the first still-unsatisfied gate each pass. The lease query already
--     only hands out the lowest unfinished authored step, so a blocked gate row
--     holds every step after it without any change there.

-- +goose Up
ALTER TABLE agent_task ADD COLUMN milestone text;
ALTER TABLE agent_task ADD COLUMN step_script text;
ALTER TABLE agent_task ADD COLUMN wait_for jsonb;
CREATE INDEX agent_task_milestone_idx ON agent_task (deployed_object_id) WHERE milestone IS NOT NULL OR step_script IS NOT NULL;

-- +goose Down
DROP INDEX agent_task_milestone_idx;
ALTER TABLE agent_task DROP COLUMN wait_for;
ALTER TABLE agent_task DROP COLUMN step_script;
ALTER TABLE agent_task DROP COLUMN milestone;
