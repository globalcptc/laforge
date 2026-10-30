package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// Recoverable build_failed causes (deployed_object.failure_kind). Each is a
// judgment the orchestrator made about something outside the host -- its agent
// went quiet, or a host it waits on failed -- so each is undone automatically
// once that stops being true. A failed step has no kind: only an operator
// (restart, rebuild) clears that.
const (
	FailureAgentLost           = "agent_lost"
	FailureAgentNeverCheckedIn = "agent_never_checked_in"
	FailureGate                = "gate"
	FailureDependency          = "dependency"
)

func failureKind(o db.DeployedObject) string {
	if o.Status != "build_failed" || o.FailureKind == nil {
		return ""
	}
	return *o.FailureKind
}

// agentFailure: failed because its agent was missing -- what a heartbeat
// after the failure disproves.
func agentFailure(o db.DeployedObject) bool {
	k := failureKind(o)
	return k == FailureAgentLost || k == FailureAgentNeverCheckedIn
}

// recoverObject moves a recoverably-failed object back to status, guarded on
// the kind it was failed for, and records why on the timeline. Returns whether
// it moved.
func recoverObject(ctx context.Context, q *db.Queries, buildID pgtype.UUID, o db.DeployedObject, status, why string) (bool, error) {
	kind := failureKind(o)
	n, err := q.RecoverDeployedObject(ctx, db.RecoverDeployedObjectParams{ID: o.ID, Status: status, FailureKind: db.StrPtr(kind)})
	if err != nil {
		return false, fmt.Errorf("recovering %s: %w", o.ID.String(), err)
	}
	if n == 0 {
		return false, nil
	}
	decide(ctx, q, buildID, o.ID, "lifecycle.recovered", fmt.Sprintf("%s: build_failed -> %s: %s", objectLabel(o, 0), status, why),
		map[string]any{"failure_kind": kind, "to": status})
	return true, nil
}

// recoverReturnedAgent: a host failed for a missing agent whose agent has
// heartbeated since goes back to building (steps queued) or running (not yet),
// and the lifecycle carries on from there as if it had never been failed.
// lastHeartbeat must already exclude a replaced instance's session.
func recoverReturnedAgent(ctx context.Context, q *db.Queries, buildID pgtype.UUID, o db.DeployedObject, lastHeartbeat time.Time) (string, error) {
	if !agentFailure(o) || lastHeartbeat.IsZero() {
		return "", nil
	}
	if o.FailedAt.Valid && !lastHeartbeat.After(o.FailedAt.Time) {
		return "", nil
	}
	status := "running"
	if o.StepsMaterializedAt.Valid {
		status = "building"
	}
	moved, err := recoverObject(ctx, q, buildID, o, status, fmt.Sprintf("its agent checked in again (heartbeat %s)", lastHeartbeat.Format(time.TimeOnly)))
	if err != nil || !moved {
		return "", err
	}
	return status, nil
}

// recoverCascade: a host failed because a dependency or wait_for target
// failed goes back once nothing it waits on is failed any more (that host
// recovered, or was restarted or rebuilt). It resumes waiting -- Dependency
// Blocked, or running its steps if the target has since finished. Returns the
// status it moved to, or "" if it stays failed.
func recoverCascade(ctx context.Context, q *db.Queries, buildID pgtype.UUID, o db.DeployedObject, dependsOn []string, placed, failedDeps map[string]bool, gates *teamGates) (string, error) {
	switch failureKind(o) {
	case FailureDependency:
		for _, d := range dependsOn {
			if placed[d] && failedDeps[d] {
				return "", nil
			}
		}
	case FailureGate:
		if gates == nil {
			return "", nil // no readiness loaded: never guess
		}
		failed, err := anyGateFailed(ctx, q, o, gates)
		if err != nil || failed {
			return "", err
		}
	default:
		return "", nil
	}
	status := "running"
	if o.StepsMaterializedAt.Valid {
		status = "building"
	}
	moved, err := recoverObject(ctx, q, buildID, o, status, "nothing it waits on has failed any more")
	if err != nil || !moved {
		return "", err
	}
	return status, nil
}

// anyGateFailed: whether any wait_for still holding this object's steps points
// at a target that has failed -- the same test releaseStepGates fails it on.
func anyGateFailed(ctx context.Context, q *db.Queries, o db.DeployedObject, gates *teamGates) (bool, error) {
	rows, err := q.ListBlockedStepGatesForObject(ctx, o.ID)
	if err != nil {
		return false, fmt.Errorf("listing step gates: %w", err)
	}
	for _, r := range rows {
		var entries []loader.WaitFor
		if err := json.Unmarshal(r.WaitFor, &entries); err != nil {
			return false, fmt.Errorf("decoding wait_for on step %d: %w", r.StepIndex, err)
		}
		for _, w := range entries {
			if gates.evaluate(w) == gateFailed {
				return true, nil
			}
		}
	}
	return false, nil
}

// reopenBuild puts a settled build back in the build phase once one of its
// hosts recovered, so reconcile runs for it again (releasing that host's
// steps, and recovering hosts that waited on it) -- the same move a restart
// from a step makes.
func reopenBuild(ctx context.Context, q *db.Queries, buildID pgtype.UUID) error {
	b, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading build: %w", err)
	}
	if b.Status != "finished" && b.Status != "failed" {
		return nil
	}
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: buildID, Status: "building"}); err != nil {
		return fmt.Errorf("re-entering the build phase: %w", err)
	}
	return nil
}
