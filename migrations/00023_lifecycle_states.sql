-- Redesign the host/container and build lifecycle vocabulary so status
-- distinguishes "infrastructure created" from "agent building" from
-- "everything finished." The old model collapsed the whole lifecycle into
-- one success value ('deployed') the moment the hoster returned an
-- instance -- a host was reported deployed and green before a single
-- authored step had run. An environment must not read as finished (green)
-- until every agent step and validator has actually completed.
--
-- New deployed_object.status vocabulary:
--   pending       - row created, nothing built yet
--   deploying      - the hoster is creating the instance
--   running        - the instance is up; the agent has not checked in yet
--   building       - the agent has checked in and is running steps/validators
--   finished       - up, all steps done, all validators passed (the only green)
--   deploy_failed  - the hoster couldn't create the instance (was 'failed')
--   build_failed   - a build step couldn't be completed
--   invalid        - steps completed but a validator failed
--   destroying     - being torn down
--   destroyed      - removed from the hoster
--
-- New build.status vocabulary mirrors the phase split:
--   planned -> deploying (infra) -> building (agents) -> finished, with
--   'failed' the aggregate when any object failed at any layer. teardown
--   states ('tearing_down','torn_down','purged') are unchanged.
--
-- Existing rows: an old 'deployed' object maps to 'running' (infra is up;
-- the object-lifecycle poll re-derives building/finished from the real
-- agent_task + validator_result rows on the next tick), and an old
-- 'failed' object maps to 'deploy_failed' (the only failure the old model
-- could record was an infra one). An old 'deployed' build maps to
-- 'finished'.

-- +goose Up

ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
UPDATE deployed_object SET status = 'running' WHERE status = 'deployed';
UPDATE deployed_object SET status = 'deploy_failed' WHERE status = 'failed';
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check
    CHECK (status IN ('pending', 'deploying', 'running', 'building', 'finished',
                      'deploy_failed', 'build_failed', 'invalid',
                      'destroying', 'destroyed'));

ALTER TABLE build DROP CONSTRAINT build_status_check;
UPDATE build SET status = 'finished' WHERE status = 'deployed';
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'building', 'finished', 'failed',
                      'tearing_down', 'torn_down', 'purged'));

-- +goose Down

ALTER TABLE build DROP CONSTRAINT build_status_check;
UPDATE build SET status = 'deployed' WHERE status IN ('building', 'finished');
ALTER TABLE build ADD CONSTRAINT build_status_check
    CHECK (status IN ('planned', 'deploying', 'deployed', 'failed',
                      'tearing_down', 'torn_down', 'purged'));

ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_status_check;
UPDATE deployed_object SET status = 'deployed'
    WHERE status IN ('running', 'building', 'finished');
UPDATE deployed_object SET status = 'failed'
    WHERE status IN ('deploy_failed', 'build_failed', 'invalid');
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_status_check
    CHECK (status IN ('pending', 'deploying', 'deployed', 'failed',
                      'destroying', 'destroyed'));
