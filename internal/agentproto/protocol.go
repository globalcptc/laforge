// Package agentproto is the wire protocol between an agent and the
// agent-gateway: "mTLS over TCP, with minimal framing inside... a small
// length-prefixed binary framing carrying three verbs: heartbeat,
// get-task, report-status." This package is the framing and the Go side
// of the verbs; agent/src/protocol.rs is the Rust agent's independent
// implementation of the exact same wire format -- independent on
// purpose, so a fuzzed or malformed frame from either side is a protocol
// question, never a shared-code assumption.
//
// Frame shape, every message, both directions:
//
//	[4 bytes: payload length, big-endian u32][1 byte: message type][payload]
//
// The payload itself is JSON. That's not the "custom protocol" the plan
// specifically warns against -- "an earlier draft of this plan proposed
// [a hand-rolled handshake], and it was the weakest idea in it." The
// cryptography here is entirely TLS (rustls on the agent side, crypto/tls
// on the gateway side), well-scrutinized and unmodified; this package
// only ever decides "how many bytes, what verb, then hand the payload to
// encoding/json." An observer of the raw TCP stream sees an ordinary TLS
// connection and learns nothing about what's inside it.
package agentproto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

type MessageType byte

const (
	HeartbeatRequest     MessageType = 0x01
	HeartbeatResponse    MessageType = 0x02
	GetTaskRequest       MessageType = 0x03
	GetTaskResponse      MessageType = 0x04
	ReportStatusRequest  MessageType = 0x05
	ReportStatusResponse MessageType = 0x06
	// LogBatchRequest carries a batch of an object's captured console lines
	// (a container's supervised app stdout/stderr); LogBatchResponse is the
	// gateway's ack. Fire-and-ack like every other verb: the agent sends what
	// it has buffered, the gateway enriches and ships it to the log sink.
	LogBatchRequest  MessageType = 0x07
	LogBatchResponse MessageType = 0x08

	// --- Interactive shell relay (0x09-0x0C) ---
	//
	// These four are NOT the request/response, poll-driven shape the verbs
	// above use. They are a bidirectional byte stream for an interactive PTY,
	// spoken on a connection DEDICATED to one shell session -- the agent opens
	// a second mTLS connection for it, and the api opens one to the gateway's
	// internal relay listener. The gateway is a dumb relay keyed by session id:
	// it consumes ShellAttach (to pair the two halves) and then forwards
	// ShellData/ShellResize/ShellClose between them unchanged. There is no ack;
	// closing the connection (or a ShellClose) ends the session.

	// ShellAttach is the first frame on a relay connection, identifying the
	// session and which half is connecting (ShellAttachPayload).
	ShellAttach MessageType = 0x09
	// ShellData carries RAW PTY bytes as its payload -- no JSON, no base64; the
	// length-prefixed framing already delimits it. client->agent is stdin,
	// agent->client is stdout/stderr.
	ShellData MessageType = 0x0A
	// ShellResize carries a new terminal size (ShellResizePayload), client->agent.
	ShellResize MessageType = 0x0B
	// ShellClose ends the session from either direction (ShellClosePayload).
	ShellClose MessageType = 0x0C
)

func (m MessageType) String() string {
	switch m {
	case HeartbeatRequest:
		return "HeartbeatRequest"
	case HeartbeatResponse:
		return "HeartbeatResponse"
	case GetTaskRequest:
		return "GetTaskRequest"
	case GetTaskResponse:
		return "GetTaskResponse"
	case ReportStatusRequest:
		return "ReportStatusRequest"
	case ReportStatusResponse:
		return "ReportStatusResponse"
	case LogBatchRequest:
		return "LogBatchRequest"
	case LogBatchResponse:
		return "LogBatchResponse"
	case ShellAttach:
		return "ShellAttach"
	case ShellData:
		return "ShellData"
	case ShellResize:
		return "ShellResize"
	case ShellClose:
		return "ShellClose"
	default:
		return fmt.Sprintf("Unknown(0x%02x)", byte(m))
	}
}

