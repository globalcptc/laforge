-- Builder backend health samples (see docs/plans/builder-monitoring.md).

-- name: InsertBuilderMetricSample :exec
INSERT INTO builder_metric_sample (builder_config_id, key, label, node, kind, unit, value, max, sampled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListBuilderMetricSamples :many
-- A builder's samples within a window, oldest first, for the time-series charts.
-- Gauges come back too; the caller takes the latest per (key,node) for those.
SELECT key, label, node, kind, unit, value, max, sampled_at
FROM builder_metric_sample
WHERE builder_config_id = $1 AND sampled_at >= $2
ORDER BY sampled_at ASC;

-- name: DeleteBuilderMetricSamplesOlderThan :execrows
DELETE FROM builder_metric_sample WHERE sampled_at < $1;
