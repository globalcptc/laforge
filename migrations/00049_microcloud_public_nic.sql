-- +goose Up
ALTER TABLE builder_config ADD COLUMN microcloud_public_access jsonb;
ALTER TABLE deployed_object ADD COLUMN public_address text NOT NULL DEFAULT '';

-- Reservations survive failed/ambiguous creates and runner restarts. Do not
-- cascade on builder deletion: a configuration must not free a live VM's IP.
CREATE TABLE microcloud_public_address (
    builder_id uuid NOT NULL REFERENCES builder_config(id),
    project text NOT NULL,
    instance_name text NOT NULL,
    external_name text NOT NULL,
    network text NOT NULL,
    address inet NOT NULL CHECK (family(address) = 4 AND masklen(address) = 32),
    settings jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (builder_id, project, instance_name),
    UNIQUE (builder_id, external_name),
    -- All MicroCloud builds in this database, including overlapping pools in
    -- different builder configs/networks, share this address namespace.
    UNIQUE (address)
);

-- +goose Down
DROP TABLE microcloud_public_address;
ALTER TABLE deployed_object DROP COLUMN public_address;
ALTER TABLE builder_config DROP COLUMN microcloud_public_access;
