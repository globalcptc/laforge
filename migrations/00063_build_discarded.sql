-- +goose Up
-- 'discarded': the terminal state of a planned build that was never deployed
-- and never will be. The build state diagram (migration 00003) runs
-- planned -> deploying -> ... -> torn_down -> purged, with every exit from
-- 'planned' going through a real deploy; a planned build that is simply no
-- longer wanted had no state to land in, so they accumulated forever (one per
-- configured_build per content revision, plus every planned build whose
-- configured_build was later deleted) and each one kept a 2-second reconcile
-- lane in the orchestrator -- checkout + content load + per-object DB work --
-- starving the live builds on a small box. 'torn_down' was the wrong word for
-- it (the UI reads it as "Destroyed", and nothing was ever created) and
-- deleting the row would drop the build's events and resolved teams/objects,
-- which every other terminal transition in this system keeps as history.
--
-- A planned build is discarded when (a) a newer planned build is created for
-- the same configured_build -- the newer commit supersedes it -- or (b) its
-- configured_build is removed. Nothing at a hoster is touched: a planned build
-- has never created anything (reconcile.go's deployTasks gate), so there is
-- nothing to destroy. The orchestrator stops reconciling a discarded build,
-- the UI shows it read-only like torn_down/purged, and it never blocks
-- removing its configured_build.
ALTER TABLE build DROP CONSTRAINT build_status_check;
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'building', 'finished', 'failed',
                      'tearing_down', 'torn_down', 'purged', 'discarded'));

-- +goose Down
-- A discarded build never had infrastructure, so folding it back into
-- 'torn_down' (the closest pre-existing terminal state) loses nothing real.
UPDATE build SET status = 'torn_down' WHERE status = 'discarded';
ALTER TABLE build DROP CONSTRAINT build_status_check;
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'building', 'finished', 'failed',
                      'tearing_down', 'torn_down', 'purged'));
