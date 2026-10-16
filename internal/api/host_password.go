package api

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// hostCredential is one account's current password on a host as LaForge
// knows it. LaForge sets these itself -- the automatic first "Set admin
// password" step, and any authored or ad-hoc `set_password` / `create_user` --
// so the last one the agent applied is what's actually on the box. A script
// that changes a password some other way is invisible here; content should use
// set_password (with `generate:` for a generated one) instead.
type hostCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Applied is true once the agent has completed the step this value came
	// from. False means it's the next planned value (the step hasn't run yet),
	// or the environment root password with no step behind it.
	Applied bool `json:"applied"`
	// Source is "step" (a set_password/create_user step) or "environment" (the
	// environment's root_password, for an admin account with no step).
	Source string `json:"source"`
	// Admin marks the built-in admin account (Administrator or root) -- the
	// login the external-access export hands out.
	Admin bool `json:"admin"`
}

// hostPassword is a host's admin credential reduced to what the export needs.
type hostPassword struct {
	Password string `json:"password"`
	Applied  bool   `json:"applied"`
	Source   string `json:"source"`
}

func isAdminAccount(username string) bool {
	u := strings.ToLower(username)
	return u == "administrator" || u == "root"
}

// currentCredentials resolves every host's current password per account in a
// build, keyed by deployed object, admin account first then by name. Per
// account (usernames compared case-insensitively, as Windows does): the last
// APPLIED step wins, by when it finished, so an operator's later ad-hoc change
// beats the authored sequence; with none applied yet, the earliest
// still-pending authored step is what the host will get next. Failed or
// ignored steps never count -- the box didn't take that password.
func currentCredentials(ctx context.Context, q *db.Queries, buildID pgtype.UUID) (map[pgtype.UUID][]hostCredential, error) {
	rows, err := q.ListCredentialStepsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	type pick struct {
		applied     *db.ListCredentialStepsByBuildRow
		nextPlanned *db.ListCredentialStepsByBuildRow
	}
	type key struct {
		obj  pgtype.UUID
		user string
	}
	picks := map[key]*pick{}
	for i := range rows {
		r := &rows[i]
		k := key{r.DeployedObjectID, strings.ToLower(r.Username)}
		p := picks[k]
		if p == nil {
			p = &pick{}
			picks[k] = p
		}
		switch r.Status {
		case "done":
			if p.applied == nil || laterFinish(r.FinishedAt, p.applied.FinishedAt) {
				p.applied = r
			}
		case "pending", "leased", "blocked":
			if !r.AdHoc && (p.nextPlanned == nil || r.StepIndex < p.nextPlanned.StepIndex) {
				p.nextPlanned = r
			}
		}
	}
	out := map[pgtype.UUID][]hostCredential{}
	for k, p := range picks {
		var c hostCredential
		switch {
		case p.applied != nil:
			c = hostCredential{Username: p.applied.Username, Password: p.applied.Password, Applied: true, Source: "step"}
		case p.nextPlanned != nil:
			c = hostCredential{Username: p.nextPlanned.Username, Password: p.nextPlanned.Password, Source: "step"}
		default:
			continue
		}
		c.Admin = isAdminAccount(c.Username)
		out[k.obj] = append(out[k.obj], c)
	}
	for _, list := range out {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Admin != list[j].Admin {
				return list[i].Admin
			}
			return strings.ToLower(list[i].Username) < strings.ToLower(list[j].Username)
		})
	}
	return out, nil
}

// passwordFor is a host's admin credential from currentCredentials, or the
// environment root password when the host has no admin password step at all.
func passwordFor(creds map[pgtype.UUID][]hostCredential, id pgtype.UUID, rootPassword string) hostPassword {
	for _, c := range creds[id] {
		if c.Admin {
			return hostPassword{Password: c.Password, Applied: c.Applied, Source: c.Source}
		}
	}
	return hostPassword{Password: rootPassword, Source: "environment"}
}

func laterFinish(a, b pgtype.Timestamptz) bool {
	if !a.Valid {
		return false
	}
	if !b.Valid {
		return true
	}
	return a.Time.After(b.Time)
}

// buildCredentialRow is one account on one host, build-wide -- the
// `laforge passwords` listing.
type buildCredentialRow struct {
	Team             int32       `json:"team"`
	Host             string      `json:"host"`
	DeployedObjectID pgtype.UUID `json:"deployed_object_id"`
	hostCredential
}

// handleListCredentials lists every account LaForge has set a password for,
// across every host in the build, by team then host -- the build-wide form of
// the host Info tab's Credentials section. levelRead, the same floor as the
// per-host infra endpoint and the external-access export that already expose
// these passwords.
func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	creds, err := currentCredentials(r.Context(), s.Queries, build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	teams, err := s.Queries.ListTeamsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	teamNum := map[pgtype.UUID]int32{}
	for _, t := range teams {
		teamNum[t.ID] = t.TeamNumber
	}
	objs, err := s.Queries.ListDeployedObjectsByBuild(r.Context(), build.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := []buildCredentialRow{}
	for _, o := range objs {
		host := o.ObjectName
		if o.AsName != nil && *o.AsName != "" {
			host = *o.AsName
		}
		for _, c := range creds[o.ID] {
			out = append(out, buildCredentialRow{Team: teamNum[o.TeamID], Host: host, DeployedObjectID: o.ID, hostCredential: c})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Team != out[j].Team {
			return out[i].Team < out[j].Team
		}
		return out[i].Host < out[j].Host
	})
	writeJSON(w, http.StatusOK, out)
}
