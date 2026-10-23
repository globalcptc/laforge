-- +goose Up
-- destroy_queued: a destroy has been asked for (a rebuild, or a content change
-- that needs the instance recreated) but hasn't started yet. The object keeps
-- its hoster reference until the runner destroys it; only then does it go to
-- pending for a fresh deploy. It used to be reset to pending at once, which
-- looked exactly like a host waiting for its first deploy.
ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check CHECK (status = ANY (ARRAY['pending', 'deploying', 'running', 'building', 'finished', 'deploy_failed', 'build_failed', 'invalid', 'destroy_queued', 'destroying', 'destroy_failed', 'destroyed']));

-- +goose Down
UPDATE deployed_object SET status = 'pending' WHERE status = 'destroy_queued';
ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check CHECK (status = ANY (ARRAY['pending', 'deploying', 'running', 'building', 'finished', 'deploy_failed', 'build_failed', 'invalid', 'destroying', 'destroy_failed', 'destroyed']));
