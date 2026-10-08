-- Mark operator-dispatched tasks (immediate `run` and scheduled dispatch) as
-- ad-hoc, distinct from authored deploy `steps:`. Ad-hoc tasks run in their own
-- lane (see NextAgentTaskForHost): they are always eligible and must run even
-- when an authored step has terminally failed and stopped the deploy -- exactly
-- when an operator needs to run commands on the box to debug why a step failed.
-- Authored steps (ad_hoc = false) keep strict ordered, failure-blocking
-- semantics. Default false so existing rows and the materialized-step path are
-- unchanged.

-- +goose Up
ALTER TABLE agent_task ADD COLUMN ad_hoc boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE agent_task DROP COLUMN ad_hoc;
