package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agenthealth"
	"github.com/globalcptc/laforge/internal/db"
)

func (s *Server) registerTools() {
	addTool(s, "whoami", "Who Am I",
		"Shows which LaForge account and API key this connection uses, when the key expires, and which repositories it can read.",
		toolWhoami)
	addTool(s, "list_repos", "List Repositories",
		"Lists the content repositories this API key can read. Each repository's builds are listed by list_builds.",
		toolListRepos)
	addTool(s, "list_builds", "List Builds",
		"Lists a repository's builds, newest first: status, environment, branch and commit, builder, team count.",
		toolListBuilds)
	addTool(s, "get_build", "Get Build",
		"Summarises one build: its repository, commit, environment and builder; its teams and their access state; and counts of its deployed objects by status, kind, failure kind, agent health and cluster location. Use list_objects to see individual hosts.",
		toolGetBuild)
	addTool(s, "list_objects", "List Deployed Objects",
		"Lists a build's deployed hosts, containers, networks and DNS records with their status, power state, agent health and last check-in, the cluster member or server each runs on (location), public address and last error. Filter by team, kind, status, agent status, location or name.",
		toolListObjects)
	s.registerDiagTools()
	s.registerPageTools()
	s.registerPowerTool()
}

// --- whoami ---

type whoamiIn struct{}

type repoRef struct {
	ID   string `json:"id"`
	Name string `json:"name" jsonschema:"owner/repository"`
}

