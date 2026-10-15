// Teardown made real: "teardown (infrastructure)...
// available from the UI and the API", found completely
// missing by direct audit -- build.status's own CHECK constraint
// (migration 00003) already named 'torn_down' as a real terminal state,
// but nothing anywhere ever created a way to reach it. The only destroy
// machinery that existed was Reconcile's own per-object diff against
// desired content, which only ever destroys an object that fell OUT of
// content -- never a whole build on a deliberate operator action, and
// content might not even be resolvable by the time someone wants a build
// torn down (repository access revoked, branch deleted, checkout gone).
package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// teardownRequeueBase and teardownRequeueMax bound how soon Teardown asks for
// another destroy after an object's latest destroy task failed for a
// transient reason. Each failed task has already had the runner's own
// MaxAttempts with its own backoff, so a fresh task every orchestrator pass
// (every two seconds) was pure churn: the base is long enough for the
// motivating transient -- a network reporting "currently in use" while its
// instances detach -- to clear, and the interval doubles per failed task so a
// destroy that keeps failing for a reason nobody has classified as permanent
// still settles down to one attempt every ten minutes rather than hammering
// the builder forever.
const (
	teardownRequeueBase = 30 * time.Second
	teardownRequeueMax  = 10 * time.Minute
)

// teardownRequeueBackoff is the wait before re-requesting a destroy for an
// object whose destroy tasks have failed failedCount times: exponential from
// teardownRequeueBase, capped at teardownRequeueMax.
func teardownRequeueBackoff(failedCount int) time.Duration {
	d := teardownRequeueBase
	for i := 1; i < failedCount && d < teardownRequeueMax; i++ {
		d *= 2
	}
	if d > teardownRequeueMax {
		d = teardownRequeueMax
	}
	return d
}

