// The read (and a few write) endpoints the UI shell needs that
// the api didn't have yet: "Repository → Builds" and "Build →
// Overview/Hosts". Deliberately thin JSON
// projections of the real orchestrator/runner rows (internal/db) rather
// than a bespoke read model -- there is exactly one source of truth for
// build state, and it's the same Postgres tables internal/orchestrator
// and internal/runner already write.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/gateway"
	"github.com/globalcptc/laforge/internal/ghclient"
	"github.com/globalcptc/laforge/internal/loader"
)

// buildAndOwningRepository is builds_ui.go's own version of builds.go's
// buildAndOwningRepo, resolving a *build* (not a configured_build) id to
// its repository via GetBuildRepository -- the two are unrelated ids in
// unrelated tables.
func (s *Server) buildAndOwningRepository(r *http.Request) (db.Build, db.Repository, error) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		return db.Build{}, db.Repository{}, err
	}
	build, err := s.Queries.GetBuild(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Build{}, db.Repository{}, errors.New("no such build")
		}
		return db.Build{}, db.Repository{}, err
	}
	repo, err := s.Queries.GetBuildRepository(r.Context(), id)
	if err != nil {
		return db.Build{}, db.Repository{}, err
	}
	return build, repo, nil
}

// handleListBuilds is "Repository → Builds: every build of this
// repository" -- read access is enough (all four access levels
// include read).
func (s *Server) handleListBuilds(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	builds, err := s.Queries.ListBuildsByRepository(r.Context(), repo.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, builds)
}

// teamSummary and buildDetail together are "Build → Overview"'s data:
// enough for the health-band stat tiles (provisioning count, failures,
// queued work) to be computed either server-side here or client-side from
// this one payload -- deliberately not yet the full server-aggregated
// query the HealthBand wants ("all fed by server-side
// aggregates rather than counted in the browser").
type teamSummary struct {
	db.Team
	Objects []objectView `json:"objects"`
}

type buildDetail struct {
	db.Build
	Teams []teamSummary `json:"teams"`
	// Access is the environment's own authored schedule ("Build →
	// Access... the schedule from the environment file")
	// -- the top bar's access countdown and the Access screen both need
	// it alongside each team's live access_state/access_override_until.
	// Absent (nil) if the environment carries none.
	Access []loader.AccessWindow `json:"access,omitempty"`
	// Commit + builder facts for the build header, resolved best-effort from
	// the build's content_revision and configured_build. Empty when a hop
	// can't be resolved (e.g. a build with no configured build behind it).
	CommitSha         string `json:"commit_sha,omitempty"`
	CommitMessage     string `json:"commit_message,omitempty"`
	BuilderConfigName string `json:"builder_config_name,omitempty"`
	BuilderKind       string `json:"builder_kind,omitempty"`
	// Branch this build's configured build tracks. Empty for an ad-hoc build
	// with no configured build behind it.
	Branch string `json:"branch,omitempty"`
}

func (s *Server) handleGetBuild(w http.ResponseWriter, r *http.Request) {
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
	// One query for every object in the build, one for every agent
	// session, rather than a query per team -- the matrix/table already
	// want the whole build's data at once, and this is the same shape
	// handleListDeployedObjects uses.
	allObjs, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views, err := attachAgentHealth(r.Context(), s.Queries, build.ID, allObjs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	byTeam := make(map[string][]objectView, len(teams))
	for _, v := range views {
		key := v.TeamID.String()
		byTeam[key] = append(byTeam[key], v)
	}
	detail := buildDetail{Build: build, Teams: make([]teamSummary, 0, len(teams))}
	if env, err := s.Queries.GetEnvironmentByRevisionAndName(r.Context(), db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: build.ContentRevisionID, Name: build.EnvironmentName,
	}); err == nil {
		json.Unmarshal(env.Access, &detail.Access) // malformed/absent -- leave Access nil, not a request failure
	}
	// Commit + builder for the header, best-effort per hop.
	if rev, err := s.Queries.GetContentRevision(r.Context(), build.ContentRevisionID); err == nil {
		detail.CommitSha = rev.CommitSha
		detail.CommitMessage = db.StrOrEmpty(rev.CommitMessage)
	}
	if build.ConfiguredBuildID.Valid {
		if cb, err := s.Queries.GetConfiguredBuild(r.Context(), build.ConfiguredBuildID); err == nil {
			detail.BuilderConfigName = cb.BuilderConfigName
			detail.Branch = cb.Branch
			if bc, err := s.Queries.GetBuilderConfigByName(r.Context(), cb.BuilderConfigName); err == nil {
				detail.BuilderKind = bc.Kind
			}
		}
	}
	for _, tm := range teams {
		objs := byTeam[tm.ID.String()]
		if objs == nil {
			objs = []objectView{}
		}
		detail.Teams = append(detail.Teams, teamSummary{Team: tm, Objects: objs})
	}
	writeJSON(w, http.StatusOK, detail)
}

