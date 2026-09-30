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

type HeartbeatResponsePayload struct {
	NextPollMS int `json:"next_poll_ms"`
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
