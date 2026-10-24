-- Splits "what about the MicroCloud builder" into what it always should
-- have been: two genuinely separate builder kinds, not one client wearing
-- two names. Confirmed directly: Incus and MicroCloud are real, diverging
-- products with different infrastructure shapes --
--
--   'microcloud': one real MicroCloud cluster (Ceph + OVN + LXD, any
--   member answers for the whole cluster) -- exactly what the existing
--   incus_api_url/incus_client_cert_path/... columns already describe,
--   just correctly named now that 'incus' means something else. No
--   column changes needed for this kind; only the CHECK constraint and
--   what internal/builderconfig.Resolve does with a kind='microcloud' row
--   are different.
--
--   'incus': a POOL of independent, non-clustered Incus hosts -- each its
--   own daemon, its own credentials, no shared storage or OVN control
--   plane between them. Content never says which host a team lands on;
--   internal/builder/incuspool picks one deterministically per team
--   (team number modulo host count -- "round-robin style... spread the
--   load"), so the same team always resolves to the same host across
--   retries without needing anywhere to persist that assignment. Each
--   pool member needs its own full connection shape (api_url, client
--   cert/key paths, server cert, OVN uplink, storage pool, timeout) --
--   the single incus_* columns can't express more than one, so this kind
--   uses the new incus_hosts array instead and leaves incus_api_url etc.
--   NULL. incus_images/incus_sizes stay shared across the whole pool:
--   content's os/image/size names are pool-wide, not per-host (an
--   operator is expected to keep each host's own image catalog
--   consistent, the same operational assumption a real MicroCloud
--   cluster's shared storage makes moot for that kind).

-- +goose Up

ALTER TABLE builder_config DROP CONSTRAINT builder_config_kind_check;
ALTER TABLE builder_config ADD CONSTRAINT builder_config_kind_check
    CHECK (kind IN ('fake', 'incus', 'microcloud'));

-- Each element: {api_url, client_cert_path, client_key_path,
-- server_cert_pem, ovn_uplink_network, storage_pool, operation_timeout_seconds}
-- -- the exact same per-host shape the single incus_* columns already
-- have, just one entry per pool member instead of one set of columns
-- total. Validated by internal/builderconfig.Resolve the same way the
-- single-host columns already are (a missing required field per host is
-- a clear error at registration time, not a runtime surprise) -- there is
-- deliberately no JSON-shape CHECK constraint here, matching this
-- table's own existing incus_images/incus_sizes (also unvalidated JSONB
-- at the schema level, validated in Go).
ALTER TABLE builder_config ADD COLUMN incus_hosts jsonb NOT NULL DEFAULT '[]';

-- +goose Down

ALTER TABLE builder_config DROP COLUMN incus_hosts;

ALTER TABLE builder_config DROP CONSTRAINT builder_config_kind_check;
ALTER TABLE builder_config ADD CONSTRAINT builder_config_kind_check
    CHECK (kind IN ('fake', 'incus'));
