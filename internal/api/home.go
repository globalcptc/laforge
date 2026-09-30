// Home: what is live right now and whether it's
// healthy, without walking into each repository -- active builds grouped
// by repository, each with its hosting picture, an attention list for
// anything failing, and how much each builder is carrying. Everything is
// aggregated here, server-side, from the same rows and the same agent
// health rule (attachAgentHealth) the build pages use.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// homeCounts is the hosting picture for one build, or summed across all.
type homeCounts struct {
	Hosts            int `json:"hosts"`
	Containers       int `json:"containers"`
	Networks         int `json:"networks"`
	ObjectsFailed    int `json:"objects_failed"`
	AgentsHealthy    int `json:"agents_healthy"`
	AgentsLate       int `json:"agents_late"`
	AgentsMissing    int `json:"agents_missing"`
	AgentsBooting    int `json:"agents_booting"`
	TasksOutstanding int `json:"tasks_outstanding"`
	TasksFailed      int `json:"tasks_failed"`
}

func (c *homeCounts) add(o homeCounts) {
	c.Hosts += o.Hosts
	c.Containers += o.Containers
	c.Networks += o.Networks
	c.ObjectsFailed += o.ObjectsFailed
	c.AgentsHealthy += o.AgentsHealthy
	c.AgentsLate += o.AgentsLate
	c.AgentsMissing += o.AgentsMissing
	c.AgentsBooting += o.AgentsBooting
	c.TasksOutstanding += o.TasksOutstanding
	c.TasksFailed += o.TasksFailed
}

type homeBuild struct {
	ID                string                `json:"id"`
	EnvironmentName   string                `json:"environment_name"`
	Status            string                `json:"status"`
	CreatedAt         time.Time             `json:"created_at"`
	CommitSha         string                `json:"commit_sha"`
	Ref               string                `json:"ref"`
	BuilderConfigName string                `json:"builder_config_name"`
	Teams             int                   `json:"teams"`
	TeamsOpen         int                   `json:"teams_open"`
	Counts            homeCounts            `json:"counts"`
	Access            []loader.AccessWindow `json:"access,omitempty"`
}

type homeRepository struct {
	ID          string      `json:"id"`
	GithubOwner string      `json:"github_owner"`
	GithubRepo  string      `json:"github_repo"`
	Builds      []homeBuild `json:"builds"`
}

// homeBuilder is one builder's load: active builds on it and what they
// have deployed there.
type homeBuilder struct {
	Name         string     `json:"name"`
	Kind         string     `json:"kind"`
	ActiveBuilds int        `json:"active_builds"`
	Counts       homeCounts `json:"counts"`
}

type homeAttention struct {
	RepositoryID    string `json:"repository_id"`
	Repository      string `json:"repository"`
	BuildID         string `json:"build_id"`
	EnvironmentName string `json:"environment_name"`
	Reason          string `json:"reason"`
}

type homeView struct {
	ActiveBuilds int              `json:"active_builds"`
	Totals       homeCounts       `json:"totals"`
	Repositories []homeRepository `json:"repositories"`
	Builders     []homeBuilder    `json:"builders"`
	Attention    []homeAttention  `json:"attention"`
}

