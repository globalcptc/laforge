-- Real execution for `schedule:` entries (internal/schedule's own
-- natural-language grammar)
-- AND ad-hoc scheduled tasks from the UI -- one shared table, one
-- dispatch loop, both landing on the exact same real machinery
-- internal/api/tasks.go's handleCreateAdHocTask already uses for
-- immediate ad-hoc dispatch (NextStepIndexForHost + CreateAgentTask):
-- a live agent polling get-task actually picks these up and runs them.
--
-- source distinguishes the two origins, since they resolve their target
-- differently: a content-sourced row already knows its one
-- deployed_object_id (materialized when that object is deployed); an
-- adhoc-sourced row carries a target selector (the same adHocTarget
-- shape tasks.go already has) re-resolved against deployed_object at
-- fire time, since the matching set of hosts can change between when
-- the schedule was created and when it fires.

-- +goose Up

CREATE TABLE scheduled_task (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id           uuid NOT NULL REFERENCES build(id) ON DELETE CASCADE,
    source             text NOT NULL CHECK (source IN ('content', 'adhoc')),
    deployed_object_id uuid REFERENCES deployed_object(id) ON DELETE CASCADE, -- content only
    schedule_index     integer,      -- content only: which Schedule[] entry
    target             jsonb,        -- adhoc only: {ids, team, kind, search}, same shape as adHocTarget
    when_expr          text NOT NULL, -- raw string (internal/schedule.Parse input), kept for display
    command            text NOT NULL,
    payload            jsonb NOT NULL DEFAULT '{}',
    anchor             text NOT NULL DEFAULT '', -- '' | competition_start | competition_end | access_open | access_close
    fires_once         boolean NOT NULL,         -- from schedule.Expr.FiresOnce()
    next_fire_at       timestamptz,              -- null once a fires_once row has fired
    status             text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'fired', 'canceled')),
    created_by         text,                     -- github login, adhoc only
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (source = 'content' AND deployed_object_id IS NOT NULL AND schedule_index IS NOT NULL AND target IS NULL)
        OR
        (source = 'adhoc' AND deployed_object_id IS NULL AND schedule_index IS NULL AND target IS NOT NULL)
    )
);

-- The dispatch loop's own query: every pending row whose next_fire_at
-- has passed. Partial index keyed on exactly that predicate, same
-- pattern as task_leasable_idx / agent_task_leasable_idx.
CREATE INDEX scheduled_task_due_idx ON scheduled_task (next_fire_at)
    WHERE status = 'pending';

CREATE INDEX scheduled_task_build_idx ON scheduled_task (build_id, status);

-- A content-sourced schedule entry is materialized once per deployed
-- object per schedule index -- re-deploying the same commit must not
-- duplicate it.
CREATE UNIQUE INDEX scheduled_task_content_once ON scheduled_task (deployed_object_id, schedule_index)
    WHERE source = 'content';

-- +goose Down

DROP TABLE scheduled_task;
