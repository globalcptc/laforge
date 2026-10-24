-- Interactive root/admin shell sessions opened on a host/container through the
-- agent, from the UI or CLI. This table is the audit trail (who opened a shell
-- on which object, when, from where), the source of the small global
-- concurrency cap (only one or two at a time), and a future "active sessions"
-- view. The byte relay itself is in-memory in the gateway; only the api writes
-- this table, so the restricted gateway role needs no access to it.

-- +goose Up
CREATE TABLE shell_session (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    deployed_object_id uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    opened_by_account_id uuid REFERENCES account(id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'pending',
    client_addr text,
    started_at timestamptz NOT NULL DEFAULT now(),
    ended_at timestamptz,
    CONSTRAINT shell_session_status_check CHECK (status = ANY (ARRAY['pending'::text, 'active'::text, 'closed'::text]))
);

-- Partial index: the only hot query is "how many are live right now" (the cap)
-- and "list the live ones" -- closed rows are history, not queried in the hot path.
CREATE INDEX shell_session_live_idx ON shell_session (status) WHERE status = ANY (ARRAY['pending'::text, 'active'::text]);

-- +goose Down
DROP TABLE shell_session;
