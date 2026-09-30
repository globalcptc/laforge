// The environment dashboard's charts -- "Build → Overview: health band...
// a row of stat tiles and a few charts, all fed by server-side
// aggregates rather than counted in the browser".
// BuildOverview.tsx's stat tiles already compute simple
// counts client-side from the whole build payload, which is
// the wrong long-term answer -- this is
// that real server-side aggregation, for the four charts specifically
// named: provisioning progress, agent check-ins over time, state by
// team, and failures grouped by cause.
package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

type teamStateCounts struct {
	TeamNumber int32          `json:"team_number"`
	ByStatus   map[string]int `json:"by_status"`
	// ByHealth is the Overview "state by team" breakdown the UI actually
	// colours: not raw provisioning status but the four outcomes an operator
	// cares about per team -- completed, plus the three distinct ways a host
	// can be wrong (infra never came up, a step failed, or the agent stopped
	// checking in), kept apart because each points at a different fix. An
	// in_progress bucket holds everything still deploying so the blocks are
	// honest mid-deploy rather than pretending a booting host is done.
	ByHealth map[string]int `json:"by_health"`
}

// Health categories for teamStateCounts.ByHealth -- one per object, worst
// signal winning (a host that both failed a step and went missing counts as
// failed_infra/failed_steps first, since that's the actionable cause).
const (
	healthCompleted     = "completed"      // deployed, agent healthy (or a deployed network, which has no agent)
	healthFailedInfra   = "failed_infra"   // the infrastructure isn't up: never provisioned, or the instance is now stopped/gone at the hoster
	healthFailedSteps   = "failed_steps"   // a step (agent_task) failed on an otherwise-up host
	healthFailedCheckin = "failed_checkin" // the instance is up, but the agent is late/missing -- genuinely a check-in problem, not an infra one
	healthInProgress    = "in_progress"    // still deploying / booting -- not yet an outcome
)

// classifyHealth maps one object to a single ByHealth bucket. Teardown
// states (destroying/destroyed) return "" and are dropped by the caller:
// "is this build healthy right now" is not a meaningful question for
// something being intentionally destroyed.
//
// The object's own lifecycle status carries most of the answer now that it
// distinguishes infra, build, and validation outcomes: deploy_failed is an
// infra failure, build_failed a failed step, invalid a failed validator
// (grouped under failed_steps -- both are "the host came up but its content
// is wrong," the actionable class this bucket names), finished is completed.
// Instance power state (builder truth, from the 30s poll) still folds in for
// the up-but-degraded case: a finished host the hoster has since stopped or
// lost is failed_infra, not a "failed check-in" -- the agent going quiet is
// the symptom, the down instance the cause. failed_checkin is reserved for a
// host that really is up (or whose power state hasn't been polled yet) but
// whose agent has gone dark.
func classifyHealth(o objectView, failedSteps map[string]bool) string {
	switch o.Status {
	case "deploy_failed":
		return healthFailedInfra
	case "build_failed", "invalid":
		return healthFailedSteps
	case "destroying", "destroyed":
		return ""
	}
	// The instance is gone or powered off at the hoster -- an infra failure,
	// regardless of what the agent's own health looks like. Only "missing"/
	// "stopped" count; "" (unpolled) and "other" (transient) fall through to
	// the classification below.
	switch o.PowerState {
	case "missing", "stopped":
		return healthFailedInfra
	}
	if failedSteps[o.ID.String()] {
		return healthFailedSteps
	}
	if o.Status != "finished" {
		return healthInProgress // pending / deploying / running / building
	}
	if o.Kind == "network" {
		return healthCompleted // finished, and no agent is ever expected on a network
	}
	if o.Agent == nil {
		return healthInProgress
	}
	switch o.Agent.Status {
	case "healthy":
		return healthCompleted
	case "late", "missing":
		return healthFailedCheckin // instance is up (power not stopped/missing) but the agent is dark
	default: // booting
		return healthInProgress
	}
}

// isFailedStatus reports whether an object status is one of the terminal
// failure states -- an infra deploy failure, a failed build step, or a
// failed validator -- each of which carries a last_error worth grouping.
func isFailedStatus(status string) bool {
	switch status {
	case "deploy_failed", "build_failed", "invalid":
		return true
	}
	return false
}

// failureGroup is "40 hosts: install-mysql exit 1 as one entry, not
// forty" -- grouped by the exact last_error text a real failed object
// carries (set at the point a deploy exhausts MaxAttempts, or the
// object-lifecycle poll records a failed step/validator -- see
// internal/runner/runner.go and internal/orchestrator/lifecycle.go), not
// re-derived or fuzzy-matched.
type failureGroup struct {
	Message string   `json:"message"`
	Count   int      `json:"count"`
	Objects []string `json:"objects"` // as_name (or object_name) of every affected instance, so a click can jump straight to them
}

// activityBucket is real ongoing agent activity, not just "first ever
// checked in" -- as of migrations/00007's append-only agent_heartbeat
// log (every real heartbeat, not the agent_session upsert's single
// latest-only row), "how many distinct objects heartbeated during this
// window" is a genuine cliff detector: a batch of agents going quiet
// shows up as Active dropping, not just as the curve flattening at its
// final value the way a first-seen-only cumulative count would.
type activityBucket struct {
	At     string `json:"at"` // bucket start, RFC3339
	Active int    `json:"active"`
}

