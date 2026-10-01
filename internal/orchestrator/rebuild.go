package orchestrator

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
)

// RebuildAffected is one object a rebuild will tear down and recreate -- the
// blast radius, returned to the operator both for a dry-run preview (so the
// confirm dialog can show exactly what's affected) and after the real action.
type RebuildAffected struct {
	ID         string `json:"id"`
	ObjectName string `json:"object_name"`
	AsName     string `json:"as_name"`
	Kind       string `json:"kind"`
	TeamNumber int32  `json:"team_number"`
}

// Rebuild forces a rebuild of the objects matched by target and -- when
// includeDependents -- everything in the same team that transitively depends on
// them (loader.Content.Dependents: rebuilding a thing invalidates everything
// built on top of it). dryRun resolves and returns the affected set without
// touching anything; otherwise it enqueues the work and returns the same set.
//
// How a non-dry-run rebuild actually re-enters the deploy pipeline, race-free:
// for each affected object it (1) enqueues a destroy task with remove=false --
// the runner tears the instance down at the hoster, resets the row for
// redeploy, and clears its materialized steps so they re-run -- and (2) resets
// the row to 'pending' synchronously, right now. Step 2 matters: an object left
// 'finished'/'running' would let finalizeBuildStatus mark the whole build
// finished in the window before the destroy runs, and the orchestrator only
// reconciles non-terminal builds, so the reset would then never redeploy.
// Holding the objects at 'pending' (infraPending > 0) keeps the build in the
// deploy phase until the rebuild genuinely completes. The destroy is enqueued
// first so it is the one open task per object; a premature deploy task is
// blocked by task_one_open_per_object until the teardown finishes. Finally the
// build is flipped back to 'deploying' so the poll loop recreates everything.
func Rebuild(ctx context.Context, pool *pgxpool.Pool, build db.Build, content *loader.Content, target AdHocTarget, includeDependents, dryRun bool) ([]RebuildAffected, error) {
	q := db.New(pool)

	affected, err := resolveRebuildSet(ctx, q, build.ID, content, target, includeDependents)
	if err != nil {
		return nil, err
	}

	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		return nil, err
	}
	teamNumberByID := make(map[string]int32, len(teams))
	for _, tm := range teams {
		teamNumberByID[tm.ID.String()] = tm.TeamNumber
	}

	out := make([]RebuildAffected, 0, len(affected))
	for _, o := range affected {
		out = append(out, RebuildAffected{
			ID: o.ID.String(), ObjectName: o.ObjectName, AsName: db.StrOrEmpty(o.AsName),
			Kind: o.Kind, TeamNumber: teamNumberByID[o.TeamID.String()],
		})
	}
	if dryRun || len(affected) == 0 {
		return out, nil
	}

	for _, o := range affected {
		if _, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
			BuildID: build.ID, DeployedObjectID: o.ID, Kind: "destroy_" + o.Kind, Payload: []byte(`{"remove":false}`),
		}); err != nil {
			return nil, fmt.Errorf("enqueuing rebuild destroy for %s: %w", o.ID.String(), err)
		}
		if _, err := q.ResetDeployedObjectForRedeploy(ctx, o.ID); err != nil {
			return nil, fmt.Errorf("resetting %s for rebuild: %w", o.ID.String(), err)
		}
	}
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		return nil, fmt.Errorf("re-entering deploy phase: %w", err)
	}
	return out, nil
}

// resolveRebuildSet returns the deployed_object rows a rebuild touches: the
// objects matched by target (networks already excluded by MatchAdHocTargets),
// plus -- when includeDependents -- every object in the SAME team whose content
// object_name transitively depends on a matched object. No writes.
func resolveRebuildSet(ctx context.Context, q *db.Queries, buildID pgtype.UUID, content *loader.Content, target AdHocTarget, includeDependents bool) ([]db.DeployedObject, error) {
	matched, err := MatchAdHocTargets(ctx, q, buildID, target)
	if err != nil {
		return nil, err
	}
	if !includeDependents || len(matched) == 0 || content == nil {
		return matched, nil
	}

	all, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	// Map (team_id, object_name) -> row, so a dependent object NAME resolves back
	// to THIS team's deployed_object (names repeat across teams).
	byTeamName := make(map[string]db.DeployedObject, len(all))
	for _, o := range all {
		if o.Kind == "network" {
			continue
		}
		byTeamName[o.TeamID.String()+"\x00"+o.ObjectName] = o
	}

	result := make(map[string]db.DeployedObject, len(matched))
	for _, m := range matched {
		result[m.ID.String()] = m // always rebuild the target itself
		for _, name := range content.Dependents([]string{m.ObjectName}) {
			if o, ok := byTeamName[m.TeamID.String()+"\x00"+name]; ok {
				result[o.ID.String()] = o
			}
		}
	}
	out := make([]db.DeployedObject, 0, len(result))
	for _, o := range result {
		out = append(out, o)
	}
	return out, nil
}