// Teardown requests a real, terminal destroy for every deployed_object
// belonging to buildID -- unlike Reconcile, it never loads content at
// all, since a build has to be destroyable even when its content no
// longer is. Idempotent and safe to call repeatedly (the same
// `CreateTaskIfNoneOpen` partial-unique-index guard Reconcile's own
// requestDestroy already relies on), and safe to call concurrently from
// more than one orchestrator replica for the same reason every other
// function in this package is. Transitions buildID itself to 'torn_down'
// exactly once every object has genuinely reached the real terminal
// 'destroyed' state (MarkDeployedObjectDestroyed, via
// internal/runner.executeDestroy's own `terminal` payload flag) -- not
// merely once every destroy task has been created, so a caller polling
// the build's own status sees 'torn_down' only once it's actually true.
//
// A destroy that fails is handled by what its failure was:
//
//   - transient (the runner exhausted its attempts on an ordinary error):
//     the object self-heals -- a fresh destroy task is requested once a
//     backoff (teardownRequeueBackoff, scaled by how many destroy tasks the
//     object has already failed) has elapsed since the last failure. This is
//     the "network currently in use while its instances were still
//     detaching" case: the instances are gone a pass or two later and the
//     retry succeeds.
//   - permanent (the runner failed the task on builder.ErrPermanentConfig,
//     recorded as task.permanent): retrying can never succeed, so the
//     object is moved to the terminal 'destroy_failed' state, an event is
//     logged on it, and the build's reconcile_error says what is stuck and
//     why. No further destroy task is ever created for it; an operator fixes
//     the builder configuration and calls RetryTeardown, or ForceTornDown.
//     Before this distinction a permanent failure re-queued forever: one
//     failed destroy task per object per two-second pass, for days.
func Teardown(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) error {
	q := db.New(pool)

	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing deployed objects: %w", err)
	}
	latestRows, err := q.ListLatestDestroyTasksByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing latest destroy tasks: %w", err)
	}
	latest := make(map[pgtype.UUID]db.ListLatestDestroyTasksByBuildRow, len(latestRows))
	for _, row := range latestRows {
		latest[row.DeployedObjectID] = row
	}

	allDestroyed := true
	var stuck []string
	for _, obj := range objs {
		switch obj.Status {
		case "destroyed":
			continue
		case "destroy_failed":
			// Terminal until an operator retries: never re-queued here.
			allDestroyed = false
			stuck = append(stuck, describeStuckObject(obj, db.StrOrEmpty(obj.LastError)))
			continue
		}
		allDestroyed = false

		last, seen := latest[obj.ID]
		if seen {
			switch last.Status {
			case "pending", "leased":
				// A destroy is genuinely in flight; CreateTaskIfNoneOpen would
				// be a no-op anyway.
				continue
			case "failed":
				errText := db.StrOrEmpty(last.LastError)
				if last.Permanent {
					if _, err := q.MarkDeployedObjectDestroyFailed(ctx, db.MarkDeployedObjectDestroyFailedParams{
						ID: obj.ID, LastError: db.StrPtr(errText),
					}); err != nil {
						return fmt.Errorf("marking %s %q destroy_failed: %w", obj.Kind, obj.ObjectName, err)
					}
					payload, _ := json.Marshal(map[string]any{
						"object": obj.ObjectName, "kind": obj.Kind, "error": errText, "failed_destroy_tasks": last.FailedCount,
					})
					q.CreateObjectEvent(ctx, db.CreateObjectEventParams{
						BuildID: buildID, DeployedObjectID: obj.ID, Kind: "object.destroy_failed",
						Message: fmt.Sprintf("teardown stopped for %s %s: %s (permanent builder error; fix the builder configuration and retry the teardown)", obj.Kind, obj.ObjectName, errText),
						Payload: payload,
					})
					stuck = append(stuck, describeStuckObject(obj, errText))
					continue
				}
				if wait := teardownRequeueBackoff(int(last.FailedCount)); time.Since(last.UpdatedAt.Time) < wait {
					continue
				}
			}
		}

		created, err := requestTerminalDestroy(ctx, q, buildID, obj)
		if err != nil {
			return err
		}
		if created && seen && last.Status == "failed" {
			q.CreateObjectEvent(ctx, db.CreateObjectEventParams{
				BuildID: buildID, DeployedObjectID: obj.ID, Kind: "teardown.retry",
				Message: fmt.Sprintf("re-requested destroy of %s %s after %d failed destroy task(s): %s", obj.Kind, obj.ObjectName, last.FailedCount, db.StrOrEmpty(last.LastError)),
				Payload: []byte("{}"),
			})
		}
	}

	if len(stuck) > 0 {
		// Surface on the build itself (the same field the UI already shows as
		// a banner), rewritten only when the text actually changes so a
		// stuck teardown doesn't touch the build row every pass.
		msg := teardownStuckMessage(stuck)
		build, err := q.GetBuild(ctx, buildID)
		if err != nil {
			return fmt.Errorf("loading build: %w", err)
		}
		if db.StrOrEmpty(build.ReconcileError) != msg {
			if err := q.SetBuildReconcileError(ctx, db.SetBuildReconcileErrorParams{ID: buildID, ReconcileError: db.StrPtr(msg)}); err != nil {
				return fmt.Errorf("recording stuck teardown: %w", err)
			}
		}
	}

	if allDestroyed {
		return finishTeardown(ctx, q, buildID)
	}
	return nil
}

// requestTerminalDestroy enqueues the terminal destroy task for obj unless one
// is already open. Returns whether a task was actually created.
func requestTerminalDestroy(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject) (bool, error) {
	payload, err := json.Marshal(map[string]bool{"terminal": true})
	if err != nil {
		return false, err
	}
	if _, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("requesting destroy for %s %q: %w", obj.Kind, obj.ObjectName, err)
	}
	return true, nil
}

