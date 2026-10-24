-- +goose Up
-- A host's own events (its panel's Events tab) include those of its tasks,
-- found through event.task_id, which had no index: every panel open scanned
-- the whole event table (1.4M rows on a busy server).
CREATE INDEX IF NOT EXISTS event_task_id_idx ON event (task_id) WHERE task_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS event_task_id_idx;
