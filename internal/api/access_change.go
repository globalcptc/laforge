package api

import (
	"strconv"
	"time"

	"github.com/globalcptc/laforge/internal/db"
)

// accessChange is a team's in-progress or last-failed access change, for the
// Access page to show Opening / Closing while the builder call is queued or
// running. A team's access_state only flips once the call completes, which on a
// big team can take minutes, so without this the page looks like nothing is
// happening.
type accessChange struct {
	// Direction is "opening" or "closing".
	Direction string `json:"direction"`
	// State is "queued" (waiting for a runner), "running" (a runner has it), or
	// "failed" (the last attempt gave up; Error says why).
	State string `json:"state"`
	// Since is when the change was requested (queued/running) or gave up (failed).
	Since    time.Time `json:"since"`
	Attempts int32     `json:"attempts,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// teamAccessChanges reduces a build's recent access tasks (newest first) to
// each team's current change, keyed by team number. A team with queued or
// running tasks is changing toward its newest one's direction -- running if any
// of them is leased. Otherwise its newest task decides: a failure is reported
// unless the team already sits in the state it was trying to reach; a success
// means nothing is in flight.
func teamAccessChanges(rows []db.ListRecentTeamAccessTasksRow, teams []db.Team) map[int32]*accessChange {
	stateByTeam := make(map[int32]string, len(teams))
	for _, t := range teams {
		stateByTeam[t.TeamNumber] = t.AccessState
	}
	byTeam := map[int32][]db.ListRecentTeamAccessTasksRow{}
	for _, r := range rows {
		n, err := strconv.ParseInt(r.Team, 10, 32)
		if err != nil {
			continue
		}
		byTeam[int32(n)] = append(byTeam[int32(n)], r)
	}
	out := make(map[int32]*accessChange, len(byTeam))
	for team, rs := range byTeam {
		var newestInFlight *db.ListRecentTeamAccessTasksRow
		running := false
		var since time.Time
		var attempts int32
		for i := range rs {
			r := &rs[i]
			if r.Status != "pending" && r.Status != "leased" {
				continue
			}
			if newestInFlight == nil {
				newestInFlight = r
			}
			if r.Status == "leased" {
				running = true
			}
			since = r.CreatedAt.Time // rows are newest first, so this ends on the oldest
			if r.Attempts > attempts {
				attempts = r.Attempts
			}
		}
		if newestInFlight != nil {
			state := "queued"
			if running {
				state = "running"
			}
			out[team] = &accessChange{Direction: direction(newestInFlight.Kind), State: state, Since: since, Attempts: attempts}
			continue
		}
		newest := rs[0]
		if newest.Status != "failed" || stateByTeam[team] == target(newest.Kind) {
			continue
		}
		out[team] = &accessChange{
			Direction: direction(newest.Kind), State: "failed", Since: newest.UpdatedAt.Time,
			Attempts: newest.Attempts, Error: db.StrOrEmpty(newest.LastError),
		}
	}
	return out
}

func direction(kind string) string {
	if kind == "open_access" {
		return "opening"
	}
	return "closing"
}

func target(kind string) string {
	if kind == "open_access" {
		return "open"
	}
	return "closed"
}
