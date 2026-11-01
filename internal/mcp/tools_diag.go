package mcp

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// Diagnostic tools that answer questions no single API page does, straight
// from the database as laforge_mcp.

func (s *Server) registerDiagTools() {
	addTool(s, "heartbeat_gaps", "Heartbeat Gaps",
		"Finds agents that went quiet: every silence between check-ins longer than min_gap_seconds in the window, longest first, plus a roll-up per cluster member or server (location) and the agents silent right now. Many gaps starting together on one location points at that member, not the hosts.",
		toolHeartbeatGaps)
	addTool(s, "search_agent_tasks", "Search Agent Tasks",
		"Searches agent tasks across a whole build -- authored steps, ad-hoc commands and scheduled runs -- by status (failed, leased = running now, blocked, pending, done, ignored), ad-hoc or not, command kind, team and time. Each comes with its payload (arguments), error and the tail of its output.",
		toolSearchAgentTasks)
	addTool(s, "get_step_output", "Get Step Output",
		"One agent task's full output and error, read in pages (offset and max_bytes), with its payload.",
		toolGetStepOutput)
	addTool(s, "list_runner_tasks", "List Runner Tasks",
		"The runner's queue for a build: deploy, destroy, power and network work items with status, attempts, who holds the lease, and the last error. Shows deploys that are stuck or retrying.",
		toolListRunnerTasks)
	addTool(s, "list_shell_sessions", "List Shell Sessions",
		"Who opened an interactive terminal on which host in a build, from where, and when.",
		toolListShellSessions)
	addTool(s, "list_content_revisions", "List Content Revisions",
		"The commits LaForge has ingested for a repository, newest first, with whether each validated and its validation errors.",
		toolListContentRevisions)
}

// --- heartbeat_gaps ---

type heartbeatGapsIn struct {
	BuildID       string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	WindowMinutes int    `json:"window_minutes,omitempty" jsonschema:"how far back to look (default 60, at most 360)"`
	MinGapSeconds int    `json:"min_gap_seconds,omitempty" jsonschema:"shortest silence to report (default 120)"`
	Limit         int    `json:"limit,omitempty" jsonschema:"how many gaps to list, longest first (default 100, at most 500)"`
}

type gapRow struct {
	Object     string    `json:"object"`
	ObjectID   string    `json:"object_id"`
	Team       int32     `json:"team"`
	Location   string    `json:"location,omitempty"`
	GapStart   time.Time `json:"gap_start"`
	GapEnd     time.Time `json:"gap_end"`
	GapSeconds int64     `json:"gap_seconds"`
}

type gapLocation struct {
	Location     string     `json:"location"`
	Gaps         int        `json:"gaps"`
	Objects      int        `json:"objects_with_gaps"`
	LongestSecs  int64      `json:"longest_seconds"`
	FirstStarted *time.Time `json:"first_gap_started,omitempty"`
	LastStarted  *time.Time `json:"last_gap_started,omitempty"`
	SilentNow    int        `json:"silent_now"`
}

