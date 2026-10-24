package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/globalcptc/laforge/internal/agentproto"
)

// cliTarget mirrors orchestrator.AdHocTarget's JSON shape -- the server-side
// host selector both /tasks and /scheduled-tasks resolve. Target an exact host
// with --id; a set with --team / --kind / --tag / --search / --network. --id,
// when given, is the whole match on its own (hostnames repeat across teams, so
// an id is the only build-wide-unique handle).
type cliTarget struct {
	IDs     []string          `json:"ids,omitempty"`
	Team    *int              `json:"team,omitempty"`
	Kind    string            `json:"kind,omitempty"`
	Search  string            `json:"search,omitempty"`
	Network string            `json:"network,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// stringList is a repeatable string flag (--id a --id b).
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// targetFlags registers the shared selector flags on fs and returns a closure
// that assembles the resolved cliTarget once fs has been parsed.
func targetFlags(fs *flag.FlagSet) func() cliTarget {
	team := fs.Int("team", -1, "target every host/container in this team number")
	kind := fs.String("kind", "", "restrict to 'host' or 'container'")
	search := fs.String("search", "", "match hosts whose name contains this text")
	network := fs.String("network", "", "restrict to hosts on this network")
	var ids stringList
	var tags stringList
	fs.Var(&ids, "id", "target an exact deployed_object id (repeatable)")
	fs.Var(&tags, "tag", "match hosts carrying this tag: k or k=v (repeatable)")
	return func() cliTarget {
		t := cliTarget{Kind: *kind, Search: *search, Network: *network}
		if len(ids) > 0 {
			t.IDs = ids
		}
		if *team >= 0 {
			tm := *team
			t.Team = &tm
		}
		if len(tags) > 0 {
			t.Tags = map[string]string{}
			for _, kv := range tags {
				k, v, _ := strings.Cut(kv, "=")
				t.Tags[k] = v
			}
		}
		return t
	}
}

// buildCommandPayload turns the --command kind plus its inputs into the task's
// command string and JSON payload. execute (the default) takes the shell
// command as the positional args after `--`; any other kind takes a raw
// --payload JSON (its shape is that command's agentproto payload). reboot needs
// neither.
func buildCommandPayload(command, rawPayload string, shell []string, timeout int) (string, json.RawMessage, error) {
	if command == "" {
		command = agentproto.CmdExecute
	}
	if rawPayload != "" {
		if !json.Valid([]byte(rawPayload)) {
			return "", nil, fmt.Errorf("--payload is not valid JSON")
		}
		return command, json.RawMessage(rawPayload), nil
	}
	switch command {
	case agentproto.CmdExecute:
		if len(shell) == 0 {
			return "", nil, fmt.Errorf("nothing to run -- put the command after `--`, e.g. laforge run --build ID --team 1 -- systemctl restart nginx")
		}
		p := agentproto.ExecutePayload{Command: "/bin/sh", Args: []string{"-c", strings.Join(shell, " ")}, TimeoutSec: timeout}
		b, err := json.Marshal(p)
		return command, b, err
	case agentproto.CmdReboot:
		return command, json.RawMessage("{}"), nil
	default:
		return "", nil, fmt.Errorf("command %q needs --payload '<json>' (its agentproto payload shape)", command)
	}
}

// runRun dispatches an ad-hoc command to the matched hosts immediately
// (POST /builds/{id}/tasks). Manage-level: the caller's stored token
// (laforge login) must have manage on the repo.
func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	build := fs.String("build", "", "build id to run against (required)")
	command := fs.String("command", "", "agent command kind (default: execute)")
	payload := fs.String("payload", "", "raw JSON payload, for a non-execute command")
	timeout := fs.Int("timeout", 300, "execute timeout, in seconds")
	dryRun := fs.Bool("dry-run", false, "show the matched hosts without running anything")
	resolveTarget := targetFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" {
		return fmt.Errorf("usage: laforge run --build <id> [--team N | --id OBJID | --tag k=v | --kind host] [--dry-run] -- <command>")
	}
	cmd, pl, err := buildCommandPayload(*command, *payload, fs.Args(), *timeout)
	if err != nil {
		return err
	}
	req := map[string]any{"target": resolveTarget(), "command": cmd, "payload": pl, "dry_run": *dryRun}

	var res struct {
		Matched int `json:"matched"`
		Objects []struct {
			ObjectName string  `json:"object_name"`
			AsName     *string `json:"as_name"`
			Kind       string  `json:"kind"`
		} `json:"objects"`
		Created []json.RawMessage `json:"created"`
	}
	if err := newAPIClient().do("POST", "/builds/"+url.PathEscape(*build)+"/tasks", req, &res); err != nil {
		return err
	}
	if *dryRun {
		fmt.Printf("would run %q on %d host(s):\n", cmd, res.Matched)
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		for _, o := range res.Objects {
			name := o.ObjectName
			if o.AsName != nil && *o.AsName != "" {
				name = *o.AsName
			}
			fmt.Fprintf(w, "  %s\t%s\n", name, o.Kind)
		}
		return w.Flush()
	}
	fmt.Printf("queued %q on %d host(s)\n", cmd, res.Matched)
	return nil
}

// runSchedule registers a recurring/future command (POST /builds/{id}/scheduled-tasks)
// using the natural-language `--when` grammar (e.g. "every 30 minutes",
// "45 minutes after competition start"). Same target + command flags as run.
func runSchedule(args []string) error {
	fs := flag.NewFlagSet("schedule", flag.ExitOnError)
	build := fs.String("build", "", "build id to schedule against (required)")
	when := fs.String("when", "", "when to run, natural language (required) -- e.g. \"every 30 minutes\"")
	command := fs.String("command", "", "agent command kind (default: execute)")
	payload := fs.String("payload", "", "raw JSON payload, for a non-execute command")
	timeout := fs.Int("timeout", 300, "execute timeout, in seconds")
	resolveTarget := targetFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" || *when == "" {
		return fmt.Errorf("usage: laforge schedule --build <id> --when \"<phrase>\" [--team N | --id OBJID | --tag k=v] -- <command>")
	}
	cmd, pl, err := buildCommandPayload(*command, *payload, fs.Args(), *timeout)
	if err != nil {
		return err
	}
	req := map[string]any{"target": resolveTarget(), "command": cmd, "payload": pl, "when": *when}

	var res map[string]any
	if err := newAPIClient().do("POST", "/builds/"+url.PathEscape(*build)+"/scheduled-tasks", req, &res); err != nil {
		return err
	}
	fmt.Printf("scheduled %q — when: %v", cmd, res["when_expr"])
	if nf, ok := res["next_fire_at"]; ok && nf != nil && nf != "" {
		fmt.Printf(", next fire: %v", nf)
	}
	fmt.Println()
	return nil
}

// runRestartSteps runs a host's build again from an authored step (POST
// /builds/{id}/restart-steps; docs/plans/restart-from-step.md). --from is the
// step's index or its label; a label is resolved through a dry run from step
// 0, since the server lists steps by index. The dry run is always shown, and
// the restart asks before acting unless --yes.
func runRestartSteps(args []string) error {
	fs := flag.NewFlagSet("restart-steps", flag.ExitOnError)
	build := fs.String("build", "", "build id (required)")
	from := fs.String("from", "", "the authored step to restart from: its index (0 is the first) or its label (required)")
	dryRun := fs.Bool("dry-run", false, "show what would run again and change nothing")
	force := fs.Bool("force", false, "restart a host even while a step is running on it")
	noDeps := fs.Bool("no-dependents", false, "don't resume hosts that were failed for depending on these")
	yes := fs.Bool("yes", false, "don't ask before restarting")
	resolveTarget := targetFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *build == "" || *from == "" {
		return fmt.Errorf("usage: laforge restart-steps --build <id> --from <step> [--team N | --id OBJID | --tag k=v | --search text] [--dry-run] [--force] [--no-dependents] [--yes]")
	}
	target := resolveTarget()
	c := newAPIClient()
	path := "/builds/" + url.PathEscape(*build) + "/restart-steps"

	type host struct {
		ID             string `json:"id"`
		AsName         string `json:"as_name"`
		TeamNumber     int    `json:"team_number"`
		Status         string `json:"status"`
		Eligible       bool   `json:"eligible"`
		Running        bool   `json:"running"`
		ContentChanged bool   `json:"content_changed"`
		Steps          []struct {
			Group    int    `json:"group"`
			Label    string `json:"label"`
			Commands int    `json:"commands"`
		} `json:"steps"`
		Dependents []struct {
			AsName string `json:"as_name"`
		} `json:"dependents"`
		Restarted bool   `json:"restarted"`
		Skipped   string `json:"skipped"`
	}
	type result struct {
		Hosts     []host `json:"hosts"`
		Restarted int    `json:"restarted"`
	}

	group, err := strconv.Atoi(*from)
	if err != nil {
		// A label: find it among the steps the dry run lists from step 0.
		var probe result
		if err := c.do("POST", path, map[string]any{"target": target, "from_group": 0, "dry_run": true}, &probe); err != nil {
			return err
		}
		group = -1
		for _, h := range probe.Hosts {
			for _, st := range h.Steps {
				if st.Label == *from {
					group = st.Group
				}
			}
		}
		if group < 0 {
			return fmt.Errorf("no step labelled %q on these hosts (use its index, or a label as the steps panel shows it)", *from)
		}
	}

	var preview result
	if err := c.do("POST", path, map[string]any{"target": target, "from_group": group, "resume_dependents": !*noDeps, "force": *force, "dry_run": true}, &preview); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	eligible := 0
	for _, h := range preview.Hosts {
		state := h.Status
		switch {
		case h.Skipped != "":
			state += " -- " + h.Skipped
		case h.Running:
			state += " -- a step is running (forced)"
		default:
			eligible++
		}
		if h.ContentChanged {
			state += " (content changed since it was built)"
		}
		fmt.Fprintf(w, "  team %d\t%s\t%s\n", h.TeamNumber, h.AsName, state)
	}
	w.Flush()
	if len(preview.Hosts) > 0 && len(preview.Hosts[0].Steps) > 0 {
		fmt.Println("steps that run again, in full:")
		for _, st := range preview.Hosts[0].Steps {
			label := st.Label
			if label == "" {
				label = "(unnamed)"
			}
			fmt.Printf("  %d\t%s\t%d command(s)\n", st.Group, label, st.Commands)
		}
	}
	deps := 0
	for _, h := range preview.Hosts {
		deps += len(h.Dependents)
	}
	if deps > 0 {
		fmt.Printf("%d host(s) that depend on these will resume once they finish\n", deps)
	}
	if *dryRun {
		return nil
	}
	if eligible == 0 {
		return fmt.Errorf("nothing to restart")
	}
	fmt.Println("steps should be safe to run twice: they run again on the boxes as they are.")
	if !*yes {
		fmt.Printf("restart %d host(s) from step %d? [y/N] ", eligible, group)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" && answer != "yes" {
			return fmt.Errorf("cancelled")
		}
	}
	var res result
	if err := c.do("POST", path, map[string]any{"target": target, "from_group": group, "resume_dependents": !*noDeps, "force": *force}, &res); err != nil {
		return err
	}
	for _, h := range res.Hosts {
		if h.Restarted {
			fmt.Printf("  team %d  %s: restarted\n", h.TeamNumber, h.AsName)
		} else if h.Skipped != "" {
			fmt.Printf("  team %d  %s: skipped -- %s\n", h.TeamNumber, h.AsName, h.Skipped)
		}
	}
	fmt.Printf("restarted %d host(s) from step %d\n", res.Restarted, group)
	return nil
}
