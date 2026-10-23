package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/gateway"
	"github.com/globalcptc/laforge/internal/loader"
)

// Restart from a step (docs/plans/restart-from-step.md): a host whose build
// stopped -- a step failed, a validator didn't pass -- runs the chosen
// authored step and everything after it again, in full, on the box it
// already has. Nothing is destroyed and the hoster isn't called: the host's
// authored agent_task rows from that step on are deleted and queued again
// from the build's current content, the host goes back to building, and the
// agent picks up the first re-queued step on its next poll. The lifecycle
// poll then finishes or fails the host as it always does.
//
// Picked once for a host definition and applied to its instances across the
// build's teams: every target must share one object_name, so "step 3" means
// the same thing on each.

// RestartOptions is what the operator chose.
type RestartOptions struct {
	// FromGroup is the authored step (index into the host's `steps:`) to
	// restart from; it and every later step run again.
	FromGroup int
	// ResumeDependents puts hosts that were failed for depending on a
	// restarted one ("dependency X failed to configure", before their steps
	// ever ran) back to running, so their steps queue once it finishes.
	ResumeDependents bool
	// Force restarts a host even while its agent is mid-step (the step's row
	// is deleted; the gateway drops the late result).
	Force bool
	// DryRun reports what would happen and changes nothing.
	DryRun bool
}

// RestartStep is one authored step that will run again.
type RestartStep struct {
	Group    int    `json:"group"`
	Label    string `json:"label,omitempty"`
	Commands int    `json:"commands"`
}

// RestartHostResult is one targeted host: what it is, whether it was (or
// would be) restarted, and why not.
type RestartHostResult struct {
	ID         string `json:"id"`
	ObjectName string `json:"object_name"`
	AsName     string `json:"as_name"`
	TeamNumber int32  `json:"team_number"`
	Status     string `json:"status"`
	// Eligible: in a state a restart applies to (build_failed, invalid, or
	// finished with steps changed under it).
	Eligible bool `json:"eligible"`
	// Running: an authored step is leased to the agent right now.
	Running bool `json:"running"`
	// ContentChanged: the build's content moved on since this host was
	// built, and it was held (steps_changed_at).
	ContentChanged bool `json:"content_changed"`
	// Steps that run again, from FromGroup on, as the current content expands
	// them.
	Steps []RestartStep `json:"steps"`
	// Dependents that resume (would resume) with this host.
	Dependents []RebuildAffected `json:"dependents"`
	// Restarted is true once the host's steps were re-queued; otherwise
	// Skipped says why not.
	Restarted bool   `json:"restarted"`
	Skipped   string `json:"skipped,omitempty"`
}

// ErrRestartRefused is a request that can't be honoured as asked: no
// eligible target, a step that doesn't exist, targets of different hosts.
var ErrRestartRefused = errors.New("restart refused")

