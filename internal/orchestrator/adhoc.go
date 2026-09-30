package orchestrator

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// AdHocTarget is the real (scoped) filter shape ad-hoc dispatch uses --
// `object, tag, network, team, search`. Every set field
// narrows the match (AND); no filters at all means "every object in this
// build." IDs, when non-empty, is the ENTIRE match on its own (see
// MatchAdHocTargets' own doc comment).
//
// Lives here, not in internal/api, so the exact same matching logic
// serves both the immediate ad-hoc endpoint (POST /builds/{id}/tasks)
// and the scheduled-task dispatch loop -- one real implementation, not
// two that have to be kept in sync by hand.
type AdHocTarget struct {
	IDs     []string `json:"ids,omitempty"`
	Team    *int32   `json:"team,omitempty"`
	Kind    string   `json:"kind,omitempty"`    // host | container -- empty means either
	Search  string   `json:"search,omitempty"`
	Network string   `json:"network,omitempty"` // match objects on this network
	// Tags: an object matches only if it carries every one of these tags. An
	// empty value matches any value for that key ("has tag k"); a non-empty
	// value requires an exact match ("k=v"). This is the "operate on them" half
	// of tags -- the runtime tags now live on deployed_object (migration 00028).
	Tags map[string]string `json:"tags,omitempty"`
}

// MatchAdHocTargets resolves target against buildID's real
// deployed_object rows -- the exact matching rules
// internal/api/tasks.go's handleCreateAdHocTask has always used,
// factored out so the scheduled-task dispatch loop (internal/orchestrator's
// own ticker) re-resolves a stored target the identical way at fire
// time, since the matching set of hosts can genuinely change between
// when a schedule was created and when it fires.
//
// IDs, when set, is the complete match by itself (team/kind/search are
// ignored) -- added after a real bug: matching by name/search alone
// silently matched every OTHER team's identically-named host too, since
// "every team gets exactly the same network" means hostnames are never
// unique build-wide, only per team.
func MatchAdHocTargets(ctx context.Context, q *db.Queries, buildID pgtype.UUID, target AdHocTarget) ([]db.DeployedObject, error) {
	all, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	teams, err := q.ListTeamsByBuild(ctx, buildID)
	if err != nil {
		return nil, err
	}
	teamNumberByID := make(map[string]int32, len(teams))
	for _, tm := range teams {
		teamNumberByID[tm.ID.String()] = tm.TeamNumber
	}

	var wantIDs map[string]bool
	if len(target.IDs) > 0 {
		wantIDs = make(map[string]bool, len(target.IDs))
		for _, id := range target.IDs {
			wantIDs[id] = true
		}
	}

	matched := make([]db.DeployedObject, 0, len(all))
	for _, o := range all {
		if o.Kind == "network" {
			continue // ad-hoc commands target a live agent; a network has none
		}
		if wantIDs != nil {
			if wantIDs[o.ID.String()] {
				matched = append(matched, o)
			}
			continue
		}
		if target.Team != nil && teamNumberByID[o.TeamID.String()] != *target.Team {
			continue
		}
		if target.Kind != "" && o.Kind != target.Kind {
			continue
		}
		if target.Search != "" {
			needle := strings.ToLower(target.Search)
			hay := strings.ToLower(o.ObjectName + " " + db.StrOrEmpty(o.AsName))
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		if target.Network != "" && db.StrOrEmpty(o.NetworkName) != target.Network {
			continue
		}
		if len(target.Tags) > 0 && !objectHasAllTags(o.Tags, target.Tags) {
			continue
		}
		matched = append(matched, o)
	}
	return matched, nil
}

// objectHasAllTags reports whether an object's persisted tags (deployed_object.tags
// JSON) satisfy every wanted tag: the key must be present, and when the wanted
// value is non-empty it must match exactly. An empty wanted value means "has this
// key, any value."
func objectHasAllTags(objTagsJSON []byte, want map[string]string) bool {
	objTags := map[string]string{}
	if len(objTagsJSON) > 0 {
		_ = json.Unmarshal(objTagsJSON, &objTags)
	}
	for k, v := range want {
		got, ok := objTags[k]
		if !ok {
			return false
		}
		if v != "" && got != v {
			return false
		}
	}
	return true
}
