-- +goose Up
-- Revalidation: re-run every validator a host already has, on the box as it
-- is, and report the results -- nothing else runs. One click makes a report
-- (report_id) with one run per host. A run's agent_task rows are copies of the
-- host's materialized validate tasks, tagged with validation_run_id so they
-- stay out of the host's build (its step counts, its Steps tab, and the
-- validator results that decide 'invalid' at the end of a build).
--
-- status: running until every copied task is through, then passed / failed;
-- canceled when its tasks disappeared (the host was rebuilt or restarted);
-- timed_out when the agent didn't run them in time. The only change a run
-- makes to the host is finished <-> invalid (status_before / status_after).
CREATE TABLE validation_run (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id             uuid NOT NULL,
    build_id              uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    deployed_object_id    uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    created_by_account_id uuid REFERENCES account(id) ON DELETE SET NULL,
    status                text NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'passed', 'failed', 'canceled', 'timed_out')),
    status_before         text NOT NULL,
    status_after          text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    finished_at           timestamptz
);
CREATE INDEX validation_run_report_idx ON validation_run (report_id);
CREATE INDEX validation_run_object_idx ON validation_run (deployed_object_id, created_at DESC);
CREATE INDEX validation_run_open_idx ON validation_run (build_id) WHERE status = 'running';

ALTER TABLE agent_task ADD COLUMN validation_run_id uuid REFERENCES validation_run(id) ON DELETE CASCADE;
CREATE INDEX agent_task_validation_run_idx ON agent_task (validation_run_id) WHERE validation_run_id IS NOT NULL;

-- +goose Down
ALTER TABLE agent_task DROP COLUMN validation_run_id;
DROP TABLE validation_run;
