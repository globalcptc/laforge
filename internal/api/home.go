// Home: what is live right now and whether it's
// healthy, without walking into each repository -- active builds grouped
// by repository, each with its hosting picture, an attention list for
// anything failing, and how much each builder is carrying. Everything is
// aggregated here, server-side, from the same rows and the same agent
// health rule (attachAgentHealth) the build pages use.
package api

import (
	"encoding/json"
	"errors"
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

// Attention categories: the stable key a person's "close this item" dismissal
// is recorded against (the Reason text varies with counts, so it can't be the
// key). Kept in sync with the attention() calls in handleGetHome.
const (
	attnBuildFailed   = "build_failed"
	attnObjectsFailed = "objects_failed"
	attnAgentsMissing = "agents_missing"
	attnStepsFailed   = "steps_failed"
)

type homeAttention struct {
	RepositoryID    string `json:"repository_id"`
	Repository      string `json:"repository"`
	BuildID         string `json:"build_id"`
	EnvironmentName string `json:"environment_name"`
	Category        string `json:"category"`
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

	// Items this person has closed, so the needs-attention list leaves them out.
	dismissed := map[string]bool{}
	if dis, err := s.Queries.ListAttentionDismissalsByAccount(ctx, sess.AccountID); err == nil {
		for _, d := range dis {
			dismissed[d.BuildID.String()+"|"+d.Category] = true
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

		// Needs-attention is scoped to a person's OWN builds: with several
		// people sharing a repository the cross-build list grew unusable, so a
		// build created by someone else (or auto-built, which has no owner)
		// never lands in this person's list. Each item can also be individually
		// closed (attention_dismissal), which filters it out by category here.
		owned := b.CreatedByAccountID.Valid && b.CreatedByAccountID == sess.AccountID
		attention := func(category, reason string) {
			if !owned || dismissed[hb.ID+"|"+category] {
				return
			}
			view.Attention = append(view.Attention, homeAttention{
				RepositoryID: repo.ID.String(), Repository: repo.GithubOwner + "/" + repo.GithubRepo,
				BuildID: hb.ID, EnvironmentName: hb.EnvironmentName, Category: category, Reason: reason,
			})
		}
		if hb.Status == "failed" {
			attention(attnBuildFailed, "Build failed")
		}
		if n := hb.Counts.ObjectsFailed; n > 0 {
			attention(attnObjectsFailed, plural(n, "object failed", "objects failed"))
		}
		if n := hb.Counts.AgentsMissing; n > 0 {
			attention(attnAgentsMissing, plural(n, "agent missing", "agents missing"))
		}
		if n := hb.Counts.TasksFailed; n > 0 {
			attention(attnStepsFailed, plural(n, "step failed", "steps failed"))
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

// attentionCategories is the closed set a dismissal may name -- guards against
// a client writing arbitrary rows, and keeps them aligned with handleGetHome.
var attentionCategories = map[string]bool{
	attnBuildFailed: true, attnObjectsFailed: true, attnAgentsMissing: true, attnStepsFailed: true,
}

type attentionDismissRequest struct {
	BuildID  string `json:"build_id"`
	Category string `json:"category"`
}

// handleDismissAttention closes one needs-attention item for the requesting
// person (POST /home/attention/dismiss). Per-account: it only affects this
// person's own home view, never anyone else's. handleUndismissAttention
// reopens it. Both require the build to be one the person can see.
func (s *Server) handleDismissAttention(w http.ResponseWriter, r *http.Request) {
	s.setAttentionDismissed(w, r, true)
}

func (s *Server) handleUndismissAttention(w http.ResponseWriter, r *http.Request) {
	s.setAttentionDismissed(w, r, false)
}

func (s *Server) setAttentionDismissed(w http.ResponseWriter, r *http.Request, dismiss bool) {
	ctx := r.Context()
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req attentionDismissRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !attentionCategories[req.Category] {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown attention category %q", req.Category))
		return
	}
	buildID, err := parseUUID(req.BuildID)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid build_id: %w", err))
		return
	}
	// The build must be one this person can see at all (its repository is
	// visible to them) -- no writing dismissal rows against unseen builds.
	repo, err := s.Queries.GetBuildRepository(ctx, buildID)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("no such build"))
		return
	}
	if len(s.visibleRepositories(ctx, sess, []db.Repository{repo})) == 0 {
		writeError(w, http.StatusNotFound, errors.New("no such build"))
		return
	}
	if dismiss {
		err = s.Queries.DismissAttention(ctx, db.DismissAttentionParams{
			AccountID: sess.AccountID, BuildID: buildID, Category: req.Category,
		})
	} else {
		err = s.Queries.UndismissAttention(ctx, db.UndismissAttentionParams{
			AccountID: sess.AccountID, BuildID: buildID, Category: req.Category,
		})
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
