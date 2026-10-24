-- The per-builder "docker base image" build job (Docker-container design).
-- On a MicroCloud/Incus cluster a LaForge `container:` runs as a thin,
-- nesting-enabled LXD system container that itself runs Docker; that LXD
-- container boots from a docker-ready base image this job builds ON the
-- cluster (import ubuntu container base -> install docker -> publish). The
-- job is builder-scoped, tracked with a status + an append-only log streamed
-- to a live console, triggered when the builder is added and re-runnable.
--
-- Per-builder image-source columns (the base is pulled from the server the
-- cluster can actually reach -- cloud-images.ubuntu.com for Canonical LXD,
-- NOT images.linuxcontainers.org, which the LXD/Incus split left unusable):
-- defaulted to cloud-images so a fresh MicroCloud builder works out of the
-- box. docker_base_fingerprint records the job's published result.

-- +goose Up

ALTER TABLE builder_config
    ADD COLUMN container_base_server text NOT NULL DEFAULT 'https://cloud-images.ubuntu.com/releases',
    ADD COLUMN container_base_alias text NOT NULL DEFAULT '22.04',
    ADD COLUMN docker_base_fingerprint text NOT NULL DEFAULT '';

-- The earlier AWS/OpenStack draft builders are accepted by the API but were
-- never added to this CHECK, so creating one would fail here. Fix that.
ALTER TABLE builder_config DROP CONSTRAINT builder_config_kind_check;
ALTER TABLE builder_config ADD CONSTRAINT builder_config_kind_check
    CHECK (kind IN ('fake', 'incus', 'microcloud', 'aws', 'openstack'));

CREATE TABLE builder_image_build (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    builder_config_id uuid NOT NULL REFERENCES builder_config(id) ON DELETE CASCADE,
    kind              text NOT NULL DEFAULT 'docker_base',
    status            text NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    image_fingerprint text NOT NULL DEFAULT '',
    error             text NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL DEFAULT now(),
    started_at        timestamptz,
    finished_at       timestamptz
);

-- Only one build should ever be running per builder at a time; the orchestrator
-- leases a pending build by flipping it to running (see StartImageBuild).
CREATE INDEX builder_image_build_pending_idx ON builder_image_build (created_at)
    WHERE status = 'pending';
CREATE INDEX builder_image_build_by_builder_idx ON builder_image_build (builder_config_id, created_at DESC);

-- Append-only progress log, one row per streamed line, ordered by seq -- the
-- live console tails it by "everything after seq N".
CREATE TABLE builder_image_build_log (
    id         bigserial PRIMARY KEY,
    build_id   uuid NOT NULL REFERENCES builder_image_build(id) ON DELETE CASCADE,
    seq        integer NOT NULL,
    line       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX builder_image_build_log_seq_idx ON builder_image_build_log (build_id, seq);

-- +goose Down

DROP TABLE builder_image_build_log;
DROP TABLE builder_image_build;

ALTER TABLE builder_config DROP CONSTRAINT builder_config_kind_check;
ALTER TABLE builder_config ADD CONSTRAINT builder_config_kind_check
    CHECK (kind IN ('fake', 'incus', 'microcloud'));

ALTER TABLE builder_config
    DROP COLUMN container_base_server,
    DROP COLUMN container_base_alias,
    DROP COLUMN docker_base_fingerprint;
