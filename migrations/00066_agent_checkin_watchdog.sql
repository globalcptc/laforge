-- +goose Up
-- Never-checked-in watchdog and boot visibility.
--
-- deployed_object.deployed_at: when the hoster reported the instance created
-- (MarkDeployedObjectRunning), so "deploy -> first agent heartbeat" can be
-- measured and a host whose agent never arrives can be flagged. updated_at
-- can't serve: every status change bumps it.
-- deployed_object.agent_platform: the agent binary the runner delivered
-- (x86_64-pc-windows-gnu / x86_64-unknown-linux-musl). It picks the watchdog
-- deadline (Windows boots far slower than Linux) and marks the objects that
-- are expected to check in at all. agent_artifact.platform can't be used,
-- because the checked-in sweep deletes that row.
-- deployed_object.agent_checkin_overdue_at: set once by the orchestrator when
-- the agent missed its deadline (event + needs-attention); cleared by a redeploy.
ALTER TABLE deployed_object ADD COLUMN deployed_at timestamp with time zone;
ALTER TABLE deployed_object ADD COLUMN agent_platform text NOT NULL DEFAULT '';
ALTER TABLE deployed_object ADD COLUMN agent_checkin_overdue_at timestamp with time zone;

-- Boot timing reported by the agent: when the OS booted and when the agent
-- process started. Recorded per heartbeat (a reboot shows up as a new boot_at)
-- and as the latest value on the session.
ALTER TABLE agent_heartbeat ADD COLUMN boot_at timestamp with time zone;
ALTER TABLE agent_heartbeat ADD COLUMN agent_started_at timestamp with time zone;
ALTER TABLE agent_session ADD COLUMN boot_at timestamp with time zone;
ALTER TABLE agent_session ADD COLUMN agent_started_at timestamp with time zone;

-- Per-script retry override: the most attempts a step gets before a failure
-- is terminal. NULL means the gateway default (gateway.MaxAttempts).
ALTER TABLE agent_task ADD COLUMN max_attempts integer;

-- +goose Down
ALTER TABLE agent_task DROP COLUMN max_attempts;
ALTER TABLE agent_session DROP COLUMN agent_started_at;
ALTER TABLE agent_session DROP COLUMN boot_at;
ALTER TABLE agent_heartbeat DROP COLUMN agent_started_at;
ALTER TABLE agent_heartbeat DROP COLUMN boot_at;
ALTER TABLE deployed_object DROP COLUMN agent_checkin_overdue_at;
ALTER TABLE deployed_object DROP COLUMN agent_platform;
ALTER TABLE deployed_object DROP COLUMN deployed_at;
