-- +goose Up
ALTER TABLE builder_config ADD COLUMN incus_one_team_per_host boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE builder_config DROP COLUMN incus_one_team_per_host;
