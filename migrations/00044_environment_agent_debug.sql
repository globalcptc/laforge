-- agent-debug (environment YAML). When true, every agent in this environment
-- writes a local debug log file beside its binary; when false (the default) the
-- agent is silent on the box and only reports to the LaForge servers. The flag
-- is baked into each agent binary at deploy time (see internal/agentdelivery),
-- so it can't be flipped by editing a box's launcher. Default false so existing
-- environments keep the silent, production-safe posture.

-- +goose Up
ALTER TABLE environment ADD COLUMN agent_debug boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE environment DROP COLUMN agent_debug;