// finishTeardown is the single place a build becomes 'torn_down': status,
// the stuck-teardown banner cleared, and the build's scheduled tasks canceled.
func finishTeardown(ctx context.Context, q *db.Queries, buildID pgtype.UUID) error {
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: buildID, Status: "torn_down"}); err != nil {
		return fmt.Errorf("marking build torn down: %w", err)
	}
	if err := q.SetBuildReconcileError(ctx, db.SetBuildReconcileErrorParams{ID: buildID, ReconcileError: nil}); err != nil {
		return fmt.Errorf("clearing reconcile error: %w", err)
	}
	// Stop this build's `schedule:` rows from coming due forever now
	// that it's gone -- otherwise they stay pending and overdue, and
	// the dispatch loop keeps resolving the torn-down build's content
	// (and minting a GitHub token) every pass trying to fire them.
	if _, err := q.CancelScheduledTasksForBuild(ctx, buildID); err != nil {
		return fmt.Errorf("canceling scheduled tasks for torn-down build: %w", err)
	}
	return nil
}

func describeStuckObject(obj db.DeployedObject, errText string) string {
	return fmt.Sprintf("%s %s: %s", obj.Kind, obj.ObjectName, errText)
}

// teardownStuckMessage is the build-level summary of a teardown that cannot
// finish on its own: what to do, then the first few stuck objects with their
// builder errors.
func teardownStuckMessage(stuck []string) string {
	const show = 3
	var b strings.Builder
	fmt.Fprintf(&b, "Teardown is stuck: %d object(s) could not be destroyed because of a permanent builder error. Fix the builder configuration and retry the teardown, or force the build to torn down. ", len(stuck))
	n := len(stuck)
	if n > show {
		n = show
	}
	b.WriteString(strings.Join(stuck[:n], "; "))
	if len(stuck) > show {
		fmt.Fprintf(&b, " (and %d more)", len(stuck)-show)
	}
	return b.String()
}

// RetryTeardown is the operator's explicit "try again" for a tearing_down
// build after fixing whatever the builder refused: every destroy_failed
// object goes back to 'destroying', the stuck banner is cleared, and a fresh
// terminal destroy is requested right away for every object not yet destroyed
// (bypassing the transient backoff too -- the operator asked now). Returns how
// many destroy tasks were created. If the builder still refuses, the next
// Teardown pass marks the objects destroy_failed again, with the new error.
func RetryTeardown(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) (int, error) {
	q := db.New(pool)
	if _, err := q.ResetDestroyFailedObjectsForBuild(ctx, buildID); err != nil {
		return 0, fmt.Errorf("resetting destroy_failed objects: %w", err)
	}
	if err := q.SetBuildReconcileError(ctx, db.SetBuildReconcileErrorParams{ID: buildID, ReconcileError: nil}); err != nil {
		return 0, fmt.Errorf("clearing reconcile error: %w", err)
	}
	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return 0, fmt.Errorf("listing deployed objects: %w", err)
	}
	created := 0
	for _, obj := range objs {
		if obj.Status == "destroyed" {
			continue
		}
		ok, err := requestTerminalDestroy(ctx, q, buildID, obj)
		if err != nil {
			return created, err
		}
		if ok {
			created++
		}
	}
	return created, nil
}

// ForceTornDown is the operator's escape hatch for a teardown that cannot
// finish: it declares the build torn down WITHOUT any hoster call. Every open
// task of the build is closed (failed, with reason), every object not yet
// destroyed is recorded as destroyed, and the build reaches 'torn_down' the
// same way a real teardown does (finishTeardown). The caller is asserting the
// real infrastructure is already gone or will be removed by hand -- on a live
// build this orphans real instances, which is why the API gates it to instance
// admins. Unlike the admin force-delete, the build row and its history stay.
// Returns how many objects were forced.
func ForceTornDown(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, reason string) (int64, error) {
	q := db.New(pool)
	if _, err := q.CancelOpenTasksForBuild(ctx, db.CancelOpenTasksForBuildParams{BuildID: buildID, LastError: db.StrPtr(reason)}); err != nil {
		return 0, fmt.Errorf("closing open tasks: %w", err)
	}
	forced, err := q.ForceDestroyObjectsForBuild(ctx, buildID)
	if err != nil {
		return 0, fmt.Errorf("marking objects destroyed: %w", err)
	}
	if err := finishTeardown(ctx, q, buildID); err != nil {
		return forced, err
	}
	return forced, nil
}
