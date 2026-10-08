package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/gateway"
	"github.com/globalcptc/laforge/internal/loader"
)

// materializeStepsIfReady queues a host/container's authored `steps:` as real
// agent_task rows as soon as its box is up -- even while it is still waiting on
// a dependency. That is the real meaning of depends_on ("a domain controller
// must be CONFIGURED as a domain controller, not just running Windows"), but the
// wait no longer hides the steps: while a dependency is unfinished the steps are
// queued with status 'blocked' -- visible in the UI, counted as open, but never
// leased by an agent -- and the object records what it's blocked_on. Once every
// dependency finishes, the blocked steps flip to 'pending' and run in order. So
// the box deploys ahead, the operator sees the queued steps and WHY they wait,
// and the agent still never runs a step against a half-built dependency.
//
// Idempotent across reconcile passes: steps_materialized_at is stamped once per
// deploy (even with zero authored steps, so the lifecycle advancer can finish a
// stepless object); after that, each pass only releases the block when the
// dependencies have finished. A redeploy clears the stamp
// (ResetDeployedObjectForRedeploy) so steps re-materialize for the rebuilt box.
func materializeStepsIfReady(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envName string, buildID pgtype.UUID, obj db.DeployedObject, teamNum int, asName string, dependsOn []string, placed, finishedDeps, failedDeps map[string]bool, active bool) error {
	if !active {
		return nil // planned/terminal build: never queue real agent work
	}
	if obj.Kind != "host" && obj.Kind != "container" {
		return nil // networks have no agent and no steps
	}
	// Only a box that actually came up can run steps. A box still being
	// created (pending/deploying), already failed, or being torn down is not
	// eligible -- try again on a later pass if it reaches running.
	if obj.Status != "running" && obj.Status != "building" {
		return nil
	}
	// A dependency that failed terminally can never be a correct prerequisite:
	// don't run this object's steps against a broken DC/database -- fail it with
	// a clear reason instead of waiting on a dependency that will never finish.
	// (A dependency not placed in this environment can't be ordered on, so it's
	// ignored, matching reconcile's depends_on handling.)
	for _, d := range dependsOn {
		if placed[d] && failedDeps[d] {
			return q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{
				ID:        obj.ID,
				LastError: db.StrPtr(fmt.Sprintf("dependency %q failed to configure", d)),
			})
		}
	}
	// Which placed dependencies haven't finished yet -- what this object is
	// blocked on (empty = free to run).
	var blockedOn []string
	for _, d := range dependsOn {
		if placed[d] && !finishedDeps[d] {
			blockedOn = append(blockedOn, d)
		}
	}

	if obj.StepsMaterializedAt.Valid {
		// Already materialized. The only thing left is to release it the moment
		// its dependencies have all finished: flip blocked steps to pending and
		// clear the reason. No-ops once released, so this is cheap every pass.
		if len(blockedOn) == 0 {
			if err := q.UnblockAgentTasksForObject(ctx, obj.ID); err != nil {
				return fmt.Errorf("releasing blocked steps: %w", err)
			}
			return q.SetDeployedObjectBlockedOn(ctx, db.SetDeployedObjectBlockedOnParams{ID: obj.ID, BlockedOn: nil})
		}
		return nil
	}
	return materializeSteps(ctx, q, repoRoot, c, envName, buildID, obj, teamNum, asName, blockedOn)
}

// materializeSteps expands an object's authored steps into agent_task rows and
// stamps steps_materialized_at. Reuses the exact same internal/gateway.
// ExpandSteps translation the ad-hoc task endpoint and the schedule dispatcher
// use, so a step queued here and one queued by hand produce byte-identical
// commands. New rows start at NextStepIndexForHost so they never collide with
// an ad-hoc/scheduled command that happened to land first.
func materializeSteps(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envName string, buildID pgtype.UUID, obj db.DeployedObject, teamNum int, asName string, blockedOn []string) error {
	// Steps queued while a dependency is still unfinished start 'blocked':
	// visible and counted as open, but never leased until released.
	status := "pending"
	if len(blockedOn) > 0 {
		status = "blocked"
	}
	// Stored registry credentials, for a `compose:` step's `docker login`.
	registryAuth := gateway.WithRegistryAuth(func(host string) (gateway.RegistryAuth, bool) {
		cred, err := q.GetRegistryCredentialByHost(ctx, host)
		if err != nil {
			return gateway.RegistryAuth{}, false
		}
		return gateway.RegistryAuth{Username: cred.Username, Secret: cred.Secret}, true
	})
	authored, notes, err := gateway.ExpandSteps(repoRoot, c, envName, asName, teamNum, registryAuth)
	if err != nil {
		return fmt.Errorf("expanding steps: %w", err)
	}
	next, err := q.NextStepIndexForHost(ctx, obj.ID)
	if err != nil {
		return fmt.Errorf("checking existing steps: %w", err)
	}
	for i, cmd := range authored {
		payload, err := json.Marshal(cmd.Payload)
		if err != nil {
			return fmt.Errorf("encoding payload for step %d: %w", i, err)
		}
		if _, err := q.CreateAgentTask(ctx, db.CreateAgentTaskParams{
			DeployedObjectID: obj.ID, StepIndex: next + int32(i), Command: cmd.Command, Payload: payload, IgnoreErrors: cmd.IgnoreErrors, Status: status,
		}); err != nil {
			return fmt.Errorf("queuing step %d (%s): %w", i, cmd.Command, err)
		}
	}
	// Record what the (blocked) steps are waiting on, for the UI. Empty when not
	// blocked -- SetDeployedObjectBlockedOn with nil clears it.
	var blockedJSON []byte
	if len(blockedOn) > 0 {
		blockedJSON, _ = json.Marshal(blockedOn)
	}
	if err := q.SetDeployedObjectBlockedOn(ctx, db.SetDeployedObjectBlockedOnParams{ID: obj.ID, BlockedOn: blockedJSON}); err != nil {
		return fmt.Errorf("recording blocked_on: %w", err)
	}
	// Stamp even for zero steps: it is what lets the lifecycle advancer finish
	// a stepless object instead of treating it as still waiting.
	if err := q.MarkDeployedObjectStepsMaterialized(ctx, obj.ID); err != nil {
		return fmt.Errorf("marking steps materialized: %w", err)
	}
	for _, n := range notes {
		q.CreateEvent(ctx, db.CreateEventParams{BuildID: buildID, Kind: "steps.note", Message: n, Payload: []byte("{}")})
	}
	if len(authored) > 0 {
		q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: buildID, Kind: "steps.materialized",
			Message: fmt.Sprintf("%d step command(s) queued for %s", len(authored), asName),
			Payload: []byte("{}"),
		})
	}
	return nil
}
