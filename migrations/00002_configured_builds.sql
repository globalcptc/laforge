-- "Configured builds: repository, branch, environment file, builder
-- config, follow on/off, competition started on/off, and the commit
-- currently applied."
--
-- builder_config_name is a name, not a foreign key: builder configs are
-- server-side and not part of content (see "Server-side configuration"),
-- and their own table is out of scope here.

-- +goose Up

CREATE TABLE configured_build (
    id                          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id               uuid NOT NULL REFERENCES repository(id) ON DELETE CASCADE,
    branch                      text NOT NULL,
    environment_path            text NOT NULL,
    builder_config_name         text NOT NULL,
    follow_enabled              boolean NOT NULL DEFAULT true,
    competition_started         boolean NOT NULL DEFAULT false,
    current_content_revision_id uuid REFERENCES content_revision(id),
    created_at                  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repository_id, branch, environment_path)
);

-- +goose Down

DROP TABLE configured_build;
