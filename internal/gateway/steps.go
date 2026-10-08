// Package gateway is the agent-gateway service: "its own container, its
// own domain, scaled on its own. Speaks only the agent protocol." This
// file is the one piece of real logic before the protocol handling
// itself: expanding a host or container's resolved Host.Steps/
// Container.Steps into the agent's actual command set. "Step kinds are
// not 1:1 with commands... a script: step expands to download then
// execute... The server does the expanding; the agent only ever sees
// commands from this list."
package gateway

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// RegistryAuth is a stored login for one image registry.
type RegistryAuth struct {
	Username string
	Secret   string
}

// Option adjusts how steps expand. The zero set is right for anything that
// only needs the shape of the commands (a preview, a check).
type Option func(*expandOptions)

type expandOptions struct {
	registryAuth func(host string) (RegistryAuth, bool)
	// containerLogs is the environment's container_logs, applied to a compose
	// project as the Docker daemon default log driver on its host so every
	// service inherits it (see expandCompose). Nil disables forwarding.
	containerLogs *loader.ContainerLogs
}

// WithRegistryAuth supplies stored registry credentials, looked up by host
// ("docker.io" for Docker Hub). A compose container uses them to `docker login`
// before pulling; without this option it pulls unauthenticated. Only the real
// materializer passes it, so credentials never reach a preview.
func WithRegistryAuth(lookup func(host string) (RegistryAuth, bool)) Option {
	return func(o *expandOptions) { o.registryAuth = lookup }
}

// PlannedCommand is one command ExpandSteps produced, ready to become an
// agent_task row (command + JSON payload) at the next sequential
// step_index. Group/GroupLabel record which authored step it came from, so
// a UI can show the expanded sub-commands (write_file + execute + validate)
// grouped under one named heading -- ExpandSteps sets them; ExpandOneStep
// (used on its own by the schedule materializer, where grouping is moot)
// leaves them zero.
type PlannedCommand struct {
	Command    string
	Payload    interface{}
	Group      int
	GroupLabel string
	// IgnoreErrors marks a command whose terminal failure must be tolerated --
	// the step is allowed to fail without blocking later steps or failing the
	// build (it becomes an `ignored` agent_task, not `failed`). Set from a
	// `script:`'s own `ignore_errors:` field, carried onto the execute command
	// that actually runs it.
	IgnoreErrors bool
}

// languageInterpreter maps a script's language to how the agent should
// run the file ExpandSteps writes out for it. Linux (bash/sh) and Windows
// (powershell/batch) are both real now -- the agent has executors for
// both (see the Rust agent's commands.rs).
func languageInterpreter(lang string) (command string, args []string) {
	switch lang {
	case "bash":
		return "/bin/bash", nil
	case "powershell":
		return "powershell.exe", []string{"-File"}
	case "batch":
		return "cmd.exe", []string{"/c"}
	default:
		return "/bin/sh", nil
	}
}

// scriptFile is where ExpandSteps writes a script for the agent to run,
// and the mode to set on it. The path has to suit the guest OS: a Windows
// interpreter can't open a /tmp path, and PowerShell's -File requires a
// .ps1 extension. Windows files get no POSIX mode (empty -> the agent
// skips chmod/icacls on them).
func scriptFile(i int, name, lang string) (path, mode string) {
	switch lang {
	case "powershell":
		return fmt.Sprintf(`C:\Windows\Temp\laforge-step-%d-%s.ps1`, i, name), ""
	case "batch":
		return fmt.Sprintf(`C:\Windows\Temp\laforge-step-%d-%s.bat`, i, name), ""
	default:
		return fmt.Sprintf("/tmp/laforge-step-%d-%s", i, name), "0700"
	}
}

