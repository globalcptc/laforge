-- name: CreateBuild :one
INSERT INTO build (configured_build_id, content_revision_id, environment_name, auto_built, created_by_account_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBuild :one
SELECT * FROM build WHERE id = $1;

-- name: SetBuildStatus :one
UPDATE build SET status = $2 WHERE id = $1 RETURNING *;

-- name: ForceDeleteBuild :exec
-- The admin "force remove from database" escape hatch for a build whose real
-- infrastructure is already gone (a volunteer deleted the hosts out from under
-- LaForge, so teardown can never complete and it's wedged in 'tearing_down').
-- Deletes the build row; every runtime row cascades (team -> deployed_object ->
-- agent_task/session/heartbeat/artifact/shell_session/external_access, plus
-- task, event, scheduled_task, attention_dismissal). The configured_build and
-- content_revision are left intact (build.configured_build_id is ON DELETE SET
-- NULL, and the revision is a parent, not a child). This touches NO hoster --
-- it is purely a database removal, which is why it is admin-only and buried.
DELETE FROM build WHERE id = $1;

-- name: SetBuildReconcileError :exec
-- The last reconcile outcome for a build: the error text when reconcile failed
-- (e.g. build/builder incompatibility), or NULL to clear it after a success.
UPDATE build SET reconcile_error = $2 WHERE id = $1;

-- name: ApplyContentRevisionToBuild :one
-- handleApplyUpcoming's own mechanism: reuses an existing build's own
-- row for a new commit instead of creating a second one -- team/
-- deployed_object rows stay put (same build_id they've always had), so
-- orchestrator.Reconcile's existing fingerprint diff naturally destroys
-- only what changed, deploys only what's new, and leaves everything else
-- alone the next time it's called against this same buildID.
UPDATE build SET content_revision_id = $2, environment_name = $3 WHERE id = $1 RETURNING *;

-- name: ListBuildsByStatus :many
SELECT * FROM build WHERE status = ANY(sqlc.arg(statuses)::text[]);

-- name: GetLiveDeployingBuildForConfiguredBuild :one
-- "If it is already deployed, the commit is applied to it"
-- -- reconcile's real auto-deploy check: is there a build for this
-- configured build that's actually live (status = 'deploying', the only
-- build status that reaches a real hoster today -- 'deployed' is reserved
-- for once something sets it, per the build state diagram). Picks the
-- most recently created one if more than one somehow exists.
-- pgx.ErrNoRows means "nothing deployed yet" -- auto-build's own fresh
-- `planned` build is enough; there's nothing to auto-apply onto.
SELECT * FROM build
WHERE configured_build_id = $1 AND status IN ('deploying', 'building', 'finished')
ORDER BY created_at DESC
LIMIT 1;

-- name: ListBuildsByRepository :many
-- "Every build of this repository" (Build → Builds) --
-- build has no direct repository_id (a build is always FROM a specific
-- commit, and a commit belongs to a repository through content_revision),
-- so this is the one join that answers "which builds are this repo's."
SELECT build.*,
       content_revision.commit_sha,
       content_revision.commit_message,
       content_revision.committed_at,
       content_revision.ref AS branch,
       (SELECT count(*) FROM team WHERE team.build_id = build.id) AS team_count
FROM build
JOIN content_revision ON content_revision.id = build.content_revision_id
WHERE content_revision.repository_id = $1
ORDER BY build.created_at DESC;

-- name: GetBuildRepository :one
-- The reverse of ListBuildsByRepository: given a build, which repository
-- (and its owner/name) does it belong to -- needed everywhere the API
-- authorizes "does this account have access to the repo THIS build is
-- part of" without the caller already knowing the repository id.
SELECT repository.* FROM repository
JOIN content_revision ON content_revision.repository_id = repository.id
JOIN build ON build.content_revision_id = content_revision.id
WHERE build.id = $1;

-- name: EnsureTeam :one
-- A no-op update on conflict (rather than DO NOTHING) purely so RETURNING
-- still gives back the existing row -- Reconcile calls this once per team
-- number every pass and needs the team's id either way.
INSERT INTO team (build_id, team_number)
VALUES ($1, $2)
ON CONFLICT (build_id, team_number) DO UPDATE SET team_number = EXCLUDED.team_number
RETURNING *;

-- name: ListTeamsByBuild :many
SELECT * FROM team WHERE build_id = $1 ORDER BY team_number;

-- name: GetTeam :one
SELECT * FROM team WHERE id = $1;

-- name: GetTeamByNumber :one
SELECT * FROM team WHERE build_id = $1 AND team_number = $2;

-- name: SetTeamAccessState :one
-- Written once the actual OpenAccess/CloseAccess builder call completes
-- (see internal/runner's executeAccess), never optimistically before --
-- this column is "what CloseAccess/OpenAccess actually achieved," not a
-- request. Deliberately touches ONLY access_state, never the override: the
-- override records the operator's/schedule's intent and is what the access
-- reconciler reads to decide the desired state, so clearing it here would let
-- the reconciler immediately revert what this task just achieved. A lapsed
-- override is dropped by the reconciler (ClearTeamAccessOverride), not here.
UPDATE team SET access_state = $2 WHERE id = $1 RETURNING *;

-- name: SetTeamAccessOverride :one
-- Records a manual access override: a direction ('open'/'closed') plus the
-- time it holds until, after which the access reconciler
-- (internal/orchestrator/access.go) resumes the environment's own schedule. A
-- null until means the override holds until changed (there is no schedule
-- boundary to hand back to). Operational state, deliberately separate from the
-- authored schedule, and never touched by a redeploy. Extend pushes the until
-- of an 'open' override further out.
UPDATE team SET access_override_state = $2, access_override_until = $3 WHERE id = $1 RETURNING *;

-- name: ClearTeamAccessOverride :exec
-- The reconciler drops a lapsed override (its until has passed) so the team
-- falls back to being driven purely by the schedule again.
UPDATE team SET access_override_state = '', access_override_until = NULL WHERE id = $1;

-- name: HasOpenTeamAccessTask :one
-- Dedup for the access reconciler: team-level access tasks carry no
-- deployed_object_id, so the partial unique index that dedups deploy/destroy
-- tasks (CreateTaskIfNoneOpen) does not apply to them. Before enqueuing an
-- open/close for a team, the reconciler checks there isn't already one pending
-- or leased for that same team (matched by the team number in the payload), so
-- a slow builder call doesn't get a fresh duplicate every pass.
SELECT EXISTS (
    SELECT 1 FROM task
    WHERE build_id = @build_id
      AND kind IN ('open_access', 'close_access')
      AND status IN ('pending', 'leased')
      AND payload->>'team' = @team::text
) AS present;

-- name: HasCompletedTeamAccessTask :one
-- Fix 1 (reconciler drives desired-vs-applied, not the stored label): a team's
-- access_state defaults to 'closed' before any builder call has run, but a
-- freshly deployed team's instances are actually OPEN (eth0 attached, public NAT
-- configured) -- the label is an assumption, not an applied fact. This reports
-- whether an open/close task has ever actually COMPLETED for the team, so the
-- reconciler can tell "converged to the label" apart from "never applied; reality
-- is open-by-deploy" and issue the real close once the team is up.
SELECT EXISTS (
    SELECT 1 FROM task
    WHERE build_id = @build_id
      AND kind IN ('open_access', 'close_access')
      AND status = 'done'
      AND payload->>'team' = @team::text
) AS present;

-- name: SupersedeTeamAccessTasks :execrows
-- Fix 3 (serialize manual access changes): a fresh manual open/close for a team
-- cancels any still-PENDING access task for that same team (either direction), so
-- a flurry of operator clicks collapses to just the newest intent instead of
-- queuing a tug-of-war the runner then executes one stale step at a time. A task
-- already 'leased' is mid-execution against the builder and is left to finish;
-- the newest task runs after it and wins, so the team lands in the last-requested
-- state regardless of ordering.
UPDATE task SET status = 'failed', last_error = 'superseded by a newer team access change', updated_at = now()
WHERE build_id = @build_id
  AND kind IN ('open_access', 'close_access')
  AND status = 'pending'
  AND payload->>'team' = @team::text;

-- name: HasConfigureNetworkAccessTask :one
-- Dedup for the network-access reconciler: one live configure_network_access task
-- per (team, desired-config fingerprint). The fingerprint hashes the team's
-- networks' names/CIDRs/visible_from PLUS its ready guests' instance references,
-- so a content change OR an instance replacement (a rebuilt guest gets a new ref)
-- yields a new fingerprint and a fresh task, while an unchanged config never
-- re-runs (the builder call is idempotent). A previously FAILED task does NOT
-- count, so an exhausted attempt is re-queued and converges rather than leaving
-- the policy permanently half-applied.
SELECT EXISTS (
    SELECT 1 FROM task
    WHERE build_id = @build_id
      AND kind = 'configure_network_access'
      AND payload->>'team' = @team::text
      AND payload->>'fingerprint' = @fingerprint::text
      AND status <> 'failed'
) AS present;

-- name: HasConfigureExternalAccessTask :one
-- Dedup for the external-access reconciler, exactly like network access above:
-- one configure_external_access task per team and desired public ports/ready
-- instance identities, across all statuses. A new or replaced public instance
-- yields a fresh task even if content has not changed.
SELECT EXISTS (
    SELECT 1 FROM task
    WHERE build_id = @build_id
      AND kind = 'configure_external_access'
      AND payload->>'team' = @team::text
      AND payload->>'fingerprint' = @fingerprint::text
) AS present;

-- name: HasOpenConfigureAccessTask :one
-- Fix 1 gate: is a team's network/external-access configuration still settling?
-- A close must run AFTER external access attaches its public-port ACLs (so the
-- close can capture and strip them); otherwise the close records a NIC with no
-- public ACL and a later external-access pass silently re-opens ingress on a
-- team the label calls closed. True while any configure_network_access or
-- configure_external_access task for the team is still pending or leased.
SELECT EXISTS (
    SELECT 1 FROM task
    WHERE build_id = @build_id
      AND kind IN ('configure_network_access', 'configure_external_access')
      AND status IN ('pending', 'leased')
      AND payload->>'team' = @team::text
) AS present;

-- name: EnsureDeployedObject :one
-- Same no-op-update-on-conflict trick as EnsureTeam: Reconcile calls this
-- for every COPY desired in every team, every pass, and needs the row's id
-- and CURRENT (not freshly-inserted) fingerprint/status back either way so
-- it can decide whether a deploy/destroy task is actually needed. The
-- conflict target matches deployed_object_identity_idx exactly: identity
-- is per copy (as_name) for hosts/containers, per definition (object_name)
-- for networks, which have no as_name.
INSERT INTO deployed_object (team_id, kind, object_name, as_name, network_name, fingerprint)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (team_id, kind, (COALESCE(as_name, object_name))) DO UPDATE SET team_id = EXCLUDED.team_id
RETURNING *;

-- name: SetDeployedObjectTags :exec
-- Carry a host/container's authored tags onto its runtime row so tasks can
-- target by tag. Set every reconcile pass (cheap; tags aren't in the
-- fingerprint), so an edited tag propagates without a redeploy.
UPDATE deployed_object SET tags = $2 WHERE id = $1;

-- name: GetDeployedObject :one
SELECT * FROM deployed_object WHERE id = $1;

-- name: ListDeployedObjectsByBuild :many
SELECT deployed_object.* FROM deployed_object
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = $1;

-- name: ListDeployedObjectsByTeam :many
SELECT * FROM deployed_object WHERE team_id = $1;

-- name: MarkDeployedObjectDeploying :one
UPDATE deployed_object SET status = 'deploying', updated_at = now() WHERE id = $1 RETURNING *;

-- name: MarkDeployedObjectRunning :one
-- Infra is up: the hoster created the instance. NOT "finished" -- the agent
-- has not checked in yet and no authored step has run. The object-lifecycle
-- poll (internal/orchestrator/lifecycle.go) advances this to building ->
-- finished/build_failed/invalid from the real agent_task/validator rows.
-- deployed_at restarts the never-checked-in watchdog's clock, and a redeploy
-- clears any earlier overdue flag.
UPDATE deployed_object SET status = 'running', external_ref = $2, fingerprint = $3, last_error = NULL, updated_at = now(),
    deployed_at = now(), agent_checkin_overdue_at = NULL
WHERE id = $1 RETURNING *;

-- name: SetDeployedObjectAgentPlatform :exec
-- Which agent binary the runner delivered to this host. Marks the object as one
-- whose agent is expected to check in, and picks its watchdog deadline.
UPDATE deployed_object SET agent_platform = $2 WHERE id = $1;

-- name: ListAgentCheckinWatchdogCandidates :many
-- Hosts in this build whose agent has not checked in since they were deployed
-- and which the watchdog hasn't flagged yet. The caller applies the per-OS
-- deadline. "Checked in since deployed" compares the session's last heartbeat
-- with deployed_at, because a rebuilt host keeps its old agent_session row.
SELECT o.id, o.object_name, o.as_name, o.external_ref, o.agent_platform, o.deployed_at, t.team_number
FROM deployed_object o
JOIN team t ON t.id = o.team_id
LEFT JOIN agent_session s ON s.deployed_object_id = o.id
WHERE t.build_id = $1
  AND o.status IN ('running', 'building')
  AND o.agent_platform <> ''
  AND o.deployed_at IS NOT NULL
  AND o.agent_checkin_overdue_at IS NULL
  AND (s.last_heartbeat_at IS NULL OR s.last_heartbeat_at < o.deployed_at)
ORDER BY o.deployed_at;

-- name: MarkAgentCheckinOverdue :execrows
-- Flag one host as never-checked-in. Guarded so concurrent orchestrator
-- replicas flag (and emit the event for) a host only once per deploy.
UPDATE deployed_object SET agent_checkin_overdue_at = now()
WHERE id = $1 AND agent_checkin_overdue_at IS NULL;

-- name: MarkDeployedObjectDeployFailed :one
-- The hoster couldn't create the instance -- an infrastructure failure,
-- distinct from a build-step or validator failure that happens later on an
-- instance that did come up (build_failed / invalid).
UPDATE deployed_object SET status = 'deploy_failed', last_error = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: SetDeployedObjectBuilding :exec
-- The agent has checked in and is running its steps/validators. Only moves
-- an object that is still 'running' (guards against racing a later terminal
-- state back to building on a subsequent poll).
UPDATE deployed_object SET status = 'building', updated_at = now()
WHERE id = $1 AND status = 'running';

-- name: SetDeployedObjectFinished :exec
-- Up, every step done, every validator passed -- the only success state,
-- the only one the UI paints green. Only advances from running/building.
UPDATE deployed_object SET status = 'finished', last_error = NULL, updated_at = now()
WHERE id = $1 AND status IN ('running', 'building');

-- name: SetDeployedObjectBuildFailed :exec
-- A build step could not be completed (a failed agent_task). Only advances
-- from running/building, never from a terminal state.
UPDATE deployed_object SET status = 'build_failed', last_error = $2, updated_at = now()
WHERE id = $1 AND status IN ('running', 'building');

-- name: SetDeployedObjectInvalid :exec
-- Every step completed but a validator failed: the host built but does not
-- match what the environment asserts about it. Only advances from
-- running/building.
UPDATE deployed_object SET status = 'invalid', last_error = $2, updated_at = now()
WHERE id = $1 AND status IN ('running', 'building');

-- name: MarkDeployedObjectStepsMaterialized :exec
-- Record that this object's authored steps have been turned into agent_task
-- rows (the orchestrator does this only once every dependency has finished).
-- Set even when an object has zero authored steps, so the lifecycle advancer
-- can tell "finished with nothing to do" from "still waiting to materialize".
-- Idempotent: only the first materialization stamps the time.
UPDATE deployed_object SET steps_materialized_at = now()
WHERE id = $1 AND steps_materialized_at IS NULL;

-- name: SetDeployedObjectBlockedOn :exec
-- Record which dependencies an object's steps are waiting on (a jsonb array of
-- names), for the UI to show "Dependency Blocked - waiting on X". Pass NULL to
-- clear it once released.
UPDATE deployed_object SET blocked_on = $2 WHERE id = $1;

-- name: SetDeployedObjectExternalRef :exec
-- Record the hoster ref for an object whose deploy did NOT fully succeed,
-- so a partially-created instance is still destroyable by teardown. The
-- builder returns the deterministic instance name even on a config-drive/
-- create failure (see incus.deployInstance); without this, that orphan is
-- untracked and only cleanable by hand. Only fills a ref in, never clears
-- one, and leaves status/last_error to the caller's own mark-failed.
UPDATE deployed_object SET external_ref = $2 WHERE id = $1 AND (external_ref IS NULL OR external_ref = '');

-- name: SetDeployedObjectPowerState :exec
-- The live hoster power state from Builder.Inspect (running/stopped/other/
-- missing), tracked separately from the deploy status and the agent's
-- health. Deliberately does NOT touch updated_at: a background power poll
-- isn't a lifecycle change, and shouldn't make an object look freshly
-- modified. checked_at records when it was last confirmed.
UPDATE deployed_object SET power_state = $2, power_state_checked_at = now() WHERE id = $1;

-- name: MarkDeployedObjectDestroying :one
UPDATE deployed_object SET status = 'destroying', updated_at = now() WHERE id = $1 RETURNING *;

-- name: MarkDeployedObjectDestroyed :one
-- The real terminal destroy outcome, for a build teardown
-- (internal/orchestrator.Teardown) -- distinct from both existing
-- destroy-completion paths: not deleted (DeleteDeployedObject, for a
-- content-driven removal, which drops the row and its history), and not
-- reset back to pending (ResetDeployedObjectForRedeploy, for "rebuild
-- means recreate" on a fingerprint change, which expects a redeploy to
-- follow). A torn-down build's objects stay in Postgres, permanently
-- destroyed, exactly the same "row and its history stay" shape every
-- other terminal state in this schema already has.
-- Clears power_state too: the instance is gone, so the last-polled "running"
-- (or any) power state is no longer true and must not linger as a stale
-- "Infra: Running" next to a "Destroyed" lifecycle badge.
UPDATE deployed_object SET status = 'destroyed', public_address = '', power_state = '', power_state_checked_at = now(), updated_at = now() WHERE id = $1 RETURNING *;

-- name: MarkDeployedObjectDestroyFailed :one
-- The destroy-side twin of MarkDeployedObjectDeployFailed: the hoster could not
-- remove this object and the failure is permanent (builder configuration), so
-- Teardown stops re-queuing destroys for it. Terminal until an operator retries
-- the teardown (ResetDestroyFailedObjectsForBuild) or forces the build down.
UPDATE deployed_object SET status = 'destroy_failed', last_error = $2, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: ResetDestroyFailedObjectsForBuild :many
-- The explicit operator retry after fixing the builder: every destroy_failed
-- object of the build goes back to 'destroying' so a fresh terminal destroy can
-- be requested for it.
UPDATE deployed_object SET status = 'destroying', last_error = NULL, updated_at = now()
WHERE status = 'destroy_failed'
  AND team_id IN (SELECT id FROM team WHERE build_id = $1)
RETURNING *;

-- name: ForceDestroyObjectsForBuild :execrows
-- The operator's "declare it gone" escape hatch for a wedged teardown: every
-- object not yet destroyed is recorded as destroyed WITHOUT any hoster call
-- (same column resets as MarkDeployedObjectDestroyed). The caller is asserting
-- the real infrastructure is already gone or will be cleaned up by hand.
UPDATE deployed_object SET status = 'destroyed', public_address = '', power_state = '', power_state_checked_at = now(), last_error = NULL, updated_at = now()
WHERE status <> 'destroyed'
  AND team_id IN (SELECT id FROM team WHERE build_id = $1);

-- name: ResetDeployedObjectForRedeploy :one
-- Used after a destroy that happened only because the fingerprint changed
-- ("rebuild means recreate"): back to pending, cleared external_ref/
-- fingerprint, so the *next* Reconcile pass sees it as needing a fresh
-- deploy task with the new fingerprint. Also clears steps_materialized_at so
-- the redeploy re-materializes this object's steps once its dependencies are
-- finished again (its agent_task rows are deleted on the same path).
UPDATE deployed_object SET status = 'pending', external_ref = NULL, fingerprint = '', last_error = NULL, steps_materialized_at = NULL, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteDeployedObject :exec
-- Used after a destroy for an object no longer in desired state at all
-- (removed from the environment's topology) -- cascades away any
-- remaining task row for it too.
DELETE FROM deployed_object WHERE id = $1;

-- name: CreateTaskIfNoneOpen :one
-- The idempotent half of "the orchestrator diffs desired vs observed state
-- ... and creates tasks": the partial unique index on
-- (deployed_object_id) WHERE status IN ('pending','leased') means a second
-- call for the same object while one is already open returns zero rows
-- rather than creating a duplicate -- safe to call from every reconcile
-- pass, and from more than one stateless orchestrator replica at once.
INSERT INTO task (build_id, deployed_object_id, kind, payload)
VALUES ($1, $2, $3, $4)
ON CONFLICT (deployed_object_id) WHERE status IN ('pending', 'leased') AND deployed_object_id IS NOT NULL
DO NOTHING
RETURNING *;

-- name: CreateTeamTask :one
-- Team-level tasks (open_access/close_access) have no deployed_object_id,
-- so the partial unique index above never applies to them -- multiple NULL
-- deployed_object_id rows never conflict with each other under a unique
-- index, by ordinary SQL NULL semantics.
INSERT INTO task (build_id, deployed_object_id, kind, payload)
VALUES ($1, NULL, $2, $3)
RETURNING *;

-- name: LeaseTask :one
-- FOR UPDATE SKIP LOCKED claims either a pending task or one
-- whose lease has expired, atomically, so concurrent runners never block
-- each other or double-claim the same row.
--
-- Per-builder concurrency cap: a candidate task is skipped while its
-- builder_config already has max_concurrency tasks leased-and-live (so a
-- capacity-limited hoster like MicroCloud can be turned down below the global
-- RUNNER_CONCURRENCY pool). max_concurrency = 0 means unlimited; tasks with no
-- builder_config (ad-hoc builds, or builds whose configured_build was cleared)
-- are never capped. The cap is a SOFT limit -- concurrent leasers each read the
-- in-flight count before committing, so it can transiently overshoot by the
-- number of simultaneous leasers, which is fine for a "don't overwhelm it" knob.
UPDATE task
SET status = 'leased', lease_owner = $1, lease_expires_at = now() + $2::interval,
    attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT t.id FROM task t
    LEFT JOIN build b ON b.id = t.build_id
    LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
    LEFT JOIN builder_config bc ON bc.name = cb.builder_config_name
    WHERE (t.status = 'pending' OR (t.status = 'leased' AND t.lease_expires_at < now()))
      AND (
        bc.max_concurrency IS NULL OR bc.max_concurrency = 0
        OR (
            SELECT count(*) FROM task it
            JOIN build ib ON ib.id = it.build_id
            JOIN configured_build icb ON icb.id = ib.configured_build_id
            WHERE icb.builder_config_name = bc.name
              AND it.status = 'leased' AND it.lease_expires_at >= now()
        ) < bc.max_concurrency
      )
    -- Critical path first: a deploy task's payload carries its priority (how
    -- many objects depends_on it; networks highest -- see the orchestrator's
    -- deployTaskPayload), so every team's domain controller is leased ahead of
    -- the workstations waiting on it, then oldest first. Guarded so a
    -- non-numeric value can never break leasing.
    ORDER BY CASE WHEN jsonb_typeof(t.payload->'priority') = 'number'
                  THEN (t.payload->>'priority')::numeric ELSE 0 END DESC,
             t.created_at
    FOR UPDATE OF t SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: LeaseTaskForBuild :one
-- LeaseTask scoped to ONE build. This is the production lease path for the
-- per-build-lane runner (each build's lane leases only its own tasks, so no
-- build's queue ever sits behind another's -- there is no cross-build FIFO).
-- It is ALSO what the tests use for isolation: several packages' tests run
-- concurrently against the same shared laforge_dev database, each driving its
-- own build, and scoping by build_id keeps one test's runner from completing
-- another's task. The per-builder max_concurrency cap is still enforced here --
-- it is a deliberate CROSS-build cap on a shared hoster, the one place two
-- builds on the same capped builder legitimately share a limit.
UPDATE task
SET status = 'leased', lease_owner = $1, lease_expires_at = now() + $2::interval,
    attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT t.id FROM task t
    LEFT JOIN build b ON b.id = t.build_id
    LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
    LEFT JOIN builder_config bc ON bc.name = cb.builder_config_name
    WHERE t.build_id = $3
      AND (t.status = 'pending' OR (t.status = 'leased' AND t.lease_expires_at < now()))
      AND (
        bc.max_concurrency IS NULL OR bc.max_concurrency = 0
        OR (
            SELECT count(*) FROM task it
            JOIN build ib ON ib.id = it.build_id
            JOIN configured_build icb ON icb.id = ib.configured_build_id
            WHERE icb.builder_config_name = bc.name
              AND it.status = 'leased' AND it.lease_expires_at >= now()
        ) < bc.max_concurrency
      )
    -- Critical path first: a deploy task's payload carries its priority (how
    -- many objects depends_on it; networks highest -- see the orchestrator's
    -- deployTaskPayload), so every team's domain controller is leased ahead of
    -- the workstations waiting on it, then oldest first. Guarded so a
    -- non-numeric value can never break leasing.
    ORDER BY CASE WHEN jsonb_typeof(t.payload->'priority') = 'number'
                  THEN (t.payload->>'priority')::numeric ELSE 0 END DESC,
             t.created_at
    FOR UPDATE OF t SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: LeaseTaskForBuilder :one
