// Real execution for `schedule:` entries and ad-hoc scheduled tasks,
// meeting the "Real
// execution, dispatched to real agents, not a stub" requirement. This
// dispatches through the exact same CreateAgentTask/agent_task machinery
// internal/api/tasks.go's handleCreateAdHocTask already uses for
// immediate ad-hoc dispatch: a live agent polling get-task actually
// picks these up and runs them. No new agent-side protocol, no new
// gateway code -- only what decides WHEN to call that existing path.
package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/gateway"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
	"github.com/globalcptc/laforge/internal/schedule"
)

// ContentResolver answers "where's buildID's content on disk" --
// letting DispatchDueScheduledTasks resolve a different checkout per
// build within the same pass, instead of being forced through one fixed
// path for every build regardless of which repository it actually
// belongs to. This is the one place in internal/orchestrator that
// already loops over more than one build in a single call (every other
// entry point -- Reconcile, DiffUpcoming, ResolveAnchors -- handles
// exactly one build per call already, so its caller can resolve the
// right path once, before calling it, with no signature change needed
// there at all). A plain func
// type, not an interface: every real implementation is either
// FixedRepoRoot (no checkout cache configured) or a *checkout.Cache's
// own ForBuild method, which already has exactly this signature.
type ContentResolver func(ctx context.Context, buildID pgtype.UUID) (string, error)

// FixedRepoRoot is the ContentResolver every caller effectively used
// before this type existed: the same path, regardless of which build is
// asking. Correct as long as only one repository is ever in play in
// this process -- the same limitation as before, now named rather than
// implicit, and easy to swap for a real *checkout.Cache once more than
// one repository needs resolving concurrently.
func FixedRepoRoot(repoRoot string) ContentResolver {
	return func(ctx context.Context, buildID pgtype.UUID) (string, error) {
		return repoRoot, nil
	}
}

// DispatchDueScheduledTasks is cmd/laforge-orchestrator's own third
// ticker (alongside reconcileAll and cleanupHeartbeats): find every
// scheduled_task past its next_fire_at, dispatch it, and compute its
// next fire time (or mark it fired for good, for a fires-once entry).
// limit caps one pass so a burst of due rows can't starve the ticker's
// own interval; a full backlog drains over a few ticks instead of one
// long one.
func DispatchDueScheduledTasks(ctx context.Context, pool *pgxpool.Pool, resolve ContentResolver, limit int32) error {
	q := db.New(pool)
	due, err := q.ListDueScheduledTasks(ctx, limit)
	if err != nil {
		return fmt.Errorf("listing due scheduled tasks: %w", err)
	}
	if len(due) == 0 {
		return nil
	}

	byBuild := make(map[string]buildContext)
	for _, st := range due {
		buildKey := st.BuildID.String()
		bc, ok := byBuild[buildKey]
		if !ok {
			bc, err = loadBuildContext(ctx, q, resolve, st.BuildID)
			if err != nil {
				log.Printf("scheduled_task %s: loading build %s: %v", st.ID, buildKey, err)
				continue
			}
			byBuild[buildKey] = bc
		}
		if err := dispatchOneScheduledTask(ctx, q, st, bc); err != nil {
			log.Printf("scheduled_task %s: %v", st.ID, err)
		}
	}
	return nil
}

// buildContext is one build's own resolved checkout, content, and
// environment, loaded once per DispatchDueScheduledTasks pass and
// reused across every due row that belongs to it -- a content-sourced
// row's dispatch re-resolves the live schedule entry against this same
// load (see dispatchOneScheduledTask's "content" case), so a burst of
// due rows for one build doesn't reload and re-parse the same checkout
// (and, now, doesn't ask the checkout cache to resolve the same build
// more than once either) per row.
type buildContext struct {
	Build    db.Build
	RepoRoot string
	Content  *loader.Content
	Anchors  schedule.Anchors
}

func loadBuildContext(ctx context.Context, q *db.Queries, resolve ContentResolver, buildID pgtype.UUID) (buildContext, error) {
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return buildContext{}, fmt.Errorf("loading build: %w", err)
	}
	repoRoot, err := resolve(ctx, buildID)
	if err != nil {
		return buildContext{}, fmt.Errorf("resolving content location: %w", err)
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return buildContext{}, fmt.Errorf("loading content: %w", err)
	}
	env := findEnvironment(c, build.EnvironmentName)
	if env == nil {
		return buildContext{}, fmt.Errorf("environment %q not found in content at %s", build.EnvironmentName, repoRoot)
	}
	return buildContext{Build: build, RepoRoot: repoRoot, Content: c, Anchors: anchorsFromEnvironment(env)}, nil
}

