package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/globalcptc/laforge/internal/db"
)

// runMCP serves the CLI's server-backed operations as an MCP server over
// stdio, so an assistant (e.g. `claude mcp add laforge -- laforge mcp`) can
// check host status and run power actions on a deployed build. It is the same
// API, token store, and target selector the other commands use -- nothing
// here talks to the hoster or the database directly. stdout belongs to the
// protocol, so nothing in this path may print to it.
func runMCP(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("usage: laforge mcp (takes no arguments; set LAFORGE_API_URL)")
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "laforge", Version: version}, nil)

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_repos",
		Description: "List the repositories registered with LaForge, as owner/repo.",
		Annotations: readOnly,
	}, mcpListRepos)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_builds",
		Description: "List a repository's builds, newest first: build id, environment, status, builder, whether the competition has started (locked), and when it was created. The build id is what every other tool takes. Torn-down and discarded builds are hidden unless include_old=true.",
		Annotations: readOnly,
	}, mcpListBuilds)
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_hosts",
		Description: "List the deployed hosts and containers in a build with their status: team, name, id, build status, power state, " +
			"agent health (healthy | late | missing | booting) and last heartbeat, public address, and last error. " +
			"Filter by team, name, or problems_only to see just the hosts that need attention.",
		Annotations: readOnly,
	}, mcpListHosts)
	mcp.AddTool(s, &mcp.Tool{
		Name: "host_events",
		Description: "The newest events for one host: deploy steps, power-action results, errors. Power actions run in the " +
			"background, so this is where each host's start/stop/reboot outcome shows up. Pick the host by id, or by team plus name.",
		Annotations: readOnly,
	}, mcpHostEvents)
	mcp.AddTool(s, &mcp.Tool{
		Name: "power",
		Description: "Start, stop, or reboot hosts at the hoster (works even if the host's agent or OS is wedged). " +
			"preview=true lists the hosts the target matches and changes nothing -- always preview first and confirm the set with the user. " +
			"force=true is a hard power operation instead of a graceful OS-level one. The action runs in the background: " +
			"check list_hosts (power_state) or host_events afterwards for each host's result. Needs manage access on the repo.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true)},
	}, mcpPower)

	return s.Run(context.Background(), &mcp.StdioTransport{})
}

func ptr[T any](v T) *T { return &v }

// jsonResult hands a value back as the tool's text content, pretty-printed:
// what the model reads.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

type noInput struct{}

func mcpListRepos(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, any, error) {
	var repos []db.Repository
	if err := newAPIClient().do("GET", "/repos", nil, &repos); err != nil {
		return nil, nil, err
	}
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		out = append(out, r.GithubOwner+"/"+r.GithubRepo)
	}
	sort.Strings(out)
	return jsonResult(out)
}

type listBuildsInput struct {
	Repo       string `json:"repo" jsonschema:"the repository as owner/repo (see list_repos)"`
	IncludeOld bool   `json:"include_old,omitempty" jsonschema:"also list torn-down and discarded builds (hidden by default)"`
}

func mcpListBuilds(_ context.Context, _ *mcp.CallToolRequest, in listBuildsInput) (*mcp.CallToolResult, any, error) {
	c := newAPIClient()
	repo, err := resolveRepoID(c, in.Repo)
	if err != nil {
		return nil, nil, err
	}
	var builds []db.Build
	if err := c.do("GET", "/repos/"+repo.ID.String()+"/builds", nil, &builds); err != nil {
		return nil, nil, err
	}
	// The builder and the competition-started lock live on the configured
	// build behind each build.
	var cbs []db.ConfiguredBuild
	if err := c.do("GET", "/repos/"+repo.ID.String()+"/configured-builds", nil, &cbs); err != nil {
		return nil, nil, err
	}
	cbByID := make(map[string]db.ConfiguredBuild, len(cbs))
	for _, cb := range cbs {
		cbByID[cb.ID.String()] = cb
	}
	sort.Slice(builds, func(i, j int) bool { return builds[i].CreatedAt.Time.After(builds[j].CreatedAt.Time) })
	type row struct {
		ID                 string `json:"id"`
		Environment        string `json:"environment"`
		Status             string `json:"status"`
		Builder            string `json:"builder,omitempty"`
		CompetitionStarted bool   `json:"competition_started,omitempty"`
		CreatedAt          string `json:"created_at"`
		Error              string `json:"error,omitempty"`
	}
	out := make([]row, 0, len(builds))
	for _, b := range builds {
		if !in.IncludeOld && (b.Status == "torn_down" || b.Status == "discarded") {
			continue
		}
		r := row{ID: b.ID.String(), Environment: b.EnvironmentName, Status: b.Status, CreatedAt: b.CreatedAt.Time.Format(time.RFC3339)}
		if cb, ok := cbByID[b.ConfiguredBuildID.String()]; ok {
			r.Builder = cb.BuilderConfigName
			r.CompetitionStarted = cb.CompetitionStarted
		}
		if b.ReconcileError != nil {
			r.Error = *b.ReconcileError
		}
		out = append(out, r)
	}
	return jsonResult(out)
}

