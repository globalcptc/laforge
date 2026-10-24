package orchestrator

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// UpcomingChange is one object whose desired state, resolved from
// repoRoot's current content, no longer matches what's actually deployed
// for buildID -- "what deploying [a newer commit] would alter, at
// environment, team, network, and host level". Grouping by Team and NetworkName is left to
// the caller; this is deliberately a flat list, the same granularity
// Reconcile itself operates at.
type UpcomingChange struct {
	Team        int    `json:"team"`
	Kind        string `json:"kind"` // "network", "host", "container", "dns"
	ObjectName  string `json:"object_name"`
	AsName      string `json:"as_name,omitempty"` // empty for a network
	NetworkName string `json:"network_name,omitempty"`
	// Change is "new" (nothing deployed yet has this identity), "changed"
	// (a real fingerprint mismatch -- would destroy then redeploy), or
	// "removed" (still deployed, no longer in the desired topology at
	// all -- would be destroyed and not replaced).
	Change string `json:"change"`
}

// DiffUpcoming computes what a Reconcile pass against repoRoot's CURRENT
// content would do to buildID, without writing anything -- no
// EnsureTeam, no EnsureDeployedObject, no task creation. It mirrors
// Reconcile's own resolution (render.Resolve + Fingerprint, per team per
// copy) and its own removal logic (reconcileRemovals), but reads
// deployed_object's existing rows instead of writing to them, and
// compares against those instead of creating tasks.
//
// Like Reconcile itself, this assumes repoRoot's checkout reflects
// whatever commit the comparison is meant to be against -- typically the
// configured build's current_content_revision_id, once that has
// advanced past the build's own content_revision_id. The caller is
// responsible for deciding whether there's actually a newer commit worth
// diffing against before calling this -- DiffUpcoming has no way to know
// which commit repoRoot is checked out to.
func DiffUpcoming(ctx context.Context, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID) ([]UpcomingChange, error) {
	q := db.New(pool)

	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("loading build: %w", err)
	}

	c, err := loader.Load(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("loading content: %w", err)
	}
	if len(c.Errors) > 0 {
		return nil, fmt.Errorf("content at %s is not valid, refusing to diff: %d schema/reference error(s) (first: %s)",
			repoRoot, len(c.Errors), c.Errors[0].Message)
	}

	env := findEnvironment(c, build.EnvironmentName)
	if env == nil {
		return nil, fmt.Errorf("environment %q not found in content at %s", build.EnvironmentName, repoRoot)
	}

	containerNames := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		containerNames[ct.Name] = true
	}

	teams, err := q.ListTeamsByBuild(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}
	teamNumByID := make(map[pgtype.UUID]int, len(teams))
	for _, t := range teams {
		teamNumByID[t.ID] = int(t.TeamNumber)
	}

	allExisting, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("listing deployed objects: %w", err)
	}
	// existingByTeam[teamNum][kind+"/"+identity] -> the deployed_object
	// row, so both the per-copy comparison and the end-of-team removal
	// sweep can look objects up the same way Reconcile's own `desired`
	// map does.
	existingByTeam := make(map[int]map[string]db.DeployedObject, len(teams))
	for _, obj := range allExisting {
		teamNum, ok := teamNumByID[obj.TeamID]
		if !ok {
			continue // an object from a team this build no longer lists -- can't happen via normal Reconcile, skip defensively
		}
		identity := db.StrOrEmpty(obj.AsName)
		if identity == "" {
			identity = obj.ObjectName
		}
		if existingByTeam[teamNum] == nil {
			existingByTeam[teamNum] = make(map[string]db.DeployedObject)
		}
		existingByTeam[teamNum][obj.Kind+"/"+identity] = obj
	}

	var changes []UpcomingChange

	for teamNum := 1; teamNum <= env.Teams; teamNum++ {
		existing := existingByTeam[teamNum]
		desired := make(map[string]bool)

		for networkName := range env.Networks {
			net := findNetwork(c, networkName)
			if net == nil {
				return nil, fmt.Errorf("network %q not found in content", networkName)
			}
			key := "network/" + networkName
			desired[key] = true
			fp := networkFingerprint(net)
			if ch, ok := diffOne(existing[key], fp); ok {
				changes = append(changes, UpcomingChange{Team: teamNum, Kind: "network", ObjectName: networkName, Change: ch})
			}
		}

		for networkName, objs := range env.Networks {
			for objectName, copies := range objs {
				kind := "host"
				if containerNames[objectName] {
					kind = "container"
				}
				for _, cp := range copies {
					key := kind + "/" + cp.As
					desired[key] = true

					rctx, err := render.Resolve(c, env.Name, cp.As, teamNum)
					if err != nil {
						return nil, fmt.Errorf("team %d, %s %q: resolving: %w", teamNum, kind, cp.As, err)
					}
					var disk int
					var ports loader.Ports
					var dependsOn []string
					if kind == "host" {
						h := findHost(c, objectName)
						if h == nil {
							return nil, fmt.Errorf("host %q not found in content", objectName)
						}
						disk, ports, dependsOn = h.Disk, h.Ports, h.DependsOn
					} else {
						ct := findContainer(c, objectName)
						if ct == nil {
							return nil, fmt.Errorf("container %q not found in content", objectName)
						}
						disk, ports, dependsOn = ct.Disk, ct.Ports, ct.DependsOn
					}
					fp, err := Fingerprint(repoRoot, c, rctx, disk, ports, dependsOn)
					if err != nil {
						return nil, fmt.Errorf("team %d, %s %q: fingerprinting: %w", teamNum, kind, cp.As, err)
					}
					if ch, ok := diffOne(existing[key], fp); ok {
						changes = append(changes, UpcomingChange{
							Team: teamNum, Kind: kind, ObjectName: objectName, AsName: cp.As, NetworkName: networkName, Change: ch,
						})
					}
				}
			}
		}

		for key, obj := range existing {
			if desired[key] || obj.Status == "destroyed" {
				continue
			}
			changes = append(changes, UpcomingChange{
				Team: teamNum, Kind: obj.Kind, ObjectName: obj.ObjectName, AsName: db.StrOrEmpty(obj.AsName), NetworkName: db.StrOrEmpty(obj.NetworkName), Change: "removed",
			})
		}
	}

	return changes, nil
}

// diffOne is the one-object version of reconcileObject's own diff logic,
// read-only: ok is false when nothing would change (matches, or an
// in-flight destroy/not-yet-deployed object Reconcile would leave alone
// this pass anyway).
func diffOne(existing db.DeployedObject, desiredFP string) (change string, ok bool) {
	if !existing.ID.Valid {
		return "new", true
	}
	if infraUp(existing.Status) {
		if existing.Fingerprint == desiredFP {
			return "", false
		}
		return "changed", true
	}
	return "", false // pending/deploying/deploy_failed/destroying/destroyed -- Reconcile itself would just (re)try the same desired state, not a "change" from this commit
}
