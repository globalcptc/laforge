-- +goose Up
-- Per-step timing: when the agent first started a step and when it finished, so
-- the steps UI can show how long each authored step took. Nullable -- a step
-- that hasn't started (or finished) yet has none. started_at is set once (the
-- first lease, via coalesce); finished_at is set when the step reaches a
-- terminal status (done/failed/ignored).
ALTER TABLE agent_task ADD COLUMN started_at timestamp with time zone;
ALTER TABLE agent_task ADD COLUMN finished_at timestamp with time zone;

-- +goose Down
ALTER TABLE agent_task DROP COLUMN started_at;
ALTER TABLE agent_task DROP COLUMN finished_at;
