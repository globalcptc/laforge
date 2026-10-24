-- `ignore_errors` on a script was authored, validated, and stored but honored
-- NOWHERE: a step that failed still hard-blocked every later step (get-task's
-- NextAgentTaskForHost stops at the first not-done row) AND drove the object to
-- build_failed (AdvanceObjectLifecycle). This wires it through the agent_task:
-- a task carries its script's ignore_errors, and on terminal failure the gateway
-- marks such a task 'ignored' instead of 'failed' -- a new terminal status that
-- (a) does NOT block subsequent steps and (b) does NOT count as a build failure,
-- while still recording the error and a distinct step event so an operator can
-- see it happened. See internal/gateway (report-status), internal/orchestrator
-- (lifecycle), and internal/db/queries/agent.sql (NextAgentTaskForHost).

-- +goose Up

ALTER TABLE agent_task ADD COLUMN ignore_errors boolean NOT NULL DEFAULT false;

ALTER TABLE agent_task DROP CONSTRAINT agent_task_status_check;
ALTER TABLE agent_task ADD CONSTRAINT agent_task_status_check
    CHECK (status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text, 'ignored'::text]));

-- +goose Down

ALTER TABLE agent_task DROP CONSTRAINT agent_task_status_check;
-- Any 'ignored' rows become 'failed' so the tighter old constraint can apply.
UPDATE agent_task SET status = 'failed' WHERE status = 'ignored';
ALTER TABLE agent_task ADD CONSTRAINT agent_task_status_check
    CHECK (status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text]));

ALTER TABLE agent_task DROP COLUMN ignore_errors;
