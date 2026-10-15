-- +goose Up
-- Per-user favorites on the builds page: a user stars the configured builds
-- (branches) they care about, and those sort to the top of their list. Purely a
-- per-viewer preference; cascades away with the account or the configured build.
CREATE TABLE configured_build_favorite (
    account_id uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    configured_build_id uuid NOT NULL REFERENCES configured_build(id) ON DELETE CASCADE,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, configured_build_id)
);

-- +goose Down
DROP TABLE configured_build_favorite;