type whoamiOut struct {
	Login        string    `json:"login"`
	KeyName      string    `json:"key_name"`
	KeyPrefix    string    `json:"key_prefix"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
	Repositories []repoRef `json:"repositories"`
	// LimitedToRepositories is true when the key itself is limited to a list
	// of repositories, rather than seeing everything its owner can.
	LimitedToRepositories bool `json:"limited_to_repositories"`
}

func toolWhoami(ctx context.Context, c *call, _ whoamiIn) (whoamiOut, error) {
	repos, err := c.repos(ctx)
	if err != nil {
		return whoamiOut{}, err
	}
	return whoamiOut{
		Login: c.key.Login, KeyName: c.key.Name, KeyPrefix: c.key.Prefix, Scope: c.key.Scope,
		ExpiresAt: c.key.ExpiresAt, Repositories: repos, LimitedToRepositories: len(c.key.Repos) > 0,
	}, nil
}

// repos is every readable repository, by name.
func (c *call) repos(ctx context.Context) ([]repoRef, error) {
	ids := make([]pgtype.UUID, 0, len(c.readable))
	for id := range c.readable {
		ids = append(ids, id)
	}
	rows, err := c.q().McpListRepositories(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]repoRef, len(rows))
	for i, r := range rows {
		out[i] = repoRef{ID: r.ID.String(), Name: r.GithubOwner + "/" + r.GithubRepo}
	}
	return out, nil
}

// --- list_repos ---

type listReposIn struct{}

type listReposOut struct {
	Repositories []repoRef `json:"repositories"`
}

func toolListRepos(ctx context.Context, c *call, _ listReposIn) (listReposOut, error) {
	repos, err := c.repos(ctx)
	return listReposOut{Repositories: repos}, err
}

// repo resolves a repository given as an id or as owner/name.
func (c *call) repo(ctx context.Context, ref string) (repoRef, error) {
	ref = strings.TrimSpace(ref)
	repos, err := c.repos(ctx)
	if err != nil {
		return repoRef{}, err
	}
	for _, r := range repos {
		if strings.EqualFold(r.ID, ref) || strings.EqualFold(r.Name, ref) {
			return r, nil
		}
	}
	return repoRef{}, errDenied
}

// --- list_builds ---

type listBuildsIn struct {
	Repository string `json:"repository" jsonschema:"the repository, as owner/name or its id"`
	Status     string `json:"status,omitempty" jsonschema:"only builds with this status: planned, deploying, building, finished, failed, tearing_down, torn_down, purged or discarded"`
	Limit      int    `json:"limit,omitempty" jsonschema:"how many builds to return, newest first (default 20, at most 100)"`
}

type buildRow struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"`
	Environment    string    `json:"environment"`
	Branch         string    `json:"branch,omitempty"`
	Commit         string    `json:"commit"`
	CommitMessage  string    `json:"commit_message,omitempty"`
	Builder        string    `json:"builder,omitempty"`
	Teams          int64     `json:"teams"`
	AutoBuilt      bool      `json:"auto_built"`
	ReconcileError string    `json:"reconcile_error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type listBuildsOut struct {
	Repository string     `json:"repository"`
	Builds     []buildRow `json:"builds"`
}

func toolListBuilds(ctx context.Context, c *call, in listBuildsIn) (listBuildsOut, error) {
	repo, err := c.repo(ctx, in.Repository)
	if err != nil {
		return listBuildsOut{}, err
	}
	var id pgtype.UUID
	if err := id.Scan(repo.ID); err != nil {
		return listBuildsOut{}, err
	}
	var status *string
	if in.Status != "" {
		status = &in.Status
	}
	rows, err := c.q().McpListBuildsByRepository(ctx, db.McpListBuildsByRepositoryParams{
		RepositoryID: id, Status: status, MaxRows: int32(clamp(in.Limit, 20, 100)),
	})
	if err != nil {
		return listBuildsOut{}, err
	}
	out := listBuildsOut{Repository: repo.Name, Builds: make([]buildRow, len(rows))}
	for i, b := range rows {
		out.Builds[i] = buildRow{
			ID: b.ID.String(), Status: b.Status, Environment: b.EnvironmentName, Branch: str(b.Branch),
			Commit: short(b.CommitSha, 12), CommitMessage: firstLine(str(b.CommitMessage)), Builder: str(b.BuilderConfigName),
			Teams: b.TeamCount, AutoBuilt: b.AutoBuilt, ReconcileError: str(b.ReconcileError), CreatedAt: b.CreatedAt.Time,
		}
	}
	return out, nil
}

// --- get_build ---

type buildIn struct {
	BuildID string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
}

type teamRow struct {
	Team          int32      `json:"team"`
	Access        string     `json:"access" jsonschema:"open or closed, after any override"`
	Override      string     `json:"override,omitempty"`
	OverrideUntil *time.Time `json:"override_until,omitempty"`
}

type getBuildOut struct {
	ID                 string           `json:"id"`
	Repository         string           `json:"repository"`
	Status             string           `json:"status"`
	ReconcileError     string           `json:"reconcile_error,omitempty"`
	Environment        string           `json:"environment"`
	EnvironmentPath    string           `json:"environment_path,omitempty"`
	Description        string           `json:"description,omitempty"`
	StartAt            *time.Time       `json:"start_at,omitempty"`
	StopAt             *time.Time       `json:"stop_at,omitempty"`
	AccessWindows      any              `json:"access_windows,omitempty"`
	Branch             string           `json:"branch,omitempty"`
	Commit             string           `json:"commit"`
	CommitMessage      string           `json:"commit_message,omitempty"`
	Builder            string           `json:"builder,omitempty"`
	BuilderKind        string           `json:"builder_kind,omitempty"`
	CompetitionStarted bool             `json:"competition_started"`
	AutoBuilt          bool             `json:"auto_built"`
	CreatedAt          time.Time        `json:"created_at"`
	Teams              []teamRow        `json:"teams"`
	Objects            int              `json:"objects"`
	ByStatus           map[string]int   `json:"objects_by_status"`
	ByKind             map[string]int   `json:"objects_by_kind"`
	ByFailureKind      map[string]int   `json:"objects_by_failure_kind,omitempty"`
	ByAgentStatus      map[string]int   `json:"agents_by_status" jsonschema:"hosts and containers whose instance is up, by agent health: healthy, late, missing or booting"`
	ByLocation         []locationCounts `json:"by_location" jsonschema:"objects per cluster member or server (location), with how many are failed and how many agents are unhealthy there -- a lopsided location points at a sick member"`
}

type locationCounts struct {
	Location      string `json:"location" jsonschema:"cluster member or server; empty when the builder hasn't reported one yet"`
	Objects       int    `json:"objects"`
	Failed        int    `json:"failed"`
	AgentsHealthy int    `json:"agents_healthy"`
	AgentsLate    int    `json:"agents_late"`
	AgentsMissing int    `json:"agents_missing"`
	AgentsBooting int    `json:"agents_booting"`
}

func toolGetBuild(ctx context.Context, c *call, in buildIn) (getBuildOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return getBuildOut{}, err
	}
	b, err := c.q().McpGetBuild(ctx, id)
	if err != nil {
		return getBuildOut{}, err
	}
	out := getBuildOut{
		ID: b.ID.String(), Repository: b.GithubOwner + "/" + b.GithubRepo, Status: b.Status,
		ReconcileError: str(b.ReconcileError), Environment: b.EnvironmentName, Branch: str(b.Branch),
		Commit: short(b.CommitSha, 12), CommitMessage: firstLine(str(b.CommitMessage)), Builder: str(b.BuilderConfigName),
		BuilderKind: str(b.BuilderKind), CompetitionStarted: b.CompetitionStarted != nil && *b.CompetitionStarted,
		AutoBuilt: b.AutoBuilt, CreatedAt: b.CreatedAt.Time,
		ByStatus: map[string]int{}, ByKind: map[string]int{}, ByFailureKind: map[string]int{}, ByAgentStatus: map[string]int{},
	}
	if env, err := c.q().McpGetBuildEnvironment(ctx, db.McpGetBuildEnvironmentParams{ContentRevisionID: b.ContentRevisionID, Name: b.EnvironmentName}); err == nil {
		out.EnvironmentPath, out.Description = env.Path, str(env.Description)
		out.StartAt, out.StopAt = ts(env.StartAt), ts(env.StopAt)
		if len(env.Access) > 0 && string(env.Access) != "[]" {
			out.AccessWindows = jsonValue(env.Access)
		}
	}
	teams, err := c.q().McpListTeamsByBuild(ctx, id)
	if err != nil {
		return getBuildOut{}, err
	}
	now := time.Now()
	for _, t := range teams {
		row := teamRow{Team: t.TeamNumber, Access: t.AccessState, Override: t.AccessOverrideState, OverrideUntil: ts(t.AccessOverrideUntil)}
		if t.AccessOverrideState != "" && (!t.AccessOverrideUntil.Valid || t.AccessOverrideUntil.Time.After(now)) {
			row.Access = t.AccessOverrideState
		}
		out.Teams = append(out.Teams, row)
	}
	objs, err := c.q().McpListObjectsByBuild(ctx, id)
	if err != nil {
		return getBuildOut{}, err
	}
	locs := map[string]*locationCounts{}
	for _, o := range objs {
		out.Objects++
		out.ByStatus[o.Status]++
		out.ByKind[o.Kind]++
		if o.FailureKind != nil {
			out.ByFailureKind[*o.FailureKind]++
		}
		l := locs[o.InstanceLocation]
		if l == nil {
			l = &locationCounts{Location: o.InstanceLocation}
			locs[o.InstanceLocation] = l
		}
		l.Objects++
		if strings.HasSuffix(o.Status, "failed") {
			l.Failed++
		}
		if status, _, ok := agentStatus(o, now); ok {
			out.ByAgentStatus[status]++
			switch status {
			case agenthealth.StatusHealthy:
				l.AgentsHealthy++
			case agenthealth.StatusLate:
				l.AgentsLate++
			case agenthealth.StatusMissing:
				l.AgentsMissing++
			case agenthealth.StatusBooting:
				l.AgentsBooting++
			}
		}
	}
	for _, l := range locs {
		out.ByLocation = append(out.ByLocation, *l)
	}
	sort.Slice(out.ByLocation, func(i, j int) bool { return out.ByLocation[i].Location < out.ByLocation[j].Location })
	if len(out.ByFailureKind) == 0 {
		out.ByFailureKind = nil
	}
	return out, nil
}

func agentStatus(o db.McpListObjectsByBuildRow, now time.Time) (string, time.Time, bool) {
	in := agenthealth.Object{Kind: o.Kind, Status: o.Status, CheckinOverdue: o.AgentCheckinOverdueAt.Valid}
	if o.DeployedAt.Valid {
		in.DeployedAt = o.DeployedAt.Time
	}
	if o.LastHeartbeatAt.Valid {
		in.LastHeartbeat = o.LastHeartbeatAt.Time
	}
	return agenthealth.Classify(in, now)
}

// --- list_objects ---

type listObjectsIn struct {
	BuildID     string `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Team        int    `json:"team,omitempty" jsonschema:"only this team number"`
	Kind        string `json:"kind,omitempty" jsonschema:"only this kind: host, container, network or dns"`
	Status      string `json:"status,omitempty" jsonschema:"only this status, e.g. finished, building, build_failed, deploy_failed; 'failed' matches every failed status"`
	AgentStatus string `json:"agent_status,omitempty" jsonschema:"only objects whose agent is healthy, late, missing or booting"`
	Location    string `json:"location,omitempty" jsonschema:"only objects on this cluster member or server"`
	Name        string `json:"name,omitempty" jsonschema:"only objects whose name contains this text"`
	Limit       int    `json:"limit,omitempty" jsonschema:"how many objects to return (default 200, at most 500)"`
}

