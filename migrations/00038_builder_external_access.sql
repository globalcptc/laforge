-- +goose Up
-- Per-cluster external-access config for the shared-IP builders (Incus,
-- MicroCloud): the single external IP that content `public:` ports are NAT'd in
-- on, and the port window allocated on it (a distinct external port per team,
-- host and port). All optional -- empty external_access_ip means this builder
-- doesn't do external ingress yet, and ConfigureExternalAccess then errors
-- clearly. Public-IP builders (AWS, OpenStack) ignore these (a host gets its own
-- IP, no shared port window).
ALTER TABLE builder_config
    ADD COLUMN external_access_ip text,
    ADD COLUMN external_port_min  integer,
    ADD COLUMN external_port_max  integer;

-- +goose Down
ALTER TABLE builder_config
    DROP COLUMN external_access_ip,
    DROP COLUMN external_port_min,
    DROP COLUMN external_port_max;
