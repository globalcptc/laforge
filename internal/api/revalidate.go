// Revalidate: re-run every validator on the selected hosts, on the boxes as
// they are, and report what each check says now. Nothing else runs -- no step,
// no fix, no restart -- and the only change it can make is a host's status
// between finished and invalid (orchestrator.SettleValidationRuns).
// Manage-level, like rebuild and ad-hoc dispatch: it runs commands on hosts.
package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

type revalidateRequest struct {
	Target adHocTarget `json:"target"`
}

type revalidateHost struct {
	ObjectID   string `json:"object_id"`
	Name       string `json:"name"`
	TeamNumber int32  `json:"team_number"`
	RunID      string `json:"run_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type revalidateResponse struct {
	// ReportID is empty when no host could be revalidated.
	ReportID string           `json:"report_id,omitempty"`
	Started  []revalidateHost `json:"started"`
	Skipped  []revalidateHost `json:"skipped"`
}

func (s *Server) handleRevalidate(w http.ResponseWriter, r *http.Request) {
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
	switch build.Status {
	case "deploying", "building", "finished", "failed":
	default:
		writeError(w, http.StatusConflict, errors.New("build is "+build.Status+" -- only a deployed build's hosts can be revalidated"))
		return
	}
	var req revalidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	matched, err := orchestrator.MatchAdHocTargets(r.Context(), s.Queries, build.ID, req.Target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(req.Target.IDs) > 0 {
		if missing := missingIDs(req.Target.IDs, matched); len(missing) > 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%d target id(s) do not belong to this build: %s", len(missing), strings.Join(missing, ", ")))
			return
		}
	}
	teams, err := s.Queries.ListTeamsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	teamNum := make(map[string]int32, len(teams))
	for _, t := range teams {
		teamNum[t.ID.String()] = t.TeamNumber
	}

	var reportID pgtype.UUID
	reportID.Valid = true
	if _, err := rand.Read(reportID.Bytes[:]); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	reportID.Bytes[6] = (reportID.Bytes[6] & 0x0f) | 0x40 // UUID v4
	reportID.Bytes[8] = (reportID.Bytes[8] & 0x3f) | 0x80

	res := revalidateResponse{Started: []revalidateHost{}, Skipped: []revalidateHost{}}
	now := time.Now()
	for _, o := range matched {
		h := revalidateHost{ObjectID: o.ID.String(), Name: db.StrOrEmpty(o.AsName), TeamNumber: teamNum[o.TeamID.String()]}
		if h.Name == "" {
			h.Name = o.ObjectName
		}
		if reason := s.revalidateBlockedReason(r, o, now); reason != "" {
			h.Reason = reason
			res.Skipped = append(res.Skipped, h)
			continue
		}
		run, err := orchestrator.StartValidationRun(r.Context(), s.Pool, reportID, build.ID, o, sess.AccountID)
		switch {
		case errors.Is(err, orchestrator.ErrNoValidators), errors.Is(err, orchestrator.ErrValidationRunning):
			h.Reason = err.Error()
			res.Skipped = append(res.Skipped, h)
			continue
		case err != nil:
			writeError(w, http.StatusInternalServerError, fmt.Errorf("revalidating %s: %w", h.Name, err))
			return
		}
		h.RunID = run.ID.String()
		res.Started = append(res.Started, h)
	}
	if len(res.Started) > 0 {
		res.ReportID = reportID.String()
	}
	writeJSON(w, http.StatusOK, res)
}

// revalidateBlockedReason returns "" if the host can be revalidated now, or why
// not. Like a shell, it needs a deployed host with a live agent to run checks.
func (s *Server) revalidateBlockedReason(r *http.Request, o db.DeployedObject, now time.Time) string {
	if o.Kind != "host" && o.Kind != "container" {
		return "only hosts and containers have validators"
	}
	if !infraUp(o.Status) {
		return fmt.Sprintf("the host is %s -- it has to be deployed", o.Status)
	}
	sess, err := s.Queries.GetAgentSessionByDeployedObject(r.Context(), o.ID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && o.DeployedAt.Valid && sess.LastHeartbeatAt.Time.Before(o.DeployedAt.Time)) {
		return "its agent hasn't checked in"
	}
	if err != nil {
		return "couldn't read its agent's status"
	}
	if age := now.Sub(sess.LastHeartbeatAt.Time); age > lateWindow {
		return fmt.Sprintf("its agent hasn't checked in for %s", age.Round(time.Minute))
	}
	return ""
}

// --- reports ---

type validationCheckView struct {
	Kind    string          `json:"kind"`
	Args    json.RawMessage `json:"args"`
	Passed  *bool           `json:"passed"` // nil: not run yet
	Message string          `json:"message,omitempty"`
}

type validationStepView struct {
	StepIndex int32 `json:"step_index"`
	// Status is the copied check task's: pending/leased (not run yet), done,
	// failed, or ignored (failed on a step marked ignore_errors -- reported,
	// never fails the run).
	Status       string                `json:"status"`
	IgnoreErrors bool                  `json:"ignore_errors"`
	Error        string                `json:"error,omitempty"`
	Checks       []validationCheckView `json:"checks"`
}

type validationRunView struct {
	ID           string               `json:"id"`
	ReportID     string               `json:"report_id"`
	ObjectID     string               `json:"object_id"`
	Name         string               `json:"name,omitempty"`
	Kind         string               `json:"kind,omitempty"`
	TeamNumber   int32                `json:"team_number,omitempty"`
	Status       string               `json:"status"`
	StatusBefore string               `json:"status_before"`
	StatusAfter  string               `json:"status_after,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	FinishedAt   *time.Time           `json:"finished_at,omitempty"`
	Steps        []validationStepView `json:"steps"`
}