// ExpandSteps resolves as's context in envName/team and turns its step
// list into the ordered, agent-ready command sequence. repoRoot must be
// the same checkout the content was loaded from (RenderScript reads real
// script files off disk). notes carries anything ExpandSteps couldn't
// turn into a fully-working command -- an `upload` step, which still has
// no file store to receive it -- flagged rather than silently dropped,
// matching this whole project's "flags anything it cannot translate
// rather than guessing." (download and extract are real now.)
// `schedule:` is no longer a step kind at all (see internal/schedule and
// the top-level `schedule:` field on Host/Container), so it no longer
// appears here -- a schedule entry's own translation goes through
// ExpandOneStep directly (internal/orchestrator's schedule materializer),
// since it dispatches independently, not as part of this ordered list.
func ExpandSteps(repoRoot string, c *loader.Content, envName, asName string, team int, opts ...Option) ([]PlannedCommand, []string, error) {
	ctx, err := render.Resolve(c, envName, asName, team)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving %s: %w", asName, err)
	}

	var out []PlannedCommand
	var notes []string
	// A compose container's project comes up first, as its own group, so the
	// container's authored steps (and their validators) run against a live stack.
	first := 0
	if ctx.Compose != "" {
		var o expandOptions
		for _, opt := range opts {
			opt(&o)
		}
		// A compose project ships its logs via the Docker daemon default on its
		// host (expandCompose), set from the environment's container_logs.
		for i := range c.Environments {
			if c.Environments[i].Name == envName {
				o.containerLogs = c.Environments[i].ContainerLogs
				break
			}
		}
		cmds, err := expandCompose(repoRoot, ctx.Compose, ctx.ObjectName, o)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", asName, err)
		}
		for k := range cmds {
			cmds[k].GroupLabel = ComposeGroupLabel
		}
		out = append(out, cmds...)
		first = 1
	}
	for i, step := range ctx.Steps {
		cmds, ns, err := ExpandOneStep(repoRoot, c, ctx, i, step, opts...)
		if err != nil {
			return nil, nil, err
		}
		// Stamp every command this authored step expanded into with the same
		// group, so its sub-commands stay visually together under one heading.
		label := StepGroupLabel(step)
		for k := range cmds {
			cmds[k].Group = first + i
			cmds[k].GroupLabel = label
		}
		out = append(out, cmds...)
		notes = append(notes, ns...)
	}
	return out, notes, nil
}

// ComposeGroupLabel heads the commands that start a compose container's project.
const ComposeGroupLabel = "Start Compose project"

// StepGroupLabel is the human heading for one authored step -- the name a UI
// shows above the sub-commands it expands into. Scripts carry their own name;
// the file/user/service actions add their primary argument when it's easy to
// read, so "Write file: /etc/motd" beats a bare "Write file".
func StepGroupLabel(step loader.Step) string {
	switch step.ActionKey() {
	case "script":
		if n, _ := step["script"].(string); n != "" {
			return "Script: " + n
		}
		return "Script"
	case "run":
		return "Run command"
	case "write_file":
		return withArg("Write file", asMap(step["write_file"]), "path")
	case "append_file":
		return withArg("Append file", asMap(step["append_file"]), "path")
	case "delete":
		return withArg("Delete", asMap(step["delete"]), "path")
	case "change_perms":
		return withArg("Change permissions", asMap(step["change_perms"]), "path")
	case "create_user":
		return withArg("Create user", asMap(step["create_user"]), "name")
	case "set_password":
		return withArg("Set password", asMap(step["set_password"]), "name")
	case "add_to_group":
		return withArg("Add to group", asMap(step["add_to_group"]), "user")
	case "service":
		return withArg("Service", asMap(step["service"]), "name")
	case "reboot":
		return "Reboot"
	case "download":
		return "Download"
	case "upload":
		return "Upload"
	case "extract":
		return "Extract"
	default:
		return step.ActionKey()
	}
}

// withArg appends a step's primary argument to its base label when present.
func withArg(base string, m map[string]interface{}, key string) string {
	if v := strField(m, key); v != "" {
		return base + ": " + v
	}
	return base
}

