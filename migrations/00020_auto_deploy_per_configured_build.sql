-- Auto-deploy is per configured build (per branch), not repository-wide,
-- and the separate "follow" (auto-build) switch is gone: auto-build is
-- always on for a configured build (it's free and validates content), so
-- the only per-branch choice is whether to also deploy. A locked build
-- (competition_started) stays frozen -- no auto-build, no auto-deploy.
--
-- Migration of intent: a configured build that had follow_enabled=false
-- was deliberately not tracking its branch; that "frozen" state now lives
-- on competition_started (the lock), so carry it over before dropping the
-- column. repository.auto_deploy_enabled is removed entirely.

-- +goose Up

ALTER TABLE configured_build ADD COLUMN auto_deploy_enabled boolean NOT NULL DEFAULT true;

UPDATE configured_build SET competition_started = true WHERE follow_enabled = false;
ALTER TABLE configured_build DROP COLUMN follow_enabled;

ALTER TABLE repository DROP COLUMN auto_deploy_enabled;

-- +goose Down

ALTER TABLE repository ADD COLUMN auto_deploy_enabled boolean NOT NULL DEFAULT true;

ALTER TABLE configured_build ADD COLUMN follow_enabled boolean NOT NULL DEFAULT true;
ALTER TABLE configured_build DROP COLUMN auto_deploy_enabled;