type silentRow struct {
	Object        string     `json:"object"`
	ObjectID      string     `json:"object_id"`
	Team          int32      `json:"team"`
	Location      string     `json:"location,omitempty"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	SilentSeconds *int64     `json:"silent_seconds,omitempty"`
	AgentStatus   string     `json:"agent_status"`
}

type heartbeatGapsOut struct {
	Build      string        `json:"build"`
	Since      time.Time     `json:"since"`
	MinGapSecs int           `json:"min_gap_seconds"`
	Gaps       []gapRow      `json:"gaps"`
	Truncated  bool          `json:"truncated"`
	ByLocation []gapLocation `json:"by_location"`
	SilentNow  []silentRow   `json:"silent_now" jsonschema:"agents whose instance is up but that haven't checked in for longer than min_gap_seconds (or never)"`
}

func toolHeartbeatGaps(ctx context.Context, c *call, in heartbeatGapsIn) (heartbeatGapsOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return heartbeatGapsOut{}, err
	}
	window := time.Duration(clamp(in.WindowMinutes, 60, 360)) * time.Minute
	minGap := clamp(in.MinGapSeconds, 120, 86400)
	limit := clamp(in.Limit, 100, 500)
	since := time.Now().Add(-window)
	rows, err := c.q().McpHeartbeatGaps(ctx, db.McpHeartbeatGapsParams{
		BuildID: id, Since: pgtype.Timestamptz{Time: since, Valid: true}, MinGapSeconds: float64(minGap), MaxRows: int32(limit + 1),
	})
	if err != nil {
		return heartbeatGapsOut{}, err
	}
	out := heartbeatGapsOut{Build: id.String(), Since: since, MinGapSecs: minGap, Gaps: []gapRow{}, SilentNow: []silentRow{}}
	if len(rows) > limit {
		rows, out.Truncated = rows[:limit], true
	}
	locs := map[string]*gapLocation{}
	objsAt := map[string]map[string]bool{}
	loc := func(name string) *gapLocation {
		if l := locs[name]; l != nil {
			return l
		}
		l := &gapLocation{Location: name}
		locs[name] = l
		objsAt[name] = map[string]bool{}
		return l
	}
	for _, r := range rows {
		name := r.ObjectName
		if r.AsName != nil && *r.AsName != "" {
			name = *r.AsName
		}
		g := gapRow{Object: name, ObjectID: r.DeployedObjectID.String(), Team: r.TeamNumber, Location: r.InstanceLocation,
			GapStart: r.GapStart.Time, GapEnd: r.GapEnd.Time, GapSeconds: int64(r.GapSeconds)}
		out.Gaps = append(out.Gaps, g)
		l := loc(r.InstanceLocation)
		l.Gaps++
		objsAt[r.InstanceLocation][g.ObjectID] = true
		if g.GapSeconds > l.LongestSecs {
			l.LongestSecs = g.GapSeconds
		}
		start := g.GapStart
		if l.FirstStarted == nil || start.Before(*l.FirstStarted) {
			l.FirstStarted = &start
		}
		if l.LastStarted == nil || start.After(*l.LastStarted) {
			l.LastStarted = &start
		}
	}

	objs, err := c.q().McpListObjectsByBuild(ctx, id)
	if err != nil {
		return heartbeatGapsOut{}, err
	}
	now := time.Now()
	for _, o := range objs {
		status, last, ok := agentStatus(o, now)
		if !ok {
			continue
		}
		var silent *int64
		if !last.IsZero() {
			secs := int64(now.Sub(last).Seconds())
			if secs < int64(minGap) {
				continue
			}
			silent = &secs
		} else if status == "booting" {
			continue // not up long enough to be overdue
		}
		name := o.ObjectName
		if o.AsName != nil && *o.AsName != "" {
			name = *o.AsName
		}
		row := silentRow{Object: name, ObjectID: o.ID.String(), Team: o.TeamNumber, Location: o.InstanceLocation, SilentSeconds: silent, AgentStatus: status}
		if !last.IsZero() {
			row.LastHeartbeat = &last
		}
		out.SilentNow = append(out.SilentNow, row)
		loc(o.InstanceLocation).SilentNow++
	}
	sort.Slice(out.SilentNow, func(i, j int) bool {
		return out.SilentNow[i].Location+out.SilentNow[i].Object < out.SilentNow[j].Location+out.SilentNow[j].Object
	})
	for name, l := range locs {
		l.Objects = len(objsAt[name])
		out.ByLocation = append(out.ByLocation, *l)
	}
	sort.Slice(out.ByLocation, func(i, j int) bool {
		return out.ByLocation[i].Gaps+out.ByLocation[i].SilentNow > out.ByLocation[j].Gaps+out.ByLocation[j].SilentNow
	})
	return out, nil
}

// --- search_agent_tasks ---

type searchTasksIn struct {
	BuildID      string   `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Statuses     []string `json:"statuses,omitempty" jsonschema:"only these statuses: pending, leased (running now), done, failed, ignored, blocked. Default: failed"`
	AdHoc        *bool    `json:"ad_hoc,omitempty" jsonschema:"true for only ad-hoc/scheduled commands, false for only authored steps; omit for both"`
	Command      string   `json:"command,omitempty" jsonschema:"only this command kind, e.g. exec, write_file, set_password, validate, reboot"`
	Team         int      `json:"team,omitempty" jsonschema:"only this team number"`
	SinceMinutes int      `json:"since_minutes,omitempty" jsonschema:"only tasks changed in the last N minutes (default: any time)"`
	OutputTail   int      `json:"output_tail,omitempty" jsonschema:"how many characters of the end of each task's output to include (default 1000, at most 8000)"`
	Limit        int      `json:"limit,omitempty" jsonschema:"how many tasks, most recently changed first (default 100, at most 500)"`
}