// ExpandOneStep translates a single step -- or a schedule entry, the
// identical shape (one action key plus an optional validate) -- into its
// real agent command(s): the action itself, plus a trailing validate
// command when present. i only numbers a script step's generated file
// path and any note messages; unique within whatever slice/dispatch this
// call is part of is enough, it's never persisted as an index of record.
//
// The one real piece of logic in this package, extracted here (out of
// ExpandSteps' own loop) so internal/orchestrator's schedule-entry
// materializer calls the exact same translation an authored `steps:`
// entry gets -- script rendering, templated password fields, the same
// per-action payload shapes -- rather than a second, partial
// reimplementation that would silently support fewer action kinds than
// steps: does.
func ExpandOneStep(repoRoot string, c *loader.Content, ctx *render.Context, i int, step loader.Step, opts ...Option) ([]PlannedCommand, []string, error) {
	var o expandOptions
	for _, opt := range opts {
		opt(&o)
	}
	var out []PlannedCommand
	var notes []string
	// ignoreErrors is set by an action whose failure the author chose to
	// tolerate (a `script:` with `ignore_errors:`). It rides on that action's
	// own command AND on the step's trailing validate, since "a failed check
	// fails its step, unless the step sets ignore_errors" (common.schema.json).
	var ignoreErrors bool

	switch step.ActionKey() {
	case "script":
		name, _ := step["script"].(string)
		script := findScript(c, name)
		if script == nil {
			return nil, []string{fmt.Sprintf("step %d: script %q not found, skipped", i, name)}, nil
		}
		rendered, err := render.RenderScript(repoRoot, script, ctx, c)
		if err != nil {
			return nil, nil, fmt.Errorf("step %d (script %s): %w", i, name, err)
		}
		path, mode := scriptFile(i, script.Name, script.Language)
		out = append(out, PlannedCommand{Command: agentproto.CmdWriteFile, Payload: agentproto.WriteFilePayload{
			Path: path, Content: rendered, Mode: mode,
		}})
		ignoreErrors = script.IgnoreErrors
		interp, prefixArgs := languageInterpreter(script.Language)
		// interpreter flags, then the script path, then the script's own
		// `args:` -- e.g. `bash /path/to/script.sh --seed 3` for
		// `args: ["--seed", "3"]`.
		execArgs := append(append([]string{}, prefixArgs...), path)
		execArgs = append(execArgs, script.Args...)
		out = append(out, PlannedCommand{Command: agentproto.CmdExecute, IgnoreErrors: ignoreErrors, Payload: agentproto.ExecutePayload{
			Command: interp, Args: execArgs, TimeoutSec: script.Timeout,
		}})
		// A script's OWN `validate:` block runs after its execution, on every
		// host that runs it -- on top of (before) any validate the step itself
		// authors, which the generic trailing-validate handler below appends.
		if len(script.Validate) > 0 {
			raw := make([]interface{}, len(script.Validate))
			for j := range script.Validate {
				raw[j] = script.Validate[j]
			}
			checks, err := renderValidateChecks(raw)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (script %s) validate: %w", i, name, err)
			}
			out = append(out, PlannedCommand{Command: agentproto.CmdValidate, IgnoreErrors: ignoreErrors, Payload: agentproto.ValidatePayload{Checks: checks}})
		}

	case "run":
		raw, _ := step["run"].(string)
		rendered, err := render.RenderString(fmt.Sprintf("step[%d].run", i), raw, ctx, c)
		if err != nil {
			return nil, nil, fmt.Errorf("step %d (run): %w", i, err)
		}
		out = append(out, PlannedCommand{Command: agentproto.CmdExecute, Payload: agentproto.ExecutePayload{
			Command: "/bin/sh", Args: []string{"-c", rendered},
		}})

	case "write_file":
		p, err := renderedField(ctx, c, step, "write_file", "content")
		if err != nil {
			return nil, nil, err
		}
		m := asMap(step["write_file"])
		out = append(out, PlannedCommand{Command: agentproto.CmdWriteFile, Payload: agentproto.WriteFilePayload{
			Path: strField(m, "path"), Content: p, Mode: strField(m, "mode"),
		}})

	case "append_file":
		p, err := renderedField(ctx, c, step, "append_file", "content")
		if err != nil {
			return nil, nil, err
		}
		m := asMap(step["append_file"])
		out = append(out, PlannedCommand{Command: agentproto.CmdAppendFile, Payload: agentproto.AppendFilePayload{
			Path: strField(m, "path"), Content: p,
		}})

	case "delete":
		m := asMap(step["delete"])
		out = append(out, PlannedCommand{Command: agentproto.CmdDelete, Payload: agentproto.DeletePayload{Path: strField(m, "path")}})

	case "change_perms":
		m := asMap(step["change_perms"])
		out = append(out, PlannedCommand{Command: agentproto.CmdChangePerms, Payload: agentproto.ChangePermsPayload{
			Path: strField(m, "path"), Mode: strField(m, "mode"), Owner: strField(m, "owner"), Group: strField(m, "group"),
		}})

	case "create_user":
		pw, err := renderedField(ctx, c, step, "create_user", "password")
		if err != nil {
			return nil, nil, err
		}
		m := asMap(step["create_user"])
		out = append(out, PlannedCommand{Command: agentproto.CmdCreateUser, Payload: agentproto.CreateUserPayload{
			Username: strField(m, "name"), Password: pw, Groups: strSliceField(m, "groups"),
		}})

	case "set_password":
		pw, err := renderedField(ctx, c, step, "set_password", "password")
		if err != nil {
			return nil, nil, err
		}
		m := asMap(step["set_password"])
		out = append(out, PlannedCommand{Command: agentproto.CmdSetPassword, Payload: agentproto.SetPasswordPayload{
			Username: strField(m, "name"), Password: pw,
		}})

	case "add_to_group":
		m := asMap(step["add_to_group"])
		out = append(out, PlannedCommand{Command: agentproto.CmdAddToGroup, Payload: agentproto.AddToGroupPayload{
			Username: strField(m, "user"), Group: strField(m, "group"),
		}})

	case "service":
		m := asMap(step["service"])
		out = append(out, PlannedCommand{Command: agentproto.CmdService, Payload: agentproto.ServicePayload{
			Name: strField(m, "name"), Action: strField(m, "action"),
		}})

	case "reboot":
		delaySec, err := rebootDelaySec(step)
		if err != nil {
			return nil, nil, fmt.Errorf("step %d (reboot): %w", i, err)
		}
		out = append(out, PlannedCommand{Command: agentproto.CmdReboot, Payload: agentproto.RebootPayload{DelaySec: delaySec}})

	case "download":
		// A real HTTP(S) GET the agent performs directly (host -> the URL's
		// host, e.g. files.cp.tc), NOT proxied through the gateway -- so it
		// adds no gateway bandwidth. No warning note: it works.
		m := asMap(step["download"])
		out = append(out, PlannedCommand{Command: agentproto.CmdDownload, Payload: agentproto.DownloadPayload{
			From: strField(m, "from"), To: strField(m, "to"),
		}})

	case "upload":
		m := asMap(step["upload"])
		out = append(out, PlannedCommand{Command: agentproto.CmdUpload, Payload: agentproto.UploadPayload{From: strField(m, "from")}})
		notes = append(notes, fmt.Sprintf("step %d: upload has no file store yet -- the agent will report this as not implemented", i))

	case "extract":
		m := asMap(step["extract"])
		out = append(out, PlannedCommand{Command: agentproto.CmdExtract, Payload: agentproto.ExtractPayload{
			Src: strField(m, "src"), Dest: strField(m, "dest"),
		}})

	default:
		return nil, []string{fmt.Sprintf("step %d: unrecognized action %q, skipped", i, step.ActionKey())}, nil
	}

	if rawChecks, ok := step["validate"].([]interface{}); ok && len(rawChecks) > 0 {
		checks, err := renderValidateChecks(rawChecks)
		if err != nil {
			return nil, nil, fmt.Errorf("step %d validate: %w", i, err)
		}
		out = append(out, PlannedCommand{Command: agentproto.CmdValidate, IgnoreErrors: ignoreErrors, Payload: agentproto.ValidatePayload{Checks: checks}})
	}

	return out, notes, nil
}