-- Lease the next leasable task that belongs to one CONFIGURED BUILDER (across
-- every build using it), oldest first. This is the production lease path: the
-- runner runs one worker pool PER configured builder, sized to that builder's
-- own concurrency limit (builder_config.max_concurrency), because concurrency is
-- a property of the hardware -- the cluster/account/host-pool -- not of a build.
-- All builds on a builder share its pool; different builders never contend. A
-- build with no configured_build (ad-hoc/fake) has builder key '' and its own
-- pool. There is NO cap subquery here -- the per-builder pool's size is the
-- limit, so the scheduler stays a dumb, fair lease.
UPDATE task
SET status = 'leased', lease_owner = $1, lease_expires_at = now() + $2::interval,
    attempts = attempts + 1, updated_at = now()
WHERE id = (
    SELECT t.id FROM task t
    LEFT JOIN build b ON b.id = t.build_id
    LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
    WHERE coalesce(cb.builder_config_name, '') = $3
      AND (t.status = 'pending' OR (t.status = 'leased' AND t.lease_expires_at < now()))
    -- Critical path first: a deploy task's payload carries its priority (how
    -- many objects depends_on it; networks highest -- see the orchestrator's
    -- deployTaskPayload), so every team's domain controller is leased ahead of
    -- the workstations waiting on it, then oldest first. Guarded so a
    -- non-numeric value can never break leasing.
    ORDER BY CASE WHEN jsonb_typeof(t.payload->'priority') = 'number'
                  THEN (t.payload->>'priority')::numeric ELSE 0 END DESC,
             t.created_at
    FOR UPDATE OF t SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: ListBuildersWithLeasableTasks :many
