// Infrastructure power actions: start / stop / reboot a deployed instance
// at the hoster, hard or graceful. Distinct from an ad-hoc agent task
// (tasks.go): this goes through the build's builder, not the in-guest
// agent, so it works on a host whose agent is dead or whose OS is wedged --
// the whole reason a "hard reset" exists. Manage-level, like ad-hoc
// dispatch, and the matched blast radius is the same target selector.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

type powerRequest struct {
	Target adHocTarget `json:"target"`
	Action string      `json:"action"` // start | stop | reboot
	Force  bool        `json:"force"`  // hard power operation vs graceful OS-level
}

func (s *Server) handlePowerAction(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelManage); err != nil {
		writeAuthError(w, err)
		return
	}
	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch req.Action {
	case builder.PowerStart, builder.PowerStop, builder.PowerReboot:
	default:
		writeError(w, http.StatusBadRequest, errors.New("action must be start, stop, or reboot"))
		return
	}

	job, err := orchestrator.PreparePower(r.Context(), s.Pool, build.ID, req.Target, req.Action, req.Force)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(job.Results) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("no hosts match this target"))
		return
	}

	kind := req.Action
	if req.Force {
		kind = "force " + kind
	}
	eventKind := "power." + strings.ReplaceAll(kind, " ", "_")
	actor := s.actorFor(r)

	// Attribute an event to each host the action hit, so it shows on that
	// host's own Events timeline (ListEventsByDeployedObject) -- not just the
	// build-wide Logs tab.
	record := func(ctx context.Context, res orchestrator.PowerResult) {
		objID, perr := parseUUID(res.ObjectID)
		if perr != nil {
			return
		}
		msg := kind + " done"
		payload := []byte("{}")
		if res.Err != "" {
			msg = kind + " — failed: " + res.Err
			if p, merr := json.Marshal(map[string]any{"error": res.Err}); merr == nil {
				payload = p
			}
		}
		s.Queries.CreateObjectEvent(ctx, db.CreateObjectEventParams{
			BuildID: build.ID, DeployedObjectID: objID, Kind: eventKind, Message: msg, Payload: payload,
		})
	}
	// Hosts that were never deployed fail now, before anything runs.
	notDeployed := 0
	for _, res := range job.Results {
		if res.Err != "" {
			notDeployed++
			record(r.Context(), res)
		}
	}

	// The action itself runs in the background, not inside this request: a
	// bulk action across teams takes minutes (graceful stops wait for each
	// guest), and Cloudflare ends a request after 100 seconds -- which used to
	// cancel the rest of the action part-way through and report it failed.
	// Detached from the request so a closed tab doesn't stop it either; each
	// host's result lands on its Events tab as it finishes, and the build log
	// gets the totals at the end.
	pending := job.Pending()
	// What this request answers with: taken before the background run starts
	// filling in results, which it does concurrently.
	immediate := append([]orchestrator.PowerResult(nil), job.Results...)
	if pending > 0 {
		s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
			BuildID: build.ID, Kind: eventKind + ".started", Payload: []byte("{}"),
			Message: fmt.Sprintf("%s started on %d host(s) by %s; each host's result appears on its Events tab", kind, pending, actor),
		})
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), powerJobTimeout)
		go func() {
			defer cancel()
			started := time.Now()
			results := job.Run(ctx, func(res orchestrator.PowerResult) { record(ctx, res) })
			failed := 0
			for _, res := range results {
				if res.Err != "" {
					failed++
				}
			}
			msg := powerMessage(kind, len(results), failed) + fmt.Sprintf(" in %s", time.Since(started).Round(time.Second))
			log.Printf("api: build %s: %s (by %s)", build.ID.String()[:8], msg, actor)
			s.Queries.CreateEvent(ctx, db.CreateEventParams{BuildID: build.ID, Kind: eventKind, Message: msg, Payload: []byte("{}")})
		}()
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"started": pending, "results": immediate, "failed": notDeployed})
}

// powerJobTimeout bounds one background bulk power action: long enough for
// every host of a large build to stop gracefully, eight at a time.
const powerJobTimeout = 45 * time.Minute

func powerMessage(kind string, total, failed int) string {
	noun := "host"
	if total != 1 {
		noun = "hosts"
	}
	msg := strconv.Itoa(total) + " " + noun + ": " + kind
	if failed > 0 {
		msg += " (" + strconv.Itoa(failed) + " failed)"
	}
	return msg
}
