-- Dependency-blocked steps. A host/container's steps are now materialized as
-- soon as its box is up -- even while it waits on a dependency -- so the UI can
-- show the queued steps and WHY they're waiting, instead of an empty "Running
-- Steps". Steps queued while blocked get status 'blocked': visible, counted as
-- open (not finished), but never leased by an agent (NextAgentTaskForHost only
-- leases pending/expired-leased). When the dependency finishes, the orchestrator
-- flips 'blocked' -> 'pending'. deployed_object.blocked_on records which
-- dependencies it's waiting on (for the UI); NULL/empty means not blocked.

-- +goose Up
ALTER TABLE agent_task DROP CONSTRAINT agent_task_status_check;
ALTER TABLE agent_task ADD CONSTRAINT agent_task_status_check
  CHECK (status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text, 'ignored'::text, 'blocked'::text]));
ALTER TABLE deployed_object ADD COLUMN blocked_on jsonb;

-- +goose Down
ALTER TABLE deployed_object DROP COLUMN blocked_on;
ALTER TABLE agent_task DROP CONSTRAINT agent_task_status_check;
ALTER TABLE agent_task ADD CONSTRAINT agent_task_status_check
  CHECK (status = ANY (ARRAY['pending'::text, 'leased'::text, 'done'::text, 'failed'::text, 'ignored'::text]));
