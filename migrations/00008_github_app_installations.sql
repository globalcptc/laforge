-- Replaces the OAuth-App-plus-manually-pasted-webhook-secret model
-- with a real GitHub App, which addresses the two problems this
-- fixes (one secret handled by every repo's admin; the webhook payload's
-- own clone_url trusted with no cross-check).
--
-- Two trust boundaries, two tables. github_installation and
-- installation_repository together are "what's technically reachable" --
-- populated ENTIRELY from GitHub's own installation/installation_
-- repositories webhook events, never written to by anything else.
-- repository (existing, migration 00001) stays "what LaForge is actually
-- configured to build from" -- a LaForge admin has to explicitly approve
-- a repository out of the installed set before it gets one.

-- +goose Up

CREATE TABLE github_installation (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id bigint NOT NULL UNIQUE,
    account_login   text NOT NULL,
    account_type    text NOT NULL, -- "Organization" or "User", exactly as GitHub reports it
    suspended       boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- One row per repository a GitHub App installation currently covers.
-- Populated from the `installation` event's own repositories list (first
-- install) and the `installation_repositories` event's added/removed
-- lists (as the selection changes later). A row disappearing means the
-- repo is no longer installed, full stop -- deleting it (rather than a
-- soft "removed" flag) keeps "currently installed" answerable with a
-- plain SELECT; the event journal is where history belongs if it's ever
-- needed, not this table.
CREATE TABLE installation_repository (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id  uuid NOT NULL REFERENCES github_installation(id) ON DELETE CASCADE,
    github_owner     text NOT NULL,
    github_repo      text NOT NULL,
    github_repo_id   bigint NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (installation_id, github_repo_id)
);

-- Nullable: existing rows (and anything created directly, e.g. content
-- authored before any App is configured, or a test fixture) predate
-- this and simply have no installation behind them. A repository with a
-- null installation_id falls back to GITHUB_SERVICE_TOKEN for verifying
-- its real clone URL (still real verification, just more broadly scoped
-- than an installation token) -- see internal/api/webhook.go's reconcile.
ALTER TABLE repository ADD COLUMN installation_id uuid REFERENCES github_installation(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE repository DROP COLUMN installation_id;
DROP TABLE installation_repository;
DROP TABLE github_installation;
