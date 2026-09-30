// Ad-hoc scheduling: "a way to schedule an ad-hoc task on hosts in the
// UI, much like we can run an ad-hoc task immediately" -- the same
// target selector and command whitelist handleCreateAdHocTask already
// uses (see tasks.go), plus a `when:` natural-language expression
// (internal/schedule), landing in the same scheduled_task table a
// content-authored `schedule:` entry would. Dispatch is real, not a
// stub: internal/orchestrator.DispatchDueScheduledTasks picks up a due
// row and calls the exact same CreateAgentTask machinery immediate
// ad-hoc dispatch already uses -- a live agent polling get-task actually
// runs it.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/schedule"
)

// matchAdHocTargets and buildAnchors are thin wrappers over
// orchestrator's own real logic (MatchAdHocTargets, ResolveAnchors) --
// see those functions' own doc comments for why this matching and this
// anchor resolution have to be the exact same code the dispatch loop
// itself uses, not a second copy that can drift.
func (s *Server) matchAdHocTargets(ctx context.Context, buildID pgtype.UUID, target adHocTarget) ([]db.DeployedObject, error) {
	return orchestrator.MatchAdHocTargets(ctx, s.Queries, buildID, target)
}

func (s *Server) buildAnchors(ctx context.Context, build db.Build) (schedule.Anchors, error) {
	return orchestrator.ResolveAnchors(ctx, s.Queries, s.RepoRoot, build.ID)
}

type createScheduledTaskRequest struct {
	Target  adHocTarget     `json:"target"`
	When    string          `json:"when"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Server) handleCreateScheduledTask(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	// Same level as immediate ad-hoc dispatch (tasks.go): "run agent
	// commands during an event" is manage, not build.
	sess, err := s.requireLevel(r.Context(), r, repo, levelManage)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req createScheduledTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !allowedAdHocCommands[req.Command] {
		writeError(w, http.StatusBadRequest, errors.New("unsupported ad-hoc command"))
		return
	}
	expr, err := schedule.Parse(req.When)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// A real, immediate dry run of the target, exactly like tasks.go's
	// own DryRun -- "the matched set is shown before anything runs"
	// applies just as much to something that won't run until later.
	// Empty target is refused here (not just left to fire against
	// nothing at whatever time it comes due): a scheduled task with no
	// real host to reach is a mistake worth catching now, not silently
	// accepted to be discovered as a no-op days later.
	matched, err := s.matchAdHocTargets(r.Context(), build.ID, req.Target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(matched) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("no hosts match this target -- nothing would ever run"))
		return
	}

	targetJSON, err := json.Marshal(req.Target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	payload := req.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	anchors, err := s.buildAnchors(r.Context(), build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	next, ok := expr.NextFireAfter(time.Now().UTC(), anchors)
	if !ok {
		writeError(w, http.StatusBadRequest, errors.New("this expression has no future occurrence for this build (its anchor has already passed)"))
		return
	}

	login := sess.GithubLogin
	task, err := s.Queries.CreateAdHocScheduledTask(r.Context(), db.CreateAdHocScheduledTaskParams{
		BuildID:    build.ID,
		Target:     targetJSON,
		WhenExpr:   req.When,
		Command:    req.Command,
		Payload:    payload,
		Anchor:     expr.AnchorName(),
		FiresOnce:  expr.FiresOnce(),
		NextFireAt: pgtype.Timestamptz{Time: next, Valid: true},
		CreatedBy:  &login,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

type previewScheduledTaskRequest struct {
	When string `json:"when"`
}

// previewScheduledTaskResponse is a dry look at what a `when:` expression
// would actually do -- the next few real fire times, computed with the
// exact same schedule.NextFireAfter + resolved anchors the dispatch loop
// uses, so "here's when it will run" is the truth, not an approximation.
// A parse error comes back as valid:false with the message (200, not 400)
// so the UI can preview live as the operator types.
type previewScheduledTaskResponse struct {
	Valid       bool     `json:"valid"`
	Error       string   `json:"error,omitempty"`
	FiresOnce   bool     `json:"fires_once"`
	Anchor      string   `json:"anchor,omitempty"`
	Occurrences []string `json:"occurrences"` // RFC3339 UTC, soonest first
}

func (s *Server) handlePreviewScheduledTask(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	// Same capability as creating one -- previewing is part of that flow.
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	var req previewScheduledTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	expr, err := schedule.Parse(req.When)
	if err != nil {
		writeJSON(w, http.StatusOK, previewScheduledTaskResponse{Valid: false, Error: err.Error(), Occurrences: []string{}})
		return
	}
	anchors, err := s.buildAnchors(r.Context(), build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Walk forward from now, feeding each fire time back in as the new
	// "after" so successive occurrences are distinct -- exactly how the
	// dispatch loop reschedules a recurring task. A fires-once expression
	// yields at most one.
	const maxOccurrences = 5
	occ := make([]string, 0, maxOccurrences)
	cur := time.Now().UTC()
	for i := 0; i < maxOccurrences; i++ {
		next, ok := expr.NextFireAfter(cur, anchors)
		if !ok {
			break
		}
		occ = append(occ, next.Format(time.RFC3339))
		cur = next
		if expr.FiresOnce() {
			break
		}
	}
	writeJSON(w, http.StatusOK, previewScheduledTaskResponse{
		Valid:       true,
		FiresOnce:   expr.FiresOnce(),
		Anchor:      expr.AnchorName(),
		Occurrences: occ,
	})
}

func (s *Server) handleListScheduledTasks(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	tasks, err := s.Queries.ListScheduledTasksByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tasks == nil {
		tasks = []db.ScheduledTask{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleCancelScheduledTask(w http.ResponseWriter, r *http.Request) {
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
	taskID, err := parseUUID(r.PathValue("taskId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	task, err := s.Queries.CancelScheduledTask(r.Context(), db.CancelScheduledTaskParams{ID: taskID, BuildID: build.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("no pending scheduled task with that id in this build"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
		BuildID: build.ID, Kind: "schedule.canceled",
		Message: sess.GithubLogin + " canceled scheduled task \"" + task.WhenExpr + "\"",
		Payload: []byte("{}"),
	})
	writeJSON(w, http.StatusOK, task)
}
