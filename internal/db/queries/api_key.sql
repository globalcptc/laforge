-- name: CreateAPIKey :one
-- Personal API keys (migrations/00080). Every query here names its columns
-- rather than `*`, so token_hash never comes back in a row a handler returns.
INSERT INTO api_key (account_id, name, prefix, token_hash, repository_ids, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, account_id, name, prefix, scope, repository_ids, created_at, expires_at, last_used_at, revoked_at;

-- name: ListAPIKeysByAccount :many
-- A person's own keys, newest first, active and inactive alike (the page hides
-- inactive ones behind a toggle).
SELECT id, account_id, name, prefix, scope, repository_ids, created_at, expires_at, last_used_at, revoked_at
FROM api_key
WHERE account_id = $1
ORDER BY created_at DESC;

-- name: CountActiveAPIKeysByAccount :one
SELECT count(*) FROM api_key
WHERE account_id = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: ListAPIKeysWithOwner :many
-- Every key on the instance with its owner, for the admin page.
SELECT k.id, k.account_id, k.name, k.prefix, k.scope, k.repository_ids, k.created_at, k.expires_at,
       k.last_used_at, k.revoked_at, a.github_login AS owner_login, a.avatar_url AS owner_avatar_url
FROM api_key k
JOIN account a ON a.id = k.account_id
ORDER BY k.created_at DESC;

-- name: RevokeOwnAPIKey :one
-- Revoke one of the caller's own keys. No row means it isn't theirs, doesn't
-- exist, or is already revoked.
UPDATE api_key SET revoked_at = now()
WHERE id = $1 AND account_id = $2 AND revoked_at IS NULL
RETURNING id, account_id, name, prefix, scope, repository_ids, created_at, expires_at, last_used_at, revoked_at;

-- name: RevokeAnyAPIKey :one
-- An instance admin revoking anyone's key.
UPDATE api_key SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL
RETURNING id, account_id, name, prefix, scope, repository_ids, created_at, expires_at, last_used_at, revoked_at;

-- name: DeleteMcpAuditOlderThan :execrows
-- MCP call audit rows are kept for 30 days (the orchestrator's cleanup pass).
DELETE FROM mcp_audit WHERE created_at < $1;

-- name: ListMcpActivityByAccount :many
-- A person's own recent MCP calls, newest first, for the API Keys page.
SELECT m.id, m.tool, m.args, m.build_id, m.status, m.error, m.duration_ms, m.result_bytes,
       m.remote_addr, m.user_agent, m.created_at, k.id AS key_id, k.name AS key_name, k.prefix AS key_prefix
FROM mcp_audit m
JOIN api_key k ON k.id = m.api_key_id
WHERE m.account_id = $1
ORDER BY m.created_at DESC
LIMIT $2;

-- name: ListMcpActivity :many
-- Everyone's recent MCP calls, newest first, for the admin API Keys page.
SELECT m.id, m.tool, m.args, m.build_id, m.status, m.error, m.duration_ms, m.result_bytes,
       m.remote_addr, m.user_agent, m.created_at, k.id AS key_id, k.name AS key_name, k.prefix AS key_prefix,
       a.github_login AS owner_login
FROM mcp_audit m
JOIN api_key k ON k.id = m.api_key_id
JOIN account a ON a.id = m.account_id
ORDER BY m.created_at DESC
LIMIT $1;