type taskRow struct {
	ID            string     `json:"id"`
	Object        string     `json:"object"`
	ObjectID      string     `json:"object_id"`
	Team          int32      `json:"team"`
	Location      string     `json:"location,omitempty"`
	StepIndex     int32      `json:"step_index"`
	Command       string     `json:"command"`
	Script        string     `json:"script,omitempty"`
	Milestone     string     `json:"milestone,omitempty"`
	WaitFor       any        `json:"wait_for,omitempty"`
	Payload       any        `json:"payload,omitempty"`
	Status        string     `json:"status"`
	Attempts      int32      `json:"attempts"`
	MaxAttempts   *int32     `json:"max_attempts,omitempty"`
	IgnoreErrors  bool       `json:"ignore_errors,omitempty"`
	AdHoc         bool       `json:"ad_hoc"`
	ScheduledTask string     `json:"scheduled_task_id,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	OutputTail    string     `json:"output_tail,omitempty"`
	OutputBytes   int32      `json:"output_bytes"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type searchTasksOut struct {
	Build     string    `json:"build"`
	Statuses  []string  `json:"statuses"`
	Tasks     []taskRow `json:"tasks"`
	Truncated bool      `json:"truncated"`
}

func toolSearchAgentTasks(ctx context.Context, c *call, in searchTasksIn) (searchTasksOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return searchTasksOut{}, err
	}
	statuses := in.Statuses
	if len(statuses) == 0 {
		statuses = []string{"failed"}
	}
	since := time.Time{}
	if in.SinceMinutes > 0 {
		since = time.Now().Add(-time.Duration(in.SinceMinutes) * time.Minute)
	}
	limit := clamp(in.Limit, 100, 500)
	rows, err := c.q().McpSearchAgentTasks(ctx, db.McpSearchAgentTasksParams{
		BuildID: id, Statuses: statuses, AdHoc: in.AdHoc, Command: in.Command, Team: int32(in.Team),
		Since: pgtype.Timestamptz{Time: since, Valid: true}, OutputTail: int32(clamp(in.OutputTail, 1000, 8000)), MaxRows: int32(limit + 1),
	})
	if err != nil {
		return searchTasksOut{}, err
	}
	out := searchTasksOut{Build: id.String(), Statuses: statuses, Tasks: []taskRow{}}
	if len(rows) > limit {
		rows, out.Truncated = rows[:limit], true
	}
	for _, r := range rows {
		name := r.ObjectName
		if r.AsName != nil && *r.AsName != "" {
			name = *r.AsName
		}
		t := taskRow{
			ID: r.ID.String(), Object: name, ObjectID: r.DeployedObjectID.String(), Team: r.TeamNumber, Location: r.InstanceLocation,
			StepIndex: r.StepIndex, Command: r.Command, Script: str(r.StepScript), Milestone: str(r.Milestone),
			Status: r.Status, Attempts: r.Attempts, MaxAttempts: r.MaxAttempts, IgnoreErrors: r.IgnoreErrors, AdHoc: r.AdHoc,
			LastError: clip(str(r.LastError), 2000), OutputTail: r.OutputTail, OutputBytes: r.OutputBytes,
			StartedAt: ts(r.StartedAt), FinishedAt: ts(r.FinishedAt), UpdatedAt: r.UpdatedAt.Time,
		}
		if len(r.Payload) > 0 && string(r.Payload) != "{}" {
			t.Payload = jsonValue(r.Payload)
		}
		if len(r.WaitFor) > 0 && string(r.WaitFor) != "null" {
			t.WaitFor = jsonValue(r.WaitFor)
		}
		if r.ScheduledTaskID.Valid {
			t.ScheduledTask = r.ScheduledTaskID.String()
		}
		out.Tasks = append(out.Tasks, t)
	}
	return out, nil
}

// --- get_step_output ---

type stepOutputIn struct {
	TaskID   string `json:"task_id" jsonschema:"the agent task's id, from list_steps or search_agent_tasks"`
	Offset   int    `json:"offset,omitempty" jsonschema:"byte offset into the output to start from (default 0)"`
	MaxBytes int    `json:"max_bytes,omitempty" jsonschema:"how many bytes of output to return (default 32000, at most 200000)"`
}

