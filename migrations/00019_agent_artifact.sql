-- Auto-delivery: agents aren't baked into images, they're fetched at first
-- boot. At deploy time the runner issues a per-host client certificate
-- (CN = deployed_object.id, the identity the gateway keys agents by),
-- patches it into a platform base binary, and stores the result here. The
-- instance's cloud-init user-data then downloads it by token from the api
-- (GET /agent-binary/{id}?token=...), installs it as a service, and runs
-- it -- the binary carries its own identity, so the download needs no
-- session, only the one-time token.
--
-- One row per deployed object. The binary is derived and regenerable, and
-- the largest artifact class, so it's deleted once the host checks in
-- (orchestrator sweep) -- the plan's "per-host agent binaries disappear
-- once hosts check in." laforge_gateway gets no access: it never serves or
-- cleans these up.

-- +goose Up

CREATE TABLE agent_artifact (
    deployed_object_id uuid PRIMARY KEY REFERENCES deployed_object(id) ON DELETE CASCADE,
    platform           text NOT NULL,
    token              text NOT NULL UNIQUE,
    agent_binary       bytea NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE agent_artifact;
