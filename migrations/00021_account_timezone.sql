-- Per-user timezone preference (direct product feedback: timestamps should
-- render in each viewer's own timezone across the UI). Stored on the
-- account, not per session or per device, so a person's choice follows them
-- everywhere they sign in. Empty means "no preference": the UI falls back to
-- the viewer's browser-local timezone, which is exactly the prior behavior,
-- so existing rows need no backfill.

-- +goose Up

ALTER TABLE account ADD COLUMN timezone text NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE account DROP COLUMN timezone;
