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
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
// states (destroying/destroy_failed/destroyed) return "" and are dropped by the caller:
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
	case "destroy_queued", "destroying", "destroy_failed", "destroyed":
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

// resourceBucket is the build-wide AVERAGE of each host metric over one time
// bucket, across every heartbeat in it -- the data behind the CPU / memory /
// disk / network graphs. Percentages are 0-100; network is bytes/sec (rx =
// download, tx = upload). A metric missing from a heartbeat (older agent, or a
// metric it couldn't sample) is left out of its own average, not counted as 0.
type resourceBucket struct {
	At    string  `json:"at"` // bucket start, RFC3339
	Cpu   float64 `json:"cpu"`
	Mem   float64 `json:"mem"`
	Disk  float64 `json:"disk"`
	NetRx float64 `json:"net_rx"`
	NetTx float64 `json:"net_tx"`
}

type dashboardData struct {
	ProvisioningByStatus map[string]int    `json:"provisioning_by_status"`
	ByTeam               []teamStateCounts `json:"by_team"`
	FailuresByCause      []failureGroup    `json:"failures_by_cause"`
	AgentActivity        []activityBucket  `json:"agent_activity"`
	ResourceUsage        []resourceBucket  `json:"resource_usage"`
}

// maxChartBuckets is how many time buckets the dashboard's agent-activity and
// resource charts are summarized into.
const maxChartBuckets = 40

// bucketLayout spreads a build's heartbeat history, first to last, over even
// time buckets: each bucket's width and how many there are (capped). The
// width is whole microseconds, the database's timestamp resolution, so the
// buckets the database fills (AgentHeartbeatBucketsByBuild) line up exactly
// with the labels written here. Shared by both charts so they share an x-axis.
func bucketLayout(first, last time.Time) (width time.Duration, count int) {
	span := last.Sub(first)
	if span <= 0 {
		span = time.Second
	}
	width = (span / maxChartBuckets).Truncate(time.Microsecond)
	if width <= 0 {
		width = time.Microsecond
	}
	count = int(span/width) + 1
	if count > maxChartBuckets {
		count = maxChartBuckets
	}
	return width, count
}

// chartBuckets turns the database's per-bucket summaries into the two chart
// series, one point per bucket. A bucket with no heartbeats reads as 0 active
// and 0 for every metric.
//
// agent activity counts DISTINCT objects seen in each bucket, not heartbeats,
// so a batch of agents going quiet shows up as a drop rather than tracking
// agent count x poll rate. Resource usage is the average of each metric over
// the bucket's heartbeats, a missing metric left out of its own average.
func chartBuckets(first time.Time, width time.Duration, count int, rows []db.AgentHeartbeatBucketsByBuildRow) ([]activityBucket, []resourceBucket) {
	activity := make([]activityBucket, count)
	resources := make([]resourceBucket, count)
	for i := 0; i < count; i++ {
		at := first.Add(time.Duration(i) * width).Format(time.RFC3339)
		activity[i] = activityBucket{At: at}
		resources[i] = resourceBucket{At: at}
	}
	for _, r := range rows {
		i := int(r.Bucket)
		if i < 0 || i >= count {
			continue
		}
		activity[i].Active = int(r.Active)
		resources[i].Cpu, resources[i].Mem, resources[i].Disk = r.Cpu, r.Mem, r.Disk
		resources[i].NetRx, resources[i].NetTx = r.NetRx, r.NetTx
	}
	return activity, resources
}

// heartbeatCharts is the dashboard's agent-activity and resource-usage series
// for a build. The database summarizes the heartbeat log into at most
// maxChartBuckets rows; loading the raw log instead held millions of rows (3
// GB and more) in the API per build by the end of an event. Summarizing still
// reads every row (seconds, by then), so a build's series is reused for
// chartCacheTTL: each bucket spans the history / maxChartBuckets (hours, mid-
// event), so a minute-old series looks the same as a fresh one. The rest of
// the dashboard (statuses, failures) is computed fresh as before.
func (s *Server) heartbeatCharts(ctx context.Context, buildID pgtype.UUID) ([]activityBucket, []resourceBucket, error) {
	if a, r, ok := s.charts.get(buildID); ok {
		return a, r, nil
	}
	a, r, err := s.computeHeartbeatCharts(ctx, buildID)
	if err == nil && len(a) > 0 { // nothing to chart yet costs one quick query; don't hold it
		s.charts.put(buildID, a, r)
	}
	return a, r, err
}

func (s *Server) computeHeartbeatCharts(ctx context.Context, buildID pgtype.UUID) ([]activityBucket, []resourceBucket, error) {
	span, err := s.Queries.AgentHeartbeatSpanByBuild(ctx, buildID)
	if err != nil {
		return nil, nil, err
	}
	if !span.FirstAt.Valid || !span.LastAt.Valid {
		return []activityBucket{}, []resourceBucket{}, nil
	}
	width, count := bucketLayout(span.FirstAt.Time, span.LastAt.Time)
	rows, err := s.Queries.AgentHeartbeatBucketsByBuild(ctx, db.AgentHeartbeatBucketsByBuildParams{
		FirstAt: span.FirstAt, WidthUs: width.Microseconds(), Buckets: int32(count), BuildID: buildID,
	})
	if err != nil {
		return nil, nil, err
	}
	activity, resources := chartBuckets(span.FirstAt.Time, width, count, rows)
	return activity, resources, nil
}

// chartCacheTTL is how long a build's heartbeat chart series is reused.
const chartCacheTTL = time.Minute

// chartCache keeps each build's most recent heartbeat chart series.
type chartCache struct {
	mu      sync.Mutex
	entries map[pgtype.UUID]chartEntry
}

type chartEntry struct {
	at        time.Time
	activity  []activityBucket
	resources []resourceBucket
}

func (c *chartCache) get(buildID pgtype.UUID) ([]activityBucket, []resourceBucket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[buildID]
	if !ok || time.Since(e.at) >= chartCacheTTL {
		return nil, nil, false
	}
	return e.activity, e.resources, true
}

func (c *chartCache) put(buildID pgtype.UUID, activity []activityBucket, resources []resourceBucket) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[pgtype.UUID]chartEntry{}
	}
	for id, e := range c.entries { // drop builds nobody has looked at lately
		if time.Since(e.at) >= 10*chartCacheTTL {
			delete(c.entries, id)
		}
	}
	c.entries[buildID] = chartEntry{at: time.Now(), activity: activity, resources: resources}
}

func (s *Server) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	buildID, repo, err := s.buildRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	// The same for everyone who may read the build: computed once and
	// shared by every open page polling it (sharedread.go).
	s.shared.serve(w, r, "dashboard:"+buildID.String(), func(w http.ResponseWriter, r *http.Request) {
		build, err := s.Queries.GetBuild(r.Context(), buildID)
		if err != nil {
			writeError(w, http.StatusNotFound, errors.New("no such build"))
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
			ResourceUsage:        []resourceBucket{},
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

		data.AgentActivity, data.ResourceUsage, err = s.heartbeatCharts(r.Context(), build.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, data)
	})
}
