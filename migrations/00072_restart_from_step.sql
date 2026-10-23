-- +goose Up
-- Restarting a host's build from a step (docs/plans/restart-from-step.md).
--
-- step_group: which authored step an agent_task row came from (the index
-- into the host's `steps:`), NULL for ad-hoc commands. Materialization sets
-- it from gateway.PlannedCommand.Group. It's what "restart from step N" deletes
-- and re-queues by, and what the steps panel groups rows under, instead of
-- re-expanding the content and lining the result up by step_index (which
-- misattributes rows once the content has changed).
ALTER TABLE agent_task ADD COLUMN step_group integer;

-- infra_fingerprint: the part of a deployed object's fingerprint that the
-- hoster built -- os, image, size, disk, compose archive, ports, depends_on --
-- recorded with the full fingerprint when the instance comes up. When the
-- full fingerprint later differs but this one doesn't, only the steps
-- changed. Empty on objects deployed before this migration: "unknown", so a
-- fingerprint change recreates them as it always has.
ALTER TABLE deployed_object ADD COLUMN infra_fingerprint text NOT NULL DEFAULT '';

-- steps_changed_at: set by the reconciler when a configured build holds
-- hosts on step-only changes (below) and this object's steps changed under
-- it; cleared when the object is restarted from a step or rebuilt.
ALTER TABLE deployed_object ADD COLUMN steps_changed_at timestamptz;
-- steps_changed_fingerprint: the fingerprint the object would have if it were
-- recreated now -- what a restart from a step sets its fingerprint to, since
-- after the restart its steps ARE the current revision's.
ALTER TABLE deployed_object ADD COLUMN steps_changed_fingerprint text NOT NULL DEFAULT '';

-- hold_on_step_change: on, a content change that touches only a host's steps
-- or scripts marks the host (steps_changed_at) instead of recreating it, so an
-- operator can restart it from the changed step. Off (the default), the
-- fingerprint rule recreates it, as before.
ALTER TABLE configured_build ADD COLUMN hold_on_step_change boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE configured_build DROP COLUMN hold_on_step_change;
ALTER TABLE deployed_object DROP COLUMN steps_changed_fingerprint;
ALTER TABLE deployed_object DROP COLUMN steps_changed_at;
ALTER TABLE deployed_object DROP COLUMN infra_fingerprint;
ALTER TABLE agent_task DROP COLUMN step_group;
