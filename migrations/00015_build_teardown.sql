-- Real teardown, found missing entirely by direct audit: the design
-- specifies
-- "teardown (infrastructure), purge artifacts (object storage), delete
-- build... each available from the UI and the API," and build.status's
-- own CHECK constraint (migration 00003) already named 'torn_down' and
-- 'purged' as real terminal states -- but nothing anywhere ever
-- transitioned a build into either one. The only destroy machinery that
-- existed was Reconcile's own per-object diff against desired content,
-- which only ever destroys an object that fell OUT of content, never a
-- whole build on a deliberate operator action, and content might not
-- even be resolvable anymore by the time someone wants to tear a build
-- down (repo access revoked, branch deleted).
--
-- 'tearing_down' is the real trigger state this was missing: set once,
-- by a real POST /builds/{id}/teardown, then internal/orchestrator.Teardown
-- (content-independent, unlike Reconcile) destroys every real object
-- terminally until none are left, at which point the build itself
-- becomes 'torn_down' for real.
--
-- deployed_object.status's own 'destroyed' value has exactly the same
-- shape: already real in migration 00003's CHECK constraint, already
-- read by internal/orchestrator/reconcile.go's own reconcileObject
-- ("let the in-flight (or already-completed-but-not-yet-reset) destroy
-- finish"), but nothing ever wrote it -- every existing destroy path
-- either deletes the row outright (remove: true, a content-driven
-- removal) or resets it to pending for a redeploy (a fingerprint change,
-- "rebuild means recreate"). Teardown is a real third case that column
-- was always missing: destroyed for good, row and its history kept, not
-- coming back.

-- +goose Up

ALTER TABLE build DROP CONSTRAINT build_status_check;
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'deployed', 'failed', 'tearing_down', 'torn_down', 'purged'));

-- +goose Down

ALTER TABLE build DROP CONSTRAINT build_status_check;
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'deployed', 'failed', 'torn_down', 'purged'));