type validationReportView struct {
	ReportID   string              `json:"report_id"`
	CreatedAt  time.Time           `json:"created_at"`
	FinishedAt *time.Time          `json:"finished_at,omitempty"`
	Hosts      int64               `json:"hosts"`
	Running    int64               `json:"running"`
	Passed     int64               `json:"passed"`
	Failed     int64               `json:"failed"`
	Incomplete int64               `json:"incomplete"`
	Runs       []validationRunView `json:"runs,omitempty"`
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

// validationSteps reads a run's copied checks with each check's result -- or,
// for a check that hasn't reported yet, what it will check.
func (s *Server) validationSteps(r *http.Request, runID pgtype.UUID) ([]validationStepView, error) {
	tasks, err := s.Queries.ListValidationRunTasks(r.Context(), runID)
	if err != nil {
		return nil, err
	}
	out := make([]validationStepView, 0, len(tasks))
	for _, t := range tasks {
		v := validationStepView{StepIndex: t.StepIndex, Status: t.Status, IgnoreErrors: t.IgnoreErrors, Error: db.StrOrEmpty(t.LastError), Checks: []validationCheckView{}}
		results, err := s.Queries.ListValidatorResultsByTask(r.Context(), t.ID)
		if err != nil {
			return nil, err
		}
		for _, vr := range results {
			passed := vr.Passed
			v.Checks = append(v.Checks, validationCheckView{Kind: vr.Kind, Args: vr.Args, Passed: &passed, Message: db.StrOrEmpty(vr.Message)})
		}
		if len(results) == 0 {
			// Not run yet (or it errored before checking): list what it checks.
			var p struct {
				Checks []struct {
					Kind string          `json:"kind"`
					Args json.RawMessage `json:"args"`
				} `json:"checks"`
			}
			if json.Unmarshal(t.Payload, &p) == nil {
				for _, c := range p.Checks {
					v.Checks = append(v.Checks, validationCheckView{Kind: c.Kind, Args: c.Args})
				}
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Server) handleListValidationReports(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListValidationReportsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]validationReportView, 0, len(rows))
	for _, row := range rows {
		out = append(out, validationReportView{
			ReportID: row.ReportID.String(), CreatedAt: row.CreatedAt.Time, FinishedAt: tsPtr(row.FinishedAt),
			Hosts: row.Hosts, Running: row.Running, Passed: row.Passed, Failed: row.Failed, Incomplete: row.Incomplete,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetValidationReport(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	reportID, err := parseUUID(r.PathValue("reportId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	runs, err := s.Queries.ListValidationRunsByReport(r.Context(), db.ListValidationRunsByReportParams{BuildID: build.ID, ReportID: reportID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(runs) == 0 {
		writeError(w, http.StatusNotFound, errors.New("no such validation report in this build"))
		return
	}
	rep := validationReportView{ReportID: reportID.String(), CreatedAt: runs[0].CreatedAt.Time, Runs: make([]validationRunView, 0, len(runs))}
	allDone := true
	for _, run := range runs {
		steps, err := s.validationSteps(r, run.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if run.CreatedAt.Time.Before(rep.CreatedAt) {
			rep.CreatedAt = run.CreatedAt.Time
		}
		rep.Hosts++
		switch run.Status {
		case "running":
			rep.Running++
			allDone = false
		case "passed":
			rep.Passed++
		case "failed":
			rep.Failed++
		default:
			rep.Incomplete++
		}
		if run.FinishedAt.Valid && (rep.FinishedAt == nil || run.FinishedAt.Time.After(*rep.FinishedAt)) {
			rep.FinishedAt = tsPtr(run.FinishedAt)
		}
		rep.Runs = append(rep.Runs, validationRunView{
			ID: run.ID.String(), ReportID: reportID.String(), ObjectID: run.DeployedObjectID.String(), Name: run.ObjectLabel, Kind: run.ObjectKind, TeamNumber: run.TeamNumber,
			Status: run.Status, StatusBefore: run.StatusBefore, StatusAfter: db.StrOrEmpty(run.StatusAfter),
			CreatedAt: run.CreatedAt.Time, FinishedAt: tsPtr(run.FinishedAt), Steps: steps,
		})
	}
	if !allDone {
		rep.FinishedAt = nil
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleListObjectValidationRuns(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	objectID, err := s.objectInBuild(r, build.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	runs, err := s.Queries.ListValidationRunsByObject(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]validationRunView, 0, len(runs))
	for _, run := range runs {
		steps, err := s.validationSteps(r, run.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, validationRunView{
			ID: run.ID.String(), ReportID: run.ReportID.String(), ObjectID: run.DeployedObjectID.String(),
			Status: run.Status, StatusBefore: run.StatusBefore, StatusAfter: db.StrOrEmpty(run.StatusAfter),
			CreatedAt: run.CreatedAt.Time, FinishedAt: tsPtr(run.FinishedAt), Steps: steps,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
