package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// Every decision the orchestrator takes about a build is recorded twice: as
// an event on the build's timeline (the UI's log, keyed to the object it's
// about) and as a line in the orchestrator's own log. The reconciler runs
// every couple of seconds and most passes decide nothing; what's recorded is
// each change it makes or asks for -- a deploy or destroy requested and
// why, a host held, a host failed because of its network or a dependency,
// steps queued or released, a status moving -- so that when a build stalls
// the log says what LaForge itself last did and why, not only what the
// hoster was asked.

// decide records one decision about obj (nil object id for a build-wide one).
func decide(ctx context.Context, q *db.Queries, buildID, objectID pgtype.UUID, kind, message string, detail map[string]any) {
	payload := []byte("{}")
	if len(detail) > 0 {
		if b, err := json.Marshal(detail); err == nil {
			payload = b
		}
	}
	log.Printf("orchestrator: build %s %s: %s -- %s %s", shortID(buildID), shortID(objectID), kind, message, string(payload))
	if objectID.Valid {
		q.CreateObjectEvent(ctx, db.CreateObjectEventParams{BuildID: buildID, DeployedObjectID: objectID, Kind: kind, Message: message, Payload: payload})
		return
	}
	q.CreateEvent(ctx, db.CreateEventParams{BuildID: buildID, Kind: kind, Message: message, Payload: payload})
}

// decideRepeating is decide for a condition the reconciler meets on every
// pass while it lasts -- an object waiting on an open task, a network, an
// agent. It is logged once a minute (throttled on key), but written to the
// build's journal only once every 30 minutes: the journal should say the
// wait is happening and what on, not fill with the same line every minute
// for every host of a stuck build (a Logs page that can't load records
// nothing). Returns whether it reported at all.
func decideRepeating(ctx context.Context, q *db.Queries, key string, buildID, objectID pgtype.UUID, kind, message string, detail map[string]any) bool {
	if !throttled(key, time.Minute) {
		return false
	}
	if throttled(key+"/journal", 30*time.Minute) {
		decide(ctx, q, buildID, objectID, kind, message, detail)
		return true
	}
	payload := "{}"
	if len(detail) > 0 {
		if b, err := json.Marshal(detail); err == nil {
			payload = string(b)
		}
	}
	log.Printf("orchestrator: build %s %s: %s -- %s %s", shortID(buildID), shortID(objectID), kind, message, payload)
	return true
}

// objectLabel names an object the way the UI does: "web01 (host, team 3)".
func objectLabel(obj db.DeployedObject, team int32) string {
	name := db.StrOrEmpty(obj.AsName)
	if name == "" {
		name = obj.ObjectName
	}
	if team > 0 {
		return fmt.Sprintf("%s (%s, team %d)", name, obj.Kind, team)
	}
	return fmt.Sprintf("%s (%s)", name, obj.Kind)
}

func shortID(id pgtype.UUID) string {
	if !id.Valid {
		return "-"
	}
	return id.String()[:8]
}

// throttled reports whether key was last reported more than every ago, and
// records it if so -- for a condition the reconciler meets on every pass
// while it lasts (an object waiting on an open task, a host waiting for its
// network), which is worth a line every minute, not every two seconds.
func throttled(key string, every time.Duration) bool {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	now := time.Now()
	if last, ok := throttleLast[key]; ok && now.Sub(last) < every {
		return false
	}
	throttleLast[key] = now
	if len(throttleLast) > 10000 {
		for k, t := range throttleLast {
			if now.Sub(t) > every {
				delete(throttleLast, k)
			}
		}
	}
	return true
}

var (
	throttleMu   sync.Mutex
	throttleLast = map[string]time.Time{}
)