-- The distinct configured builders (by builder_config name; '' for builds with
-- no configured_build) that currently have at least one leasable task. The
-- runner's supervisor polls this to decide which builders need a worker pool
-- running -- cheap, and builders never couple to each other.
SELECT DISTINCT coalesce(cb.builder_config_name, '') AS builder_config_name
FROM task t
LEFT JOIN build b ON b.id = t.build_id
LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE t.status = 'pending' OR (t.status = 'leased' AND t.lease_expires_at < now());

-- name: CountLeasableTasksForBuilder :one
-- How many leasable tasks (pending or expired-lease) one configured builder
-- still has across all its builds. A builder's pool uses this to know when its
-- queue is drained (0) so it can exit; the supervisor restarts it when work
-- reappears.
SELECT count(*) FROM task t
LEFT JOIN build b ON b.id = t.build_id
LEFT JOIN configured_build cb ON cb.id = b.configured_build_id
WHERE coalesce(cb.builder_config_name, '') = $1
  AND (t.status = 'pending' OR (t.status = 'leased' AND t.lease_expires_at < now()));

-- name: HeartbeatTask :execrows
-- Zero rows affected means the lease was already reclaimed out from under
-- the caller (it should stop working immediately -- see internal/runner).
UPDATE task SET lease_expires_at = now() + $3::interval, updated_at = now()
WHERE id = $1 AND lease_owner = $2 AND status = 'leased';

