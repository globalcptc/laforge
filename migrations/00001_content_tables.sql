-- Content-side schema: everything authored in git, immutable per commit.
-- JSONB only for vars/tags/attributes (and the similarly free-form
-- steps/validate/findings/ports shapes); everything else that needs to be
-- queried or joined on is a real column.
--
-- Deliberately NOT here: the "Runtime (internal only)" tables (build, team,
-- task, lease, agent_session, event journal) -- those belong to the
-- "Orchestrator + runner contract" and "New agent + agent-gateway" work,
-- not this "Schema + loader."

-- +goose Up

CREATE TABLE repository (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    github_owner  text NOT NULL,
    github_repo   text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (github_owner, github_repo)
);

-- One row per commit that was ever ingested and checked, whether or not it
-- passed. "Every push is a content revision. The api ingests the commit,
-- runs full schema validation, and reports pass/fail."
CREATE TABLE content_revision (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id      uuid NOT NULL REFERENCES repository(id) ON DELETE CASCADE,
    commit_sha         text NOT NULL,
    ref                text,
    valid              boolean NOT NULL DEFAULT false,
    validation_errors  jsonb NOT NULL DEFAULT '[]',
    validated_at       timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, commit_sha)
);

CREATE TABLE environment (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    schema_version      integer,
    description         text,
    teams               integer NOT NULL,
    admin_cidrs         jsonb NOT NULL DEFAULT '[]',
    vdi_ports           jsonb NOT NULL DEFAULT '[]',
    root_password       text,
    start_at            timestamptz,
    stop_at             timestamptz,
    dns                 jsonb,
    access              jsonb NOT NULL DEFAULT '[]',
    vars                jsonb NOT NULL DEFAULT '{}',
    tags                jsonb NOT NULL DEFAULT '{}',
    findings            jsonb NOT NULL DEFAULT '[]',
    extends             text,
    UNIQUE (content_revision_id, name)
);

CREATE TABLE network (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    cidr                text NOT NULL,
    vdi_visible         boolean NOT NULL DEFAULT false,
    vars                jsonb NOT NULL DEFAULT '{}',
    tags                jsonb NOT NULL DEFAULT '{}',
    findings            jsonb NOT NULL DEFAULT '[]',
    extends             text,
    UNIQUE (content_revision_id, name)
);

-- Hosts and containers share one namespace at the config level ("Names are
-- unique per type. Hosts and containers share one namespace") but are kept
-- as separate tables here since their columns genuinely differ (disk vs
-- image) -- the loader enforces the shared-namespace rule; the database
-- does not need a shared table to do that.
CREATE TABLE host (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    os                  text,
    size                text,
    disk_gb             integer,
    ports               jsonb NOT NULL DEFAULT '{}',
    depends_on          jsonb NOT NULL DEFAULT '[]',
    steps               jsonb NOT NULL DEFAULT '[]',
    vars                jsonb NOT NULL DEFAULT '{}',
    tags                jsonb NOT NULL DEFAULT '{}',
    findings            jsonb NOT NULL DEFAULT '[]',
    people              jsonb NOT NULL DEFAULT '[]',
    extends             text,
    UNIQUE (content_revision_id, name)
);

CREATE TABLE container (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    image               text,
    size                text,
    ports               jsonb NOT NULL DEFAULT '{}',
    depends_on          jsonb NOT NULL DEFAULT '[]',
    steps               jsonb NOT NULL DEFAULT '[]',
    vars                jsonb NOT NULL DEFAULT '{}',
    tags                jsonb NOT NULL DEFAULT '{}',
    findings            jsonb NOT NULL DEFAULT '[]',
    people              jsonb NOT NULL DEFAULT '[]',
    extends             text,
    UNIQUE (content_revision_id, name)
);

CREATE TABLE script (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    description         text,
    language            text,
    source_path         text,
    timeout_seconds     integer,
    args                jsonb NOT NULL DEFAULT '[]',
    ignore_errors       boolean NOT NULL DEFAULT false,
    tags                jsonb NOT NULL DEFAULT '{}',
    findings            jsonb NOT NULL DEFAULT '[]',
    people              jsonb NOT NULL DEFAULT '[]',
    validate            jsonb NOT NULL DEFAULT '[]',
    UNIQUE (content_revision_id, name)
);

-- One row per people/*.csv file.
CREATE TABLE people_source (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_revision_id uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    path                text NOT NULL,
    name                text NOT NULL,
    UNIQUE (content_revision_id, path)
);

-- One row per CSV row. Everything beyond username is a free-form bag --
-- "A people CSV is data, not configuration" -- so a repo can add whatever
-- columns its scripts need without a schema change here.
CREATE TABLE person (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    people_source_id uuid NOT NULL REFERENCES people_source(id) ON DELETE CASCADE,
    username         text NOT NULL,
    attributes       jsonb NOT NULL DEFAULT '{}',
    UNIQUE (people_source_id, username)
);

-- The environment's `networks:` topology: network -> host/container -> a
-- list of copies. object_kind + object_name is deliberately denormalized
-- (text, not a foreign key to host/container) rather than two nullable FKs
-- with a check constraint -- the loader already validates the reference
-- exists as part of `laforge check`, and a build resolves it again from a
-- known-valid content_revision, so the extra FK plumbing doesn't buy much
-- at this stage. Worth revisiting if that stops being true.
CREATE TABLE placement (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id uuid NOT NULL REFERENCES environment(id) ON DELETE CASCADE,
    network_name   text NOT NULL,
    object_kind    text NOT NULL CHECK (object_kind IN ('host', 'container')),
    object_name    text NOT NULL,
    as_name        text NOT NULL,
    last_octet     integer NOT NULL,
    public         jsonb,
    -- `as` becomes the hostname (and the DNS name), so it must be unique
    -- within the environment -- matches internal/loader/checks.go.
    UNIQUE (environment_id, as_name)
);

-- +goose Down

DROP TABLE placement;
DROP TABLE person;
DROP TABLE people_source;
DROP TABLE script;
DROP TABLE container;
DROP TABLE host;
DROP TABLE network;
DROP TABLE environment;
DROP TABLE content_revision;
DROP TABLE repository;
