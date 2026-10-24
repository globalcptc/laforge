-- A per-build secret for the password generators (passphrase / wordchain /
-- randompassword, and set_password's `generate:`). Generated passwords must be
-- stable across every render of the same build -- materialization, each
-- reconcile pass, scheduled fires, the Render tab -- so they are derived from
-- this secret (HMAC over the value's identity), never drawn fresh. Created on
-- first use (EnsureBuildRenderSecret) and kept for the build's life, so a host
-- rebuilt within a build gets the same passwords back.
--
-- Its own table, not a build column: build rows are returned whole as JSON in
-- many places, and this must never ride along.

-- +goose Up
CREATE TABLE build_render_secret (
    build_id   uuid PRIMARY KEY REFERENCES build(id) ON DELETE CASCADE,
    secret     bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE build_render_secret;
