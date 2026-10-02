-- Interactive shell sessions -- the audit trail for who opened a root/admin
-- prompt on which host and when. Only the api (full laforge role) touches this
-- table; the gateway relays bytes purely in memory.

-- name: CreateShellSession :one
INSERT INTO shell_session (deployed_object_id, opened_by_account_id, status, client_addr)
VALUES ($1, $2, 'pending', $3)
RETURNING *;

-- name: MarkShellSessionActive :exec
-- The relay to the gateway was established (client half attached). Only moves a
-- still-pending row forward.
UPDATE shell_session SET status = 'active'
WHERE id = $1 AND status = 'pending';

-- name: MarkShellSessionClosed :exec
-- The session ended (either side closed). Stamps ended_at; idempotent for a row
-- already closed.
UPDATE shell_session SET status = 'closed', ended_at = now()
WHERE id = $1 AND status <> 'closed';