// MaxFrameSize caps how large a single frame's payload is allowed to be.
// Without a cap, a length prefix is an invitation to allocate however much
// memory an attacker names in four bytes, before a single byte of the
// actual payload has even been read -- the textbook framing-protocol
// mistake. 10MiB comfortably covers the largest real payload this
// protocol carries (a rendered script inside a task, see
// internal/gateway/steps.go) with a lot of headroom, not a tight fit.
const MaxFrameSize = 10 * 1024 * 1024

// WriteFrame writes one frame: length prefix, message type, payload.
func WriteFrame(w io.Writer, msgType MessageType, payload []byte) error {
	if len(payload) > MaxFrameSize-1 {
		return fmt.Errorf("payload of %d bytes exceeds MaxFrameSize", len(payload))
	}
	header := make([]byte, 5)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(payload)+1))
	header[4] = byte(msgType)
	if _, err := w.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// ReadFrame reads one frame. Returns an error for anything malformed --
// a length that's absurd, a truncated stream, zero bytes after the length
// prefix (no room for even the message-type byte) -- rather than ever
// panicking or reading past what the prefix promised. This is the
// function FuzzReadFrame exercises directly.
func ReadFrame(r io.Reader) (MessageType, []byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length == 0 {
		return 0, nil, fmt.Errorf("frame length is 0, need at least 1 byte for the message type")
	}
	if length > MaxFrameSize {
		return 0, nil, fmt.Errorf("frame length %d exceeds MaxFrameSize %d", length, MaxFrameSize)
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, err
	}
	return MessageType(body[0]), body[1:], nil
}

// --- JSON payload shapes, the shared vocabulary between the Go gateway
// and the Rust agent (agent/src/protocol.rs mirrors these field-for-field,
// independently). ---

// HeartbeatRequestPayload carries the basic host metrics the agent samples on
// each check-in. cpu/mem/disk are percentages (0-100); net_rx/tx are bytes per
// second since the previous heartbeat. Every field is a pointer so "not
// collected" (an older agent, a metric the agent couldn't read, or the first
// heartbeat of a session before a network rate can be computed) is distinct
// from a real zero. The gateway stores them on the agent_heartbeat row. Mirrors
// the Rust agent's HeartbeatRequestPayload (agent/src/protocol.rs).
type HeartbeatRequestPayload struct {
	CPUPct   *float64 `json:"cpu_pct,omitempty"`
	MemPct   *float64 `json:"mem_pct,omitempty"`
	DiskPct  *float64 `json:"disk_pct,omitempty"`
	NetRxBps *float64 `json:"net_rx_bps,omitempty"`
	NetTxBps *float64 `json:"net_tx_bps,omitempty"`
}

type HeartbeatResponsePayload struct {
	NextPollMS int `json:"next_poll_ms"`
	// PendingSessions are interactive-shell session ids a client has requested
	// on this object but that have no agent connection yet. The agent opens a
	// second mTLS connection per id and ShellAttaches as the "agent" half. Empty
	// (omitted) on the common case; when non-empty the gateway also shortens
	// NextPollMS so the shell opens sub-second. Mirrors the Rust agent's
	// HeartbeatResponsePayload.
	PendingSessions []string `json:"pending_sessions,omitempty"`
}

// ShellRole is which half of a relay connection this is: the agent that holds
// the PTY, or the client (api, on behalf of a UI/CLI user) that holds the user.
const (
	ShellRoleAgent  = "agent"
	ShellRoleClient = "client"
)

// ShellDir is set on the AGENT half only, and splits it into two strictly
// one-directional connections. rustls (the agent's TLS) is a single state
// machine that cannot be read and written from two threads at once, and a
// single-connection read/write interleave via a socket read timeout behaved
// differently on Windows (keystrokes never arrived). So the agent opens TWO
// mTLS connections per session -- one it only writes PTY output on (Out) and
// one it only reads client input on (In) -- each owned by one thread doing
// pure blocking reads XOR writes, identical on Linux and Windows. The gateway
// pairs both against the single (duplex) client connection. The client half
// leaves ShellDir empty; Go's crypto/tls allows one concurrent reader and one
// writer, so the client side stays a single connection.
const (
	ShellDirOut = "out" // agent -> client: PTY stdout/stderr
	ShellDirIn  = "in"  // client -> agent: stdin / resize / close
)

