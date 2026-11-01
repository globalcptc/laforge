-- +goose Up
-- The full record of one MCP call, shown when a row on the API Keys
-- activity log is opened: the complete arguments, the client, timing per
-- phase, every database query and internal request the call made, and a
-- preview of the result. laforge_mcp's table-wide INSERT covers the column.
ALTER TABLE mcp_audit ADD COLUMN detail jsonb;

-- Paging through the history, newest first.
CREATE INDEX mcp_audit_time_id_idx ON mcp_audit (created_at DESC, id DESC);
DROP INDEX mcp_audit_time_idx;

-- +goose Down
CREATE INDEX mcp_audit_time_idx ON mcp_audit (created_at);
DROP INDEX mcp_audit_time_id_idx;
ALTER TABLE mcp_audit DROP COLUMN detail;