// ResolveAnchors loads the real environment start/stop/access windows a
// build's schedule entries resolve against -- the same content
// Reconcile itself reads, so a schedule anchor and the build it fires
// against never disagree about what "competition start" means. Exported
// so internal/api can call it too, at schedule-creation time, to refuse
// an expression with no possible future occurrence rather than silently
// accepting a scheduled task that can never fire.
func ResolveAnchors(ctx context.Context, q *db.Queries, repoRoot string, buildID pgtype.UUID) (schedule.Anchors, error) {
	bc, err := loadBuildContext(ctx, q, FixedRepoRoot(repoRoot), buildID)
	if err != nil {
		return schedule.Anchors{}, err
	}
	return bc.Anchors, nil
}

// anchorsFromEnvironment is ResolveAnchors' own real conversion, factored
// out so Reconcile -- which already has env in scope, no extra DB round
// trip or content reload needed -- can build the identical schedule.Anchors
// itself when materializing a host/container's own schedule: entries at
// deploy time.
func anchorsFromEnvironment(env *loader.Environment) schedule.Anchors {
	var a schedule.Anchors
	if env.Start != "" {
		a.CompetitionStart, _ = time.Parse(time.RFC3339, env.Start)
	}
	if env.Stop != "" {
		a.CompetitionEnd, _ = time.Parse(time.RFC3339, env.Stop)
	}
	for _, w := range env.Access {
		open, errO := time.Parse(time.RFC3339, w.Open)
		close, errC := time.Parse(time.RFC3339, w.Close)
		if errO != nil || errC != nil {
			continue // caught by loader/schema validation already; skip rather than fail the whole pass
		}
		a.AccessWindows = append(a.AccessWindows, schedule.AccessWindow{Open: open, Close: close})
	}
	return a
}

