-- +goose Up
-- Where external access actually lands. Content's `public:` ports (a topology
-- copy's subset of its host's own ports, meaning "reachable from outside the
-- competition network") are builder-agnostic INTENT; the builder realizes them
-- -- a public IP per host on AWS, a port-NAT on a shared uplink IP on
-- Incus/MicroCloud -- and reports back the real address:port an operator or
-- competitor connects to. That mapping is recorded here, one row per
-- (object, protocol, internal port), so it survives reconciles, never collides,
-- and is the single place the UI/CLI read connection endpoints from. Rows cascade
-- away with their deployed_object. external_port equals internal_port on a
-- public-IP builder (no remap); it is the allocated shared-IP port otherwise.
CREATE TABLE external_access (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    deployed_object_id uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    protocol           text NOT NULL,   -- 'tcp' | 'udp'
    internal_port      text NOT NULL,   -- the host's own port (e.g. '3389' for RDP)
    public_address     text NOT NULL,   -- 'ip:port' to connect to from outside
    external_port      text NOT NULL,   -- the external-side port (shared-IP builders); = internal_port on a public-IP builder
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deployed_object_id, protocol, internal_port)
);

CREATE INDEX external_access_object_idx ON external_access (deployed_object_id);

-- +goose Down
DROP TABLE external_access;