-- name: CompleteTask :one
UPDATE task SET status = 'done', last_error = NULL, updated_at = now()
WHERE id = $1 AND lease_owner = $2
RETURNING *;

-- name: RetryTask :one
-- Holds the task for a backoff window before it can be re-leased, rather
-- than failing it -- used while attempts is still under the runner's cap.
-- The row stays 'leased' with its lease_expires_at pushed out by the
-- backoff interval; LeaseTask only reclaims a leased task once its
-- lease_expires_at is in the past, so this is exactly "don't retry before
-- now()+backoff" with no extra column. This is the "retries use backoff
-- and a cap" the runner documents: a bare 'pending' is re-leased on the
-- very next tick (every attempt within one second), which is useless for a
-- transient failure like a network still draining its used_by after its
-- instances detach, or a builder still mid-operation.
UPDATE task SET status = 'leased', last_error = $3,
    lease_expires_at = now() + $4::interval, updated_at = now()
WHERE id = $1 AND lease_owner = $2
RETURNING *;

-- name: DeferTask :one
-- Waiting for VM startup is not a failed forwarding attempt.
UPDATE task SET status = 'leased', last_error = $3,
    attempts = GREATEST(attempts - 1, 0),
    lease_expires_at = now() + $4::interval, updated_at = now()
