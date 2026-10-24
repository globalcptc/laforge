-- Instance admins, managed from the UI (Admin -> Admins) instead of only
-- from LAFORGE_ADMIN_LOGINS. The env var used to be the whole list, read at
-- every start; now it only seeds this table the first time laforge-api
-- starts against an empty one, and the table is the source of truth after.
--
-- Keyed by GitHub login, not account id: an admin is usually added before
-- they have ever signed in (being on this list is what lets them sign in at
-- all -- signInAuthorized), so there is no account row to point at yet.
-- Logins are case-insensitive on GitHub, hence the lower() index.
--
-- added_by is the login of the admin who added the row; NULL means it was
-- seeded from LAFORGE_ADMIN_LOGINS.

-- +goose Up

CREATE TABLE instance_admin (
    github_login text NOT NULL,
    added_by     text,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX instance_admin_login_idx ON instance_admin (lower(github_login));

-- +goose Down

DROP TABLE instance_admin;