// handleListDeployedObjects is the flat form of the same data -- "Build →
// Hosts (table)": every deployed object across every team in one list,
// which is what a sortable/filterable table view (and the ad-hoc task
// TargetPicker) actually wants, rather than walking handleGetBuild's
// team-nested shape.
func (s *Server) handleListDeployedObjects(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	objs, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views, err := attachAgentHealth(r.Context(), s.Queries, build.ID, objs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if views == nil {
		views = []objectView{}
	}
	// Fill in the human team number per object so a client (e.g. `laforge shell`)
	// can resolve a single host by team + name without mapping team_id itself.
	if teams, err := s.Queries.ListTeamsByBuild(r.Context(), build.ID); err == nil {
		num := make(map[string]int32, len(teams))
		for _, t := range teams {
			num[t.ID.String()] = t.TeamNumber
		}
		for i := range views {
			views[i].TeamNumber = num[views[i].TeamID.String()]
		}
	}
	writeJSON(w, http.StatusOK, views)
}

// handleListEvents is "Build → Logs", the plain
// (non-live) read of the whole journal -- handleLiveStatus (live.go) is
// the same data, tailed.
func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	rows, err := s.Queries.ListEventsByBuildWithObject(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]eventView, 0, len(rows))
	for _, row := range rows {
		views = append(views, makeEventView(row.Event, row.ObjAsName, row.ObjObjectName, row.TeamNumber))
	}
	writeJSON(w, http.StatusOK, views)
}

// eventView is a journal event flattened together with the host it's
// about, so the Logs page can show "which host" per line without a second
// round trip. Host is the object's `as` name (or its content name), empty
// for a build-level event that belongs to no single object.
type eventView struct {
	db.Event
	Host       string `json:"host,omitempty"`
	TeamNumber *int32 `json:"team_number,omitempty"`
}

func makeEventView(ev db.Event, asName, objectName *string, teamNumber *int32) eventView {
	host := ""
	if asName != nil && *asName != "" {
		host = *asName
	} else if objectName != nil {
		host = *objectName
	}
	return eventView{Event: ev, Host: host, TeamNumber: teamNumber}
}

