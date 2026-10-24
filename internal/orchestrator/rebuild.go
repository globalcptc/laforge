package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"

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
// How a non-dry-run rebuild re-enters the deploy pipeline: for each affected
// object it (1) enqueues a destroy task with remove=false and (2) marks the
// object destroy_queued -- waiting to be destroyed, its hoster reference kept.
// The runner then marks it destroying, tears the instance down, and only then
// resets it to pending (clearing its steps so they re-run), and the reconciler
// deploys it fresh. destroy_queued counts as infrastructure in progress, so the
// build can't be finalized as finished in the window before the destroy runs
// (the orchestrator only reconciles non-terminal builds; a build finalized
// then would never redeploy). The destroy is the object's one open task, so no
// deploy can be queued until it ends. Finally the build is flipped back to
// 'deploying' so the poll loop recreates everything.
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
		// Capture the real hoster reference INTO the destroy payload before the
		// reset below clears it, so the runner tears down the actual recorded
		// instance (e.g. Incus's `lf-...-efc5c7`) rather than falling back to the
		// logical name and "succeeding" as a no-op against a name that doesn't
		// exist -- which left the original VM running (the rebuild bug).
		payload, err := json.Marshal(map[string]any{"remove": false, "external_ref": db.StrOrEmpty(o.ExternalRef)})
		if err != nil {
			return nil, fmt.Errorf("building rebuild destroy payload for %s: %w", o.ID.String(), err)
		}
		team := teamNumberByID[o.TeamID.String()]
		// A deploy queued but not started is withdrawn: running it would
		// adopt the very instance being rebuilt and make the rebuild a no-op.
		if cancelled, err := q.CancelPendingDeployForObject(ctx, db.CancelPendingDeployForObjectParams{
			DeployedObjectID: o.ID, Reason: "withdrawn: the host is being rebuilt",
		}); err == nil {
			decide(ctx, q, build.ID, o.ID, "rebuild.withdrew_deploy", fmt.Sprintf("%s: withdrew its queued deploy (not started) in favour of the rebuild", objectLabel(o, team)), map[string]any{"task_id": cancelled.ID.String()})
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("withdrawing %s's queued deploy: %w", o.ID.String(), err)
		}
		task, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
			BuildID: build.ID, DeployedObjectID: o.ID, Kind: "destroy_" + o.Kind, Payload: payload,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			// Something is running for it right now (a deploy mid-create, an
			// earlier destroy). Record the request and leave the object as it
			// is -- hoster reference included -- and the reconciler queues the
			// destroy the moment that task ends. (Resetting it here, as this
			// once did, lost both the reference and the request.)
			if err := q.SetRebuildRequested(ctx, o.ID); err != nil {
				return nil, fmt.Errorf("recording the rebuild request for %s: %w", o.ID.String(), err)
			}
			what := "a task"
			if open, err := q.GetOpenTaskForObject(ctx, o.ID); err == nil {
				what = fmt.Sprintf("a %s task (%s, attempt %d)", open.Kind, open.Status, open.Attempts)
			}
			decide(ctx, q, build.ID, o.ID, "rebuild.waiting", fmt.Sprintf("%s: rebuild recorded; it has %s running, and its destroy is queued the moment that ends", objectLabel(o, team), what), nil)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enqueuing rebuild destroy for %s: %w", o.ID.String(), err)
		}
		decide(ctx, q, build.ID, o.ID, "rebuild.destroy", fmt.Sprintf("%s: destroy task queued for the rebuild (hoster ref %s); it redeploys once destroyed", objectLabel(o, team), db.StrOrEmpty(o.ExternalRef)),
			map[string]any{"task_id": task.ID.String(), "external_ref": db.StrOrEmpty(o.ExternalRef)})
		if err := q.ClearRebuildRequested(ctx, o.ID); err != nil {
			return nil, err
		}
		if _, err := q.MarkDeployedObjectDestroyQueued(ctx, o.ID); err != nil {
			return nil, fmt.Errorf("marking %s waiting to be destroyed: %w", o.ID.String(), err)
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
	// Map (team_id, object_name) -> ALL this team's deployed_objects of that name.
	// A name resolves back to THIS team's rows (names repeat across teams), and
	// per-person instances (the `people:` field) share one object_name, so a name
	// maps to SEVERAL rows -- keep them all rather than collapsing to one arbitrary
	// sibling, which both dragged in the wrong instance and under-counted a
	// duplicated dependent.
	byTeamName := make(map[string][]db.DeployedObject, len(all))
	for _, o := range all {
		if o.Kind == "network" {
			continue
		}
		key := o.TeamID.String() + "\x00" + o.ObjectName
		byTeamName[key] = append(byTeamName[key], o)
	}
	// The selected objects' own names. content.Dependents seeds its walk with the
	// roots themselves, so those names come back in the result; excluding them
	// here keeps a sibling instance that merely shares the target's object_name
	// (another person's copy of the same host) from being treated as a dependent.
	// The targets are already included by their exact ids below.
	matchedNames := make(map[string]bool, len(matched))
	for _, m := range matched {
		matchedNames[m.ObjectName] = true
	}

	result := make(map[string]db.DeployedObject, len(matched))
	for _, m := range matched {
		result[m.ID.String()] = m // always rebuild the target itself
		for _, name := range content.Dependents([]string{m.ObjectName}) {
			if matchedNames[name] {
				continue
			}
			for _, o := range byTeamName[m.TeamID.String()+"\x00"+name] {
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
