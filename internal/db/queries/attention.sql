-- Per-person "close this item" state for the home page's needs-attention list.
-- A dismissal is keyed by (account, build, category) so one person closing an
-- item never hides it from anyone else, and it survives until the build itself
-- goes away (ON DELETE CASCADE).

-- name: DismissAttention :exec
-- Close one attention item for this person. Idempotent: closing an already-
-- closed item is a no-op.
INSERT INTO attention_dismissal (account_id, build_id, category)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, build_id, category) DO NOTHING;

-- name: UndismissAttention :exec
-- Reopen one previously-closed attention item for this person.
DELETE FROM attention_dismissal
WHERE account_id = $1 AND build_id = $2 AND category = $3;

-- name: ListAttentionDismissalsByAccount :many
-- Every item this person has closed, so Home can filter them out of the
-- aggregate it computes.
SELECT build_id, category FROM attention_dismissal WHERE account_id = $1;
