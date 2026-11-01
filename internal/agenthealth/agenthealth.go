// Package agenthealth is the one rule for "is this object's agent checked in":
// healthy (heard from within the expected window), late (overdue but not yet
// written off), missing (long overdue, or the check-in watchdog gave up on it)
// or booting (up, but no check-in yet). The REST API's object views and the
// MCP server both classify agents with it, so they always agree.
package agenthealth

import "time"

// Healthy/Late are fixed thresholds for the "a few tens of seconds"
// heartbeat the gateway uses; they are not read from the gateway's configured
// poll interval, which is a deploy-time setting not stored per build.
const (
	Healthy = 90 * time.Second
	Late    = 5 * time.Minute
)

// The statuses Classify returns.
const (
	StatusBooting = "booting"
	StatusHealthy = "healthy"
	StatusLate    = "late"
	StatusMissing = "missing"
)

// InfraUp reports whether an object's status means its instance is actually
// up at the hoster -- the point past which asking "is the agent checked in"
// becomes meaningful. Mirrors internal/orchestrator.infraUp.
func InfraUp(status string) bool {
	switch status {
	case "running", "building", "finished", "build_failed", "invalid":
		return true
	}
	return false
}

// Object is what Classify needs to know about one deployed object.
type Object struct {
	Kind   string
	Status string
	// DeployedAt is when the current instance was deployed (zero if unknown).
	DeployedAt time.Time
	// CheckinOverdue is set once the orchestrator's watchdog has flagged the
	// agent as never having checked in.
	CheckinOverdue bool
	// LastHeartbeat is the agent session's last check-in; zero if there is no
	// session.
	LastHeartbeat time.Time
}

// Classify returns the object's agent status, and false when the question
// doesn't apply (a network, or an object whose instance isn't up).
// lastHeartbeat in the result is zero when the status isn't based on one.
func Classify(o Object, now time.Time) (status string, lastHeartbeat time.Time, ok bool) {
	if o.Kind == "network" || !InfraUp(o.Status) {
		return "", time.Time{}, false
	}
	last := o.LastHeartbeat
	// A rebuilt host keeps its old session row; a heartbeat from before the
	// redeploy says nothing about the new instance's agent.
	if !last.IsZero() && !o.DeployedAt.IsZero() && last.Before(o.DeployedAt) {
		last = time.Time{}
	}
	if last.IsZero() {
		if o.CheckinOverdue {
			return StatusMissing, time.Time{}, true
		}
		return StatusBooting, time.Time{}, true
	}
	switch age := now.Sub(last); {
	case age <= Healthy:
		return StatusHealthy, last, true
	case age <= Late:
		return StatusLate, last, true
	}
	return StatusMissing, last, true
}
