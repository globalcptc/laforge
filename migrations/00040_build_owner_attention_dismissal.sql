-- The home page's "needs attention" list aggregated failures across every
-- build a person could see, which becomes unusable once several people share a
-- repository. Two changes scope it down:
--   1. build.created_by_account_id records who created a build (NULL for an
--      auto-built webhook build, and for builds created before this column).
--      Home shows a person only their OWN builds' attention items.
--   2. attention_dismissal lets a person close an attention item (per build +
--      category); it stays closed for that person until the build goes away.

-- +goose Up
ALTER TABLE build ADD COLUMN created_by_account_id uuid REFERENCES account(id) ON DELETE SET NULL;

CREATE TABLE attention_dismissal (
    account_id uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    build_id uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    category text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, build_id, category)
);

-- +goose Down
DROP TABLE attention_dismissal;
ALTER TABLE build DROP COLUMN created_by_account_id;