func findScript(c *loader.Content, name string) *loader.Script {
	for i := range c.Scripts {
		if c.Scripts[i].Name == name {
			return &c.Scripts[i]
		}
	}
	return nil
}

// asMap normalizes a step action's value to a plain map, regardless of
// which of the two equivalent-but-distinct Go types it actually arrived
// as. yaml.v3, decoding a Host/Container's Steps ([]loader.Step) field,
// decodes each step as loader.Step -- but a *nested* mapping inside one
// (write_file's own {path, content, mode} object, say) also comes back
// typed as loader.Step, not map[string]interface{}, even though the two
// are structurally identical (`type Step map[string]interface{}`). A
// plain `.(map[string]interface{})` type assertion checks the exact
// dynamic type, not structural equivalence, so it silently fails on the
// nested case -- found by running ExpandSteps against a real host
// (webserver.yaml's download step) and seeing empty fields, not by
// reasoning about yaml.v3 in the abstract. render.walkStrings hits the
// same thing and handles it the same way, one call site earlier.
func asMap(v interface{}) map[string]interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return x
	case loader.Step:
		return map[string]interface{}(x)
	default:
		return nil
	}
}

func strField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func strSliceField(m map[string]interface{}, key string) []string {
	raw, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// renderedField renders one templated string field of a step's action
// object (e.g. write_file's `content`, create_user's `password`) --
// mirrors what internal/render.RenderStepFields does generically for
// `laforge check`, but returns the single rendered value ExpandSteps
// actually needs to build a command payload rather than only collecting
// errors.
func renderedField(ctx *render.Context, c *loader.Content, step loader.Step, action, field string) (string, error) {
	m := asMap(step[action])
	raw, _ := m[field].(string)
	if raw == "" {
		return "", nil
	}
	return render.RenderString(fmt.Sprintf("%s.%s", action, field), raw, ctx, c)
}

func renderValidateChecks(raw []interface{}) ([]agentproto.ValidateCheck, error) {
	var out []agentproto.ValidateCheck
	for _, r := range raw {
		m := asMap(r)
		if m == nil {
			b, _ := json.Marshal(r)
			return nil, fmt.Errorf("malformed validate entry: %s", b)
		}
		// An optional `delay:` sibling waits before the check runs. Pull it out
		// (without mutating the shared loaded map -- ExpandSteps runs on it more
		// than once), then require exactly one remaining key: the check itself.
		var delayMs int64
		var kind string
		var args interface{}
		checks := 0
		for k, v := range m {
			if k == "delay" {
				ms, err := parseDelayMs(v)
				if err != nil {
					return nil, fmt.Errorf("validate entry delay: %w", err)
				}
				delayMs = ms
				continue
			}
			kind, args = k, v
			checks++
		}
		if checks != 1 {
			b, _ := json.Marshal(r)
			return nil, fmt.Errorf("validate entry must have exactly one check (plus an optional delay): %s", b)
		}
		argMap := asMap(args)
		if argMap == nil {
			// A bare-value check, e.g. `- user_exists: dbadmin` -- normalize to
			// {"value": "dbadmin"} so the agent always sees a map.
			argMap = map[string]interface{}{"value": args}
		}
		out = append(out, agentproto.ValidateCheck{Kind: kind, Args: argMap, DelayMs: delayMs})
	}
	return out, nil
}

// rebootDelaySec reads a reboot step's optional `delay:` -- a duration string
// ("30s"), the same format as a validator's delay -- and returns whole seconds
// for the agent's shutdown scheduler (which rounds to whole minutes anyway). No
// delay, or a bare `reboot: {}`, means reboot now.
func rebootDelaySec(step loader.Step) (int, error) {
	m := asMap(step["reboot"])
	if m == nil {
		return 0, nil
	}
	d, ok := m["delay"]
	if !ok {
		return 0, nil
	}
	ms, err := parseDelayMs(d)
	if err != nil {
		return 0, err
	}
	return int(ms / 1000), nil
}

// parseDelayMs turns a validator's `delay:` value into milliseconds. Like
// reboot's `delay`, it is a duration STRING ("10s", "500ms", "1m") -- the schema
// allows only a string, and this agrees. Negative is rejected.
func parseDelayMs(v interface{}) (int64, error) {
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("must be a duration string like \"10s\", got %T", v)
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%q is not a duration (try \"10s\"): %w", s, err)
	}
	if dur < 0 {
		return 0, fmt.Errorf("%q is negative", s)
	}
	return dur.Milliseconds(), nil
}
