-- name: CreateAPIKey :one
-- Personal API keys (migrations/00080). Every query here names its columns
-- rather than `*`, so token_hash never comes back in a row a handler returns.
INSERT INTO api_key (account_id, name, prefix, token_hash, repository_ids, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, account_id, name, prefix, scope, repository_ids, created_at, expires_at, last_used_at, revoked_at;

-- name: ListAPIKeysByAccount :many
-- A person's own keys, newest first, active and inactive alike (the page hides
-- inactive ones behind a toggle). Keys with an OAuth client are connected apps
-- (migrations/00083); the client's redirect URIs name where it lives.
SELECT k.id, k.account_id, k.name, k.prefix, k.scope, k.repository_ids, k.created_at, k.expires_at,
       k.last_used_at, k.revoked_at, k.oauth_client_id, c.redirect_uris AS oauth_redirect_uris
FROM api_key k
LEFT JOIN oauth_client c ON c.id = k.oauth_client_id
WHERE k.account_id = $1
ORDER BY k.created_at DESC;

-- name: CountActiveAPIKeysByAccount :one
-- Pasted keys only; connected apps have their own cap.
SELECT count(*) FROM api_key
WHERE account_id = $1 AND oauth_client_id IS NULL AND revoked_at IS NULL AND expires_at > now();

-- name: CountActiveOAuthKeysByAccount :one
SELECT count(*) FROM api_key
WHERE account_id = $1 AND oauth_client_id IS NOT NULL AND revoked_at IS NULL AND expires_at > now();

-- name: CreateOAuthAPIKey :one
-- The key an OAuth sign-in hands to a connected app: an ordinary read key,
-- tied to the client that asked for it.
INSERT INTO api_key (account_id, name, prefix, token_hash, expires_at, oauth_client_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: RevokeOAuthKeysForClient :execrows
-- Connecting the same app again replaces its earlier key.
UPDATE api_key SET revoked_at = now()
WHERE account_id = $1 AND oauth_client_id = $2 AND revoked_at IS NULL;

-- name: ListAPIKeysWithOwner :many
-- Every key on the instance with its owner, for the admin page.
SELECT k.id, k.account_id, k.name, k.prefix, k.scope, k.repository_ids, k.created_at, k.expires_at,
       k.last_used_at, k.revoked_at, k.oauth_client_id, c.redirect_uris AS oauth_redirect_uris,
       a.github_login AS owner_login, a.avatar_url AS owner_avatar_url
FROM api_key k
JOIN account a ON a.id = k.account_id
LEFT JOIN oauth_client c ON c.id = k.oauth_client_id
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
-- A person's own MCP calls, newest first, a page at a time: pass the last
-- row's created_at and id as before_at/before_id for the next page.
SELECT m.id, m.tool, m.args, m.build_id, m.status, m.error, m.duration_ms, m.result_bytes,
       m.remote_addr, m.user_agent, m.created_at, k.id AS key_id, k.name AS key_name, k.prefix AS key_prefix
FROM mcp_audit m
JOIN api_key k ON k.id = m.api_key_id
WHERE m.account_id = sqlc.arg(account_id)
  AND (sqlc.narg(before_at)::timestamptz IS NULL OR (m.created_at, m.id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_size);

-- name: CountMcpActivityByAccount :one
SELECT count(*) FROM mcp_audit WHERE account_id = $1;

-- name: ListMcpActivity :many
-- Everyone's MCP calls, newest first, a page at a time, for admins.
SELECT m.id, m.tool, m.args, m.build_id, m.status, m.error, m.duration_ms, m.result_bytes,
       m.remote_addr, m.user_agent, m.created_at, k.id AS key_id, k.name AS key_name, k.prefix AS key_prefix,
       a.github_login AS owner_login
FROM mcp_audit m
JOIN api_key k ON k.id = m.api_key_id
JOIN account a ON a.id = m.account_id
WHERE (sqlc.narg(before_at)::timestamptz IS NULL OR (m.created_at, m.id) < (sqlc.narg(before_at)::timestamptz, sqlc.narg(before_id)::uuid))
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg(page_size);

-- name: CountMcpActivity :one
SELECT count(*) FROM mcp_audit;

-- name: GetMcpActivity :one
-- One MCP call with its full detail, for an opened activity row.
SELECT m.id, m.account_id, m.tool, m.args, m.build_id, m.status, m.error, m.duration_ms, m.result_bytes,
       m.remote_addr, m.user_agent, m.created_at, m.detail, k.id AS key_id, k.name AS key_name, k.prefix AS key_prefix,
       a.github_login AS owner_login
FROM mcp_audit m
JOIN api_key k ON k.id = m.api_key_id
JOIN account a ON a.id = m.account_id
WHERE m.id = $1;
