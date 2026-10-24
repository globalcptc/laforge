-- Access is now actually enforced from the environment's authored `access:`
-- windows by the orchestrator's access reconciler (internal/orchestrator/
-- access.go), not just displayed. Two changes make that real:
--
-- 1. Teams start CLOSED, not open. A team no operator and no schedule has
--    opened is closed -- "if I haven't touched it, it isn't open." The old
--    'open' default silently contradicted the schedule before the first
--    window. Existing rows are left as-is; the
--    reconciler converges them to the schedule on its next pass.
--
-- 2. access_override_state records the DIRECTION of a manual override next to
--    its existing expiry (access_override_until). A hand open/close now means
--    "hold this state until the next scheduled boundary" (the API stores that
--    boundary as the expiry); the reconciler reads the direction to know
--    whether to hold open or closed, then resumes the schedule when it lapses.
--    '' means no override -- follow the schedule. This is what finally makes
--    Extend do something: it pushes the expiry of an open-direction override.

-- +goose Up

ALTER TABLE team ALTER COLUMN access_state SET DEFAULT 'closed';

ALTER TABLE team ADD COLUMN access_override_state text NOT NULL DEFAULT ''
    CHECK (access_override_state IN ('', 'open', 'closed'));

-- +goose Down

ALTER TABLE team DROP COLUMN access_override_state;
ALTER TABLE team ALTER COLUMN access_state SET DEFAULT 'open';
