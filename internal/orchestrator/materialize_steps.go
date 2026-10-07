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
// agent_task rows -- but only once its box is up AND every dependency has
// finished configuring. This is the real meaning of depends_on: "a domain
// controller must be configured as a domain controller, not just running
// Windows." The box itself deploys ahead of time (reconcileObject no longer
// waits on depends_on); this is where the wait moved to, so slow VM
// provisioning overlaps dependency configuration instead of serializing behind
// it.
//
// Idempotent across reconcile passes via the steps_materialized_at stamp:
// materialization runs exactly once per deploy, set even when an object has
// zero authored steps (so the lifecycle advancer can finish it rather than
// leaving it forever "waiting to materialize"). A redeploy clears the stamp
// (ResetDeployedObjectForRedeploy) so steps re-materialize for the rebuilt
// instance.
func materializeStepsIfReady(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envName string, buildID pgtype.UUID, obj db.DeployedObject, teamNum int, asName string, dependsOn []string, placed, finishedDeps, failedDeps map[string]bool, active bool) error {
	if !active {
		return nil // planned/terminal build: never queue real agent work
	}
	if obj.Kind != "host" && obj.Kind != "container" {
		return nil // networks have no agent and no steps
	}
	if obj.StepsMaterializedAt.Valid {
		return nil // already materialized for this deploy
	}
	// Only a box that actually came up can run steps. A box still being
	// created (pending/deploying), already failed, or being torn down is not
	// eligible -- try again on a later pass if it reaches running.
	if obj.Status != "running" && obj.Status != "building" {
		return nil
	}
	// A dependency that failed terminally can never be a correct prerequisite:
	// don't run this object's steps against a broken DC/database -- fail it
	// with a clear reason instead of waiting on a dependency that will never
	// finish. (A dependency not placed in this environment can't be ordered
	// on, so it's ignored, matching reconcile's depends_on handling.)
	for _, d := range dependsOn {
		if placed[d] && failedDeps[d] {
			return q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{
				ID:        obj.ID,
				LastError: db.StrPtr(fmt.Sprintf("dependency %q failed to configure", d)),
			})
		}
	}
	// Wait until every placed dependency has fully finished.
	for _, d := range dependsOn {
		if placed[d] && !finishedDeps[d] {
			return nil // not yet -- a later pass will try again
		}
	}
	return materializeSteps(ctx, q, repoRoot, c, envName, buildID, obj, teamNum, asName)
}

// materializeSteps expands an object's authored steps into agent_task rows and
// stamps steps_materialized_at. Reuses the exact same internal/gateway.
// ExpandSteps translation the ad-hoc task endpoint and the schedule dispatcher
// use, so a step queued here and one queued by hand produce byte-identical
// commands. New rows start at NextStepIndexForHost so they never collide with
// an ad-hoc/scheduled command that happened to land first.
func materializeSteps(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envName string, buildID pgtype.UUID, obj db.DeployedObject, teamNum int, asName string) error {
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
			DeployedObjectID: obj.ID, StepIndex: next + int32(i), Command: cmd.Command, Payload: payload, IgnoreErrors: cmd.IgnoreErrors,
		}); err != nil {
			return fmt.Errorf("queuing step %d (%s): %w", i, cmd.Command, err)
		}
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
