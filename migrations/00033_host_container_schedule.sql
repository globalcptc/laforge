-- Persist a host/container's `schedule:` list, so findings attribution (and any
-- future consumer that reads the persisted content row rather than the loaded
-- checkout) can see scripts a host runs only from a schedule entry, not just its
-- steps. Previously only `steps` was stored, so a script referenced solely by a
-- `schedule:` entry was invisible to the per-script findings breakdown. Additive
-- column with a default -- no writer needs stopping.

-- +goose Up

ALTER TABLE host ADD COLUMN schedule jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE container ADD COLUMN schedule jsonb NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down

ALTER TABLE host DROP COLUMN schedule;
ALTER TABLE container DROP COLUMN schedule;