// ShellAttachPayload is the first frame on a shell-relay connection: which
// session, which half, and the client's initial terminal size (so the PTY is
// opened at the right dimensions). Cols/Rows are only meaningful from the
// client half.
type ShellAttachPayload struct {
	SessionID string `json:"session_id"`
	Role      string `json:"role"` // ShellRoleAgent | ShellRoleClient
	// Dir is set only on the agent half: ShellDirOut or ShellDirIn (see
	// ShellDir). The agent opens one connection per direction; the gateway pairs
	// both with the single client connection. Empty on the client half.
	Dir string `json:"dir,omitempty"`
	// ObjectID is set only by the client (api) half: the deployed_object whose
	// agent is allowed to attach to this session. The gateway records it from
	// the trusted client attach and then admits the agent half only if the
	// agent's own cert CN equals it -- so an agent can never attach to a shell
	// aimed at a different host. The agent half leaves it empty.
	ObjectID string `json:"object_id,omitempty"`
	Cols     uint16 `json:"cols,omitempty"`
	Rows     uint16 `json:"rows,omitempty"`
}

// ShellResizePayload is a terminal-size change, client->agent.
type ShellResizePayload struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// ShellClosePayload ends a session; Reason is a short human string for logs.
type ShellClosePayload struct {
	Reason string `json:"reason,omitempty"`
}

// Task is what GetTaskResponse carries: one step for the requesting host
// to run. Task == nil (an empty {"task":null} response body) means
// "nothing to do right now" -- not an error, just poll again later.
type GetTaskResponsePayload struct {
	Task *Task `json:"task"`
}

type Task struct {
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

// ValidatorResult is one `validate:` check's real, structured outcome --
// kind/args/passed/message, matching `validator_result`'s own typed
// columns (migrations/00004) exactly. Mirrors the Rust agent's own
// ValidatorResultPayload (agent/src/protocol.rs) wire-for-wire. Only ever
// non-empty on a report for a "validate" task -- found completely
// discarded by direct audit: the agent already
// computed exactly this, per check, but flattened it into the generic
// Output/Error text below before this field existed, so the gateway had
// no way to persist anything more structured than one blob.
type ValidatorResult struct {
	Kind    string          `json:"kind"`
	Args    json.RawMessage `json:"args,omitempty"`
	Passed  bool            `json:"passed"`
	Message string          `json:"message,omitempty"`
}

type ReportStatusRequestPayload struct {
	TaskID           string            `json:"task_id"`
	Status           string            `json:"status"` // "done" | "failed"
	Output           string            `json:"output,omitempty"`
	Error            string            `json:"error,omitempty"`
	ValidatorResults []ValidatorResult `json:"validator_results,omitempty"`
}

type ReportStatusResponsePayload struct {
	OK bool `json:"ok"`
}

// LogLine is one captured console line as the agent sends it: only what the
// agent knows -- when (epoch milliseconds, the agent's wall clock), which
// stream, and the text. The gateway enriches it with the object's build / team
// / name / kind (keyed off the agent's cert, like every other message) before
// it reaches a sink. Dropped > 0 marks a synthetic record standing in for that
// many lines the agent's bounded buffer had to drop before this one (overflow),
// so a flood is visible as a gap, never silent. Mirrors the Rust agent's
// LogLine (agent/src/applog.rs) field-for-field.
type LogLine struct {
	TSMs    int64  `json:"ts_ms"`
	Stream  string `json:"stream"` // "stdout" | "stderr"
	Line    string `json:"line"`
	Dropped int    `json:"dropped,omitempty"`
}

type LogBatchRequestPayload struct {
	Records []LogLine `json:"records"`
}

type LogBatchResponsePayload struct {
	OK bool `json:"ok"`
}
