package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// The one tool that changes anything: power. It goes through the API's power
// endpoint (POST /builds/{id}/power) via the action bridge, as the key's
// owner at their real level, so it needs manage on the build's repository --
// exactly the UI's power buttons. Networks can't be powered, and rebuilds,
// deploys and teardowns aren't offered here at all.

type powerIn struct {
	BuildID string            `json:"build_id" jsonschema:"the build's id, or a unique prefix of it such as 9531b5b4"`
	Action  string            `json:"action" jsonschema:"start, stop or reboot"`
	Force   bool              `json:"force,omitempty" jsonschema:"true for a hard power operation at the hoster (pull the plug / reset) instead of a graceful OS shutdown or restart"`
	Objects []string          `json:"objects,omitempty" jsonschema:"the hosts or containers to act on, by id, id prefix or name (names with team when several teams have one)"`
	Team    int               `json:"team,omitempty" jsonschema:"only this team number (also disambiguates names in objects)"`
	Kind    string            `json:"kind,omitempty" jsonschema:"only hosts or only containers: host or container"`
	Search  string            `json:"search,omitempty" jsonschema:"only objects whose name contains this text"`
	Network string            `json:"network,omitempty" jsonschema:"only objects on this network"`
	Tags    map[string]string `json:"tags,omitempty" jsonschema:"only objects with every one of these tags (an empty value matches any value)"`
	All     bool              `json:"all,omitempty" jsonschema:"set true to act on every host and container in the build when no other filter is given"`
}

type powerResult struct {
	ObjectID string `json:"object_id"`
	Name     string `json:"name"`
	Error    string `json:"error,omitempty"`
}

type powerOut struct {
	Build   string        `json:"build"`
	Action  string        `json:"action"`
	Force   bool          `json:"force"`
	Started int           `json:"started" jsonschema:"how many hosts the action is now running on, in the background"`
	Failed  int           `json:"failed" jsonschema:"how many matched hosts were refused at once (not deployed yet)"`
	Results []powerResult `json:"results"`
	Note    string        `json:"note"`
}

func (s *Server) registerPowerTool() {
	addToolWith(s, &sdk.Tool{
		Name:  "power",
		Title: "Power Hosts",
		Description: "Starts, stops or reboots hosts and containers at the hoster -- gracefully, or with force for a hard power-off or reset. " +
			"Needs manage access to the build's repository, as in the LaForge UI. Pick targets with objects (names or ids), or with team, kind, search, network and tags; " +
			"acting on a whole build needs all=true. Networks can't be powered. The action runs in the background: each host's result appears in list_object_events, and the totals in list_build_events.",
		Annotations: destructive("Power Hosts"),
	}, toolPower)
}

func toolPower(ctx context.Context, c *call, in powerIn) (powerOut, error) {
	switch in.Action {
	case "start", "stop", "reboot":
	default:
		return powerOut{}, errors.New("action must be start, stop or reboot")
	}
	if in.Kind != "" && in.Kind != "host" && in.Kind != "container" {
		return powerOut{}, errors.New("kind must be host or container; networks can't be powered")
	}
	id, err := c.build(ctx, in.BuildID)
	if err != nil {
		return powerOut{}, err
	}

	target := map[string]any{}
	if len(in.Objects) > 0 {
		ids := make([]string, 0, len(in.Objects))
		for _, ref := range in.Objects {
			o, err := c.findObject(ctx, id, ref, in.Team)
			if err != nil {
				return powerOut{}, err
			}
			if o.Kind == "network" || o.Kind == "dns" {
				return powerOut{}, fmt.Errorf("%s is a %s; only hosts and containers can be powered", ref, o.Kind)
			}
			ids = append(ids, o.ID.String())
		}
		target["ids"] = ids
	} else {
		if in.Team != 0 {
			target["team"] = in.Team
		}
		if in.Kind != "" {
			target["kind"] = in.Kind
		}
		if in.Search != "" {
			target["search"] = in.Search
		}
		if in.Network != "" {
			target["network"] = in.Network
		}
		if len(in.Tags) > 0 {
			target["tags"] = in.Tags
		}
		if len(target) == 0 && !in.All {
			return powerOut{}, errors.New("no target: name objects, or filter by team, kind, search, network or tags -- or set all=true for every host and container in the build")
		}
	}

	if ok, wait := c.s.powerLim.allow(c.key.AccountID, time.Now()); !ok {
		return powerOut{}, fmt.Errorf("%w: power actions are limited to one every 10 seconds; retry in %s", errRateLimited, wait.Round(time.Second)+time.Second)
	}

	body, _ := json.Marshal(map[string]any{"target": target, "action": in.Action, "force": in.Force})
	path := "/builds/" + id.String() + "/power"
	began := time.Now()
	status, resp, err := c.s.cfg.Access.Act(ctx, c.key.AccountID, c.key.Login, path, body)
	step := traceStep{Kind: "power", Name: "POST " + path, Detail: string(body), Status: status, Bytes: len(resp), Response: truncUTF8(string(resp), 16<<10)}
	if err != nil {
		step.Error = err.Error()
	}
	c.trace.add(step, began)
	if err != nil {
		return powerOut{}, err
	}
	if status != http.StatusAccepted && status != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(resp, &e)
		msg := strings.TrimSpace(e.Error)
		if status == http.StatusForbidden || status == http.StatusUnauthorized {
			return powerOut{}, fmt.Errorf("%w: power needs manage access to this build's repository (%s)", errDenied, msg)
		}
		return powerOut{}, fmt.Errorf("power refused (%d): %s", status, msg)
	}
	var r struct {
		Started int           `json:"started"`
		Failed  int           `json:"failed"`
		Results []powerResult `json:"results"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return powerOut{}, fmt.Errorf("decoding the power response: %w", err)
	}
	if r.Results == nil {
		r.Results = []powerResult{}
	}
	return powerOut{
		Build: id.String(), Action: in.Action, Force: in.Force, Started: r.Started, Failed: r.Failed, Results: r.Results,
		Note: "Running in the background. Each host's result appears in list_object_events (power.* events); check power state with list_objects.",
	}, nil
}
