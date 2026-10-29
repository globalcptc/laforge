package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builderconfig"
	"github.com/globalcptc/laforge/internal/db"
)

// PowerResult is one object's outcome from a PowerObjects call -- a real
// per-host answer, since a power action can succeed on some hosts and fail
// on others (one host wedged, the rest fine).
type PowerResult struct {
	ObjectID string `json:"object_id"`
	Name     string `json:"name"`
	Err      string `json:"error,omitempty"`
}

// PowerJob is a bulk power action that has been checked and matched but not
// yet run: PreparePower resolves the builder and the target hosts quickly (so
// a bad request fails at once), and Run carries it out, which on a large
// target takes minutes. The API runs it in the background rather than inside
// the request: Cloudflare ends any request after 100 seconds, which cancelled
// the rest of a bulk action part-way through and reported it failed.
type PowerJob struct {
	pool   *pgxpool.Pool
	bldr   builder.Builder
	action string
	force  bool
	// Results holds every matched host in match order. Hosts not yet
	// deployed already carry their error; the rest are filled in by Run.
	Results []PowerResult
	todo    []powerTarget
}

type powerTarget struct {
	index    int
	objectID pgtype.UUID
	team     string
	ref      string
}

// PreparePower checks a power action and matches its target hosts, without
// touching the hoster. See PowerJob.
func PreparePower(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, target AdHocTarget, action string, force bool) (*PowerJob, error) {
	switch action {
	case builder.PowerStart, builder.PowerStop, builder.PowerReboot:
	default:
		return nil, fmt.Errorf("unsupported power action %q", action)
	}

	q := db.New(pool)
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("loading build: %w", err)
	}
	if !build.ConfiguredBuildID.Valid {
		return nil, errors.New("this build has no configured_build, so there's no builder to run a power action through")
	}
	cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID)
	if err != nil {
		return nil, fmt.Errorf("loading configured build: %w", err)
	}
	row, err := q.GetBuilderConfigByName(ctx, cb.BuilderConfigName)
	if err != nil {
		return nil, fmt.Errorf("loading builder config %q: %w", cb.BuilderConfigName, err)
	}
	bldr, err := builderconfig.Resolve(pool, row)
	if err != nil {
		return nil, fmt.Errorf("resolving builder %q: %w", cb.BuilderConfigName, err)
	}

	objs, err := MatchAdHocTargets(ctx, q, buildID, target)
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

	job := &PowerJob{pool: pool, bldr: bldr, action: action, force: force, Results: make([]PowerResult, len(objs))}
	for i, o := range objs {
		name := db.StrOrEmpty(o.AsName)
		if name == "" {
			name = o.ObjectName
		}
		job.Results[i] = PowerResult{ObjectID: o.ID.String(), Name: name}
		if o.ExternalRef == nil || *o.ExternalRef == "" {
			job.Results[i].Err = "not deployed at the hoster yet"
			continue
		}
		job.todo = append(job.todo, powerTarget{
			index: i, objectID: o.ID, ref: *o.ExternalRef,
			team: strconv.FormatInt(int64(teamNumberByID[o.TeamID.String()]), 10),
		})
	}
	return job, nil
}

// Pending is how many hosts Run will act on.
func (j *PowerJob) Pending() int { return len(j.todo) }

// powerConcurrency bounds how many hosts one bulk action works on at once.
const powerConcurrency = 8

// Run carries out the action on every pending host, at most powerConcurrency
// at a time, calling onResult as each finishes (from several goroutines;
// onResult must be safe for that). A host that succeeds gets its power state
// recorded straight away -- running after a start or reboot, stopped after a
// stop -- so the UI follows along instead of waiting for the next hoster poll.
// Returns every result, including the hosts that were never deployed.
func (j *PowerJob) Run(ctx context.Context, onResult func(PowerResult)) []PowerResult {
	q := db.New(j.pool)
	settled := builder.PowerStateRunning
	if j.action == builder.PowerStop {
		settled = builder.PowerStateStopped
	}
	sem := make(chan struct{}, powerConcurrency)
	var wg sync.WaitGroup
	for _, t := range j.todo {
		wg.Add(1)
		go func(t powerTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := j.bldr.PowerAction(ctx, t.team, t.ref, j.action, j.force); err != nil {
				j.Results[t.index].Err = err.Error()
			} else if err := q.SetDeployedObjectPowerState(ctx, db.SetDeployedObjectPowerStateParams{ID: t.objectID, PowerState: settled}); err != nil {
				log.Printf("power %s: recording %s's power state: %v", j.action, j.Results[t.index].Name, err)
			}
			if onResult != nil {
				onResult(j.Results[t.index])
			}
		}(t)
	}
	wg.Wait()
	return j.Results
}

// PowerObjects prepares and runs a power action in one call, waiting for every
// host. See PreparePower and PowerJob.Run.
func PowerObjects(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, target AdHocTarget, action string, force bool) ([]PowerResult, error) {
	job, err := PreparePower(ctx, pool, buildID, target, action, force)
	if err != nil {
		return nil, err
	}
	return job.Run(ctx, nil), nil
}