type objectRow struct {
	ID                string     `json:"id"`
	Team              int32      `json:"team"`
	Kind              string     `json:"kind"`
	Name              string     `json:"name"`
	Network           string     `json:"network,omitempty"`
	Status            string     `json:"status"`
	FailureKind       string     `json:"failure_kind,omitempty"`
	FailedAt          *time.Time `json:"failed_at,omitempty"`
	LastError         string     `json:"last_error,omitempty"`
	BlockedOn         any        `json:"blocked_on,omitempty"`
	PowerState        string     `json:"power_state,omitempty"`
	PowerCheckedAt    *time.Time `json:"power_checked_at,omitempty"`
	Location          string     `json:"location,omitempty"`
	PublicAddress     string     `json:"public_address,omitempty"`
	ExternalRef       string     `json:"external_ref,omitempty"`
	AgentStatus       string     `json:"agent_status,omitempty"`
	AgentPlatform     string     `json:"agent_platform,omitempty"`
	LastHeartbeatAt   *time.Time `json:"last_heartbeat_at,omitempty"`
	SecondsSinceCheck *int64     `json:"seconds_since_heartbeat,omitempty"`
	BootAt            *time.Time `json:"boot_at,omitempty"`
	AgentStartedAt    *time.Time `json:"agent_started_at,omitempty"`
	DeployedAt        *time.Time `json:"deployed_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type listObjectsOut struct {
	Build     string      `json:"build"`
	Matched   int         `json:"matched" jsonschema:"how many objects matched the filters"`
	Truncated bool        `json:"truncated" jsonschema:"true when more matched than the limit returned"`
	Objects   []objectRow `json:"objects"`
}

func toolListObjects(ctx context.Context, c *call, in listObjectsIn) (listObjectsOut, error) {
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return listObjectsOut{}, err
	}
	objs, err := c.q().McpListObjectsByBuild(ctx, id)
	if err != nil {
		return listObjectsOut{}, err
	}
	limit := clamp(in.Limit, 200, 500)
	out := listObjectsOut{Build: id.String(), Objects: []objectRow{}}
	now := time.Now()
	for _, o := range objs {
		name := o.ObjectName
		if o.AsName != nil && *o.AsName != "" {
			name = *o.AsName
		}
		agent, last, hasAgent := agentStatus(o, now)
		switch {
		case in.Team != 0 && int(o.TeamNumber) != in.Team,
			in.Kind != "" && !strings.EqualFold(o.Kind, in.Kind),
			in.Status == "failed" && !strings.HasSuffix(o.Status, "failed"),
			in.Status != "" && in.Status != "failed" && !strings.EqualFold(o.Status, in.Status),
			in.AgentStatus != "" && (!hasAgent || !strings.EqualFold(agent, in.AgentStatus)),
			in.Location != "" && !strings.EqualFold(o.InstanceLocation, in.Location),
			in.Name != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(in.Name)):
			continue
		}
		out.Matched++
		if len(out.Objects) >= limit {
			out.Truncated = true
			continue
		}
		row := objectRow{
			ID: o.ID.String(), Team: o.TeamNumber, Kind: o.Kind, Name: name, Network: str(o.NetworkName),
			Status: o.Status, FailureKind: str(o.FailureKind), FailedAt: ts(o.FailedAt), LastError: clip(str(o.LastError), 500),
			PowerState: o.PowerState, PowerCheckedAt: ts(o.PowerStateCheckedAt), Location: o.InstanceLocation,
			PublicAddress: o.PublicAddress, ExternalRef: str(o.ExternalRef), AgentPlatform: o.AgentPlatform,
			BootAt: ts(o.BootAt), AgentStartedAt: ts(o.AgentStartedAt), DeployedAt: ts(o.DeployedAt), UpdatedAt: o.UpdatedAt.Time,
		}
		if len(o.BlockedOn) > 0 && string(o.BlockedOn) != "null" {
			row.BlockedOn = jsonValue(o.BlockedOn)
		}
		if hasAgent {
			row.AgentStatus = agent
			if !last.IsZero() {
				row.LastHeartbeatAt = &last
				secs := int64(now.Sub(last).Seconds())
				row.SecondsSinceCheck = &secs
			}
		}
		out.Objects = append(out.Objects, row)
	}
	return out, nil
}

// --- helpers ---

// jsonValue decodes a JSON column for a tool result (a json.RawMessage would
// be described to clients as an array of bytes).
func jsonValue(raw []byte) any {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	return v
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ts(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func short(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return clip(line, 200)
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n] + fmt.Sprintf("… (%d more bytes)", len(s)-n)
	}
	return s
}

func clamp(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}