// handleListObjectEvents is "per-object logs" -- one host or container's
// own slice of the journal. The object id alone doesn't prove it
// belongs to THIS build (ids are global), so this checks the object's
// own team.build_id matches the
// build in the URL before returning anything -- otherwise a valid id
// from a different build a caller has no access to would leak its
// events through a build they do have access to.
func (s *Server) handleListObjectEvents(w http.ResponseWriter, r *http.Request) {
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
	events, err := s.Queries.ListEventsByDeployedObject(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if events == nil {
		events = []db.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

// objectInBuild parses {objectId} and confirms it genuinely belongs to
// this build before a caller can read anything scoped to it -- shared by
// handleListObjectEvents and handleListObjectHeartbeats, same reasoning
// both need: an object id alone doesn't prove which build it's in (ids
// are global), so a valid id from a DIFFERENT build a caller has no access to
// must not leak its data through a build they do have access to.
func (s *Server) objectInBuild(r *http.Request, buildID pgtype.UUID) (pgtype.UUID, error) {
	objectID, err := parseUUID(r.PathValue("objectId"))
	if err != nil {
		return pgtype.UUID{}, err
	}
	obj, err := s.Queries.GetDeployedObject(r.Context(), objectID)
	if err != nil {
		return pgtype.UUID{}, errors.New("no such object")
	}
	team, err := s.Queries.GetTeam(r.Context(), obj.TeamID)
	if err != nil || team.BuildID != buildID {
		return pgtype.UUID{}, errors.New("no such object in this build")
	}
	return objectID, nil
}

// handleListObjectHeartbeats is the other half of "live troubleshooting"
// -- one object's own real heartbeat history (migrations/00007's
// append-only agent_heartbeat log), including the real remote address
// each check-in came from -- "rules checking" signal (an address change
// mid-competition is real, actionable), not just liveness.
func (s *Server) handleListObjectHeartbeats(w http.ResponseWriter, r *http.Request) {
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
	heartbeats, err := s.Queries.ListAgentHeartbeatsByObject(r.Context(), db.ListAgentHeartbeatsByObjectParams{
		DeployedObjectID: objectID, Limit: 200,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if heartbeats == nil {
		heartbeats = []db.AgentHeartbeat{}
	}
	writeJSON(w, http.StatusOK, heartbeats)
}

// stepView is one materialized build step for a host/container -- the real
// agent_task the agent runs -- plus its own validator results, so the
// detail panel can show "where is this host in its build" step by step
// (which are done, which is running, which failed and why), not just the
// object's single rolled-up status.
type stepView struct {
	StepIndex int32  `json:"step_index"`
	Command   string `json:"command"`
	Status    string `json:"status"` // pending | leased | done | failed
	Attempts  int32  `json:"attempts"`
	LastError string `json:"last_error,omitempty"`
	Output    string `json:"output,omitempty"`
	// GroupIndex/GroupLabel tie the expanded sub-commands (write_file +
	// execute + validate) of one authored step together under a named
	// heading, recomputed from content the same way the Render tab is.
	// GroupLabel is empty for a step that couldn't be attributed (an ad-hoc
	// command past the materialized set, or when content can't be resolved).
	GroupIndex int             `json:"group_index"`
	GroupLabel string          `json:"group_label,omitempty"`
	Validators []validatorView `json:"validators"`
}

type validatorView struct {
	Kind    string `json:"kind"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

func (s *Server) handleListObjectSteps(w http.ResponseWriter, r *http.Request) {
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
	tasks, err := s.Queries.ListAgentTasksByHost(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Best-effort grouping: which authored step each materialized step_index
	// belongs to. A failure here (content not resolvable, etc.) just leaves
	// the steps ungrouped rather than failing the list.
	groups := s.stepGroups(r.Context(), build, objectID)
	views := make([]stepView, 0, len(tasks))
	for _, t := range tasks {
		results, err := s.Queries.ListValidatorResultsByTask(r.Context(), t.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		vs := make([]validatorView, 0, len(results))
		for _, vr := range results {
			vs = append(vs, validatorView{Kind: vr.Kind, Passed: vr.Passed, Message: db.StrOrEmpty(vr.Message)})
		}
		v := stepView{
			StepIndex: t.StepIndex, Command: t.Command, Status: t.Status, Attempts: t.Attempts,
			LastError: db.StrOrEmpty(t.LastError), Output: db.StrOrEmpty(t.Output), Validators: vs,
			GroupIndex: int(t.StepIndex),
		}
		if idx := int(t.StepIndex); idx >= 0 && idx < len(groups) {
			v.GroupIndex = groups[idx].group
			v.GroupLabel = groups[idx].label
		}
		views = append(views, v)
	}
	writeJSON(w, http.StatusOK, views)
}

type stepGroupRef struct {
	group int
	label string
}

// stepGroups recomputes, aligned to materialized step_index, the authored
// step each command came from -- the same content resolution the Render tab
// uses (internal/gateway.ExpandSteps). A container's steps align one-to-one
// with its authored steps, same as a host's. Best-effort: any failure returns
// nil and the caller shows the steps ungrouped.
func (s *Server) stepGroups(ctx context.Context, build db.Build, objectID pgtype.UUID) []stepGroupRef {
	obj, err := s.Queries.GetDeployedObject(ctx, objectID)
	if err != nil || obj.Kind == "network" || obj.AsName == nil {
		return nil
	}
	team, err := s.Queries.GetTeam(ctx, obj.TeamID)
	if err != nil {
		return nil
	}
	repoRoot := s.RepoRoot
	if s.Checkouts != nil {
		dir, err := s.Checkouts.ForBuild(ctx, build.ID)
		if err != nil {
			return nil
		}
		repoRoot = dir
	}
	c, err := loader.Load(repoRoot)
	if err != nil || len(c.Errors) > 0 {
		return nil
	}
	cmds, _, err := gateway.ExpandSteps(repoRoot, c, build.EnvironmentName, *obj.AsName, int(team.TeamNumber))
	if err != nil {
		return nil
	}
	// A container's materialized steps are exactly its authored steps now, in
	// the same order -- no leading Docker-run command to offset for. Its agent
	// runs inside the app container and its steps run there, just like a host's;
	// the image pull/run is the builder's business, never a materialized step.
	refs := make([]stepGroupRef, 0, len(cmds))
	for _, cmd := range cmds {
		refs = append(refs, stepGroupRef{group: cmd.Group, label: cmd.GroupLabel})
	}
	return refs
}

// --- repository_access administration ("Admin → People & access") ---

// requireRepoAdminOrInstanceAdmin is the floor for managing one
// repository's access: an instance admin, or an effective levelAdmin on
// that repository (their GitHub admin role, or an admin's grant).
func (s *Server) requireRepoAdminOrInstanceAdmin(ctx context.Context, r *http.Request, repo db.Repository) (authSession, error) {
	if sess, err := s.requireInstanceAdmin(ctx, r); err == nil {
		return sess, nil
	}
	return s.requireLevel(ctx, r, repo, levelAdmin)
}

// resolveGrantee turns a GitHub login into a real account row -- looked
// up live against GitHub (which also confirms the identity exists), and
// created if this person has never signed in, so access can be granted
// before someone's first login.
func (s *Server) resolveGrantee(ctx context.Context, sess authSession, login string) (db.Account, error) {
	// Someone LaForge already knows needs no GitHub round-trip -- changing
	// a known person's level shouldn't depend on GitHub being reachable.
	if acct, err := s.Queries.GetAccountByLogin(ctx, login); err == nil {
		return acct, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.Account{}, err
	}
	ghUser, err := s.GH.GetUserByLogin(ctx, sess.GithubToken, login)
	if err != nil {
		return db.Account{}, err
	}
	return s.Queries.UpsertAccount(ctx, db.UpsertAccountParams{
		GithubID: ghUser.ID, GithubLogin: ghUser.Login, AvatarUrl: db.StrPtr(ghUser.AvatarURL),
	})
}

// repoAccessPerson is one row of a repository's access list: someone
// GitHub lists as a collaborator, or someone an admin has set a level for,
// or both.
type repoAccessPerson struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	// GithubRole is GitHub's own word for their role ("admin", "write",
	// ...); empty when they aren't a collaborator.
	GithubRole string `json:"github_role"`
	// GithubLevel is the level that role gives them ("none" when they
	// aren't a collaborator).
	GithubLevel string `json:"github_level"`
	// Level is the admin's setting for them on this repository, which
	// replaces GithubLevel; empty when they follow GitHub.
	Level     string `json:"level"`
	Effective string `json:"effective"`
}

type repoAccessView struct {
	Repository db.Repository      `json:"repository"`
	People     []repoAccessPerson `json:"people"`
	// GithubError is set when GitHub's collaborator list couldn't be read
	// -- the list then only has people with a setting here.
	GithubError string `json:"github_error,omitempty"`
}

// repoGithubToken is the token used to read a repository's collaborators:
// its installation's token when a GitHub App is configured (it sees the
// whole list whoever is asking), else the signed-in admin's own token.
func (s *Server) repoGithubToken(ctx context.Context, sess authSession, repo db.Repository) (string, error) {
	if !repo.InstallationID.Valid || s.AppID == "" || s.AppPrivateKey == nil {
		return sess.GithubToken, nil
	}
	inst, err := s.Queries.GetGithubInstallation(ctx, repo.InstallationID)
	if err != nil {
		return "", err
	}
	jwt, err := ghclient.GenerateAppJWT(s.AppID, s.AppPrivateKey, time.Now())
	if err != nil {
		return "", fmt.Errorf("signing app jwt: %w", err)
	}
	tok, err := s.GH.CreateInstallationToken(ctx, jwt, inst.InstallationID)
	if err != nil {
		return "", fmt.Errorf("minting installation token: %w", err)
	}
	return tok.Token, nil
}

// handleListRepositoryAccess is one repository's access list: everyone
// GitHub lists as a collaborator on it (their GitHub role is their
// starting level) merged with every admin setting on it.
func (s *Server) handleListRepositoryAccess(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireRepoAdminOrInstanceAdmin(r.Context(), r, repo)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	grants, err := s.Queries.ListRepositoryAccessByRepository(r.Context(), repo.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	view := repoAccessView{Repository: repo, People: []repoAccessPerson{}}
	byLogin := map[string]*repoAccessPerson{}
	person := func(login, avatar string) *repoAccessPerson {
		key := strings.ToLower(login)
		if p := byLogin[key]; p != nil {
			return p
		}
		p := &repoAccessPerson{Login: login, AvatarURL: avatar, GithubLevel: levelNone.String()}
		byLogin[key] = p
		return p
	}

	token, err := s.repoGithubToken(r.Context(), sess, repo)
	if err == nil {
		var collabs []ghclient.Collaborator
		collabs, err = s.GH.ListCollaborators(r.Context(), token, repo.GithubOwner, repo.GithubRepo)
		for _, c := range collabs {
			p := person(c.Login, c.AvatarURL)
			p.GithubRole = c.RoleName
			p.GithubLevel = githubRoleLevel(c.Permissions).String()
		}
	}
	if err != nil {
		view.GithubError = err.Error()
	}
	for _, g := range grants {
		p := person(g.GithubLogin, db.StrOrEmpty(g.AvatarUrl))
		p.Level = g.Level
	}

	for _, p := range byLogin {
		p.Effective = p.GithubLevel
		if p.Level != "" {
			p.Effective = p.Level
		}
		view.People = append(view.People, *p)
	}
	sort.Slice(view.People, func(i, j int) bool {
		return strings.ToLower(view.People[i].Login) < strings.ToLower(view.People[j].Login)
	})
	writeJSON(w, http.StatusOK, view)
}

// handleGetMyRepositoryAccess tells the signed-in person their own level on
// a repository and whether they can manage its access -- so the UI only
// offers what they can use.
func (s *Server) handleGetMyRepositoryAccess(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.sessionFromRequest(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	level, err := s.effectiveLevel(r.Context(), sess, repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"level":             level.String(),
		"can_manage_access": level >= levelAdmin || s.isInstanceAdmin(sess),
	})
}

// errSelfLockout refuses a repository admin (who isn't an instance admin)
// taking their own admin away -- they'd have no way back.
var errSelfLockout = errors.New("you can't lower your own access below admin on a repository you manage -- ask another admin")

type setAccessRequest struct {
	Level string `json:"level"`
}

// handleSetRepositoryAccess is "Adding someone means finding their GitHub
// identity and choosing a level" -- {login} in the path, not an account
// id, since an admin thinks in GitHub usernames, not LaForge's internal
// ids; this looks the login up live against GitHub (also confirms the
// identity actually exists) and upserts (or creates) the account row for
// them, exactly like a real login would, so granting access works even
// for someone who has never signed in yet.
func (s *Server) handleSetRepositoryAccess(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireRepoAdminOrInstanceAdmin(r.Context(), r, repo)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	var req setAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Level != "none" && parseLevel(req.Level) == levelNone {
		writeError(w, http.StatusBadRequest, errors.New("level must be one of none, read, build, manage, admin"))
		return
	}
	grantee, err := s.resolveGrantee(r.Context(), sess, r.PathValue("login"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if grantee.ID == sess.AccountID && parseLevel(req.Level) < levelAdmin && !s.isInstanceAdmin(sess) {
		writeError(w, http.StatusBadRequest, errSelfLockout)
		return
	}
	granted, err := s.Queries.UpsertRepositoryAccess(r.Context(), db.UpsertRepositoryAccessParams{
		RepositoryID: repo.ID, AccountID: grantee.ID, Level: req.Level, GrantedBy: sess.AccountID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, granted)
}

// handleDeleteRepositoryAccess removes an admin's setting for a person on
// a repository, putting them back on their GitHub role. It is "grant and revoke levels"
// made whole -- found by direct audit: the grant
// half (handleSetRepositoryAccess above) and DeleteRepositoryAccess
// (internal/db/queries/access.sql) both existed, but nothing ever called
// the delete, and handleSetRepositoryAccess itself refuses an empty/"none"
// level, so there was no way to actually remove a grant once made, only
// downgrade it. A no-op (204, not 404) when the account never had a grant
// on this repository at all -- consistent with every other "safe to call
// more than once" write in this codebase, and simpler than a caller
// needing to know in advance whether a grant exists before removing it.
func (s *Server) handleDeleteRepositoryAccess(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repoByIDParam(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	sess, err := s.requireRepoAdminOrInstanceAdmin(r.Context(), r, repo)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	login := r.PathValue("login")
	account, err := s.Queries.GetAccountByLogin(r.Context(), login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent) // no account, so no grant to remove either
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// Resetting yourself to your GitHub role is a lockout too when that
	// role is below admin.
	if account.ID == sess.AccountID && !s.isInstanceAdmin(sess) {
		ghRepo, err := s.GH.GetRepo(r.Context(), sess.GithubToken, repo.GithubOwner, repo.GithubRepo)
		if err != nil || githubRoleLevel(ghRepo.Permissions) < levelAdmin {
			writeError(w, http.StatusBadRequest, errSelfLockout)
			return
		}
	}
	if err := s.Queries.DeleteRepositoryAccess(r.Context(), db.DeleteRepositoryAccessParams{
		RepositoryID: repo.ID, AccountID: account.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
