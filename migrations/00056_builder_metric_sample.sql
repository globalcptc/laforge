-- +goose Up
-- Time-series + gauge samples of a builder's own backend health (CPU, memory,
-- disk, network, API latency, quota usage), collected per builder on an interval
-- by the orchestrator and rendered on the builder page. One row per metric per
-- node per sample; append-only with time-bounded retention like agent_heartbeat.
-- See docs/plans/builder-monitoring.md.
CREATE TABLE builder_metric_sample (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    builder_config_id uuid NOT NULL REFERENCES builder_config(id) ON DELETE CASCADE,
    key text NOT NULL,
    label text NOT NULL DEFAULT '',
    node text NOT NULL DEFAULT '',
    kind text NOT NULL,
    unit text NOT NULL DEFAULT '',
    value double precision NOT NULL,
    max double precision,
    sampled_at timestamp with time zone NOT NULL DEFAULT now()
);
-- Reading a builder's recent history for one metric (the chart query).
CREATE INDEX builder_metric_sample_lookup ON builder_metric_sample (builder_config_id, key, node, sampled_at DESC);
-- Retention sweep by age.
CREATE INDEX builder_metric_sample_sampled_at ON builder_metric_sample (sampled_at);

-- +goose Down
DROP TABLE builder_metric_sample;
