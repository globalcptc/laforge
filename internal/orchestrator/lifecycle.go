package orchestrator

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// AdvanceObjectLifecycle moves each host/container off the "infra is up"
// states (running/building) toward a terminal build outcome, from the real
// agent evidence -- an environment must not read as finished until every
// authored step and validator has actually completed, and this is what
// enforces that. Pure DB: no builder, no content, so it runs on the same
// slow poll the power/build-status advancers do and works for a build whose
// revision can no longer be fetched.
//
// Per object:
//   - running: the instance is up but the agent has not been seen. Once a
//     session exists (the agent has checked in), it becomes building; a
//     network -- which has no in-guest agent -- goes straight to finished.
//   - building (or a running object that just checked in): decided from its
//     steps. Any failed step -> build_failed. Any step still pending/leased
//     -> stay building. Otherwise every step is done: a failed validator ->
//     invalid, else -> finished (an object with no steps at all finishes as
//     soon as its agent is in -- there is nothing to build).
//
// Terminal states (finished/build_failed/invalid/deploy_failed/destroyed…)
// are left untouched; the DB setters are additionally guarded to only
// advance from running/building, so a late poll can never bounce a settled
// object backward.
func AdvanceObjectLifecycle(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) error {
	q := db.New(pool)

	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing deployed objects: %w", err)
	}
	// Cheap early out: nothing is mid-lifecycle, so skip the extra queries.
	anyActive := false
	for _, o := range objs {
		if o.Status == "running" || o.Status == "building" {
			anyActive = true
			break
		}
	}
	if !anyActive {
		return nil
	}

	sessions, err := q.ListAgentSessionsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing agent sessions: %w", err)
	}
	checkedIn := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		checkedIn[s.DeployedObjectID.String()] = true
	}

	stepRows, err := q.SummarizeAgentTasksByObjectForBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("summarizing steps: %w", err)
	}
	type stepCounts struct{ total, open, failed int64 }
	steps := make(map[string]stepCounts, len(stepRows))
	for _, r := range stepRows {
		steps[r.DeployedObjectID.String()] = stepCounts{total: r.Total, open: r.Open, failed: r.Failed}
	}

	badValidatorIDs, err := q.ListObjectsWithFailedValidatorByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing failed validators: %w", err)
	}
	badValidator := make(map[string]bool, len(badValidatorIDs))
	for _, id := range badValidatorIDs {
		badValidator[id.String()] = true
	}

	// evaluateBuild decides the terminal-or-still-building state for an object
	// whose agent is in, from its step counts and validator results.
	evaluateBuild := func(id string) string {
		s := steps[id] // zero value when the object has no steps
		if s.failed > 0 {
			return "build_failed"
		}
		if s.open > 0 {
			return "building"
		}
		if badValidator[id] {
			return "invalid"
		}
		return "finished"
	}

	for _, o := range objs {
		id := o.ID.String()
		switch o.Status {
		case "running":
			if o.Kind == "network" {
				// A network has no agent -- once its infra is up there is
				// nothing more to build.
				if err := q.SetDeployedObjectFinished(ctx, o.ID); err != nil {
					return fmt.Errorf("finishing network %s: %w", id, err)
				}
				continue
			}
			if !checkedIn[id] {
				continue // instance up, agent not seen yet -- stay running
			}
			if err := applyObjectState(ctx, q, o.ID, evaluateBuild(id)); err != nil {
				return err
			}
		case "building":
			if err := applyObjectState(ctx, q, o.ID, evaluateBuild(id)); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyObjectState routes an evaluated state to its guarded DB setter. A
// result of "building" for an object already building is a no-op (nothing
// to write), so this is cheap to call every poll.
func applyObjectState(ctx context.Context, q *db.Queries, id pgtype.UUID, state string) error {
	var err error
	switch state {
	case "building":
		err = q.SetDeployedObjectBuilding(ctx, id)
	case "finished":
		err = q.SetDeployedObjectFinished(ctx, id)
	case "build_failed":
		err = q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{ID: id, LastError: db.StrPtr("a build step could not be completed")})
	case "invalid":
		err = q.SetDeployedObjectInvalid(ctx, db.SetDeployedObjectInvalidParams{ID: id, LastError: db.StrPtr("a validator did not pass")})
	}
	if err != nil {
		return fmt.Errorf("setting %s on %s: %w", state, id.String(), err)
	}
	return nil
}
