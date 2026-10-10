-- +goose Up
-- Per-builder AWS account infrastructure, previously only configurable through
-- runner AWS_* environment variables: ECS cluster, task-execution role, log
-- group, region, agent-init image, and whether hosts auto-assign a public IP.
-- Stored as one JSONB blob (internal/builder/aws.ConfigInput), the same per-kind
-- pattern microcloud_public_access uses. Empty fields fall back to the AWS_*
-- env vars in internal/builder/aws.New, so an env-only deployment is unchanged.
ALTER TABLE builder_config ADD COLUMN aws_config jsonb;

-- +goose Down
ALTER TABLE builder_config DROP COLUMN aws_config;
