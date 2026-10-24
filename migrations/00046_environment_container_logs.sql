-- container_logs (environment YAML). When set, every container in the
-- environment forwards its console output to an external collector using each
-- builder's native log mechanism (docker log driver, Fargate logConfiguration,
-- the Docker daemon default before `docker compose up`, ...). Stored as the
-- {driver, options} object the author wrote; NULL means no forwarding. Nullable
-- with no default so existing environments are unchanged.

-- +goose Up
ALTER TABLE environment ADD COLUMN container_logs jsonb;

-- +goose Down
ALTER TABLE environment DROP COLUMN container_logs;
