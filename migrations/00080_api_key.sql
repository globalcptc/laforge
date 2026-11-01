-- +goose Up
-- Personal API keys: what a person's AI agent uses to read their builds through
-- LaForge's MCP server (docs/plans/mcp-readonly.md). Only the key's sha256 is
-- stored -- the plaintext is shown once, at creation -- and every key expires.
-- A key sees at most what its owner can read; repository_ids, when set, narrows
-- that further to those repositories. Revoked and expired keys are kept (never
-- deleted) so later audit rows still name the key they came from.
CREATE TABLE api_key (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id uuid NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    name text NOT NULL,
    -- The key's first characters ('lfk_ab12cd34'), shown so a person can tell
    -- their keys apart without the secret.
    prefix text NOT NULL,
    token_hash text NOT NULL UNIQUE,
    scope text NOT NULL DEFAULT 'read' CHECK (scope IN ('read')),
    repository_ids uuid[],
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    last_used_at timestamptz,
    revoked_at timestamptz
);
CREATE INDEX api_key_account_idx ON api_key (account_id);

-- +goose Down
DROP TABLE api_key;
