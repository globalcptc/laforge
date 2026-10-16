package api

import (
	"context"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// hostPassword is a host's current built-in admin (Administrator/root)
// password as LaForge knows it. LaForge sets that password itself -- the
// automatic first "Set admin password" step, plus any authored or ad-hoc
// `set_password` for the admin account -- so the last one the agent applied is
// what's actually on the box. A script that changes the admin password some
// other way is invisible here; content should use `set_password` for that.
type hostPassword struct {
	Password string `json:"password"`
	// Applied is true once the agent has completed the set_password step this
	// value came from. False means it's the next planned value (the step hasn't
	// run yet) or the environment root password with no step behind it.
	Applied bool `json:"applied"`
	// Source is "step" (a set_password step) or "environment" (the
	// environment's root_password, when the host has no admin set_password).
	Source string `json:"source"`
}

// currentAdminPasswords resolves every host's current admin password in a
// build, keyed by deployed object. The last APPLIED admin set_password wins (by
// when it finished, so an operator's later ad-hoc change beats the authored
// sequence); with none applied yet, the earliest still-pending authored one is
// what the host will get next. Hosts with no admin set_password at all are
// left out (see passwordFor). Failed or ignored steps never count -- the box
// didn't take that password.
func currentAdminPasswords(ctx context.Context, q *db.Queries, buildID pgtype.UUID) (map[pgtype.UUID]hostPassword, error) {
	rows, err := q.ListAdminPasswordStepsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	type pick struct {
		applied     *db.ListAdminPasswordStepsByBuildRow
		nextPlanned *db.ListAdminPasswordStepsByBuildRow
	}
	picks := map[pgtype.UUID]*pick{}
	for i := range rows {
		r := &rows[i]
		if r.Password == "" {
			continue
		}
		p := picks[r.DeployedObjectID]
		if p == nil {
			p = &pick{}
			picks[r.DeployedObjectID] = p
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
	out := make(map[pgtype.UUID]hostPassword, len(picks))
	for id, p := range picks {
		switch {
		case p.applied != nil:
			out[id] = hostPassword{Password: p.applied.Password, Applied: true, Source: "step"}
		case p.nextPlanned != nil:
			out[id] = hostPassword{Password: p.nextPlanned.Password, Source: "step"}
		}
	}
	// Hosts with no admin set_password are absent; passwordFor falls back to the
	// environment root password for them.
	return out, nil
}

// passwordFor is a host's entry from currentAdminPasswords, or the environment
// root password when the host has no admin set_password at all.
func passwordFor(m map[pgtype.UUID]hostPassword, id pgtype.UUID, rootPassword string) hostPassword {
	if hp, ok := m[id]; ok {
		return hp
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
