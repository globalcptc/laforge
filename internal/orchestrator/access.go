package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/schedule"
)

// ReconcileAccess enforces a build's authored `access:` windows -- the half
// that was missing: the windows were displayed and used
// as schedule anchors, but nothing ever opened or closed a team from them, so
// access_state sat at its default regardless of the schedule and the UI's
// countdown ticked toward a transition that never fired.
//
// This is the enforcer. For each team it computes the desired open/closed state
// -- a live manual override if one is active, otherwise whether now falls in an
// authored window -- and, when the team's real access_state doesn't match yet,
// enqueues the exact open_access/close_access task the runner already executes
// against the real builder (internal/runner's executeAccess). It never talks to
// a builder itself, same as the rest of this package: it only diffs desired vs
// observed and creates task rows. Safe to call repeatedly and from more than one
// replica -- HasOpenTeamAccessTask keeps a slow builder call from getting a
// duplicate every pass, and access_state only flips once the task completes, so
// a converged team enqueues nothing.
//
// Windows come straight from the persisted environment row (not a content
// checkout), so this stays a pure-Postgres pass that works even for a finished
// build whose revision no longer resolves -- exactly when access enforcement
// matters most.
func ReconcileAccess(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, now time.Time) error {
	q := db.New(pool)

	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading build: %w", err)
	}

	// Best-effort: if the environment row can't be read we still honor active
	// manual overrides (windows stays nil -> ScheduledOpen is always false), we
	// just can't drive the schedule this pass. A real build's env is persisted
	// per revision and normally present.
	var windows []schedule.AccessWindow
	if env, err := q.GetEnvironmentByRevisionAndName(ctx, db.GetEnvironmentByRevisionAndNameParams{
		ContentRevisionID: build.ContentRevisionID, Name: build.EnvironmentName,
	}); err == nil {
		windows = schedule.ParseAccessWindows(env.Access)
	}

	teams, err := q.ListTeamsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing teams: %w", err)
	}

	for _, team := range teams {
		want, enforce := desiredAccess(windows, team, now)

		// Drop a lapsed override so the team is driven purely by the schedule
		// again -- but not too early for a lapsed OPEN override, which represents
		// a manually-set access end (extend/reduce): it must linger to keep the
		// window it shortened closed, and is only safe to clear once a genuinely
		// new window has opened past that end (otherwise clearing it would let the
		// schedule re-open the very window just trimmed). A lapsed CLOSED override
		// has no such tail and clears immediately, as before.
		if team.AccessOverrideState != "" && team.AccessOverrideUntil.Valid && !now.Before(team.AccessOverrideUntil.Time) {
			clear := team.AccessOverrideState != "open" ||
				schedule.OpenInWindowStartingAfter(windows, now, team.AccessOverrideUntil.Time)
			if clear {
				if err := q.ClearTeamAccessOverride(ctx, team.ID); err != nil {
					return fmt.Errorf("team %d: clearing lapsed override: %w", team.TeamNumber, err)
				}
			}
		}

		if !enforce || team.AccessState == want {
			continue
		}

		present, err := q.HasOpenTeamAccessTask(ctx, db.HasOpenTeamAccessTaskParams{
			BuildID: buildID, Team: strconv.FormatInt(int64(team.TeamNumber), 10),
		})
		if err != nil {
			return fmt.Errorf("team %d: checking for open access task: %w", team.TeamNumber, err)
		}
		if present {
			continue // one is already in flight; don't pile on
		}

		kind := "close_access"
		if want == "open" {
			kind = "open_access"
		}
		payload, _ := json.Marshal(map[string]string{"team": strconv.FormatInt(int64(team.TeamNumber), 10)})
		task, err := q.CreateTeamTask(ctx, db.CreateTeamTaskParams{BuildID: buildID, Kind: kind, Payload: payload})
		if err != nil {
			return fmt.Errorf("team %d: creating %s task: %w", team.TeamNumber, kind, err)
		}
		q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: buildID, TaskID: task.ID, Kind: "access." + want + ".reconciled",
			Message: fmt.Sprintf("schedule set team %d access to %s", team.TeamNumber, want),
			Payload: []byte("{}"),
		})
	}
	return nil
}

// desiredAccess is the pure decision the reconciler and the manual endpoint
// share the pieces of: an active manual override wins (hold its direction until
// it lapses); otherwise the authored windows decide. enforce is false only when
// there is nothing to assert -- no override and no windows at all -- so a
// schedule-less environment leaves a team's state under pure manual control
// instead of the reconciler fighting it.
func desiredAccess(windows []schedule.AccessWindow, team db.Team, now time.Time) (want string, enforce bool) {
	// An active manual override wins while it holds.
	if team.AccessOverrideState != "" {
		if !team.AccessOverrideUntil.Valid || now.Before(team.AccessOverrideUntil.Time) {
			return team.AccessOverrideState, true
		}
	}
	if len(windows) == 0 {
		return "", false // no schedule and no live override -- manual control
	}
	// A lapsed OPEN override is a manually-set access END (extend/reduce) that has
	// passed: the team stays CLOSED for the rest of the window that end belonged
	// to, and only re-opens for a genuinely new window that STARTS after it. This
	// is what makes "reduce the end by N" actually stick instead of the schedule
	// re-opening the trimmed window. A lapsed CLOSED override has no such tail and
	// just falls through to the schedule.
	if team.AccessOverrideState == "open" && team.AccessOverrideUntil.Valid && !now.Before(team.AccessOverrideUntil.Time) &&
		!schedule.OpenInWindowStartingAfter(windows, now, team.AccessOverrideUntil.Time) {
		return "closed", true
	}
	if schedule.ScheduledOpen(windows, now) {
		return "open", true
	}
	return "closed", true
}
