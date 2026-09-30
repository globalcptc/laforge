// Teardown made real: "teardown (infrastructure)...
// available from the UI and the API", found completely
// missing by direct audit -- build.status's own CHECK constraint
// (migration 00003) already named 'torn_down' as a real terminal state,
// but nothing anywhere ever created a way to reach it. The only destroy
// machinery that existed was Reconcile's own per-object diff against
// desired content, which only ever destroys an object that fell OUT of
// content -- never a whole build on a deliberate operator action, and
// content might not even be resolvable by the time someone wants a build
// torn down (repository access revoked, branch deleted, checkout gone).
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
)

// Teardown requests a real, terminal destroy for every deployed_object
// belonging to buildID -- unlike Reconcile, it never loads content at
// all, since a build has to be destroyable even when its content no
// longer is. Idempotent and safe to call repeatedly (the same
// `CreateTaskIfNoneOpen` partial-unique-index guard Reconcile's own
// requestDestroy already relies on), and safe to call concurrently from
// more than one orchestrator replica for the same reason every other
// function in this package is. Transitions buildID itself to 'torn_down'
// exactly once every object has genuinely reached the real terminal
// 'destroyed' state (MarkDeployedObjectDestroyed, via
// internal/runner.executeDestroy's own `terminal` payload flag) -- not
// merely once every destroy task has been created, so a caller polling
// the build's own status sees 'torn_down' only once it's actually true.
func Teardown(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) error {
	q := db.New(pool)

	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("listing deployed objects: %w", err)
	}

	allDestroyed := true
	for _, obj := range objs {
		if obj.Status == "destroyed" {
			continue
		}
		allDestroyed = false
		// Re-request a destroy for anything not yet terminally destroyed --
		// including an object stuck in 'destroying' whose destroy task has
		// since failed and exhausted its retries. CreateTaskIfNoneOpen is a
		// no-op while a destroy is genuinely in flight (its partial unique
		// index conflicts on an open pending/leased task), so this both
		// starts the first destroy and, on a later teardown pass, self-heals
		// a wedged one: e.g. a network that returned "currently in use" while
		// its instances were still detaching, exhausted its 3 attempts, and
		// would otherwise sit in 'destroying' forever even though the
		// instances are long gone. Each pass that finds no open task queues a
		// fresh one (attempts reset), so a transient failure can no longer
		// permanently strand a teardown -- it just retries until the object
		// truly reaches 'destroyed'.
		payload, err := json.Marshal(map[string]bool{"terminal": true})
		if err != nil {
			return err
		}
		if _, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
			BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload,
		}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("requesting destroy for %s %q: %w", obj.Kind, obj.ObjectName, err)
		}
	}

	if allDestroyed {
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: buildID, Status: "torn_down"}); err != nil {
			return fmt.Errorf("marking build torn down: %w", err)
		}
	}
	return nil
}
