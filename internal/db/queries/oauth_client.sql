-- name: CreateOAuthClient :one
-- An MCP client registering itself (RFC 7591, docs/plans/mcp-readonly.md §14).
INSERT INTO oauth_client (client_id, client_name, redirect_uris, client_secret_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, client_id, client_name, redirect_uris, client_secret_hash, created_at;

-- name: GetOAuthClientByClientID :one
SELECT id, client_id, client_name, redirect_uris, client_secret_hash, created_at
FROM oauth_client
WHERE client_id = $1;

-- name: CountUnusedOAuthClients :one
-- Registered clients that never got a key: the cap on open registration.
SELECT count(*) FROM oauth_client c
WHERE NOT EXISTS (SELECT 1 FROM api_key k WHERE k.oauth_client_id = c.id);

-- name: DeleteUnusedOAuthClients :execrows
-- Clients that registered but never got a key are dropped after a day (the
-- orchestrator's cleanup pass).
DELETE FROM oauth_client c
WHERE c.created_at < $1
  AND NOT EXISTS (SELECT 1 FROM api_key k WHERE k.oauth_client_id = c.id);
