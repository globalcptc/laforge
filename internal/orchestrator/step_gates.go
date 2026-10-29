package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// Step-level gates: a step's `wait_for:` holds that step, and every step after
// it, until the same team's copy of another host reaches a milestone, a script
// step, or the end of its steps. Steps before the gate run straight away.
//
// The gate lives on the waiting host's agent_task rows: the first command of a
// gated step carries the step's wait_for entries, and from that row on the
// steps are queued 'blocked'. Each reconcile pass releases the blocked rows
// below the first gate that is still unsatisfied. The lease query only ever
// hands out the lowest unfinished authored step, so one blocked row holds
// everything after it with no further bookkeeping.

// gateState is where one wait_for entry stands.
type gateState int

const (
	gateWaiting gateState = iota
	gateSatisfied
	gateFailed
)

// pointProgress is one copy's progress on one milestone/script point.
type pointProgress struct{ total, reached, failed int64 }

// teamGates is a team's readiness, as one reconcile pass sees it, for every
// point a wait_for can name. Built once per team per pass, and only when the
// content uses wait_for at all.
type teamGates struct {
	placed   map[string]bool
	finished map[string]bool // every copy finished all its steps
	failed   map[string]bool // some copy failed terminally
	// copies lists each host/container's live copies in the team (deployed
	// object IDs) -- every one must reach a point for it to count.
	copies map[string][]string
	// milestone[object][name][copyID] / script[object][name][copyID].
	milestone map[string]map[string]map[string]*pointProgress
	script    map[string]map[string]map[string]*pointProgress
}

// contentUsesWaitFor reports whether any host/container step has a wait_for,
// so a pass over content without one skips the per-team gate query entirely.
func contentUsesWaitFor(c *loader.Content) bool {
	for _, h := range c.Hosts {
		for _, st := range h.Steps {
			if _, ok := st["wait_for"]; ok {
				return true
			}
		}
	}
	for _, ct := range c.Containers {
		for _, st := range ct.Steps {
			if _, ok := st["wait_for"]; ok {
				return true
			}
		}
	}
	return false
}

// loadTeamGates reads a team's per-point step progress. finished/failed are
// the same per-object readiness depends_on uses (teamDepReadiness).
func loadTeamGates(ctx context.Context, q *db.Queries, teamID pgtype.UUID, placed, finished, failed map[string]bool) (*teamGates, error) {
	objs, err := q.ListDeployedObjectsByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("listing team objects: %w", err)
	}
	g := &teamGates{
		placed: placed, finished: finished, failed: failed,
		copies:    map[string][]string{},
		milestone: map[string]map[string]map[string]*pointProgress{},
		script:    map[string]map[string]map[string]*pointProgress{},
	}
	for _, o := range objs {
		if (o.Kind == "host" || o.Kind == "container") && o.Status != "destroyed" {
			g.copies[o.ObjectName] = append(g.copies[o.ObjectName], o.ID.String())
		}
	}
	rows, err := q.ListStepGateProgressByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("reading step gate progress: %w", err)
	}
	add := func(m map[string]map[string]map[string]*pointProgress, object, name, copyID string, r db.ListStepGateProgressByTeamRow) {
		if m[object] == nil {
			m[object] = map[string]map[string]*pointProgress{}
		}
		if m[object][name] == nil {
			m[object][name] = map[string]*pointProgress{}
		}
		p := m[object][name][copyID]
		if p == nil {
			p = &pointProgress{}
			m[object][name][copyID] = p
		}
		p.total += r.Total
		p.reached += r.Reached
		p.failed += r.Failed
	}
	for _, r := range rows {
		id := r.DeployedObjectID.String()
		if r.Milestone != nil {
			add(g.milestone, r.ObjectName, *r.Milestone, id, r)
		}
		if r.StepScript != nil {
			add(g.script, r.ObjectName, *r.StepScript, id, r)
		}
	}
	return g, nil
}

// evaluate decides one wait_for entry. A target this environment doesn't place
// can't be waited on, so it counts as satisfied (the same as depends_on); the
// materializer leaves a note when that happens.
func (g *teamGates) evaluate(w loader.WaitFor) gateState {
	if !g.placed[w.Host] {
		return gateSatisfied
	}
	var points map[string]map[string]*pointProgress
	var name string
	switch {
	case w.Milestone != "":
		points, name = g.milestone[w.Host], w.Milestone
	case w.Step != "":
		points, name = g.script[w.Host], w.Step
	default:
		// Whole-host wait: the same readiness host-level depends_on uses.
		if g.finished[w.Host] {
			return gateSatisfied
		}
		if g.failed[w.Host] {
			return gateFailed
		}
		return gateWaiting
	}
	byCopy := points[name]
	copies := g.copies[w.Host]
	reachedAll := len(copies) > 0
	for _, id := range copies {
		p := byCopy[id]
		if p != nil && p.failed > 0 {
			return gateFailed
		}
		if p == nil || p.total == 0 || p.reached < p.total {
			reachedAll = false
		}
	}
	if reachedAll {
		return gateSatisfied
	}
	// Not reached, and the target has failed for good (a later or earlier step,
	// or its deploy): it will never get there.
	if g.failed[w.Host] {
		return gateFailed
	}
	return gateWaiting
}

