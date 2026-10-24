-- Surface WHY a build is stuck. The orchestrator's reconcile loop can fail for
-- reasons that aren't content errors and so can't be caught by `laforge check`
-- -- most notably build/builder compatibility (the chosen builder has no image
-- configured for an os the environment uses). Until now that only went to the
-- orchestrator log and the build sat silently at its status. This column holds
-- the last reconcile error (or NULL when the last reconcile succeeded) so the
-- UI can show it on the build.

-- +goose Up
ALTER TABLE build ADD COLUMN reconcile_error text;

-- +goose Down
ALTER TABLE build DROP COLUMN reconcile_error;
