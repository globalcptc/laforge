-- +goose Up
-- OAuth sign-in for the MCP server (docs/plans/mcp-readonly.md §14): ChatGPT and
-- Claude.ai can't send a pasted API key, so they register here (RFC 7591), send
-- the person through GitHub sign-in and a consent page, and get back an ordinary
-- api_key. A key with oauth_client_id set is a "connected app" on the API Keys
-- page; everything else about it -- what it can see, revoking it -- is the same
-- as a pasted key. Pending sign-ins and their one-time codes live in the API's
-- memory, not here.
CREATE TABLE oauth_client (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The id the client sends; random, handed out at registration.
    client_id text NOT NULL UNIQUE,
    client_name text NOT NULL,
    redirect_uris text[] NOT NULL,
    -- sha256 of the secret, only for clients that asked for one
    -- (client_secret_post / client_secret_basic).
    client_secret_hash text,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE api_key ADD COLUMN oauth_client_id uuid REFERENCES oauth_client(id) ON DELETE SET NULL;
CREATE INDEX api_key_oauth_client_idx ON api_key (oauth_client_id) WHERE oauth_client_id IS NOT NULL;

-- +goose Down
DROP INDEX api_key_oauth_client_idx;
ALTER TABLE api_key DROP COLUMN oauth_client_id;
DROP TABLE oauth_client;
