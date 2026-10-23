-- +goose Up
-- Make the builder part of a configured build's identity, so the SAME branch +
-- environment can be built on DIFFERENT builders (e.g. the same lab on Incus and
-- on AWS at once). The old key was (repository, branch, environment_path); the
-- new one adds builder_config_name. This only widens the key -- every row unique
-- under the old constraint is still unique under the new one -- so it applies
-- cleanly with no data conflict and needs no writer stop.
ALTER TABLE configured_build
  DROP CONSTRAINT configured_build_repository_id_branch_environment_path_key;
ALTER TABLE configured_build
  ADD CONSTRAINT configured_build_repo_branch_env_builder_key
  UNIQUE (repository_id, branch, environment_path, builder_config_name);

-- +goose Down
ALTER TABLE configured_build
  DROP CONSTRAINT configured_build_repo_branch_env_builder_key;
ALTER TABLE configured_build
  ADD CONSTRAINT configured_build_repository_id_branch_environment_path_key
  UNIQUE (repository_id, branch, environment_path);