// buildObject is one row of GET /builds/{id}/objects: db.DeployedObject plus
// the team number and agent health the api attaches (internal/api/health.go).
type buildObject struct {
	db.DeployedObject
	TeamNumber int `json:"team_number"`
	Agent      *struct {
		LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
		Status          string     `json:"agent_status"`
	} `json:"agent"`
}

func (o buildObject) name() string {
	if o.AsName != nil && *o.AsName != "" {
		return *o.AsName
	}
	return o.ObjectName
}

// needsAttention is list_hosts' problems_only: a failed build step, an
// error recorded, an agent that stopped checking in, or a box that's off.
func (o buildObject) needsAttention() bool {
	if o.LastError != nil && *o.LastError != "" {
		return true
	}
	if strings.Contains(o.Status, "fail") || o.Status == "invalid" {
		return true
	}
	if o.Agent != nil && (o.Agent.Status == "late" || o.Agent.Status == "missing") {
		return true
	}
	return o.PowerState != "" && o.PowerState != "running"
}

func listBuildObjects(c *apiClient, build string) ([]buildObject, error) {
	var objs []buildObject
	if err := c.do("GET", "/builds/"+url.PathEscape(build)+"/objects", nil, &objs); err != nil {
		return nil, err
	}
	hosts := objs[:0]
	for _, o := range objs {
		if o.Kind == "host" || o.Kind == "container" {
			hosts = append(hosts, o)
		}
	}
	sort.Slice(hosts, func(i, j int) bool {
		if hosts[i].TeamNumber != hosts[j].TeamNumber {
			return hosts[i].TeamNumber < hosts[j].TeamNumber
		}
		return hosts[i].name() < hosts[j].name()
	})
	return hosts, nil
}

type listHostsInput struct {
	Build        string `json:"build" jsonschema:"build id (see list_builds)"`
	Team         int    `json:"team,omitempty" jsonschema:"only this team number"`
	Search       string `json:"search,omitempty" jsonschema:"only hosts whose name contains this text"`
	ProblemsOnly bool   `json:"problems_only,omitempty" jsonschema:"only hosts with a failed status, an error, a late or missing agent, or not powered on"`
}

