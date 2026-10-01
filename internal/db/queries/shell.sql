-- Interactive shell sessions. Only the api (full laforge role) touches this
-- table; the gateway relays bytes purely in memory. These back the audit trail
-- and the small global concurrency cap.

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

-- name: CountLiveShellSessions :one
-- How many sessions are pending or active right now, across the whole instance
-- -- the global cap is enforced against this.
SELECT COUNT(*) FROM shell_session WHERE status = ANY (ARRAY['pending'::text, 'active'::text]);

-- name: ListLiveShellSessions :many
-- The live sessions with the object they target, newest first -- for an
-- operator "who's in a shell right now" view.
SELECT shell_session.*, deployed_object.object_name, deployed_object.as_name, deployed_object.kind
FROM shell_session
JOIN deployed_object ON deployed_object.id = shell_session.deployed_object_id
WHERE shell_session.status = ANY (ARRAY['pending'::text, 'active'::text])
ORDER BY shell_session.started_at DESC;
