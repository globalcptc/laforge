-- +goose Up
-- The read-only MCP server (docs/plans/mcp-readonly.md, internal/mcp) connects
-- as its own role, laforge_mcp, and this grant surface IS its read-only
-- guarantee: Postgres itself refuses every write and every column not listed
-- here, whatever the Go code tries. Same idea as laforge_gateway (00004).
--
-- MCP shows what a read-level person sees in the UI, passwords and step
-- payloads included; most of that comes through the API's own read pages
-- (internal/api/mcp_bridge.go), and these grants cover the tools that query
-- directly. Grants are column-level: what the API never shows anyone --
-- builder and registry credentials, sessions and GitHub tokens, agent
-- download tokens, the password-generator secret -- is never granted.
--
-- The role is created only if missing: roles are cluster-wide, so a second
-- database on the same Postgres (a test database) must not fail on it.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'laforge_mcp') THEN
        CREATE ROLE laforge_mcp LOGIN PASSWORD 'laforge_mcp_dev_only_do_not_use_in_prod';
    END IF;
END
$$;
-- +goose StatementEnd

-- A runaway query from an agent never holds a connection for long, and the
-- role can't open more than a handful.
ALTER ROLE laforge_mcp SET statement_timeout = '5s';
ALTER ROLE laforge_mcp CONNECTION LIMIT 6;

-- One row per MCP tool call (who, which key, what, how long), shown on the
-- API Keys page. Rows are pruned after 30 days by the orchestrator.
CREATE TABLE mcp_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id uuid NOT NULL REFERENCES api_key(id) ON DELETE CASCADE,
    account_id uuid NOT NULL,
    tool text NOT NULL,
    args jsonb NOT NULL DEFAULT '{}'::jsonb,
    build_id uuid,
    status text NOT NULL CHECK (status IN ('ok', 'denied', 'error', 'rate_limited')),
    error text NOT NULL DEFAULT '',
    duration_ms integer NOT NULL DEFAULT 0,
    result_bytes integer NOT NULL DEFAULT 0,
    remote_addr text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX mcp_audit_account_time_idx ON mcp_audit (account_id, created_at);
CREATE INDEX mcp_audit_time_idx ON mcp_audit (created_at);

