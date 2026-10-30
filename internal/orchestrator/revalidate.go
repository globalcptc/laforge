package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// Revalidation re-runs every validator a host already has, on the box as it
// is, and reports what each check says now. Nothing else runs: no step, no
// restart, no fix. The host's own build is untouched -- its steps, Steps tab and
// build-time validator results -- and the one change a revalidation makes is
// to the host's status, and only finished <-> invalid.

// ValidationRunTimeout is how long a run's checks may wait for the agent to run
// them before the run is written off as timed out (and its unrun checks
// dropped, so they don't fire long after anyone is looking).
var ValidationRunTimeout = 15 * time.Minute

// ErrNoValidators: the host has no validate checks to re-run.
var ErrNoValidators = errors.New("the host has no validators")

// ErrValidationRunning: a revalidation is already running on the host.
var ErrValidationRunning = errors.New("a revalidation is already running on this host")

// StartValidationRun queues one host's revalidation under reportID: a copy of
// each of its validate checks, as its build rendered them. Returns the run.
func StartValidationRun(ctx context.Context, pool *pgxpool.Pool, reportID pgtype.UUID, buildID pgtype.UUID, obj db.DeployedObject, accountID pgtype.UUID) (db.ValidationRun, error) {
	var run db.ValidationRun
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		// Serialize against a second click on the same host.
		if _, err := tx.Exec(ctx, "SELECT 1 FROM deployed_object WHERE id = $1 FOR UPDATE", obj.ID); err != nil {
			return err
		}
		open, err := q.HasOpenValidationRunForObject(ctx, obj.ID)
		if err != nil {
			return err
		}
		if open {
			return ErrValidationRunning
		}
		checks, err := q.ListValidateTasksForObject(ctx, obj.ID)
		if err != nil {
			return err
		}
		if len(checks) == 0 {
			return ErrNoValidators
		}
		run, err = q.CreateValidationRun(ctx, db.CreateValidationRunParams{
			ReportID: reportID, BuildID: buildID, DeployedObjectID: obj.ID, CreatedByAccountID: accountID, StatusBefore: obj.Status,
		})
		if err != nil {
			return err
		}
		next, err := q.NextStepIndexForHost(ctx, obj.ID)
		if err != nil {
			return err
		}
		for i, c := range checks {
			if _, err := q.CreateValidationRunTask(ctx, db.CreateValidationRunTaskParams{
				DeployedObjectID: obj.ID, StepIndex: next + int32(i), Payload: c.Payload, IgnoreErrors: c.IgnoreErrors, ValidationRunID: run.ID,
			}); err != nil {
				return err
			}
		}
		decide(ctx, q, buildID, obj.ID, "validation.started", fmt.Sprintf("%s: revalidating -- re-running %d validator step(s)", objectLabel(obj, 0), len(checks)),
			map[string]any{"validation_run_id": run.ID, "report_id": reportID, "checks": len(checks)})
		return nil
	})
	return run, err
}

// SettleValidationRuns closes every run in the build whose checks have all
// reported (passed / failed), whose checks vanished (canceled: the host was
// rebuilt), or that waited too long (timed_out), and applies a finished run's
// verdict to the host: a failing check turns finished into invalid, every
// check passing turns invalid into finished. Any other host status is left as
// it is -- a host still building, or failed by a step, isn't a validator call.
// Returns whether any host's status changed (so the build's own status, which
// is derived from its hosts', can be re-settled).
func SettleValidationRuns(ctx context.Context, q *db.Queries, buildID pgtype.UUID) (bool, error) {
	runs, err := q.ListOpenValidationRunsByBuild(ctx, buildID)
	if err != nil {
		return false, fmt.Errorf("listing open validation runs: %w", err)
	}
	changed := false
	for _, run := range runs {
		sum, err := q.SummarizeValidationRun(ctx, run.ID)
		if err != nil {
			return changed, fmt.Errorf("summarizing validation run %s: %w", run.ID.String(), err)
		}
		obj, err := q.GetDeployedObject(ctx, run.DeployedObjectID)
		if err != nil {
			return changed, fmt.Errorf("loading %s: %w", run.DeployedObjectID.String(), err)
		}
		label := objectLabel(obj, 0)
		switch {
		case sum.Total == 0:
			// Its checks were deleted under it: a rebuild or restart cleared the
			// host's tasks. Nothing to report.
			if _, err := q.FinishValidationRun(ctx, db.FinishValidationRunParams{ID: run.ID, Status: "canceled"}); err != nil {
				return changed, err
			}
			decide(ctx, q, buildID, obj.ID, "validation.canceled", fmt.Sprintf("%s: revalidation canceled -- the host's tasks were cleared (rebuilt or restarted)", label),
				map[string]any{"validation_run_id": run.ID})
		case sum.Open > 0:
			if time.Since(run.CreatedAt.Time) < ValidationRunTimeout {
				continue
			}
			if err := q.DeleteOpenValidationRunTasks(ctx, run.ID); err != nil {
				return changed, err
			}
			if _, err := q.FinishValidationRun(ctx, db.FinishValidationRunParams{ID: run.ID, Status: "timed_out"}); err != nil {
				return changed, err
			}
			decide(ctx, q, buildID, obj.ID, "validation.timed_out", fmt.Sprintf("%s: revalidation timed out -- %d of %d check step(s) never ran in %s (is the agent checking in?)", label, sum.Open, sum.Total, ValidationRunTimeout),
				map[string]any{"validation_run_id": run.ID, "open": sum.Open, "total": sum.Total})
		default:
			passed := sum.Failed == 0
			status := "passed"
			after := obj.Status
			if passed {
				n, err := q.SetDeployedObjectFinishedFromInvalid(ctx, obj.ID)
				if err != nil {
					return changed, err
				}
				if n > 0 {
					after = "finished"
				}
			} else {
				status = "failed"
				msg := fmt.Sprintf("revalidation: %d of %d validator step(s) failed", sum.Failed, sum.Total)
				n, err := q.SetDeployedObjectInvalidFromFinished(ctx, db.SetDeployedObjectInvalidFromFinishedParams{ID: obj.ID, LastError: db.StrPtr(msg)})
				if err != nil {
					return changed, err
				}
				if n > 0 {
					after = "invalid"
				}
			}
			if _, err := q.FinishValidationRun(ctx, db.FinishValidationRunParams{ID: run.ID, Status: status, StatusAfter: db.StrPtr(after)}); err != nil {
				return changed, err
			}
			msg := fmt.Sprintf("%s: revalidation %s (%d of %d validator step(s) failed)", label, status, sum.Failed, sum.Total)
			if after != obj.Status {
				msg += fmt.Sprintf("; %s -> %s", obj.Status, after)
				changed = true
			}
			decide(ctx, q, buildID, obj.ID, "validation."+status, msg,
				map[string]any{"validation_run_id": run.ID, "failed": sum.Failed, "total": sum.Total, "from": obj.Status, "to": after})
		}
	}
	return changed, nil
}