// RestartFromStep applies opts to the targets in build (content is the
// build's current revision at repoRoot). With DryRun, every target is
// described and nothing changes; otherwise each eligible host is restarted
// in its own transaction and the result says what happened to each.
func RestartFromStep(ctx context.Context, pool *pgxpool.Pool, build db.Build, content *loader.Content, repoRoot string, target AdHocTarget, opts RestartOptions) ([]RestartHostResult, error) {
	q := db.New(pool)
	matched, err := MatchAdHocTargets(ctx, q, build.ID, target)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no hosts match this target", ErrRestartRefused)
	}
	names := map[string]bool{}
	for _, o := range matched {
		names[o.ObjectName] = true
	}
	if len(names) > 1 {
		list := make([]string, 0, len(names))
		for n := range names {
			list = append(list, n)
		}
		sort.Strings(list)
		return nil, fmt.Errorf("%w: a restart is for one host definition at a time (its steps are numbered the same on every team); these are %s", ErrRestartRefused, strings.Join(list, ", "))
	}
	if opts.FromGroup < 0 {
		return nil, fmt.Errorf("%w: from_group must be 0 or more", ErrRestartRefused)
	}

	env := findEnvironment(content, build.EnvironmentName)
	if env == nil {
		return nil, fmt.Errorf("environment %q not found in content", build.EnvironmentName)
	}
	placed := placedNames(env.Networks)
	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		return nil, err
	}
	teamByID := make(map[string]db.Team, len(teams))
	for _, tm := range teams {
		teamByID[tm.ID.String()] = tm
	}

	results := make([]RestartHostResult, 0, len(matched))
	anyEligible := false
	for _, o := range matched {
		tm := teamByID[o.TeamID.String()]
		res := RestartHostResult{ID: o.ID.String(), ObjectName: o.ObjectName, AsName: db.StrOrEmpty(o.AsName), TeamNumber: tm.TeamNumber,
			Status: o.Status, Eligible: restartable(o), ContentChanged: o.StepsChangedAt.Valid, Steps: []RestartStep{}, Dependents: []RebuildAffected{}}
		running, err := q.CountRunningAuthoredTasks(ctx, o.ID)
		if err != nil {
			return nil, err
		}
		res.Running = running > 0

		// What runs again: the current content's expansion of this host's
		// steps, from the chosen one on.
		steps, err := plannedRestartSteps(ctx, q, repoRoot, content, build, o, tm, opts.FromGroup)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", res.AsName, err)
		}
		res.Steps = steps
		if len(steps) == 0 {
			res.Eligible = false
			res.Skipped = fmt.Sprintf("step %d is past this host's last step", opts.FromGroup)
		} else if groups, err := q.ListAuthoredStepGroups(ctx, o.ID); err != nil {
			return nil, err
		} else if len(groups) == 0 && !o.StepsMaterializedAt.Valid {
			res.Eligible = false
			res.Skipped = "its steps haven't been queued yet"
		} else if len(groups) == 0 {
			// Materialized, but every authored row lacks a step_group: queued
			// before this feature existed, so there's nothing to line the
			// restart up against.
			res.Eligible = false
			res.Skipped = "its steps were queued before restarting from a step existed; rebuild the host instead"
		} else if !res.Eligible {
			res.Skipped = "not in a state a restart applies to (" + o.Status + ")"
		} else if res.Running && !opts.Force {
			res.Skipped = "a step is running on it right now; wait for it, or force the restart"
		}
		if opts.ResumeDependents {
			deps, err := failedDependents(ctx, q, content, o, tm)
			if err != nil {
				return nil, err
			}
			res.Dependents = deps
		}
		if res.Eligible && res.Skipped == "" {
			anyEligible = true
		}
		results = append(results, res)
	}
	if !anyEligible && !opts.DryRun {
		return nil, fmt.Errorf("%w: %s", ErrRestartRefused, firstSkipReason(results))
	}
	if opts.DryRun {
		return results, nil
	}

	for i := range results {
		res := &results[i]
		if !res.Eligible || res.Skipped != "" {
			continue
		}
		var obj db.DeployedObject
		for _, o := range matched {
			if o.ID.String() == res.ID {
				obj = o
			}
		}
		tm := teamByID[obj.TeamID.String()]
		err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			return restartOne(ctx, db.New(tx), repoRoot, content, env, placed, build, obj, tm, opts, res)
		})
		if err != nil {
			return nil, fmt.Errorf("restarting %s (team %d): %w", res.AsName, res.TeamNumber, err)
		}
	}
	// The lifecycle poll and step materialization run for builds that are
	// deploying or building; a finished or failed build with a host building
	// again is building again.
	if build.Status == "finished" || build.Status == "failed" {
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "building"}); err != nil {
			return nil, fmt.Errorf("re-entering the build phase: %w", err)
		}
	}
	return results, nil
}

// restartable: the states a restart applies to.
func restartable(o db.DeployedObject) bool {
	switch o.Status {
	case "build_failed", "invalid":
		return true
	case "finished":
		return o.StepsChangedAt.Valid
	}
	return false
}

func firstSkipReason(results []RestartHostResult) string {
	for _, r := range results {
		if r.Skipped != "" {
			return r.Skipped
		}
	}
	return "no host can be restarted"
}

// plannedRestartSteps expands the host's steps from the current content and
// groups the commands from fromGroup on by authored step.
func plannedRestartSteps(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, build db.Build, o db.DeployedObject, tm db.Team, fromGroup int) ([]RestartStep, error) {
	if o.AsName == nil {
		return nil, nil
	}
	secret, err := db.RenderSecretForBuild(ctx, q, build.ID)
	if err != nil {
		return nil, fmt.Errorf("loading the build's render secret: %w", err)
	}
	cmds, _, err := gateway.ExpandSteps(repoRoot, c, build.EnvironmentName, *o.AsName, int(tm.TeamNumber),
		gateway.WithPublicAddress(o.PublicAddress), gateway.WithRenderSecret(secret))
	if err != nil {
		return nil, fmt.Errorf("expanding steps: %w", err)
	}
	var out []RestartStep
	for _, cmd := range cmds {
		if cmd.Group < fromGroup {
			continue
		}
		if n := len(out); n > 0 && out[n-1].Group == cmd.Group {
			out[n-1].Commands++
			continue
		}
		out = append(out, RestartStep{Group: cmd.Group, Label: cmd.GroupLabel, Commands: 1})
	}
	return out, nil
}

