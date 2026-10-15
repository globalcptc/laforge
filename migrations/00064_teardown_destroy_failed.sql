-- Teardown that stops instead of looping forever. internal/orchestrator.Teardown
-- re-requests a destroy for every not-yet-destroyed object on every pass so a
-- transient builder failure (a network still "in use" while its instances
-- detach) self-heals. When the failure is PERMANENT -- a builder configuration
-- the destroy can never succeed against (incuspool refusing a team because the
-- host list cannot hold it, a project name too long for the hoster) -- that same
-- loop created a fresh destroy task every pass, each failing its three attempts:
-- two builds on prod accumulated ~210,000 failed destroy rows and kept a runner
-- churning for days.
--
-- task.permanent records the runner's own classification (builder.ErrPermanentConfig)
-- on the failed row, so the orchestrator can read it back without re-parsing
-- error text. deployed_object gains the terminal 'destroy_failed' state the
-- lifecycle was missing: "the hoster could not remove this and retrying will
-- not help" -- the destroy-side twin of deploy_failed. An operator fixes the
-- builder configuration and retries the teardown explicitly (POST
-- /builds/{id}/teardown/retry), or forces the build to torn_down.
--
-- The (deployed_object_id, created_at) index serves the orchestrator's
-- latest-destroy-task-per-object lookup; until now task had no index usable
-- for a per-object history scan at all (only the partial open-task uniqueness
-- index), which the wedged prod builds turned into a 200k-row sequential scan
-- every two seconds.

-- +goose Up

ALTER TABLE task ADD COLUMN permanent boolean NOT NULL DEFAULT false;
CREATE INDEX task_object_history_idx ON task (deployed_object_id, created_at DESC) WHERE deployed_object_id IS NOT NULL;

ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check
    CHECK (status IN ('pending', 'deploying', 'running', 'building', 'finished',
                      'deploy_failed', 'build_failed', 'invalid',
                      'destroying', 'destroy_failed', 'destroyed'));

-- +goose Down

ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
UPDATE deployed_object SET status = 'destroying' WHERE status = 'destroy_failed';
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check
    CHECK (status IN ('pending', 'deploying', 'running', 'building', 'finished',
                      'deploy_failed', 'build_failed', 'invalid',
                      'destroying', 'destroyed'));

DROP INDEX task_object_history_idx;
ALTER TABLE task DROP COLUMN permanent;
