-- +goose Up
-- Basic host metrics carried on every agent heartbeat, stored per-heartbeat on
-- the append-only agent_heartbeat log so the heartbeats view can show them and a
-- usage-over-time chart can be built later. All nullable: an older agent (or one
-- that couldn't read a metric) simply sends none, and pre-existing rows have
-- none. Collected uniformly across Linux and Windows (internal to the agent, via
-- the sysinfo crate): cpu/memory/disk are instantaneous percentages, network is
-- a per-second rate computed between heartbeats. laforge_gateway's existing
-- INSERT grant on agent_heartbeat (migration 00007) already covers new columns.
ALTER TABLE agent_heartbeat
    ADD COLUMN cpu_pct    double precision,
    ADD COLUMN mem_pct    double precision,
    ADD COLUMN disk_pct   double precision,
    ADD COLUMN net_rx_bps double precision,
    ADD COLUMN net_tx_bps double precision;

-- +goose Down
ALTER TABLE agent_heartbeat
    DROP COLUMN cpu_pct,
    DROP COLUMN mem_pct,
    DROP COLUMN disk_pct,
    DROP COLUMN net_rx_bps,
    DROP COLUMN net_tx_bps;