// materializeSchedule turns a just-(re)deployed host/container's own
// `schedule:` entries into real scheduled_task rows -- the content side
// of "real execution, dispatched to real agents, not a stub" (the
// ad-hoc side already works this way; see MatchAdHocTargets/dispatchOneScheduledTask).
// Called exactly once per real deploy (reconcileCopy's own `created`
// gate), not every reconcile pass, and upserts (CreateContentScheduledTask's
// own ON CONFLICT) so a later redeploy refreshes a changed when:/action
// without duplicating the row.
//
// Deliberately does NOT resolve the entry's action into a real agent
// command here: Command/Payload are display-only for a content-sourced
// row (the "Command" column dispatchOneScheduledTask's own "content"
// case actually uses is computed fresh, from live content, at fire
// time -- see its own doc comment for why that's the real fix for a
// `script:`/`run:`/validate:-carrying entry needing more than one agent
// command, which used to be a named, logged skip here).
func materializeSchedule(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, entries []loader.Step, anchors schedule.Anchors) error {
	for idx, entry := range entries {
		when, _ := entry["when"].(string)
		expr, err := schedule.Parse(when)
		if err != nil {
			// Already caught by loader validation at commit time; reaching
			// here with a bad expression would mean deploying content that
			// never should have validated -- log and skip this one entry
			// rather than failing the whole deploy over it.
			log.Printf("deployed_object %s: schedule[%d]: %v", obj.ID, idx, err)
			continue
		}
		next, ok := expr.NextFireAfter(time.Now().UTC(), anchors)
		if !ok {
			log.Printf("deployed_object %s: schedule[%d] (%q): no future occurrence for this build, skipped", obj.ID, idx, when)
			continue
		}
		scheduleIndex := int32(idx)
		_, err = q.CreateContentScheduledTask(ctx, db.CreateContentScheduledTaskParams{
			BuildID: buildID, DeployedObjectID: obj.ID, ScheduleIndex: &scheduleIndex,
			WhenExpr: when, Command: entry.ActionKey(), Payload: []byte("{}"),
			Anchor: expr.AnchorName(), FiresOnce: expr.FiresOnce(),
			NextFireAt: pgtype.Timestamptz{Time: next, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			// ErrNoRows here means the query's own conditional upsert found
			// nothing needed updating (see its own doc comment) -- a real
			// "already correct" outcome, not a failure.
			return fmt.Errorf("schedule[%d]: %w", idx, err)
		}
	}
	return nil
}

// dispatchOneScheduledTask fires one due row. The two sources dispatch
// genuinely differently, not just a different target lookup: adhoc
// already carries its one whitelisted command (validated at creation
// time, see internal/api/scheduled_tasks.go), applied to every matched
// target; content re-resolves the live schedule entry and expands it
// (gateway.ExpandOneStep) against bc's already-loaded content, which is
// what lets a `script:`/`run:` entry, or one carrying its own
// `validate:`, dispatch as the real multi-command sequence it always
// was -- no longer the single-command-only limit an earlier version of
// this had.
func dispatchOneScheduledTask(ctx context.Context, q *db.Queries, st db.ScheduledTask, bc buildContext) error {
	created := 0
	switch st.Source {
	case "adhoc":
		var target AdHocTarget
		if err := json.Unmarshal(st.Target, &target); err != nil {
			return fmt.Errorf("decoding target: %w", err)
		}
		matched, err := MatchAdHocTargets(ctx, q, st.BuildID, target)
		if err != nil {
			return fmt.Errorf("resolving target: %w", err)
		}
		for _, obj := range matched {
			if dispatchCommand(ctx, q, obj.ID, st.Command, st.Payload) {
				created++
			}
		}

	case "content":
		if !st.DeployedObjectID.Valid || st.ScheduleIndex == nil {
			return fmt.Errorf("content-sourced row missing deployed_object_id/schedule_index")
		}
		obj, err := q.GetDeployedObject(ctx, st.DeployedObjectID)
		if err != nil {
			return fmt.Errorf("loading deployed object: %w", err)
		}
		team, err := q.GetTeam(ctx, obj.TeamID)
		if err != nil {
			return fmt.Errorf("loading team: %w", err)
		}
		rctx, err := render.Resolve(bc.Content, bc.Build.EnvironmentName, db.StrOrEmpty(obj.AsName), int(team.TeamNumber))
		if err != nil {
			return fmt.Errorf("resolving %s: %w", db.StrOrEmpty(obj.AsName), err)
		}
		idx := int(*st.ScheduleIndex)
		if idx < 0 || idx >= len(rctx.Schedule) {
			return fmt.Errorf("schedule index %d out of range -- content changed since this was materialized (%d entries now)", idx, len(rctx.Schedule))
		}
		cmds, notes, err := gateway.ExpandOneStep(bc.RepoRoot, bc.Content, rctx, idx, rctx.Schedule[idx])
		if err != nil {
			return fmt.Errorf("resolving schedule[%d]: %w", idx, err)
		}
		for _, n := range notes {
			log.Printf("scheduled_task %s: %s", st.ID, n)
		}
		for _, cmd := range cmds {
			payload, err := json.Marshal(cmd.Payload)
			if err != nil {
				log.Printf("scheduled_task %s: encoding payload: %v", st.ID, err)
				continue
			}
			if dispatchCommand(ctx, q, obj.ID, cmd.Command, payload) {
				created++
			}
		}

	default:
		return fmt.Errorf("unknown source %q", st.Source)
	}

	if _, err := q.CreateEvent(ctx, db.CreateEventParams{
		BuildID: st.BuildID,
		Kind:    "schedule.fired",
		Message: fmt.Sprintf("scheduled entry %q fired, %d agent command(s) enqueued", st.WhenExpr, created),
		Payload: []byte("{}"),
	}); err != nil {
		log.Printf("scheduled_task %s: recording event: %v", st.ID, err)
	}

	if st.FiresOnce {
		_, err := q.MarkScheduledTaskFired(ctx, st.ID)
		return err
	}

	expr, err := schedule.Parse(st.WhenExpr)
	if err != nil {
		// Already caught by loader/API validation at creation time; a
		// row reaching here with a bad expression is a real bug, not a
		// content mistake -- fire it once more, then stop rather than
		// loop forever re-attempting a parse that will never succeed.
		_, markErr := q.MarkScheduledTaskFired(ctx, st.ID)
		if markErr != nil {
			return fmt.Errorf("re-parsing when_expr %q: %w (and failed to mark fired: %v)", st.WhenExpr, err, markErr)
		}
		return fmt.Errorf("re-parsing when_expr %q: %w -- marked fired to stop retrying", st.WhenExpr, err)
	}
	next, ok := expr.NextFireAfter(time.Now().UTC(), bc.Anchors)
	if !ok {
		// No more occurrences (e.g. the last access window has passed) --
		// same terminal state as a fires-once row.
		_, err := q.MarkScheduledTaskFired(ctx, st.ID)
		return err
	}
	_, err = q.RescheduleScheduledTask(ctx, db.RescheduleScheduledTaskParams{
		ID: st.ID, NextFireAt: pgtype.Timestamptz{Time: next, Valid: true},
	})
	return err
}

// dispatchCommand appends one command to objID's own agent_task queue --
// the shared tail end of both dispatch paths above (a single ad-hoc
// command per matched target, or one of several commands a content
// schedule entry expanded to), logging rather than failing the whole
// dispatch pass over one target's error.
func dispatchCommand(ctx context.Context, q *db.Queries, objID pgtype.UUID, command string, payload []byte) bool {
	next, err := q.NextStepIndexForHost(ctx, objID)
	if err != nil {
		log.Printf("next step index for %s: %v", objID, err)
		return false
	}
	if _, err := q.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: objID, StepIndex: next, Command: command, Payload: payload,
	}); err != nil {
		log.Printf("creating agent task for %s: %v", objID, err)
		return false
	}
	return true
}
