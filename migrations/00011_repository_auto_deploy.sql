-- Real per-repository auto-deploy toggle. "Auto-deploy follows the
-- environment's state by
-- default... Both can be turned off per repository, for an event where
-- every deploy should be a decision someone makes deliberately." This is
-- deliberately a NEW column, not a reuse of configured_build.follow_enabled
-- (which already gates auto-*build*, per-configured-build): the spec
-- is explicit that "what happens then is set by an admin when the
-- repository is added," i.e. a repository-wide setting, and auto-build
-- and auto-deploy are two separately toggleable things ("Both can be
-- turned off"), not one shared switch. Defaults to true, matching the
-- spec's own stated default ("there is no reason not to" -- said of
-- auto-build, and auto-deploy is introduced the same way, right after).

-- +goose Up

ALTER TABLE repository ADD COLUMN auto_deploy_enabled boolean NOT NULL DEFAULT true;

-- +goose Down

ALTER TABLE repository DROP COLUMN auto_deploy_enabled;
