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

	"github.com/globalcptc/laforge/internal/db"
)

// healthyWindow/lateWindow are fixed defaults, not read from the actual
// gateway's configured BasePollMS/JitterMS for whichever build this is
// (internal/gateway.Server's poll interval is a deploy-time setting, not
// stored per build) -- a real gap, not pretended otherwise: these are
// reasonable thresholds for the "a few tens of seconds" heartbeat this
// project's own tests and examples use, not a value derived from what a
// specific live deployment actually configured.
const (
	healthyWindow = 90 * time.Second
	lateWindow    = 5 * time.Minute
)

// infraUp reports whether an object's status means its instance is actually
// up at the hoster -- the point past which asking "is the agent checked in"
// becomes meaningful. Mirrors internal/orchestrator.infraUp; kept as its own
// small copy rather than exported across the package boundary since the two
// are the same short list for the same reason but serve different layers.
func infraUp(status string) bool {
	switch status {
	case "running", "building", "finished", "build_failed", "invalid":
		return true
	}
	return false
}

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
	Agent *agentHealth `json:"agent,omitempty"`
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
		if !infraUp(o.Status) {
			continue // agent health only makes sense once the instance is up
		}
		sess, ok := byObject[o.ID.String()]
		if !ok {
			out[i].Agent = &agentHealth{Status: "booting"}
			continue
		}
		last := sess.LastHeartbeatAt.Time
		age := now.Sub(last)
		status := "missing"
		switch {
		case age <= healthyWindow:
			status = "healthy"
		case age <= lateWindow:
			status = "late"
		}
		out[i].Agent = &agentHealth{LastHeartbeatAt: &last, Status: status}
	}
	return out, nil
}
