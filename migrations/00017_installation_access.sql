-- Access is managed per GitHub App installation, not app-wide: a person
-- gets one level for an installation, covering every repository in it --
-- including repositories approved later. repository_access stays for
-- per-repository exceptions; a person's effective level on a repository is
-- the highest of their installation grant, their repository grant, and what
-- their own GitHub permissions imply (grants only ever raise a level).
-- An installation-level "admin" can manage that installation's access.

-- +goose Up

CREATE TABLE installation_access (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id uuid NOT NULL REFERENCES github_installation(id) ON DELETE CASCADE,
    account_id      uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    level           text NOT NULL CHECK (level IN ('read', 'build', 'manage', 'admin')),
    granted_by      uuid REFERENCES account(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (installation_id, account_id)
);

-- +goose Down

DROP TABLE installation_access;
