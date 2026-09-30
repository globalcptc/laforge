-- "Access state is operational state, not desired state... A redeploy...
-- must never silently reopen a closed team." Stored on team directly
-- (not a separate table) precisely because team rows persist across
-- redeploys within the same build ("a new commit reconciles in place"),
-- so this state naturally survives exactly the events it has to survive.
--
-- access_state is what CloseAccess/OpenAccess actually achieved (set by
-- the runner once the builder call completes, not optimistically before).
-- access_override_until is the operational "extend by N minutes" knob:
-- when set and in the future, it means "stay open regardless of the
-- environment's own schedule until this time" -- checked by whatever
-- evaluates the schedule (not built this migration; see the UI/API work
-- that reads it). Every actual state change is still an ordinary `event`
-- row (kind 'access.opened' / 'access.closed'), reusing the existing
-- journal rather than a bespoke audit table.

-- +goose Up

ALTER TABLE team ADD COLUMN access_state text NOT NULL DEFAULT 'open' CHECK (access_state IN ('open', 'closed'));
ALTER TABLE team ADD COLUMN access_override_until timestamptz;

-- +goose Down

ALTER TABLE team DROP COLUMN access_override_until;
ALTER TABLE team DROP COLUMN access_state;
