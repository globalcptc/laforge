// Agent health: "Agent healthy: checked in within the expected window.
// Agent late: overdue but not yet written off. Agent missing: long
// overdue" (the environment
// dashboard's health band). agent_session.last_heartbeat_at already
// existed, and nothing ever read it back
// through the API -- the matrix and table both need it to show anything
// richer than deployed_object.status.
package api

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/agenthealth"
	"github.com/globalcptc/laforge/internal/db"
)

// infraUp is agenthealth.InfraUp, kept under its local name for this
// package's other callers.
func infraUp(status string) bool { return agenthealth.InfraUp(status) }

// lateWindow is how long an agent can go quiet before it counts as missing.
const lateWindow = agenthealth.Late

type agentHealth struct {
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	Status          string     `json:"agent_status"` // booting | healthy | late | missing
}

// objectView is db.DeployedObject plus its agent health, when
// applicable -- Agent is nil for anything not yet (or no longer)
// deployed, since "is the agent checked in" isn't a meaningful question
// for a pending or destroyed object.
type objectView struct {
	db.DeployedObject
	// TeamNumber is the human team number (1..N) for this object's team, so
	// clients don't have to resolve the team_id UUID themselves. Populated by
	// handleListDeployedObjects; 0 where not filled in.
	TeamNumber int32        `json:"team_number,omitempty"`
	Agent      *agentHealth `json:"agent,omitempty"`
}

// attachAgentHealth is shared by handleListDeployedObjects and
// handleGetBuild so the matrix, the table, and the build detail view all
// agree on exactly the same health computation -- one query, one status
// vocabulary, not three places that could drift.
func attachAgentHealth(ctx context.Context, q *db.Queries, buildID pgtype.UUID, objs []db.DeployedObject) ([]objectView, error) {
	sessions, err := q.ListAgentSessionsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	byObject := make(map[string]db.AgentSession, len(sessions))
	for _, s := range sessions {
		byObject[s.DeployedObjectID.String()] = s
	}

	now := time.Now()
	out := make([]objectView, len(objs))
	for i, o := range objs {
		out[i] = objectView{DeployedObject: o}
		in := agenthealth.Object{Kind: o.Kind, Status: o.Status, CheckinOverdue: o.AgentCheckinOverdueAt.Valid}
		if o.DeployedAt.Valid {
			in.DeployedAt = o.DeployedAt.Time
		}
		if sess, ok := byObject[o.ID.String()]; ok && sess.LastHeartbeatAt.Valid {
			in.LastHeartbeat = sess.LastHeartbeatAt.Time
		}
		status, last, ok := agenthealth.Classify(in, now)
		if !ok {
			continue // networks, and objects whose instance isn't up, carry no agent health
		}
		h := &agentHealth{Status: status}
		if !last.IsZero() {
			h.LastHeartbeatAt = &last
		}
		out[i].Agent = h
	}
	return out, nil
}
