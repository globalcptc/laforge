package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
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
