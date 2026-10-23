-- name: EnsureBuildRenderSecret :one
-- The build's password-generator secret, creating it from $2 (fresh random
-- bytes the caller supplies) on first use. ON CONFLICT ... DO UPDATE with a
-- no-op assignment makes RETURNING hand back the EXISTING secret when one is
-- already stored, so concurrent first callers all converge on one value.
INSERT INTO build_render_secret (build_id, secret)
VALUES ($1, $2)
ON CONFLICT (build_id) DO UPDATE SET build_id = EXCLUDED.build_id
RETURNING secret;
