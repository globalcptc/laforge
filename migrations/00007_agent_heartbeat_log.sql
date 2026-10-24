-- agent_session (migration 00004) is deliberately "one row per
-- deployed_object" -- upserted on every heartbeat, so it only ever
-- answers "when did this agent last check in," never "what did its
-- check-ins look like over the last hour." That's a real gap for live
-- troubleshooting during an event ("did team 7's domain controller go
-- quiet at 14:32, or has it never been healthy") and for anything that
-- wants to reason about an agent's behavior over time (rules checking --
-- e.g. a host whose remote address changes mid-competition is a real,
-- actionable signal in a security competition, not just an oddity).
--
-- agent_heartbeat is the append-only counterpart: one row per real
-- heartbeat the gateway ever receives, never updated, never upserted.
-- agent_session stays exactly as it is -- the fast "current status"
-- lookup every health check already uses (internal/api/health.go),
-- which would be wasteful to compute by scanning this table's full
-- history on every request. Two tables, two jobs, same split the rest
-- of this schema already uses (deployed_object.status is "current",
-- event is "how we got here").

-- +goose Up

CREATE TABLE agent_heartbeat (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    deployed_object_id uuid NOT NULL REFERENCES deployed_object(id) ON DELETE CASCADE,
    cert_fingerprint   text NOT NULL,
    -- The real TCP remote address the gateway's own net.Conn saw for
    -- this connection -- not re-trusted as identity (the mTLS
    -- certificate already is that), but real, useful signal: an agent
    -- whose remote address changes mid-competition, or one heartbeating
    -- from two different addresses close together, is exactly the kind
    -- of thing "rules checking" and live troubleshooting both want to
    -- ask about later. Nullable only because a caller (a test, a
    -- backfill) might not always have one.
    remote_addr        text,
    -- What HeartbeatResponsePayload actually handed back -- lets a later
    -- query ask "was this agent's NEXT heartbeat actually within its own
    -- assigned window" instead of guessing at a fixed threshold.
    next_poll_ms       integer,
    created_at         timestamptz NOT NULL DEFAULT now()
);

-- The one access pattern this table exists to serve: one object's own
-- heartbeat history, newest or oldest first. A build-wide time-window
-- query (the dashboard's "check-ins over time" chart) joins through
-- deployed_object -> team -> build, same as ListAgentSessionsByBuild
-- already does against agent_session -- no separate build_id column
-- here either, for the same reason: this table has exactly one source
-- of truth for "which build," and it's deployed_object, not a
-- denormalized copy that could drift.
CREATE INDEX agent_heartbeat_object_time_idx ON agent_heartbeat (deployed_object_id, created_at);

-- Real, deliberately unaddressed here: retention. This table grows
-- without bound (every real heartbeat, every ~30s, for however many
-- agents, for the life of a build) -- the "Retention and cleanup"
-- guidance already anticipates exactly this kind of growth
-- for other artifact classes ("purge artifacts... storage shown per
-- build"), and a real deployment needs the same discipline applied here
-- (a time-bounded retention policy, or folding into "purge artifacts").
-- Not built this migration.

GRANT INSERT ON agent_heartbeat TO laforge_gateway;
-- Deliberately no SELECT: the gateway only ever writes its own
-- heartbeat log, exactly like agent_session's own grant reasoning
-- (migration 00004) -- reading history back is the API's job, over a
-- connection that was never scoped down to laforge_gateway's
-- restricted surface in the first place.

-- +goose Down

REVOKE ALL ON agent_heartbeat FROM laforge_gateway;
DROP TABLE agent_heartbeat;
