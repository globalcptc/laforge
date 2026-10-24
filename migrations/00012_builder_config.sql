-- The real answer to "what about the MicroCloud builder": internal/builder/incus
-- has been real, tested, and (2026-09-25) proven live against an actual
-- MicroCloud cluster -- but nothing in the running system could ever
-- actually select it. cmd/laforge-runner unconditionally constructed
-- internal/builder/fake, and configured_build.builder_config_name,
-- despite being required at build-configuration time, was never read by
-- anything downstream -- a name captured for a lookup that didn't exist.
--
-- builder_config is that lookup, made real: one row per named builder an
-- admin has registered, kind 'fake' (dev/test, no cluster needed) or
-- 'incus' (a real MicroCloud/Incus cluster). Deliberately keyed by name,
-- not referenced by builder_config_name via a hard foreign key: dozens
-- of existing tests already create a configured_build with an arbitrary
-- builder_config_name string ("microcloud") and no matching row, and
-- nothing about that needs to become an error at configuration time --
-- exactly like a `schedule:` entry firing against content that's since
-- changed, "no such builder config" is a real, clear error at the one
-- point it actually matters: when a task genuinely needs to run against
-- it (internal/runner's own Builders resolver).
--
-- Client credentials (cert/key) are stored as filesystem PATHS, not
-- key material in this table -- matching this codebase's own existing
-- pattern for every other credential (GITHUB_APP_PRIVATE_KEY_PATH is a
-- path read from disk, never a column; see cmd/laforge-api/main.go's own
-- doc comment on it), not a new, divergent way of handling secrets.
-- incus_server_cert_pem is the one real exception: it's the PINNED
-- SERVER certificate (internal/builder/incus.FetchServerCertificateInsecure's
-- own TOFU bootstrap output), a public value, not a secret -- storing it
-- inline is what makes it usable without also needing filesystem
-- provisioning just to register a cluster.

-- +goose Up

CREATE TABLE builder_config (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     text NOT NULL UNIQUE,
    kind                     text NOT NULL CHECK (kind IN ('fake', 'incus')),
    incus_api_url            text,
    incus_client_cert_path   text,
    incus_client_key_path    text,
    incus_server_cert_pem    text,
    incus_ovn_uplink_network text,
    incus_storage_pool       text,
    -- Nullable, defaulting to internal/builder/incus.NewClient's own
    -- 120s when unset. Made configurable per builder_config, not left as
    -- the client's own fixed constant, because how long a real deploy
    -- can take is a genuine property of the cluster (its storage
    -- backend, network distance) -- found live: a real VM's first image
    -- clone over Ceph RBD took ~1m46s,
    -- close enough to the 120s default that a slightly larger image or a
    -- busier cluster would exceed it.
    incus_operation_timeout_seconds integer,
    incus_images             jsonb NOT NULL DEFAULT '{}',
    incus_sizes              jsonb NOT NULL DEFAULT '{}',
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE builder_config;