// releaseStepGates moves an already-materialized object's blocked steps forward
// as far as its gates allow, and records what it is waiting on for the UI.
//
//   - hostBlockedOn non-empty: host-level depends_on still holds every step;
//     nothing to do here (materialization already recorded blocked_on).
//   - A gate whose target failed: the object is failed with a reason naming it.
//   - Otherwise blocked rows below the first unsatisfied gate are released.
//     blocked_on names that gate's unsatisfied targets only once the host has
//     worked up to it -- before that it is running steps, not blocked -- and is
//     cleared when nothing is held any more.
func releaseStepGates(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, hostBlockedOn []string, gates *teamGates) error {
	if len(hostBlockedOn) > 0 {
		return nil
	}
	rows, err := q.ListBlockedStepGatesForObject(ctx, obj.ID)
	if err != nil {
		return fmt.Errorf("listing step gates: %w", err)
	}
	limit := int32(-1)
	var waitingOn []string
	for _, r := range rows {
		var entries []loader.WaitFor
		if err := json.Unmarshal(r.WaitFor, &entries); err != nil {
			return fmt.Errorf("decoding wait_for on step %d: %w", r.StepIndex, err)
		}
		for _, w := range entries {
			state := gateWaiting // no team readiness loaded: hold, never guess
			if gates != nil {
				state = gates.evaluate(w)
			}
			switch state {
			case gateFailed:
				msg := fmt.Sprintf("waiting on %s, which failed", w.Label())
				if err := q.SetDeployedObjectBuildFailed(ctx, db.SetDeployedObjectBuildFailedParams{
					ID:        obj.ID,
					LastError: db.StrPtr(msg),
				}); err != nil {
					return err
				}
				// Record WHY on the timeline -- without this the host just flips
				// back to build_failed with no log line, which looks like a restart
				// that "didn't take": the gate target it waits on is itself failed,
				// so restarting this host alone can never succeed (restart the
				// target too). blocked_on is cleared so the UI shows the failure,
				// not a stale "waiting on <already-fired gate>".
				_ = setBlockedOn(ctx, q, obj, nil)
				decide(ctx, q, buildID, obj.ID, "lifecycle.build_failed",
					fmt.Sprintf("%s: %s; restart that host too, then this one resumes", objectLabel(obj, 0), msg),
					map[string]any{"gate": w.Label()})
				return nil
			case gateWaiting:
				waitingOn = append(waitingOn, w.Label())
			}
		}
		if len(waitingOn) > 0 {
			limit = r.StepIndex
			break
		}
	}

	if limit < 0 {
		// Every gate is satisfied (or there were none): release everything.
		if err := q.UnblockAgentTasksForObject(ctx, obj.ID); err != nil {
			return fmt.Errorf("releasing blocked steps: %w", err)
		}
		return setBlockedOn(ctx, q, obj, nil)
	}
	if err := q.UnblockAgentTasksBelowIndex(ctx, db.UnblockAgentTasksBelowIndexParams{DeployedObjectID: obj.ID, Below: limit}); err != nil {
		return fmt.Errorf("releasing steps before the gate: %w", err)
	}
	open, err := q.CountOpenAuthoredStepsBelowIndex(ctx, db.CountOpenAuthoredStepsBelowIndexParams{DeployedObjectID: obj.ID, Below: limit})
	if err != nil {
		return fmt.Errorf("checking progress toward the gate: %w", err)
	}
	if open > 0 {
		return setBlockedOn(ctx, q, obj, nil) // still working toward the gate
	}
	return setBlockedOn(ctx, q, obj, waitingOn)
}

// setBlockedOn records what the object is waiting on, writing only on change
// so an every-pass call doesn't churn the row.
func setBlockedOn(ctx context.Context, q *db.Queries, obj db.DeployedObject, on []string) error {
	var want []byte
	if len(on) > 0 {
		want, _ = json.Marshal(on)
	}
	var have []string
	if len(obj.BlockedOn) > 0 {
		_ = json.Unmarshal(obj.BlockedOn, &have)
	}
	if strings.Join(have, "\x00") == strings.Join(on, "\x00") && (len(have) > 0) == (len(on) > 0) {
		return nil
	}
	return q.SetDeployedObjectBlockedOn(ctx, db.SetDeployedObjectBlockedOnParams{ID: obj.ID, BlockedOn: want})
}
