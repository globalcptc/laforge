// Ad-hoc tasks: "a UI and API capability for acting on a live
// environment... exposes the agent command set... The matched set is
// shown before anything runs, with a count."
//
// Full targeting model (`object, tag, network, team,
// search`): team, kind (host/container), substring search over object/as
// name, network, and tag (key or key=value) -- see
// orchestrator.MatchAdHocTargets. Tag/network targeting is real now: a
// host/container's authored tags are persisted onto its runtime
// deployed_object row at reconcile (migration 00028), so the same matcher
// serves the immediate ad-hoc endpoint and the scheduled-task dispatch loop.
//
// Dispatch is real, not a stub: this inserts genuine agent_task rows via
// the exact same CreateAgentTask/agent_task machinery authored `steps:`
// use (materialized at deploy by internal/runner's materializeSteps) --
// a live agent polling get-task will actually pick these up and run
// them, appended after whatever step_index a host already has.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/schedule"
)

// missingIDs returns whichever of requested didn't end up in matched --
// used to turn "some ids weren't found in this build" into a named,
// actionable list rather than a bare count.
func missingIDs(requested []string, matched []db.DeployedObject) []string {
	found := make(map[string]bool, len(matched))
	for _, o := range matched {
		found[o.ID.String()] = true
	}
	var missing []string
	for _, id := range requested {
		if !found[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

// adHocTarget is orchestrator.AdHocTarget itself, not a copy -- the
// TargetPicker's real (scoped) filter shape, `object,
// tag, network, team, search` minus tag and network (see this file's
// package doc). Matching now lives in orchestrator.MatchAdHocTargets so
// the scheduled-task dispatch loop re-resolves a stored target the
// identical way at fire time, not a second, separately-maintained copy
// of the same rules.
type adHocTarget = orchestrator.AdHocTarget

type createAdHocTaskRequest struct {
	Target  adHocTarget     `json:"target"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
	// DryRun resolves and returns the matched set without touching
	// anything -- "the matched set is shown before anything runs."
	DryRun bool `json:"dry_run,omitempty"`
}

// allowedAdHocCommands is every agent command the Rust agent actually
// implements (agent/src/commands.rs), so an operator can run any real
// LaForge action ad-hoc or on a schedule, not just a hand-picked few. The
// three the protocol defines but the agent does not implement yet
// (download/upload/extract -- they need the artifact store) are left out,
// as is `validate` (a check block, not an action). Re-running an
// already-authored script by name is also still out: it needs the same
// content-resolution ExpandSteps does, a separate piece of work (see this
// file's package doc). This is a manage-level capability either way.
var allowedAdHocCommands = map[string]bool{
	agentproto.CmdExecute:     true,
	agentproto.CmdReboot:      true,
	agentproto.CmdService:     true,
	agentproto.CmdWriteFile:   true,
	agentproto.CmdAppendFile:  true,
	agentproto.CmdDelete:      true,
	agentproto.CmdChangePerms: true,
	agentproto.CmdCreateUser:  true,
	agentproto.CmdSetPassword: true,
	agentproto.CmdAddToGroup:  true,
}

type adHocResult struct {
	Matched int                 `json:"matched"`
	Objects []db.DeployedObject `json:"objects,omitempty"`
	Created []db.AgentTask      `json:"created,omitempty"`
}

func (s *Server) handleCreateAdHocTask(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	// "manage" level: "run agent commands
	// during an event" is explicitly a manage-level capability, above
	// build.
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	var req createAdHocTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !req.DryRun && !allowedAdHocCommands[req.Command] {
		writeError(w, http.StatusBadRequest, errors.New("unsupported ad-hoc command"))
		return
	}

	matched, err := orchestrator.MatchAdHocTargets(r.Context(), s.Queries, build.ID, req.Target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Every requested id must actually belong to this build, or this
	// fails loudly instead of silently dispatching to fewer objects than
	// asked -- the real risk this guards against isn't cross-build id
	// COLLISION (deployed_object.id is a real Postgres UUID, globally
	// unique by construction; ListDeployedObjectsByBuild above already
	// means an id from a different build can never appear in `all`, so
	// it can never silently match here either), it's a STALE id: a UI
	// that cached a selection from before a rebuild, a copy-pasted id
	// from the wrong build's tab, or any other case where the caller
	// believes an id is in this build and it isn't. Silently matching
	// zero of those ids (an earlier version of this code) looks
	// indistinguishable from "nothing needed to happen"; erroring names
	// exactly which ids were wrong so the real mistake -- pointing at the
	// wrong build -- doesn't get confused with a genuinely empty match.
	if len(req.Target.IDs) > 0 {
		if missing := missingIDs(req.Target.IDs, matched); len(missing) > 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%d target id(s) do not belong to this build: %s", len(missing), strings.Join(missing, ", ")))
			return
		}
	}

	if req.DryRun {
		writeJSON(w, http.StatusOK, adHocResult{Matched: len(matched), Objects: matched})
		return
	}

	created := make([]db.AgentTask, 0, len(matched))
	for _, o := range matched {
		next, err := s.Queries.NextStepIndexForHost(r.Context(), o.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		task, err := s.Queries.CreateAgentTask(r.Context(), db.CreateAgentTaskParams{
			DeployedObjectID: o.ID, StepIndex: next, Command: req.Command, Payload: req.Payload,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		created = append(created, task)
	}
	writeJSON(w, http.StatusOK, adHocResult{Matched: len(matched), Created: created})
}

// --- team access windows ("Build → Access") ---

type setTeamAccessRequest struct {
	// Action is "open", "close", "extend", or "reduce". A hand open/close records
	// a manual override (direction + how long it holds) AND fires the builder
	// call now for immediate effect; the override is what stops the access
	// reconciler (internal/orchestrator/access.go) from reverting the team to
	// the schedule on its next pass. "extend" pushes an open override's expiry
	// out -- and, if the team isn't open yet, opens it too, so it acts rather
	// than only lengthening. "reduce" is extend's exact inverse -- a penalty:
	// it holds the team CLOSED for N more minutes (taking that access away now),
	// after which the schedule resumes. Always available, and stacking reduces
	// add up the same way stacking extends do. A team's real open/closed state
	// still only flips once the OpenAccess/CloseAccess builder call completes
	// (internal/runner's executeAccess); the override records intent, not an
	// achieved state.
	Action string `json:"action"`
	// ExtendMinutes is only read for action "extend".
	ExtendMinutes int `json:"extend_minutes,omitempty"`
	// ReduceMinutes is only read for action "reduce".
	ReduceMinutes int `json:"reduce_minutes,omitempty"`
}

// accessWindowsForBuild reads a build's authored access: windows from the
// persisted environment row (the same source GET /builds/{id} shows), as real
// timestamps for schedule math. Empty when the environment has none or can't be
// read -- a hand action taken then has no schedule boundary to hand back to, so
// it holds until changed.
func (s *Server) accessWindowsForBuild(ctx context.Context, build db.Build) []schedule.AccessWindow {
	env, err := s.Queries.GetEnvironmentByRevisionAndName(ctx, db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: build.ContentRevisionID, Name: build.EnvironmentName,
	})
	if err != nil {
		return nil
	}
	return schedule.ParseAccessWindows(env.Access)
}

// effectiveAccessEnd is the team's current access end -- what extend/reduce move
// later/earlier: an active open override's expiry if one holds, else the close
// of the scheduled window it's in. Not ok means the team has no finite end right
// now (it's closed, or open with no schedule), so there's nothing to move.
func effectiveAccessEnd(windows []schedule.AccessWindow, team db.Team, now time.Time) (time.Time, bool) {
	if team.AccessOverrideState == "open" && team.AccessOverrideUntil.Valid && team.AccessOverrideUntil.Time.After(now) {
		return team.AccessOverrideUntil.Time, true
	}
	return schedule.CurrentWindowClose(windows, now)
}

// handleSetTeamAccess is the operator front door onto the same
// open_access/close_access task the runner already executes against the real
// builder (internal/runner's executeAccess, proven live in
// TestOpenCloseAccessRemovesAndRestoresNIC/TestRealTwoTeamIncusBuild). It sets a
// manual override so the change survives the schedule reconciler, and fires the
// builder call immediately so it takes effect now, not on the next pass.
func (s *Server) handleSetTeamAccess(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireLevel(r.Context(), r, repo, levelManage)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	teamNum, err := strconv.ParseInt(r.PathValue("team"), 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid team number"))
		return
	}
	team, err := s.Queries.GetTeamByNumber(r.Context(), db.GetTeamByNumberParams{BuildID: build.ID, TeamNumber: int32(teamNum)})
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such team on this build"))
		return
	}
	var req setTeamAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	now := time.Now()
	teamStr := strconv.FormatInt(int64(teamNum), 10)

	switch req.Action {
	case "open", "close":
		dir, kind := "open", "open_access"
		if req.Action == "close" {
			dir, kind = "closed", "close_access"
		}
		// The override holds until the schedule's next boundary, then the
		// reconciler resumes the schedule; with no future boundary it holds
		// until changed. Recording it (not just firing a one-off task) is what
		// keeps the reconciler from reverting the team on its next pass.
		until := pgtype.Timestamptz{}
		if b, ok := schedule.NextBoundary(s.accessWindowsForBuild(r.Context(), build), now); ok {
			until = pgtype.Timestamptz{Time: b, Valid: true}
		}
		if _, err := s.Queries.SetTeamAccessOverride(r.Context(), db.SetTeamAccessOverrideParams{
			ID: team.ID, AccessOverrideState: dir, AccessOverrideUntil: until,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// Fire the builder call now for immediate effect; the reconciler
		// maintains it thereafter and dedups against this very task.
		payload, _ := json.Marshal(map[string]string{"team": teamStr})
		task, err := s.Queries.CreateTeamTask(r.Context(), db.CreateTeamTaskParams{
			BuildID: build.ID, Kind: kind, Payload: payload,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
			BuildID: build.ID, TaskID: task.ID, Kind: "access." + req.Action + ".requested",
			Message: sess.GithubLogin + " requested " + req.Action + " for team " + teamStr,
			Payload: []byte("{}"),
		})
		writeJSON(w, http.StatusAccepted, task)
	case "extend", "reduce":
		// Extend and reduce both just move the team's access END time -- extend
		// pushes it later, reduce pulls it earlier -- recorded as an open override
		// whose expiry IS that end. The team stays open until then; it is never
		// shut off now (reduce takes time off the END, it doesn't punch a hole in
		// the middle). The access reconciler closes the team when the end passes
		// and -- crucially -- keeps it closed for the rest of the window the end
		// belonged to (see desiredAccess), so a reduced end actually sticks and a
		// later window still opens normally.
		mins := req.ExtendMinutes
		if req.Action == "reduce" {
			mins = req.ReduceMinutes
		}
		if mins <= 0 {
			writeError(w, http.StatusBadRequest, errors.New(req.Action+"_minutes must be positive"))
			return
		}
		windows := s.accessWindowsForBuild(r.Context(), build)
		end, haveEnd := effectiveAccessEnd(windows, team, now)
		if req.Action == "reduce" && !haveEnd {
			writeError(w, http.StatusBadRequest, errors.New("team has no open access window to reduce -- it's already closed"))
			return
		}
		// Extend from the current end if there is one, else from now (extending a
		// closed team opens it for the next N minutes). Reduce always has an end.
		base := now
		if haveEnd {
			base = end
		}
		delta := time.Duration(mins) * time.Minute
		var newEnd time.Time
		if req.Action == "extend" {
			newEnd = base.Add(delta)
		} else {
			newEnd = base.Add(-delta)
			if newEnd.Before(now) {
				newEnd = now // reducing past now just ends access now
			}
		}
		updated, err := s.Queries.SetTeamAccessOverride(r.Context(), db.SetTeamAccessOverrideParams{
			ID: team.ID, AccessOverrideState: "open", AccessOverrideUntil: pgtype.Timestamptz{Time: newEnd, Valid: true},
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// Extending a team that isn't open yet opens it now; reduce never opens a
		// team (it only moves an existing end in), and neither shuts one off now --
		// the reconciler closes at newEnd.
		if req.Action == "extend" && team.AccessState != "open" {
			payload, _ := json.Marshal(map[string]string{"team": teamStr})
			if task, err := s.Queries.CreateTeamTask(r.Context(), db.CreateTeamTaskParams{
				BuildID: build.ID, Kind: "open_access", Payload: payload,
			}); err == nil {
				s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
					BuildID: build.ID, TaskID: task.ID, Kind: "access.open.requested",
					Message: sess.GithubLogin + " opened team " + teamStr + " via extend",
					Payload: []byte("{}"),
				})
			}
		}
		past := "extended"
		if req.Action == "reduce" {
			past = "reduced"
		}
		s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
			BuildID: build.ID, Kind: "access." + past,
			Message: sess.GithubLogin + " " + past + " team " + teamStr + " by " + strconv.Itoa(mins) + " minutes",
			Payload: []byte("{}"),
		})
		writeJSON(w, http.StatusOK, updated)
	default:
		writeError(w, http.StatusBadRequest, errors.New("action must be open, close, extend, or reduce"))
	}
}
