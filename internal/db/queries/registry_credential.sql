-- Private Docker registry credentials (migration 00025).

-- name: GetRegistryCredentialByHost :one
SELECT * FROM registry_credential WHERE registry_host = $1;

-- name: ListRegistryCredentials :many
-- For the admin UI. Does NOT project the secret out -- callers that only
-- list (never authenticate) shouldn't handle it; the secret column is read
-- only by the deploy path via GetRegistryCredentialByHost.
SELECT id, registry_host, username, created_at, updated_at
FROM registry_credential ORDER BY registry_host;

-- name: UpsertRegistryCredential :one
INSERT INTO registry_credential (registry_host, username, secret)
VALUES ($1, $2, $3)
ON CONFLICT (registry_host) DO UPDATE SET
    username = EXCLUDED.username, secret = EXCLUDED.secret, updated_at = now()
RETURNING id, registry_host, username, created_at, updated_at;

-- name: DeleteRegistryCredential :exec
DELETE FROM registry_credential WHERE registry_host = $1;
