package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// AgentLostAfter is how long an agent that has checked in may go without a
// heartbeat, while its host still has steps to run, before the host is
// failed as lost. AgentNeverCheckedInAfter is the same for an agent that
// has never connected since the instance came up (longer: a Windows first
// boot is slow). Both are set from the orchestrator's flags; 0 disables.
var (
	AgentLostAfter           = 30 * time.Minute
	AgentNeverCheckedInAfter = 60 * time.Minute
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
	// When each object's current instance came up. A rebuild keeps the
	// object's agent_session row (one per object), so until the new agent
	// connects, that row still holds the OLD instance's last heartbeat.
	deployedAt := make(map[string]time.Time, len(objs))
	for _, o := range objs {
		if o.DeployedAt.Valid {
			deployedAt[o.ID.String()] = o.DeployedAt.Time
		}
	}
	checkedIn := make(map[string]bool, len(sessions))
	lastHeartbeat := make(map[string]time.Time, len(sessions))
	for _, s := range sessions {
		// A session whose last heartbeat predates the current instance belongs
		// to the instance a rebuild replaced: the new agent has not checked in
		// yet. Counting it would move the host to building at once and then,
		// with that hours-old heartbeat, fail it as a lost agent before the
		// new instance has even booted.
		if da, ok := deployedAt[s.DeployedObjectID.String()]; ok && s.LastHeartbeatAt.Time.Before(da) {
			continue
		}
		checkedIn[s.DeployedObjectID.String()] = true
		lastHeartbeat[s.DeployedObjectID.String()] = s.LastHeartbeatAt.Time
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

	// Which objects have had their steps materialized. An object's steps are
	// now queued lazily -- only once its dependencies have finished -- so a
	// host with its agent in but no agent_task rows yet is NOT done: it's still
	// waiting for a dependency, not finished-with-nothing-to-do. The stamp is
	// what tells those two apart (set even for a genuinely stepless object).
	materialized := make(map[string]bool, len(objs))
	for _, o := range objs {
		materialized[o.ID.String()] = o.StepsMaterializedAt.Valid
	}

	// evaluateBuild decides the terminal-or-still-building state for an object
	// whose agent is in, from its step counts and validator results.
	evaluateBuild := func(id string) string {
		s := steps[id] // zero value when the object has no steps
		if s.failed > 0 {
			return "build_failed"
		}
		if !materialized[id] {
			// Agent is in but steps haven't been queued yet -- a dependency is
			// still configuring. Hold at building; don't finish prematurely
			// (which would both paint it green early and wrongly satisfy its
			// own dependents).
			return "building"
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
				// An agent that has never checked in, long past any first-boot
				// deadline, isn't coming: the host stops as build_failed with
				// that reason, so it's visible and can be rebuilt, instead of
				// "running" for the rest of the event.
				if AgentNeverCheckedInAfter > 0 && o.DeployedAt.Valid && time.Since(o.DeployedAt.Time) > AgentNeverCheckedInAfter {
					msg := fmt.Sprintf("agent never checked in: the instance came up %s ago and nothing has connected (rebuild the host, or check its console)", time.Since(o.DeployedAt.Time).Round(time.Minute))
					if err := q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{ID: o.ID, LastError: db.StrPtr(msg)}); err != nil {
						return fmt.Errorf("failing %s for a missing agent: %w", id, err)
					}
					decide(ctx, q, buildID, o.ID, "lifecycle.build_failed", fmt.Sprintf("%s: running -> build_failed: %s", objectLabel(o, 0), msg), map[string]any{"deployed_at": o.DeployedAt.Time})
					continue
				}
				decideRepeating(ctx, q, "checkin-wait/"+id, buildID, o.ID, "lifecycle.waiting", fmt.Sprintf("%s: instance is up; waiting for its agent to check in", objectLabel(o, 0)), nil)
				continue // instance up, agent not seen yet -- stay running
			}
			next := evaluateBuild(id)
			if err := applyObjectState(ctx, q, o.ID, next); err != nil {
				return err
			}
			noteObjectState(ctx, q, buildID, o, next, steps[id])
		case "building":
			next := evaluateBuild(id)
			// An agent that checked in and then went silent, with steps still
			// to run, isn't going to run them: the host stops as build_failed
			// with the time of its last heartbeat. Otherwise it sits "building"
			// indefinitely with 31 pending steps and nobody the wiser.
			if next == "building" && AgentLostAfter > 0 {
				if hb, ok := lastHeartbeat[id]; ok && !hb.IsZero() && time.Since(hb) > AgentLostAfter && steps[id].open > 0 {
					msg := fmt.Sprintf("agent stopped checking in %s ago (last heartbeat %s) with %d step(s) still to run; the host may be off, rebooted into a broken state, or the agent crashed -- restart it from a step once it's back, or rebuild it", time.Since(hb).Round(time.Minute), hb.Format(time.TimeOnly), steps[id].open)
					if err := q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{ID: o.ID, LastError: db.StrPtr(msg)}); err != nil {
						return fmt.Errorf("failing %s for a lost agent: %w", id, err)
					}
					decide(ctx, q, buildID, o.ID, "lifecycle.build_failed", fmt.Sprintf("%s: building -> build_failed: %s", objectLabel(o, 0), msg), map[string]any{"last_heartbeat": hb, "steps_open": steps[id].open})
					continue
				}
			}
			if err := applyObjectState(ctx, q, o.ID, next); err != nil {
				return err
			}
			noteObjectState(ctx, q, buildID, o, next, steps[id])
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

// noteObjectState records an object's lifecycle move (running -> building ->
// finished / build_failed / invalid) once, with the step counts behind it.
func noteObjectState(ctx context.Context, q *db.Queries, buildID pgtype.UUID, o db.DeployedObject, next string, s struct{ total, open, failed int64 }) {
	if next == o.Status {
		return
	}
	var why string
	switch next {
	case "building":
		why = fmt.Sprintf("agent checked in; %d of %d step(s) still to run", s.open, s.total)
	case "finished":
		why = fmt.Sprintf("every step done (%d) and every validator passed", s.total)
	case "build_failed":
		why = fmt.Sprintf("%d step(s) failed; %d of %d never ran", s.failed, s.open, s.total)
	case "invalid":
		why = "all steps done, but a validator did not pass"
	}
	decide(ctx, q, buildID, o.ID, "lifecycle."+next, fmt.Sprintf("%s: %s -> %s: %s", objectLabel(o, 0), o.Status, next, why),
		map[string]any{"from": o.Status, "to": next, "steps_total": s.total, "steps_open": s.open, "steps_failed": s.failed})
}
