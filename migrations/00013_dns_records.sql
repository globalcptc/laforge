-- +goose Up

-- "dns" joins network/host/container as a deployed_object kind. The
-- authored DNS behavior ("A records for every host are generated
-- automatically, with custom records added in the environment's dns
-- block") gets exactly one
-- deployed_object per team, diffed by fingerprint the same way everything
-- else is -- see internal/orchestrator/dns.go's reconcileDNS.
ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_kind_check;
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_kind_check
    CHECK (kind IN ('network', 'host', 'container', 'dns'));

-- The fake builder's own DNS "hoster", the same real-Postgres-table shape
-- fake_hoster_resource already uses for network/host/container (see its
-- own comment: a real table, not in-process memory, so a killed runner
-- can't lose what "the hoster" already has). DNS doesn't fit
-- fake_hoster_resource's single-external-ref-per-resource shape (one
-- DeployDNSRecords call replaces a team's whole record set, not one
-- record at a time), so it gets its own table: current rows for a team
-- are exactly the records DeployDNSRecords was last called with for that
-- team, replaced wholesale on every ensure call.
CREATE TABLE fake_dns_record (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team       text NOT NULL,
    name       text NOT NULL,
    type       text NOT NULL,
    value      text NOT NULL,
    priority   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX fake_dns_record_team_idx ON fake_dns_record (team);

-- +goose Down

DROP TABLE fake_dns_record;

ALTER TABLE deployed_object DROP CONSTRAINT deployed_object_kind_check;
ALTER TABLE deployed_object ADD CONSTRAINT deployed_object_kind_check
    CHECK (kind IN ('network', 'host', 'container'));