// failedDependents are the objects in o's team that depend (transitively) on
// o's host and were failed before their steps ran -- the ones a restart
// resumes.
func failedDependents(ctx context.Context, q *db.Queries, c *loader.Content, o db.DeployedObject, tm db.Team) ([]RebuildAffected, error) {
	names := map[string]bool{}
	for _, n := range c.Dependents([]string{o.ObjectName}) {
		if n != o.ObjectName {
			names[n] = true
		}
	}
	out := []RebuildAffected{}
	if len(names) == 0 {
		return out, nil
	}
	objs, err := q.ListDeployedObjectsByTeam(ctx, tm.ID)
	if err != nil {
		return nil, err
	}
	for _, d := range objs {
		if names[d.ObjectName] && d.Status == "build_failed" && !d.StepsMaterializedAt.Valid {
			out = append(out, RebuildAffected{ID: d.ID.String(), ObjectName: d.ObjectName, AsName: db.StrOrEmpty(d.AsName), Kind: d.Kind, TeamNumber: tm.TeamNumber})
		}
	}
	return out, nil
}

// restartOne is the per-host transaction: drop the steps from the chosen one
// on, queue them again from the current content, reopen the host, resume its
// failed dependents, and say so in the build's log.
func restartOne(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, env *loader.Environment, placed map[string]bool, build db.Build, obj db.DeployedObject, tm db.Team, opts RestartOptions, res *RestartHostResult) error {
	if !opts.Force {
		running, err := q.CountRunningAuthoredTasks(ctx, obj.ID)
		if err != nil {
			return err
		}
		if running > 0 {
			res.Skipped = "a step started running on it meanwhile; wait for it, or force the restart"
			return nil
		}
	}
	if _, err := q.DeleteAuthoredAgentTasksFromGroup(ctx, db.DeleteAuthoredAgentTasksFromGroupParams{DeployedObjectID: obj.ID, StepGroup: int32Ptr(opts.FromGroup)}); err != nil {
		return fmt.Errorf("dropping the steps from %d on: %w", opts.FromGroup, err)
	}
	// The host ran past its dependencies to get here, so its new rows start
	// pending; a wait_for among them starts blocked, and the team's gate state
	// decides what's already satisfied.
	finished, failed, _, _, err := teamDepReadiness(ctx, q, tm.ID)
	if err != nil {
		return err
	}
	var gates *teamGates
	if contentUsesWaitFor(c) {
		if gates, err = loadTeamGates(ctx, q, tm.ID, placed, finished, failed); err != nil {
			return err
		}
	}
	queued, notes, err := queueSteps(ctx, q, repoRoot, c, env.Name, build.ID, obj, int(tm.TeamNumber), res.AsName, "pending", gates, opts.FromGroup)
	if err != nil {
		return err
	}
	if err := releaseStepGates(ctx, q, obj, nil, gates); err != nil {
		return err
	}
	if _, err := q.SetDeployedObjectRestarting(ctx, obj.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			res.Skipped = "its state changed meanwhile (" + obj.Status + " when checked)"
			return errors.New("state changed; rolled back")
		}
		return fmt.Errorf("reopening the host: %w", err)
	}
	resumed := res.Dependents[:0]
	for _, d := range res.Dependents {
		id, err := parseUUIDString(d.ID)
		if err != nil {
			return err
		}
		n, err := q.ResumeDependentForRestart(ctx, id)
		if err != nil {
			return fmt.Errorf("resuming %s: %w", d.AsName, err)
		}
		if n > 0 {
			resumed = append(resumed, d)
		}
	}
	res.Dependents = resumed
	res.Restarted = true
	for _, n := range notes {
		q.CreateEvent(ctx, db.CreateEventParams{BuildID: build.ID, Kind: "steps.note", Message: n, Payload: []byte("{}")})
	}
	label := fmt.Sprintf("step %d", opts.FromGroup)
	if len(res.Steps) > 0 && res.Steps[0].Label != "" {
		label = fmt.Sprintf("%q (step %d)", res.Steps[0].Label, opts.FromGroup)
	}
	msg := fmt.Sprintf("%s (team %d): steps restarted from %s -- %d step(s), %d command(s) run again on the box as it is", res.AsName, tm.TeamNumber, label, len(res.Steps), len(queued))
	if res.ContentChanged {
		msg += "; the content had changed since it was built, so they come from the current revision"
	}
	if len(resumed) > 0 {
		names := make([]string, 0, len(resumed))
		for _, d := range resumed {
			names = append(names, d.AsName)
		}
		msg += fmt.Sprintf("; %s resume once it finishes", strings.Join(names, ", "))
	}
	q.CreateEvent(ctx, db.CreateEventParams{BuildID: build.ID, Kind: "steps.restarted", Message: msg, Payload: []byte("{}")})
	return nil
}

func int32Ptr(v int) *int32 {
	n := int32(v)
	return &n
}

func parseUUIDString(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return id, fmt.Errorf("bad id %q: %w", s, err)
	}
	return id, nil
}
