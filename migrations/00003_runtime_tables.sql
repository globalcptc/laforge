-- Runtime tables, deliberately excluded from the content schema
-- ("Deliberately NOT here: the 'Runtime (internal only)' tables..."). This
-- is the "Orchestrator + runner contract with a fake builder, plus chaos
-- tests... that must end converged."
--
-- The schema and the leasing mechanism below, proven against real Postgres:
-- "a runner takes a task with FOR UPDATE SKIP LOCKED, gets a
-- lease, and heartbeats. A dead runner's lease expires and the task is
-- re-leased." agent_session is NOT here -- that belongs to the
-- agent-gateway and doesn't exist yet.

-- +goose Up

-- "Build and deploy are separate" state machine:
-- planned -> deploying -> deployed/failed -> torn_down -> purged.
-- content_revision_id is the resolved commit this build is FROM (see
-- configured_build.current_content_revision_id, which is what usually
-- creates one); environment_name picks which environment in that revision.
CREATE TABLE build (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    configured_build_id  uuid REFERENCES configured_build(id) ON DELETE SET NULL,
    content_revision_id  uuid NOT NULL REFERENCES content_revision(id) ON DELETE CASCADE,
    environment_name     text NOT NULL,
    status               text NOT NULL DEFAULT 'planned'
                             CHECK (status IN ('planned', 'deploying', 'deployed', 'failed', 'torn_down', 'purged')),
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE team (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id     uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    team_number  integer NOT NULL,
    UNIQUE (build_id, team_number)
);

-- One row per resolved network/host/container COPY within a team -- the
-- "observed" half of "the orchestrator diffs desired vs observed state in
-- Postgres." fingerprint is "the rendered output is the fingerprint":
-- everything that would change a single byte the agent receives, hashed
-- together (see internal/orchestrator). A fingerprint mismatch on an
-- already-deployed object means "rebuild means recreate": destroy, then
-- (on a later reconcile pass, once destroyed) deploy again fresh.
CREATE TABLE deployed_object (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       uuid NOT NULL REFERENCES team(id) ON DELETE CASCADE,
    kind          text NOT NULL CHECK (kind IN ('network', 'host', 'container')),
    object_name   text NOT NULL,           -- the content DEFINITION this copy came from, e.g. "kali"
    as_name       text,                    -- the copy's own hostname; null for networks
    network_name  text,                    -- which network a host/container sits on; null for networks themselves
    fingerprint   text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'deploying', 'deployed', 'failed', 'destroying', 'destroyed')),
    external_ref  text,                    -- builder-assigned id/name; how Inspect/adoption identify it later
    last_error    text,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Identity per COPY, not per definition: "listing a host twice puts two
-- copies of it on the network", e.g. kali01 and
-- kali02 both from the `kali` host definition, in the same team. A
-- host/container copy's real identity is its own `as` hostname, which the
-- loader's own cross-checks already guarantee is unique within an
-- environment; object_name alone would collide the two copies into one
-- row. A network has no `as` (as_name is null for it), so COALESCE falls
-- back to object_name, which is what actually identifies a network.
CREATE UNIQUE INDEX deployed_object_identity_idx
    ON deployed_object (team_id, kind, COALESCE(as_name, object_name));

-- The lease/reclaim primitive itself, proven in the spike.
-- deployed_object_id is nullable for team-level tasks (open_access /
-- close_access have no single object). The partial unique index is what
-- makes Reconcile safe to call repeatedly and concurrently from multiple
-- stateless orchestrator replicas without ever creating two open tasks for
-- the same object: `INSERT ... ON CONFLICT DO NOTHING` against it.
CREATE TABLE task (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id           uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    deployed_object_id uuid REFERENCES deployed_object(id) ON DELETE CASCADE,
    kind               text NOT NULL, -- deploy_network | deploy_host | deploy_container | destroy_network | destroy_host | destroy_container | open_access | close_access
    payload            jsonb NOT NULL DEFAULT '{}',
    status             text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'leased', 'done', 'failed')),
    attempts           integer NOT NULL DEFAULT 0,
    lease_owner        text,
    lease_expires_at   timestamptz,
    last_error         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX task_leasable_idx ON task (status, lease_expires_at);

CREATE UNIQUE INDEX task_one_open_per_object ON task (deployed_object_id)
    WHERE status IN ('pending', 'leased') AND deployed_object_id IS NOT NULL;

-- Append-only event journal: "every state change is an append-only event,
-- feeding status, audit, and post-mortems."
CREATE TABLE event (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id   uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    task_id    uuid REFERENCES task(id) ON DELETE SET NULL,
    kind       text NOT NULL,
    message    text NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX event_build_idx ON event (build_id, created_at);

-- The fake builder's "hoster": a real table, separate from any single
-- runner process, so killing a runner genuinely can't lose what "the
-- hoster" already has -- exactly like a real cloud API wouldn't. Ensure
-- semantics (INSERT ... ON CONFLICT DO NOTHING keyed on external_ref) are
-- what proves "a retry after partial success converges instead of
-- duplicating" against something that outlives the runner, not just
-- in-process state.
CREATE TABLE fake_hoster_resource (
    external_ref  text PRIMARY KEY,
    kind          text NOT NULL CHECK (kind IN ('network', 'host', 'container')),
    ensure_count  integer NOT NULL DEFAULT 1,
    destroyed     boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE fake_hoster_resource;
DROP TABLE event;
DROP TABLE task;
DROP TABLE deployed_object;
DROP TABLE team;
DROP TABLE build;