type hostRow struct {
	Team          int    `json:"team"`
	Name          string `json:"name"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Network       string `json:"network,omitempty"`
	Status        string `json:"status"`
	PowerState    string `json:"power_state,omitempty"`
	Agent         string `json:"agent,omitempty"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
	PublicAddress string `json:"public_address,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

func toHostRow(o buildObject) hostRow {
	r := hostRow{Team: o.TeamNumber, Name: o.name(), ID: o.ID.String(), Kind: o.Kind, Status: o.Status, PowerState: o.PowerState, PublicAddress: o.PublicAddress}
	if o.NetworkName != nil {
		r.Network = *o.NetworkName
	}
	if o.LastError != nil {
		r.LastError = *o.LastError
	}
	if o.Agent != nil {
		r.Agent = o.Agent.Status
		if o.Agent.LastHeartbeatAt != nil {
			r.LastHeartbeat = o.Agent.LastHeartbeatAt.Format(time.RFC3339)
		}
	}
	return r
}

func mcpListHosts(_ context.Context, _ *mcp.CallToolRequest, in listHostsInput) (*mcp.CallToolResult, any, error) {
	objs, err := listBuildObjects(newAPIClient(), in.Build)
	if err != nil {
		return nil, nil, err
	}
	rows := []hostRow{}
	byStatus, byAgent := map[string]int{}, map[string]int{}
	for _, o := range objs {
		if in.Team != 0 && o.TeamNumber != in.Team {
			continue
		}
		if in.Search != "" && !strings.Contains(strings.ToLower(o.name()), strings.ToLower(in.Search)) {
			continue
		}
		byStatus[o.Status]++
		if o.Agent != nil {
			byAgent[o.Agent.Status]++
		}
		if in.ProblemsOnly && !o.needsAttention() {
			continue
		}
		rows = append(rows, toHostRow(o))
	}
	return jsonResult(map[string]any{
		"summary": map[string]any{"by_status": byStatus, "by_agent": byAgent},
		"hosts":   rows,
	})
}

type hostEventsInput struct {
	Build string `json:"build" jsonschema:"build id"`
	ID    string `json:"id,omitempty" jsonschema:"the host's id (from list_hosts); or give team and host instead"`
	Team  int    `json:"team,omitempty" jsonschema:"team number, with host"`
	Host  string `json:"host,omitempty" jsonschema:"the host's name, with team"`
	Limit int    `json:"limit,omitempty" jsonschema:"how many of the newest events to return (default 20)"`
}

func mcpHostEvents(_ context.Context, _ *mcp.CallToolRequest, in hostEventsInput) (*mcp.CallToolResult, any, error) {
	id := in.ID
	if id == "" {
		if in.Team == 0 || in.Host == "" {
			return nil, nil, errors.New("give the host's id, or team and host")
		}
		var err error
		if id, err = resolveShellObject(in.Build, in.Team, in.Host); err != nil {
			return nil, nil, err
		}
	}
	var events []db.Event
	path := "/builds/" + url.PathEscape(in.Build) + "/objects/" + url.PathEscape(id) + "/events"
	if err := newAPIClient().do("GET", path, nil, &events); err != nil {
		return nil, nil, err
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.Time.After(events[j].CreatedAt.Time) })
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if len(events) > limit {
		events = events[:limit]
	}
	type row struct {
		At      string `json:"at"`
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}
	out := make([]row, 0, len(events))
	for _, e := range events {
		out = append(out, row{At: e.CreatedAt.Time.Format(time.RFC3339), Kind: e.Kind, Message: e.Message})
	}
	return jsonResult(out)
}

type powerInput struct {
	Build    string            `json:"build" jsonschema:"build id"`
	Action   string            `json:"action" jsonschema:"start, stop, or reboot"`
	Preview  bool              `json:"preview,omitempty" jsonschema:"list the matched hosts and change nothing"`
	Force    bool              `json:"force,omitempty" jsonschema:"hard power operation instead of a graceful OS-level one"`
	IDs      []string          `json:"ids,omitempty" jsonschema:"exact host ids (from list_hosts); when given, the other selectors are ignored"`
	Team     int               `json:"team,omitempty" jsonschema:"every host in this team number"`
	Search   string            `json:"search,omitempty" jsonschema:"hosts whose name contains this text"`
	Network  string            `json:"network,omitempty" jsonschema:"hosts on this network"`
	Kind     string            `json:"kind,omitempty" jsonschema:"host or container"`
	Tags     map[string]string `json:"tags,omitempty" jsonschema:"hosts carrying every one of these tags; an empty value matches any value"`
	AllHosts bool              `json:"all_hosts,omitempty" jsonschema:"required to act with no selector at all, i.e. on every host in every team"`
}

func (in powerInput) target() cliTarget {
	t := cliTarget{IDs: in.IDs, Kind: in.Kind, Search: in.Search, Network: in.Network, Tags: in.Tags}
	if in.Team > 0 {
		tm := in.Team
		t.Team = &tm
	}
	return t
}

func mcpPower(_ context.Context, _ *mcp.CallToolRequest, in powerInput) (*mcp.CallToolResult, any, error) {
	switch in.Action {
	case "start", "stop", "reboot":
	default:
		return nil, nil, errors.New("action must be start, stop, or reboot")
	}
	target := in.target()
	if len(target.IDs) == 0 && target.Team == nil && target.Kind == "" && target.Search == "" &&
		target.Network == "" && len(target.Tags) == 0 && !in.AllHosts {
		return nil, nil, errors.New("no selector given: this would hit every host in every team -- pass ids, team, search, network, kind or tags, or all_hosts=true if that's really meant")
	}
	c := newAPIClient()
	build := url.PathEscape(in.Build)

	if in.Preview {
		// The power endpoint has no dry run, but it matches hosts with the same
		// resolver as ad-hoc tasks (orchestrator.MatchAdHocTargets), whose dry
		// run touches nothing -- so that is an exact preview of the set.
		var res struct {
			Matched int                 `json:"matched"`
			Objects []db.DeployedObject `json:"objects"`
		}
		req := map[string]any{"target": target, "command": "reboot", "dry_run": true}
		if err := c.do("POST", "/builds/"+build+"/tasks", req, &res); err != nil {
			return nil, nil, err
		}
		// Fill in team number, power state and agent health from the host list.
		all, err := listBuildObjects(c, in.Build)
		if err != nil {
			return nil, nil, err
		}
		byID := make(map[string]buildObject, len(all))
		for _, o := range all {
			byID[o.ID.String()] = o
		}
		rows := make([]hostRow, 0, len(res.Objects))
		for _, o := range res.Objects {
			if full, ok := byID[o.ID.String()]; ok {
				rows = append(rows, toHostRow(full))
			} else {
				rows = append(rows, toHostRow(buildObject{DeployedObject: o}))
			}
		}
		verb := in.Action
		if in.Force {
			verb = "force " + verb
		}
		return jsonResult(map[string]any{
			"preview": true,
			"would":   fmt.Sprintf("%s %d host(s)", verb, res.Matched),
			"hosts":   rows,
		})
	}

	var res struct {
		Started int `json:"started"`
		Failed  int `json:"failed"`
		Results []struct {
			ObjectID string `json:"object_id"`
			Name     string `json:"name"`
			Err      string `json:"error"`
		} `json:"results"`
	}
	req := map[string]any{"target": target, "action": in.Action, "force": in.Force}
	if err := c.do("POST", "/builds/"+build+"/power", req, &res); err != nil {
		return nil, nil, err
	}
	failed := []string{}
	for _, r := range res.Results {
		if r.Err != "" {
			failed = append(failed, r.Name+": "+r.Err)
		}
	}
	return jsonResult(map[string]any{
		"started":         res.Started,
		"failed_to_start": failed,
		"note":            "runs in the background; check list_hosts (power_state) or host_events for each host's result",
	})
}