type dashboardData struct {
	ProvisioningByStatus map[string]int    `json:"provisioning_by_status"`
	ByTeam               []teamStateCounts `json:"by_team"`
	FailuresByCause      []failureGroup    `json:"failures_by_cause"`
	AgentActivity        []activityBucket  `json:"agent_activity"`
}

// bucketHeartbeats turns a build's raw heartbeat log into a fixed number
// of time buckets, each counting DISTINCT objects seen at least once --
// not total heartbeats, which would just track agent count x poll rate
// rather than "how many agents are actually alive right now." Server-
// side, per "aggregates rather than counted in the browser"
// -- and because raw per-heartbeat rows could be very
// large at real scale (no retention policy exists yet on agent_heartbeat,
// see migrations/00007's own note).
func bucketHeartbeats(heartbeats []db.AgentHeartbeat) []activityBucket {
	if len(heartbeats) == 0 {
		return []activityBucket{}
	}
	minT, maxT := heartbeats[0].CreatedAt.Time, heartbeats[0].CreatedAt.Time
	for _, h := range heartbeats {
		t := h.CreatedAt.Time
		if t.Before(minT) {
			minT = t
		}
		if t.After(maxT) {
			maxT = t
		}
	}
	const maxBuckets = 40
	span := maxT.Sub(minT)
	if span <= 0 {
		span = time.Second
	}
	width := span / maxBuckets
	if width <= 0 {
		width = time.Second
	}
	bucketCount := int(span/width) + 1
	if bucketCount > maxBuckets {
		bucketCount = maxBuckets
	}

	seenPerBucket := make([]map[string]bool, bucketCount)
	for i := range seenPerBucket {
		seenPerBucket[i] = map[string]bool{}
	}
	for _, h := range heartbeats {
		idx := int(h.CreatedAt.Time.Sub(minT) / width)
		if idx >= bucketCount {
			idx = bucketCount - 1
		}
		seenPerBucket[idx][h.DeployedObjectID.String()] = true
	}

	out := make([]activityBucket, bucketCount)
	for i := range out {
		out[i] = activityBucket{At: minT.Add(time.Duration(i) * width).Format(time.RFC3339), Active: len(seenPerBucket[i])}
	}
	return out
}

func (s *Server) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}

	teams, err := s.Queries.ListTeamsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	teamNumberByID := make(map[string]int32, len(teams))
	for _, tm := range teams {
		teamNumberByID[tm.ID.String()] = tm.TeamNumber
	}

	objs, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Agent health per object, computed exactly the way the matrix and
	// build detail compute it (attachAgentHealth), so "failed check-in"
	// here means the same thing it does everywhere else in the UI.
	views, err := attachAgentHealth(r.Context(), s.Queries, build.ID, objs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Which objects have a failed step (a failed agent_task), so a
	// step-level failure is told apart from an infra one.
	failedStepIDs, err := s.Queries.ListFailedStepObjectIDsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	failedSteps := make(map[string]bool, len(failedStepIDs))
	for _, id := range failedStepIDs {
		failedSteps[id.String()] = true
	}

	data := dashboardData{
		ProvisioningByStatus: map[string]int{},
		ByTeam:               []teamStateCounts{},
		FailuresByCause:      []failureGroup{},
		AgentActivity:        []activityBucket{},
	}
	byTeam := map[int32]map[string]int{}
	byTeamHealth := map[int32]map[string]int{}
	failureIndex := map[string]*failureGroup{}

	for _, o := range views {
		data.ProvisioningByStatus[o.Status]++

		teamNumber := teamNumberByID[o.TeamID.String()]
		if byTeam[teamNumber] == nil {
			byTeam[teamNumber] = map[string]int{}
			byTeamHealth[teamNumber] = map[string]int{}
		}
		byTeam[teamNumber][o.Status]++
		if h := classifyHealth(o, failedSteps); h != "" {
			byTeamHealth[teamNumber][h]++
		}

		if isFailedStatus(o.Status) && o.LastError != nil && *o.LastError != "" {
			g, ok := failureIndex[*o.LastError]
			if !ok {
				g = &failureGroup{Message: *o.LastError}
				failureIndex[*o.LastError] = g
			}
			g.Count++
			name := db.StrOrEmpty(o.AsName)
			if name == "" {
				name = o.ObjectName
			}
			g.Objects = append(g.Objects, name)
		}
	}

	teamNumbers := make([]int32, 0, len(byTeam))
	for tn := range byTeam {
		teamNumbers = append(teamNumbers, tn)
	}
	sort.Slice(teamNumbers, func(i, j int) bool { return teamNumbers[i] < teamNumbers[j] })
	for _, tn := range teamNumbers {
		data.ByTeam = append(data.ByTeam, teamStateCounts{TeamNumber: tn, ByStatus: byTeam[tn], ByHealth: byTeamHealth[tn]})
	}

	for _, g := range failureIndex {
		data.FailuresByCause = append(data.FailuresByCause, *g)
	}
	sort.Slice(data.FailuresByCause, func(i, j int) bool { return data.FailuresByCause[i].Count > data.FailuresByCause[j].Count })

	heartbeats, err := s.Queries.ListAgentHeartbeatsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	data.AgentActivity = bucketHeartbeats(heartbeats)

	writeJSON(w, http.StatusOK, data)
}
