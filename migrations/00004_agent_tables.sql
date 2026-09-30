-- "New agent + agent-gateway: the protocol, the full command
-- set, the starting validators, the agent factory... the gateway as its
-- own service with a restricted database role." agent_session was
-- explicitly excluded from the runtime tables ("that belongs to
-- the agent-gateway") -- this is that.

-- +goose Up

-- One row per deployed_object that has ever checked in as an agent.
-- Liveness only -- the identity that actually matters is the mTLS client
-- certificate the gateway's TLS layer already verified before any of this
-- is touched; cert_fingerprint here is for observability (which cert an
-- agent presented), never re-trusted as an auth decision on its own.
CREATE TABLE agent_session (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    deployed_object_id uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    cert_fingerprint   text NOT NULL,
    first_seen_at      timestamptz NOT NULL DEFAULT now(),
    last_heartbeat_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deployed_object_id)
);

CREATE INDEX agent_session_fingerprint_idx ON agent_session (cert_fingerprint);

-- The per-host step queue: a host's own Host.Steps/Container.Steps,
-- resolved and rendered once (see internal/gateway/steps.go), become one
-- row per step here as soon as the host is deployed. Same lease shape as
-- the `task` table on purpose -- "a runner takes a task with
-- FOR UPDATE SKIP LOCKED, gets a lease, and heartbeats" is the same
-- mechanism, with the agent itself now doing the leasing.
--
-- Ordering matters (steps run in the order authored), which is why this
-- is a queue keyed by step_index rather than a free-for-all like
-- the task table: an agent must finish step N before starting
-- N+1, so get-task only ever offers the lowest not-yet-done step_index
-- for that host.
CREATE TABLE agent_task (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    deployed_object_id uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    step_index         integer NOT NULL,
    command            text NOT NULL, -- execute | write_file | append_file | extract | delete | change_perms | download | upload | create_user | set_password | add_to_group | service | reboot | validate
    payload            jsonb NOT NULL DEFAULT '{}',
    status             text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'leased', 'done', 'failed')),
    lease_expires_at   timestamptz,
    attempts           integer NOT NULL DEFAULT 0,
    output             text,
    last_error         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deployed_object_id, step_index)
);

CREATE INDEX agent_task_leasable_idx ON agent_task (deployed_object_id, status, step_index);

CREATE TABLE validator_result (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_task_id  uuid NOT NULL REFERENCES agent_task(id) ON DELETE CASCADE,
    kind           text NOT NULL,
    args           jsonb NOT NULL DEFAULT '{}',
    passed         boolean NOT NULL,
    message        text,
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- "Least privilege in the database. It connects as its own Postgres role
-- that can record heartbeats and task results and read tasks addressed to
-- the agent that asked. It cannot read builder configs, cannot reach
-- another team's data, cannot start builds." The password here is a
-- known, documented dev-only placeholder
-- -- a real deployment issues this role a real secret out of band, the
-- same way it would for any other credential; the role and its exact
-- grant surface are the real, load-bearing part of this migration.
CREATE ROLE laforge_gateway LOGIN PASSWORD 'laforge_gateway_dev_only_do_not_use_in_prod';

GRANT SELECT, INSERT, UPDATE ON agent_session TO laforge_gateway;
GRANT SELECT, UPDATE ON agent_task TO laforge_gateway;
GRANT INSERT ON validator_result TO laforge_gateway;
GRANT SELECT ON deployed_object, team, build TO laforge_gateway;
-- Deliberately NOT granted: repository, content_revision,
-- configured_build, environment/network/host/container/script/person, or
-- anything under the content/ingest surface or a future
-- builder-config table. A compromised gateway connection can heartbeat,
-- fetch the next step for the specific host asking, and report a result
-- -- nothing else. See internal/gateway's own tests, which connect AS
-- this role and assert a forbidden query is actually rejected by
-- Postgres, not just by application code choosing not to run it.

-- +goose Down

REVOKE ALL ON deployed_object, team, build FROM laforge_gateway;
REVOKE ALL ON validator_result FROM laforge_gateway;
REVOKE ALL ON agent_task FROM laforge_gateway;
REVOKE ALL ON agent_session FROM laforge_gateway;
DROP ROLE IF EXISTS laforge_gateway;
DROP TABLE validator_result;
DROP TABLE agent_task;
DROP TABLE agent_session;