type stepOutputOut struct {
	TaskID      string     `json:"task_id"`
	Object      string     `json:"object"`
	Team        int32      `json:"team"`
	Build       string     `json:"build"`
	StepIndex   int32      `json:"step_index"`
	Command     string     `json:"command"`
	Script      string     `json:"script,omitempty"`
	Payload     any        `json:"payload,omitempty"`
	Status      string     `json:"status"`
	Attempts    int32      `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
	OutputBytes int        `json:"output_bytes"`
	Offset      int        `json:"offset"`
	Output      string     `json:"output"`
	More        bool       `json:"more" jsonschema:"true when there is output after this page; ask again with offset = offset + len(output)"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

func toolGetStepOutput(ctx context.Context, c *call, in stepOutputIn) (stepOutputOut, error) {
	id, err := parseID(in.TaskID, "task_id")
	if err != nil {
		return stepOutputOut{}, err
	}
	t, err := c.q().McpGetAgentTask(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return stepOutputOut{}, errDenied
	}
	if err != nil {
		return stepOutputOut{}, err
	}
	// The task's build must be one this key can read.
	if _, err := c.build(ctx, t.BuildID.String()); err != nil {
		return stepOutputOut{}, err
	}
	output := str(t.Output)
	offset := in.Offset
	if offset < 0 || offset > len(output) {
		offset = len(output)
	}
	end := offset + clamp(in.MaxBytes, 32000, 200000)
	if end > len(output) {
		end = len(output)
	}
	name := t.ObjectName
	if t.AsName != nil && *t.AsName != "" {
		name = *t.AsName
	}
	out := stepOutputOut{
		TaskID: t.ID.String(), Object: name, Team: t.TeamNumber, Build: t.BuildID.String(), StepIndex: t.StepIndex,
		Command: t.Command, Script: str(t.StepScript), Status: t.Status, Attempts: t.Attempts, LastError: str(t.LastError),
		OutputBytes: len(output), Offset: offset, Output: strings.ToValidUTF8(output[offset:end], "�"), More: end < len(output),
		StartedAt: ts(t.StartedAt), FinishedAt: ts(t.FinishedAt),
	}
	if len(t.Payload) > 0 && string(t.Payload) != "{}" {
		out.Payload = jsonValue(t.Payload)
	}
	return out, nil
}

// --- list_runner_tasks ---

type runnerTasksIn struct {
	BuildID  string   `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Statuses []string `json:"statuses,omitempty" jsonschema:"only these statuses: pending, leased, done, failed. Default: all"`
	Limit    int      `json:"limit,omitempty" jsonschema:"how many, most recently changed first (default 100, at most 500)"`
}

type runnerTaskRow struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Status         string     `json:"status"`
	Attempts       int32      `json:"attempts"`
	Permanent      bool       `json:"permanent_failure"`
	LeaseOwner     string     `json:"lease_owner,omitempty"`
	LeaseExpiresAt *time.Time `json:"lease_expires_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	Object         string     `json:"object,omitempty"`
	Team           *int32     `json:"team,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type runnerTasksOut struct {
	Build    string          `json:"build"`
	Counts   map[string]int  `json:"counts_by_status"`
	Tasks    []runnerTaskRow `json:"tasks"`
	Truncate bool            `json:"truncated"`
}

func toolListRunnerTasks(ctx context.Context, c *call, in runnerTasksIn) (runnerTasksOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return runnerTasksOut{}, err
	}
	limit := clamp(in.Limit, 100, 500)
	statuses := in.Statuses
	if statuses == nil {
		statuses = []string{}
	}
	rows, err := c.q().McpListRunnerTasks(ctx, db.McpListRunnerTasksParams{BuildID: id, Statuses: statuses, MaxRows: int32(limit + 1)})
	if err != nil {
		return runnerTasksOut{}, err
	}
	out := runnerTasksOut{Build: id.String(), Counts: map[string]int{}, Tasks: []runnerTaskRow{}}
	if len(rows) > limit {
		rows, out.Truncate = rows[:limit], true
	}
	for _, r := range rows {
		name := str(r.ObjectName)
		if r.AsName != nil && *r.AsName != "" {
			name = *r.AsName
		}
		out.Counts[r.Status]++
		out.Tasks = append(out.Tasks, runnerTaskRow{
			ID: r.ID.String(), Kind: r.Kind, Status: r.Status, Attempts: r.Attempts, Permanent: r.Permanent,
			LeaseOwner: str(r.LeaseOwner), LeaseExpiresAt: ts(r.LeaseExpiresAt), LastError: clip(str(r.LastError), 2000),
			Object: name, Team: r.TeamNumber, CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time,
		})
	}
	return out, nil
}

// --- list_shell_sessions ---

type shellSessionsIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
}

type shellSessionRow struct {
	ID         string     `json:"id"`
	Object     string     `json:"object"`
	Team       int32      `json:"team"`
	OpenedBy   string     `json:"opened_by,omitempty"`
	ClientAddr string     `json:"client_addr,omitempty"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

type shellSessionsOut struct {
	Build    string            `json:"build"`
	Sessions []shellSessionRow `json:"sessions"`
}

func toolListShellSessions(ctx context.Context, c *call, in shellSessionsIn) (shellSessionsOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return shellSessionsOut{}, err
	}
	rows, err := c.q().McpListShellSessions(ctx, id)
	if err != nil {
		return shellSessionsOut{}, err
	}
	out := shellSessionsOut{Build: id.String(), Sessions: []shellSessionRow{}}
	for _, r := range rows {
		name := r.ObjectName
		if r.AsName != nil && *r.AsName != "" {
			name = *r.AsName
		}
		out.Sessions = append(out.Sessions, shellSessionRow{
			ID: r.ID.String(), Object: name, Team: r.TeamNumber, OpenedBy: str(r.OpenedBy), ClientAddr: str(r.ClientAddr),
			Status: r.Status, StartedAt: r.StartedAt.Time, EndedAt: ts(r.EndedAt),
		})
	}
	return out, nil
}

// --- list_content_revisions ---

type contentRevisionsIn struct {
	Repository string `json:"repository" jsonschema:"the repository, as owner/name or its id"`
	Limit      int    `json:"limit,omitempty" jsonschema:"how many, newest first (default 20, at most 100)"`
}

type revisionRow struct {
	ID               string     `json:"id"`
	Commit           string     `json:"commit"`
	Branch           string     `json:"branch,omitempty"`
	Valid            bool       `json:"valid"`
	ValidationErrors any        `json:"validation_errors,omitempty"`
	ValidatedAt      *time.Time `json:"validated_at,omitempty"`
	Message          string     `json:"message,omitempty"`
	Author           string     `json:"author,omitempty"`
	CommittedAt      *time.Time `json:"committed_at,omitempty"`
	IngestedAt       time.Time  `json:"ingested_at"`
}

type contentRevisionsOut struct {
	Repository string        `json:"repository"`
	Revisions  []revisionRow `json:"revisions"`
}

func toolListContentRevisions(ctx context.Context, c *call, in contentRevisionsIn) (contentRevisionsOut, error) {
	repo, err := c.repo(ctx, in.Repository)
	if err != nil {
		return contentRevisionsOut{}, err
	}
	id, err := parseID(repo.ID, "repository")
	if err != nil {
		return contentRevisionsOut{}, err
	}
	rows, err := c.q().McpListContentRevisions(ctx, db.McpListContentRevisionsParams{RepositoryID: id, MaxRows: int32(clamp(in.Limit, 20, 100))})
	if err != nil {
		return contentRevisionsOut{}, err
	}
	out := contentRevisionsOut{Repository: repo.Name, Revisions: []revisionRow{}}
	for _, r := range rows {
		row := revisionRow{
			ID: r.ID.String(), Commit: r.CommitSha, Branch: str(r.Ref), Valid: r.Valid, ValidatedAt: ts(r.ValidatedAt),
			Message: firstLine(str(r.CommitMessage)), Author: str(r.CommitAuthor), CommittedAt: ts(r.CommittedAt), IngestedAt: r.CreatedAt.Time,
		}
		if len(r.ValidationErrors) > 0 && string(r.ValidationErrors) != "[]" {
			row.ValidationErrors = jsonValue(r.ValidationErrors)
		}
		out.Revisions = append(out.Revisions, row)
	}
	return out, nil
}
