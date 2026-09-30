-- Track an instance's live power state at the hoster (builder truth)
-- separately from its deploy lifecycle status and from the LaForge agent's
-- own health. deployed_object.status reflects the deploy OUTCOME and then
-- stops moving; it can't tell a running host from one that later crashed,
-- stopped, or vanished at the hoster. A periodic orchestrator poll fills
-- these from Builder.Inspect so "is the instance up" is answered by the
-- hoster, not inferred from whether the agent checked in.
--
-- power_state: '' (unknown / not yet polled / not applicable, e.g. a
-- network), 'running', 'stopped', 'other' (exists but not cleanly either),
-- or 'missing' (the hoster no longer has it). checked_at is when it was
-- last confirmed, so the UI can show staleness.

-- +goose Up

ALTER TABLE deployed_object ADD COLUMN power_state text NOT NULL DEFAULT '';
ALTER TABLE deployed_object ADD COLUMN power_state_checked_at timestamptz;

-- +goose Down

ALTER TABLE deployed_object DROP COLUMN power_state;
ALTER TABLE deployed_object DROP COLUMN power_state_checked_at;
