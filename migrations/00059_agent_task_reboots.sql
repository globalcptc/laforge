-- +goose Up
-- A step that reboots the host itself (a `script:` with `reboots: true`, e.g.
-- dcpromo) never reports a result -- the reboot kills the agent mid-command. The
-- gateway must therefore NOT re-lease and re-run it (which hits "already done"
-- errors like "This server is already domain joined"); instead, the agent
-- reconnecting after the reboot is treated as the step completing. This flag,
-- carried from the script's config through to the task, is how get-task knows a
-- leased task is a reboot-and-done one. Default false: every ordinary step is
-- unaffected.
ALTER TABLE agent_task ADD COLUMN reboots boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE agent_task DROP COLUMN reboots;