func (s *Server) handleGetHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	allRepos, err := s.Queries.ListRepositories(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	visible := map[pgtype.UUID]db.Repository{}
	for _, repo := range s.visibleRepositories(ctx, sess, allRepos) {
		visible[repo.ID] = repo
	}

	active, err := s.Queries.ListActiveBuilds(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var builds []db.ListActiveBuildsRow
	var buildIDs []pgtype.UUID
	for _, b := range active {
		if _, ok := visible[b.RepositoryID]; ok {
			builds = append(builds, b)
			buildIDs = append(buildIDs, b.ID)
		}
	}

	tasks := map[pgtype.UUID]homeCounts{}
	if len(buildIDs) > 0 {
		rows, err := s.Queries.CountAgentTasksByBuild(ctx, buildIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		for _, row := range rows {
			c := tasks[row.BuildID]
			switch row.Status {
			case "pending", "leased":
				c.TasksOutstanding += int(row.N)
			case "failed":
				c.TasksFailed += int(row.N)
			}
			tasks[row.BuildID] = c
		}
	}

	view := homeView{Repositories: []homeRepository{}, Builders: []homeBuilder{}, Attention: []homeAttention{}}
	repoIndex := map[pgtype.UUID]int{}
	builders := map[string]*homeBuilder{}

	for _, b := range builds {
		hb, err := s.homeBuild(r, b)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		t := tasks[b.ID]
		hb.Counts.TasksOutstanding, hb.Counts.TasksFailed = t.TasksOutstanding, t.TasksFailed

		repo := visible[b.RepositoryID]
		i, ok := repoIndex[repo.ID]
		if !ok {
			i = len(view.Repositories)
			repoIndex[repo.ID] = i
			view.Repositories = append(view.Repositories, homeRepository{
				ID: repo.ID.String(), GithubOwner: repo.GithubOwner, GithubRepo: repo.GithubRepo, Builds: []homeBuild{},
			})
		}
		view.Repositories[i].Builds = append(view.Repositories[i].Builds, hb)
		view.ActiveBuilds++
		view.Totals.add(hb.Counts)

		if hb.BuilderConfigName != "" {
			bl := builders[hb.BuilderConfigName]
			if bl == nil {
				bl = &homeBuilder{Name: hb.BuilderConfigName}
				builders[hb.BuilderConfigName] = bl
			}
			bl.ActiveBuilds++
			bl.Counts.add(hb.Counts)
		}

		attention := func(reason string) {
			view.Attention = append(view.Attention, homeAttention{
				RepositoryID: repo.ID.String(), Repository: repo.GithubOwner + "/" + repo.GithubRepo,
				BuildID: hb.ID, EnvironmentName: hb.EnvironmentName, Reason: reason,
			})
		}
		if hb.Status == "failed" {
			attention("Build failed")
		}
		if n := hb.Counts.ObjectsFailed; n > 0 {
			attention(plural(n, "object failed", "objects failed"))
		}
		if n := hb.Counts.AgentsMissing; n > 0 {
			attention(plural(n, "agent missing", "agents missing"))
		}
		if n := hb.Counts.TasksFailed; n > 0 {
			attention(plural(n, "step failed", "steps failed"))
		}
	}

	// Every builder's kind; an instance admin also sees builders with
	// nothing on them.
	configs, err := s.Queries.ListBuilderConfigs(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	admin := s.isInstanceAdmin(sess)
	for _, c := range configs {
		if bl := builders[c.Name]; bl != nil {
			bl.Kind = c.Kind
		} else if admin {
			builders[c.Name] = &homeBuilder{Name: c.Name, Kind: c.Kind}
		}
	}
	for _, bl := range builders {
		view.Builders = append(view.Builders, *bl)
	}
	sort.Slice(view.Builders, func(i, j int) bool {
		if view.Builders[i].ActiveBuilds != view.Builders[j].ActiveBuilds {
			return view.Builders[i].ActiveBuilds > view.Builders[j].ActiveBuilds
		}
		return view.Builders[i].Name < view.Builders[j].Name
	})
	writeJSON(w, http.StatusOK, view)
}

// homeBuild is one active build's card: its teams, what it has deployed
// and how those agents are doing, and its access schedule.
func (s *Server) homeBuild(r *http.Request, b db.ListActiveBuildsRow) (homeBuild, error) {
	ctx := r.Context()
	hb := homeBuild{
		ID: b.ID.String(), EnvironmentName: b.EnvironmentName, Status: b.Status, CreatedAt: b.CreatedAt.Time,
		CommitSha: b.CommitSha, Ref: b.Ref, BuilderConfigName: b.BuilderConfigName,
	}
	teams, err := s.Queries.ListTeamsByBuild(ctx, b.ID)
	if err != nil {
		return hb, err
	}
	hb.Teams = len(teams)
	for _, tm := range teams {
		if tm.AccessState == "open" {
			hb.TeamsOpen++
		}
	}
	objs, err := s.Queries.ListDeployedObjectsByBuild(ctx, b.ID)
	if err != nil {
		return hb, err
	}
	views, err := attachAgentHealth(ctx, s.Queries, b.ID, objs)
	if err != nil {
		return hb, err
	}
	for _, v := range views {
		if v.Status == "destroyed" {
			continue
		}
		switch v.Kind {
		case "host":
			hb.Counts.Hosts++
		case "container":
			hb.Counts.Containers++
		case "network":
			hb.Counts.Networks++
		}
		if isFailedStatus(v.Status) {
			hb.Counts.ObjectsFailed++
		}
		if v.Agent != nil {
			switch v.Agent.Status {
			case "healthy":
				hb.Counts.AgentsHealthy++
			case "late":
				hb.Counts.AgentsLate++
			case "missing":
				hb.Counts.AgentsMissing++
			default:
				hb.Counts.AgentsBooting++
			}
		}
	}
	if env, err := s.Queries.GetEnvironmentByRevisionAndName(ctx, db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: b.ContentRevisionID, Name: b.EnvironmentName,
	}); err == nil {
		json.Unmarshal(env.Access, &hb.Access) // malformed/absent -- no countdown, not a failure
	}
	return hb, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
