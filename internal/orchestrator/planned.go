package orchestrator

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// DiscardStalePlannedBuilds moves every stale planned build to the terminal
// 'discarded' status (migration 00063): a planned build whose configured_build
// has been removed, or that has been superseded by a newer planned build for
// the same configured_build. The two write paths (triggerBuild superseding on
// create, handleDeleteConfiguredBuild discarding on remove -- internal/api)
// keep this from happening going forward; this sweep heals the backlog an
// upgraded instance already carries, and backstops those paths. Each discarded
// build gets an event saying why, since the row is kept as history.
//
// Nothing here touches a hoster: a planned build has never created anything
// (reconcile.go's deployTasks gate), so discarding is purely a status change.
// Returns how many builds were discarded.
func DiscardStalePlannedBuilds(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	q := db.New(pool)
	rows, err := q.DiscardStalePlannedBuilds(ctx)
	if err != nil {
		return 0, fmt.Errorf("discarding stale planned builds: %w", err)
	}
	for _, b := range rows {
		msg := "planned build discarded: superseded by a newer planned build of the same configured build"
		if !b.ConfiguredBuildID.Valid {
			msg = "planned build discarded: its configured build was removed"
		}
		if _, err := q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: b.ID, Kind: "build.discarded", Message: msg, Payload: []byte("{}"),
		}); err != nil {
			log.Printf("recording discard event for build %s: %v", b.ID, err)
		}
	}
	return len(rows), nil
}