WHERE id = $1 AND lease_owner = $2
RETURNING *;

-- name: FailTask :one
-- permanent records the runner's own classification of the failure
-- (builder.ErrPermanentConfig): a failure retrying cannot fix. The
-- orchestrator's teardown reads it back to stop re-queuing a destroy that can
-- never succeed (see ListLatestDestroyTasksByBuild).
UPDATE task SET status = 'failed', last_error = $3, permanent = $4, updated_at = now()
WHERE id = $1 AND lease_owner = $2
RETURNING *;

-- name: ListLatestDestroyTasksByBuild :many
-- The most recent destroy task per object of a build, with how many destroy
-- tasks for that object have failed so far -- what Teardown decides from: an
-- open task means a destroy is in flight (nothing to do), a failed permanent one
-- means stop (destroy_failed), a failed transient one is re-queued only once a
-- backoff scaled by failed_count has elapsed since it failed (updated_at).
SELECT DISTINCT ON (t.deployed_object_id) t.*,
    (SELECT count(*) FROM task f
     WHERE f.deployed_object_id = t.deployed_object_id
       AND f.kind LIKE 'destroy_%' AND f.status = 'failed')::int AS failed_count
FROM task t
WHERE t.build_id = $1 AND t.deployed_object_id IS NOT NULL AND t.kind LIKE 'destroy_%'
ORDER BY t.deployed_object_id, t.created_at DESC;

