-- Builder connections are enrolled with an Incus trust token (`incus
-- config trust add <name>` on the server), not configured by hand: LaForge
-- verifies the server's certificate against the fingerprint the token
-- carries, generates its own client keypair, and enrolls it. The result
-- lives here, server-side, so a private key never travels through the
-- browser and no service needs a credential file on its own local disk
-- (which can't work once api/runner/orchestrator are separate hosted
-- services).
--
-- No grant to laforge_gateway: that role only has explicit per-table
-- grants (migration 00004), so it can't read this table.

-- +goose Up

CREATE TABLE builder_credential (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    api_url            text NOT NULL,
    server_name        text NOT NULL DEFAULT '',
    server_fingerprint text NOT NULL,
    server_cert_pem    text NOT NULL,
    client_cert_pem    text NOT NULL,
    client_key_pem     text NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);

-- kind "microcloud" references one credential here; kind "incus" pool
-- members carry their own credential_id inside incus_hosts' JSON. The
-- older path-based columns stay for configs created before this existed.
ALTER TABLE builder_config ADD COLUMN incus_credential_id uuid REFERENCES builder_credential(id);

-- +goose Down

ALTER TABLE builder_config DROP COLUMN incus_credential_id;
DROP TABLE builder_credential;
