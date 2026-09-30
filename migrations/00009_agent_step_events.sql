-- "Every object has its own log... A build, a team, a network, a host, a
-- step" -- true today for deploy/destroy/access
-- lifecycle events (`event.task_id` -> `task.deployed_object_id`), not
-- for an individual script execution on a host. internal/gateway's own
-- handleReportStatus already knows exactly when a step
-- finishes or fails; it just never wrote that into `event`.
--
-- The fix isn't routing step events through `event.task_id`: that column
-- references `task` (the orchestrator's deploy_network/deploy_host/...
-- rows), not `agent_task` (a script's own step execution) -- a step
-- happens well after its object's deploy task already reached `done`,
-- with no live `task` row left to attribute it to. `event.deployed_object_id`
-- is the direct fix -- the same identity agent_heartbeat and agent_session
-- already reference directly rather than hopping through task, and the
-- one that's actually stable for an object's whole life.
--
-- Nullable and additive: existing deploy/destroy/access events keep using
-- task_id exactly as before (not backfilled here -- a bigger, separate
-- change for no real benefit, since ListEventsByDeployedObject reads
-- both paths). Only new step events set this column; task_id stays NULL
-- for them, since no `task` row is the right owner.

-- +goose Up

ALTER TABLE event ADD COLUMN deployed_object_id uuid REFERENCES deployed_object(id) ON DELETE SET NULL;

CREATE INDEX event_deployed_object_id_idx ON event (deployed_object_id) WHERE deployed_object_id IS NOT NULL;

GRANT INSERT ON event TO laforge_gateway;
-- Deliberately no SELECT -- same reasoning as agent_heartbeat's own
-- grant (migration 00007): the gateway only ever writes its own step
-- events; reading the journal back is the API's job.

-- +goose Down

REVOKE INSERT ON event FROM laforge_gateway;
DROP INDEX event_deployed_object_id_idx;
ALTER TABLE event DROP COLUMN deployed_object_id;
