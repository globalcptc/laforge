-- name: UpsertAccount :one
-- Re-fetched from GitHub on every login (see migrations/00005's own doc
-- comment) -- login/avatar can drift (renamed, changed picture), github_id
-- never does, so it's the real identity key.
INSERT INTO account (github_id, github_login, avatar_url)
VALUES ($1, $2, $3)
ON CONFLICT (github_id) DO UPDATE SET
    github_login = EXCLUDED.github_login,
    avatar_url = EXCLUDED.avatar_url,
    updated_at = now()
RETURNING *;

-- name: GetAccount :one
SELECT * FROM account WHERE id = $1;

-- name: SetAccountTimezone :exec
-- The per-user timezone preference the UI renders every timestamp in.
-- Validated as a real IANA name by the caller (time.LoadLocation) before
-- it gets here; "" is allowed and means "follow the viewer's browser."
UPDATE account SET timezone = $2, updated_at = now() WHERE id = $1;

-- name: GetAccountByLogin :one
-- Revoking access shouldn't depend on a live GitHub API call succeeding
-- the way granting does (handleSetRepositoryAccess's own GetUserByLogin
-- -- it needs to look the identity up to create the account row in the
-- first place) -- a grantee's account row already exists by the time
-- there's anything to revoke, so this is a pure DB lookup.
SELECT * FROM account WHERE github_login = $1;

-- name: CreateSession :one
INSERT INTO session (account_id, token_hash, github_token, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSessionByTokenHash :one
-- The join is the whole point: one round trip from "cookie value" to
-- "who is this and what's their live GitHub token," which every
-- authenticated request needs.
SELECT session.*, account.github_id, account.github_login, account.avatar_url, account.timezone
FROM session
JOIN account ON account.id = session.account_id
WHERE session.token_hash = $1 AND session.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM session WHERE token_hash = $1;

-- name: DeleteExpiredSessions :execrows
-- Unbounded growth otherwise: a session row is never deleted anywhere
-- else once it expires (GetSessionByTokenHash's own WHERE just stops
-- matching it). Same "time-bounded retention policy" shape as
-- DeleteAgentHeartbeatsOlderThan (migrations/00007's own doc comment),
-- just with no configurable window -- expires_at is already the
-- retention boundary, set once at CreateSession time.
DELETE FROM session WHERE expires_at <= now();

-- name: UpsertRepositoryAccess :one
INSERT INTO repository_access (repository_id, account_id, level, granted_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (repository_id, account_id) DO UPDATE SET level = EXCLUDED.level
RETURNING *;

-- name: GetRepositoryAccess :one
SELECT * FROM repository_access WHERE repository_id = $1 AND account_id = $2;

-- name: ListRepositoryAccessByRepository :many
SELECT repository_access.*, account.github_login, account.avatar_url
FROM repository_access
JOIN account ON account.id = repository_access.account_id
WHERE repository_id = $1
ORDER BY account.github_login;

-- name: ListRepositoryAccessByAccount :many
-- Every repository an account has an explicit grant on, including 'none'
-- (an admin taking away what GitHub would give) -- callers that mean
-- "has access somewhere" skip those.
SELECT repository_access.*, repository.github_owner, repository.github_repo
FROM repository_access
JOIN repository ON repository.id = repository_access.repository_id
WHERE account_id = $1;

-- name: DeleteRepositoryAccess :exec
DELETE FROM repository_access WHERE repository_id = $1 AND account_id = $2;
