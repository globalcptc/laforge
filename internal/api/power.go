// Infrastructure power actions: start / stop / reboot a deployed instance
// at the hoster, hard or graceful. Distinct from an ad-hoc agent task
// (tasks.go): this goes through the build's builder, not the in-guest
// agent, so it works on a host whose agent is dead or whose OS is wedged --
// the whole reason a "hard reset" exists. Manage-level, like ad-hoc
// dispatch, and the matched blast radius is the same target selector.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

	results, err := orchestrator.PowerObjects(r.Context(), s.Pool, build.ID, req.Target, req.Action, req.Force)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(results) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("no hosts match this target"))
		return
	}

	failed := 0
	for _, res := range results {
		if res.Err != "" {
			failed++
		}
	}
	kind := req.Action
	if req.Force {
		kind = "force " + kind
	}
	eventKind := "power." + strings.ReplaceAll(kind, " ", "_")
	// Attribute an event to each host the action hit, so it shows on that host's
	// own Events timeline (ListEventsByDeployedObject) -- not just the build-wide
	// Logs tab. A build-level CreateEvent carries no object and never reaches a
	// host's panel, which is why a forced reboot was invisible there.
	for _, res := range results {
		objID, perr := parseUUID(res.ObjectID)
		if perr != nil {
			continue
		}
		msg := kind
		payload := []byte("{}")
		if res.Err != "" {
			msg = kind + " — failed: " + res.Err
			if p, merr := json.Marshal(map[string]any{"error": res.Err}); merr == nil {
				payload = p
			}
		}
		s.Queries.CreateObjectEvent(r.Context(), db.CreateObjectEventParams{
			BuildID: build.ID, DeployedObjectID: objID, Kind: eventKind, Message: msg, Payload: payload,
		})
	}
	// A rollup on the build log for a bulk action across several hosts; a
	// single-host action is already covered by its own per-object event above.
	if len(results) > 1 {
		s.Queries.CreateEvent(r.Context(), db.CreateEventParams{
			BuildID: build.ID, Kind: eventKind,
			Message: powerMessage(kind, len(results), failed),
			Payload: []byte("{}"),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results, "failed": failed})
}

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
