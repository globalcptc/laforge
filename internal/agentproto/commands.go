package agentproto

// The full agent command set (the "Agent commands"
// table), minus `registry` (Windows-only; the current scope is Linux
// hosts).
// "Step kinds are not 1:1 with commands... the server does the
// expanding; the agent only ever sees commands from this list" --
// internal/gateway/steps.go is that expansion; these are the payload
// shapes it produces and agent/src/commands.rs consumes, independently
// implemented on each side against this shared shape.
const (
	CmdExecute     = "execute"
	CmdWriteFile   = "write_file"
	CmdAppendFile  = "append_file"
	CmdDelete      = "delete"
	CmdChangePerms = "change_perms"
	CmdCreateUser  = "create_user"
	CmdSetPassword = "set_password"
	CmdAddToGroup  = "add_to_group"
	CmdService     = "service"
	CmdReboot      = "reboot"
	CmdValidate    = "validate"
	CmdDownload    = "download"
	CmdUpload      = "upload"
	CmdExtract     = "extract"
	// CmdDisconnectSessions forcibly ends live interactive remote-access
	// sessions on the host (RDP on Windows, SSH/VNC on Linux) WITHOUT touching
	// the agent's own outbound connection to the gateway. The access reconciler
	// fires this at a team on close: the network firewall only blocks NEW
	// connections, but on a bridge the published port is a nat-proxy whose
	// already-established conntrack (and the guest's egress replies riding
	// Incus's global stateful bypass) keep existing sessions alive. The ACL
	// can't scope statefulness or tear down a live flow, so the agent -- reached
	// over the one egress path a closed team keeps -- is the only lever that
	// reaches inside the guest to drop them.
	CmdDisconnectSessions = "disconnect_sessions"
)

type ExecutePayload struct {
	Command    string   `json:"command"`
	Args       []string `json:"args,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	TimeoutSec int      `json:"timeout_sec,omitempty"`
	// Script, when set, is the rendered script file this command runs. The
	// agent writes it just before each attempt and removes it as soon as the
	// process exits -- succeeded, failed or timed out -- because a rendered
	// script carries the environment's passwords and must not stay on a box
	// competitors control. (A separate write_file before and a delete after
	// still bracket the command, for agents that predate this field.)
	Script *WriteFilePayload `json:"script,omitempty"`
}

type WriteFilePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"`
}

type AppendFilePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type DeletePayload struct {
	Path string `json:"path"`
}

type ChangePermsPayload struct {
	Path  string `json:"path"`
	Mode  string `json:"mode,omitempty"`
	Owner string `json:"owner,omitempty"`
	Group string `json:"group,omitempty"`
}

type CreateUserPayload struct {
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"`
	Groups   []string `json:"groups,omitempty"`
}

type SetPasswordPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AddToGroupPayload struct {
	Username string `json:"username"`
	Group    string `json:"group"`
}

type ServicePayload struct {
	Name   string `json:"name"`
	Action string `json:"action"` // start | stop | restart | enable | disable
}

type RebootPayload struct {
	DelaySec int `json:"delay_sec,omitempty"`
}

// DisconnectSessionsPayload carries the optional set of TCP ports whose
// established inbound connections should also be reset, on top of the agent's
// built-in default (Windows: log off every active rdp-tcp session; Linux: reset
// established sockets on the usual remote-access ports). The access reconciler
// passes the team's actual published ports so the reset is precise; an empty
// list just uses the defaults. The agent never resets its own outbound gateway
// connection -- it only acts on inbound/listening sides.
type DisconnectSessionsPayload struct {
	Ports []int `json:"ports,omitempty"`
}

// ValidateCheck is one check from a step's or script's `validate:` block.
// Args is intentionally a free-form map -- "each is a named check with
// typed arguments," and the argument shape genuinely differs per kind
// (file_exists takes a path; user_in_group takes a username and a
// group), so a single Go/Rust struct can't type them all without either a
// tagged union per kind or this. The agent validates the shape it expects
// for whichever Kind it receives.
type ValidateCheck struct {
	Kind string                 `json:"kind"`
	Args map[string]interface{} `json:"args"`
	// DelayMs is an optional wait (milliseconds) before this check runs, from a
	// validator's `delay:` sub-item ("10s"), so a service/port has time to
	// settle. 0/omitted means run immediately. The agent sleeps on its task
	// worker thread, so this never blocks heartbeats.
	DelayMs int64 `json:"delay_ms,omitempty"`
}

type ValidatePayload struct {
	Checks []ValidateCheck `json:"checks"`
}

// DownloadPayload, UploadPayload, and ExtractPayload are part of the
// protocol's command set but not implemented by the agent yet
// -- they depend on the artifact/object-storage service the spec
// describes ("agents fetch artifacts... using short-lived presigned
// URLs"), which doesn't exist yet. The agent recognizes these commands
// and reports a clear "not implemented" failure rather than silently
// doing nothing or crashing -- see agent/src/commands.rs.
type DownloadPayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type UploadPayload struct {
	From string `json:"from"`
}

type ExtractPayload struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}
