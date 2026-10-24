package orchestrator

import (
	"context"
	"errors"
	"fmt"
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

// PowerObjects runs an infrastructure power action -- start, stop, or
// reboot, hard or graceful -- against every deployed object the target
// matches, at the hoster, through the build's own builder (the same
// build -> configured_build -> builder_config resolution DetectDrift
// uses). Unlike an ad-hoc agent task, this is a hoster operation: it acts
// on the instance itself, so it still works when the in-guest agent is
// dead or the OS is wedged -- exactly when you most need it. Objects run
// concurrently (bounded) so a graceful action that waits per instance
// doesn't serialize across a whole build.
func PowerObjects(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID, target AdHocTarget, action string, force bool) ([]PowerResult, error) {
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

	results := make([]PowerResult, len(objs))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, o := range objs {
		name := db.StrOrEmpty(o.AsName)
		if name == "" {
			name = o.ObjectName
		}
		results[i] = PowerResult{ObjectID: o.ID.String(), Name: name}
		if o.ExternalRef == nil || *o.ExternalRef == "" {
			results[i].Err = "not deployed at the hoster yet"
			continue
		}
		team := strconv.FormatInt(int64(teamNumberByID[o.TeamID.String()]), 10)
		ref := *o.ExternalRef
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := bldr.PowerAction(ctx, team, ref, action, force); err != nil {
				results[i].Err = err.Error()
			}
		}(i)
	}
	wg.Wait()
	return results, nil
}
