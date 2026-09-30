-- Private Docker registry credentials (Docker-container design). A LaForge
-- `container:` whose image ref names a private registry needs a `docker
-- login` before the pull; this stores those credentials, keyed by the
-- registry host (e.g. "registry.internal" or "registry.internal:5000").
-- Docker Hub public images need no row.
--
-- The secret is stored as-is, the same trust model as builder_credential's
-- client keys and the people CSVs' passwords: the LaForge database is trusted
-- infrastructure. It reaches the in-container agent over the mTLS gateway as
-- part of the materialized `docker login` step.

-- +goose Up

CREATE TABLE registry_credential (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    registry_host text NOT NULL UNIQUE,
    username      text NOT NULL,
    secret        text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE registry_credential;
