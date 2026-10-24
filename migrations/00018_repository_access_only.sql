-- Access is per repository, full stop: every repository is independent,
-- whatever organization it sits in. Installation-wide grants (00017) are
-- gone.
--
-- A person's level on a repository starts from their GitHub role there
-- (admin -> admin, write/maintain -> build, anything else -> read). A
-- repository_access row is an admin's decision for that person on that
-- repository and REPLACES the GitHub-derived level -- it can raise it,
-- lower it, or ('none') take access away entirely. Deleting the row puts
-- the person back on their GitHub role.

-- +goose Up

DROP TABLE installation_access;

ALTER TABLE repository_access DROP CONSTRAINT repository_access_level_check;
ALTER TABLE repository_access ADD CONSTRAINT repository_access_level_check
    CHECK (level IN ('none', 'read', 'build', 'manage', 'admin'));

-- +goose Down

DELETE FROM repository_access WHERE level = 'none';
ALTER TABLE repository_access DROP CONSTRAINT repository_access_level_check;
ALTER TABLE repository_access ADD CONSTRAINT repository_access_level_check
    CHECK (level IN ('read', 'build', 'manage', 'admin'));

CREATE TABLE installation_access (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id uuid NOT NULL REFERENCES github_installation(id) ON DELETE CASCADE,
    account_id      uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    level           text NOT NULL CHECK (level IN ('read', 'build', 'manage', 'admin')),
    granted_by      uuid REFERENCES account(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (installation_id, account_id)
);