-- Keys: looked up by hash on every request, and last-used bumped. Nothing else.
-- (last_used_at is readable because the throttled bump's WHERE reads it.)
GRANT SELECT (id, account_id, name, prefix, token_hash, scope, repository_ids, expires_at, revoked_at, last_used_at) ON api_key TO laforge_mcp;
GRANT UPDATE (last_used_at) ON api_key TO laforge_mcp;
GRANT SELECT (id, github_login) ON account TO laforge_mcp;
GRANT INSERT ON mcp_audit TO laforge_mcp;

-- Repositories, content and builds.
GRANT SELECT (id, github_owner, github_repo, created_at, installation_id) ON repository TO laforge_mcp;
GRANT SELECT (id, repository_id, branch, environment_path, builder_config_name, competition_started,
              auto_deploy_enabled, hold_on_step_change, current_content_revision_id, created_at)
    ON configured_build TO laforge_mcp;
GRANT SELECT (id, configured_build_id, content_revision_id, resolved_revision_id, environment_name, status,
              created_at, auto_built, reconcile_error, created_by_account_id)
    ON build TO laforge_mcp;
GRANT SELECT (id, repository_id, commit_sha, ref, valid, validation_errors, validated_at, created_at,
              commit_message, commit_author, committed_at)
    ON content_revision TO laforge_mcp;
-- Authored content, minus everything that can hold a secret: no vars, no
-- steps (set_password/create_user arguments), no schedule bodies, no people.
GRANT SELECT (id, content_revision_id, path, name, description, teams, start_at, stop_at, access, cidr,
              extends, schema_version)
    ON environment TO laforge_mcp;
GRANT SELECT (id, content_revision_id, path, name, cidr, tags, extends, visible_from) ON network TO laforge_mcp;
GRANT SELECT (id, content_revision_id, path, name, os, size, disk_gb, ports, depends_on, tags, extends) ON host TO laforge_mcp;
GRANT SELECT (id, content_revision_id, path, name, image, size, ports, depends_on, tags, extends) ON container TO laforge_mcp;
GRANT SELECT (environment_id, network_name, object_kind, object_name, as_name, last_octet, public) ON placement TO laforge_mcp;

-- What's deployed and how it's going.
GRANT SELECT ON team TO laforge_mcp;
GRANT SELECT ON deployed_object TO laforge_mcp;
GRANT SELECT ON event TO laforge_mcp;
GRANT SELECT ON external_access TO laforge_mcp;
GRANT SELECT ON validation_run TO laforge_mcp;
GRANT SELECT ON validator_result TO laforge_mcp;
GRANT SELECT (deployed_object_id, first_seen_at, last_heartbeat_at, boot_at, agent_started_at)
    ON agent_session TO laforge_mcp;
GRANT SELECT (id, deployed_object_id, remote_addr, next_poll_ms, created_at, cpu_pct, mem_pct, disk_pct,
              net_rx_bps, net_tx_bps, boot_at, agent_started_at)
    ON agent_heartbeat TO laforge_mcp;
-- Agent steps and ad-hoc/scheduled runs, payloads (the arguments, passwords
-- included) and output alike: anyone who can read a build sees these in the UI.
GRANT SELECT (id, deployed_object_id, step_index, command, payload, status, lease_expires_at, attempts, output, last_error,
              created_at, updated_at, ignore_errors, ad_hoc, reboots, started_at, finished_at, max_attempts,
              milestone, step_script, wait_for, scheduled_task_id, step_group, validation_run_id)
    ON agent_task TO laforge_mcp;
GRANT SELECT (id, build_id, source, deployed_object_id, schedule_index, target, when_expr, command, payload, anchor,
              fires_once, next_fire_at, status, created_by, created_at, updated_at)
    ON scheduled_task TO laforge_mcp;
-- The runner's deploy/destroy queue, without payload (builder specs).
GRANT SELECT (id, build_id, deployed_object_id, kind, status, attempts, lease_owner, lease_expires_at, last_error,
              created_at, updated_at, permanent)
    ON task TO laforge_mcp;
GRANT SELECT (id, deployed_object_id, opened_by_account_id, status, client_addr, started_at, ended_at)
    ON shell_session TO laforge_mcp;

-- Builders: their settings minus credentials (those live in
-- builder_credential, never granted) and client key/cert paths.
GRANT SELECT (id, name, kind, color, incus_api_url, incus_project, incus_storage_pool, incus_ovn_uplink_network,
              incus_operation_timeout_seconds, incus_images, incus_sizes, incus_one_team_per_host,
              incus_host_max_concurrency, microcloud_cluster_group, container_base_server, container_base_alias,
              docker_base_fingerprint, external_access_ip, external_port_min, external_port_max, max_concurrency,
              created_at, updated_at)
    ON builder_config TO laforge_mcp;
GRANT SELECT ON builder_metric_sample TO laforge_mcp;
GRANT SELECT ON builder_image_build TO laforge_mcp;
GRANT SELECT ON builder_image_build_log TO laforge_mcp;

-- Instance settings (admin-only tools): who's an admin, GitHub App
-- installations, and which registries have credentials (host names only).
GRANT SELECT ON instance_admin TO laforge_mcp;
GRANT SELECT ON github_installation TO laforge_mcp;
GRANT SELECT ON installation_repository TO laforge_mcp;
GRANT SELECT (registry_host, created_at, updated_at) ON registry_credential TO laforge_mcp;

-- +goose Down
REVOKE ALL ON api_key, account, mcp_audit, repository, configured_build, build, content_revision,
    environment, network, host, container, placement, team, deployed_object, event, external_access,
    validation_run, validator_result, agent_session, agent_heartbeat, agent_task, scheduled_task, task,
    shell_session, builder_config, builder_metric_sample, builder_image_build, builder_image_build_log,
    instance_admin, github_installation, installation_repository, registry_credential FROM laforge_mcp;
DROP TABLE mcp_audit;
-- The role is left in place: another database on this cluster may still use it.