-- name: CancelOpenTasksForBuild :execrows
-- Closes every pending/leased task of a build as failed with the given reason.
-- Used when a teardown is forced to torn_down: nothing should keep working on
-- a build the operator has declared gone.
UPDATE task SET status = 'failed', last_error = $2, updated_at = now()
WHERE build_id = $1 AND status IN ('pending', 'leased');

-- name: GetTask :one
SELECT * FROM task WHERE id = $1;

-- name: ListTasksByBuild :many
SELECT * FROM task WHERE build_id = $1 ORDER BY created_at;

-- name: CreateEvent :one
INSERT INTO event (build_id, task_id, kind, message, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateObjectEvent :one
-- An event attributed to a specific deployed_object, so it shows on that host's
-- own log timeline (ListEventsByDeployedObject) as well as the build-wide log.
-- Used for shell-session audit (who opened a prompt on which host).
INSERT INTO event (build_id, deployed_object_id, kind, message, payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListEventsByBuild :many
SELECT * FROM event WHERE build_id = $1 ORDER BY created_at;

-- name: ListEventsByBuildWithObject :many
-- The build-wide journal, each event resolved to the host it's about so
-- the Logs page can show "which host" per line. An event attributes to an
-- object either directly (event.deployed_object_id, agent step events) or
-- through its task (event.task_id -> task.deployed_object_id, lifecycle
-- events) -- COALESCE picks whichever is set. LEFT JOINs throughout: a
-- build-level event (no object, no task) still comes back, with null host.
SELECT sqlc.embed(event),
       obj.as_name AS obj_as_name,
       obj.object_name AS obj_object_name,
       tm.team_number AS team_number
FROM event
LEFT JOIN task ON task.id = event.task_id
LEFT JOIN deployed_object obj ON obj.id = COALESCE(event.deployed_object_id, task.deployed_object_id)
LEFT JOIN team tm ON tm.id = obj.team_id
WHERE event.build_id = $1
ORDER BY event.created_at;

-- name: ListEventsByBuildSinceWithObject :many
-- The SSE live path's object-resolved counterpart to
-- ListEventsByBuildSince -- same host resolution as
-- ListEventsByBuildWithObject, so a live agent step streams in already
-- labelled with its host rather than the client having to resolve it.
SELECT sqlc.embed(event),
       obj.as_name AS obj_as_name,
       obj.object_name AS obj_object_name,
       tm.team_number AS team_number
FROM event
LEFT JOIN task ON task.id = event.task_id
LEFT JOIN deployed_object obj ON obj.id = COALESCE(event.deployed_object_id, task.deployed_object_id)
LEFT JOIN team tm ON tm.id = obj.team_id
WHERE event.build_id = $1 AND event.created_at > $2
ORDER BY event.created_at
LIMIT 500;

-- name: ListEventsByBuildSince :many
-- Powers the SSE live-status endpoint's poll loop (see internal/api's
-- live.go): only what's new since the last created_at it already sent
-- (a UUID primary key carries no ordering, so the timestamp is the real
-- cursor here), so a long-lived connection doesn't re-walk the whole
-- journal every tick.
SELECT * FROM event WHERE build_id = $1 AND created_at > $2 ORDER BY created_at
LIMIT 500;

-- name: ListEventsByDeployedObject :many
-- "Per-object logs" -- two
-- paths into the same journal, unioned: deploy/destroy/access lifecycle
-- events (event.task_id -> task.deployed_object_id, since those come
-- from a real orchestrator `task` row) and agent step events
-- (event.deployed_object_id, set directly -- see migrations/00009: a
-- step happens after its object's own deploy task is long done, so
-- there's no live `task` row left to attribute it to). UNION, not
-- UNION ALL: the two paths are mutually exclusive by construction
-- (CreateEvent never sets deployed_object_id; the gateway's step events
-- never set task_id), but UNION's dedup is a free safety net against
-- that ever drifting rather than something this query depends on.
SELECT event.* FROM event
JOIN task ON task.id = event.task_id
WHERE task.deployed_object_id = $1
UNION
SELECT event.* FROM event
WHERE event.deployed_object_id = $1
ORDER BY created_at;

-- name: CreateAgentStepEvent :exec
-- The gateway's own half of "every object has its own log... a step" --
-- internal/gateway's handleReportStatus calls
-- this on every step completion/failure it already knows about (see
-- migrations/00009's own doc comment for why this needs a new column
-- rather than reusing CreateEvent's task_id). :exec, not :one -- same
-- reasoning as CreateAgentHeartbeat: RETURNING needs SELECT privilege,
-- which laforge_gateway's real INSERT-only grant on `event` doesn't
-- have, matching its own "write, never read back" design.
INSERT INTO event (build_id, deployed_object_id, kind, message, payload)
VALUES ($1, $2, $3, $4, $5);

-- name: GetBuildIDByDeployedObject :one
-- What handleReportStatus needs before it can write an event at all --
-- CreateAgentStepEvent's own build_id, resolved the same way every other
-- deployed_object -> team -> build hop in this schema already does
-- (agent.sql's ListAgentHeartbeatsByBuild, ListAgentSessionsByBuild).
SELECT team.build_id FROM deployed_object
JOIN team ON team.id = deployed_object.team_id
WHERE deployed_object.id = $1;

-- name: GetLogContextByDeployedObject :one
-- The identity the log sink needs to make one captured console line a
-- self-contained record for an external ingester: its build, team number,
-- display name and kind, resolved from the agent's deployed_object id (its
-- cert CN). Same deployed_object -> team -> build hop GetBuildIDByDeployedObject
-- uses; the laforge_gateway role already has SELECT on all three tables
-- (migrations/00004). The gateway caches this per object, so it is read once
-- per object, not once per log batch.
SELECT team.build_id, team.team_number,
       deployed_object.object_name, deployed_object.as_name, deployed_object.kind
FROM deployed_object
JOIN team ON team.id = deployed_object.team_id
WHERE deployed_object.id = $1;

-- name: EnsureFakeHosterResource :one
-- The fake builder's own idempotent "deploy": create-or-adopt, keyed on a
-- deterministic external_ref the orchestrator assigns. Backed by a real
-- table (not runner-process memory) specifically so killing a runner can
-- never lose what "the hoster" already has -- see migrations/00003's
-- comment on this table.
INSERT INTO fake_hoster_resource (external_ref, kind)
VALUES ($1, $2)
ON CONFLICT (external_ref) DO UPDATE SET ensure_count = fake_hoster_resource.ensure_count + 1, updated_at = now()
RETURNING *;

-- name: DestroyFakeHosterResource :one
UPDATE fake_hoster_resource SET destroyed = true, updated_at = now() WHERE external_ref = $1 RETURNING *;

-- name: GetFakeHosterResource :one
SELECT * FROM fake_hoster_resource WHERE external_ref = $1;

-- name: ListFakeHosterResources :many
SELECT * FROM fake_hoster_resource ORDER BY external_ref;

-- name: CountFakeHosterResourcesByRef :one
-- Used by chaos tests to assert convergence directly: exactly one row for
-- a given external_ref, no matter how many times a crashed-and-retried
-- runner called ensure for it.
SELECT count(*) FROM fake_hoster_resource WHERE external_ref = $1;

-- name: DeleteFakeDNSRecordsByTeam :exec
-- The fake builder's own DeployDNSRecords is "replace the whole set,"
-- unlike DeployNetwork/Host/Container's per-resource "create or adopt" --
-- a team has one DNS record set, not one row to ensure per record. Called
-- inside the same transaction as InsertFakeDNSRecord below (see
-- internal/builder/fake's DeployDNSRecords) so a reader never observes a
-- half-replaced set.
DELETE FROM fake_dns_record WHERE team = $1;

-- name: InsertFakeDNSRecord :exec
INSERT INTO fake_dns_record (team, name, type, value, priority)
VALUES ($1, $2, $3, $4, $5);

-- name: ListFakeDNSRecordsByTeam :many
SELECT * FROM fake_dns_record WHERE team = $1 ORDER BY name, type, value;

-- name: ListActiveBuilds :many
-- Home's "what is live right now": every build that has (or is getting,
-- or still has leftover) real infrastructure, with the repository, commit
-- and builder it runs on. A build created outside a configured build has
-- no builder recorded here.
SELECT build.*, content_revision.repository_id, content_revision.commit_sha,
       COALESCE(content_revision.ref, '')::text AS ref,
       COALESCE(configured_build.builder_config_name, '')::text AS builder_config_name
FROM build
JOIN content_revision ON content_revision.id = build.content_revision_id
LEFT JOIN configured_build ON configured_build.id = build.configured_build_id
WHERE build.status IN ('deploying', 'building', 'finished', 'failed', 'tearing_down')
ORDER BY build.created_at DESC;

-- name: CountAgentTasksByBuild :many
-- Steps outstanding and failed, per build, for Home's health strip.
SELECT team.build_id, agent_task.status, count(*)::bigint AS n
FROM agent_task
JOIN deployed_object ON deployed_object.id = agent_task.deployed_object_id
JOIN team ON team.id = deployed_object.team_id
WHERE team.build_id = ANY(sqlc.arg(build_ids)::uuid[])
GROUP BY team.build_id, agent_task.status;

-- name: SetDeployedObjectPublicAddress :exec
UPDATE deployed_object SET public_address = $2, updated_at = now() WHERE id = $1;

-- name: DiscardSupersededPlannedBuilds :many
-- A newer planned build (keep_id) was just created for this configured build:
-- every OTHER planned build it has is now a superseded revision nobody will
-- deploy, so it goes terminal as 'discarded' (migration 00063). Keeps at most
-- one planned build per configured_build, which is what stops planned builds
-- accumulating one per push forever and each holding an orchestrator lane.
-- Only 'planned' rows are touched: anything deploying/building/finished is real
-- infrastructure and is handled by apply-in-place or teardown, never this.
UPDATE build SET status = 'discarded'
WHERE configured_build_id = sqlc.arg(configured_build_id)
  AND status = 'planned'
  AND id <> sqlc.arg(keep_id)
RETURNING *;

-- name: DiscardPlannedBuildsForConfiguredBuild :many
-- The configured build is being removed: its planned builds (never deployed,
-- nothing at a hoster) have no future, so they go terminal as 'discarded'
-- rather than lingering as detached 'planned' rows that the orchestrator would
-- keep reconciling and that nobody could meaningfully deploy.
UPDATE build SET status = 'discarded'
WHERE configured_build_id = $1 AND status = 'planned'
RETURNING *;

-- name: DiscardStalePlannedBuilds :many
-- The sweep behind the two queries above, for rows that predate them: a planned
-- build is stale when its configured_build is gone (configured_build_id was set
-- NULL by the FK on delete) or when a newer planned build exists for the same
-- configured_build. Run by the orchestrator at startup and on its slow
-- housekeeping tick, so an upgraded instance heals its backlog without manual
-- SQL and a planned build never slips through the two write paths.
UPDATE build AS b SET status = 'discarded'
WHERE b.status = 'planned'
  AND (b.configured_build_id IS NULL
       OR EXISTS (SELECT 1 FROM build AS n
                  WHERE n.configured_build_id = b.configured_build_id
                    AND n.status = 'planned'
                    AND (n.created_at, n.id) > (b.created_at, b.id)))
RETURNING *;
