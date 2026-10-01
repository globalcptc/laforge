// The wire protocol, independently implemented from
// internal/agentproto/protocol.go against the same shared spec -- see
// that file's doc comment for the frame shape and why JSON-inside-TLS is
// the whole "custom protocol," not a hand-rolled handshake.

use std::io::{Error, ErrorKind, Read, Result as IoResult, Write};

pub const MAX_FRAME_SIZE: u32 = 10 * 1024 * 1024;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum MessageType {
    HeartbeatRequest = 0x01,
    HeartbeatResponse = 0x02,
    GetTaskRequest = 0x03,
    GetTaskResponse = 0x04,
    ReportStatusRequest = 0x05,
    ReportStatusResponse = 0x06,
    LogBatchRequest = 0x07,
    LogBatchResponse = 0x08,
    // Interactive shell relay (see internal/agentproto/protocol.go). These are a
    // bidirectional PTY byte stream on a connection dedicated to one session,
    // not the poll-driven request/response the verbs above use.
    ShellAttach = 0x09,
    ShellData = 0x0A,
    ShellResize = 0x0B,
    ShellClose = 0x0C,
}

impl MessageType {
    pub fn from_u8(b: u8) -> IoResult<Self> {
        match b {
            0x01 => Ok(Self::HeartbeatRequest),
            0x02 => Ok(Self::HeartbeatResponse),
            0x03 => Ok(Self::GetTaskRequest),
            0x04 => Ok(Self::GetTaskResponse),
            0x05 => Ok(Self::ReportStatusRequest),
            0x06 => Ok(Self::ReportStatusResponse),
            0x07 => Ok(Self::LogBatchRequest),
            0x08 => Ok(Self::LogBatchResponse),
            0x09 => Ok(Self::ShellAttach),
            0x0A => Ok(Self::ShellData),
            0x0B => Ok(Self::ShellResize),
            0x0C => Ok(Self::ShellClose),
            other => Err(Error::new(
                ErrorKind::InvalidData,
                format!("unknown message type 0x{other:02x}"),
            )),
        }
    }
}

pub fn write_frame<W: Write>(w: &mut W, msg_type: MessageType, payload: &[u8]) -> IoResult<()> {
    if payload.len() as u64 > (MAX_FRAME_SIZE as u64) - 1 {
        return Err(Error::new(ErrorKind::InvalidInput, "payload exceeds MAX_FRAME_SIZE"));
    }
    let len: u32 = payload.len() as u32 + 1;
    w.write_all(&len.to_be_bytes())?;
    w.write_all(&[msg_type as u8])?;
    if !payload.is_empty() {
        w.write_all(payload)?;
    }
    w.flush()
}

/// Reads one frame. Never panics on malformed input -- every failure
/// mode (a zero length, an oversized claim, a truncated body, an unknown
/// type byte) is a returned io::Error, matching
/// internal/agentproto.ReadFrame's own contract and its fuzz test.
pub fn read_frame<R: Read>(r: &mut R) -> IoResult<(MessageType, Vec<u8>)> {
    let mut header = [0u8; 4];
    r.read_exact(&mut header)?;
    let length = u32::from_be_bytes(header);
    if length == 0 {
        return Err(Error::new(ErrorKind::InvalidData, "frame length is 0"));
    }
    if length > MAX_FRAME_SIZE {
        return Err(Error::new(ErrorKind::InvalidData, "frame length exceeds MAX_FRAME_SIZE"));
    }
    let mut body = vec![0u8; length as usize];
    r.read_exact(&mut body)?;
    let mt = MessageType::from_u8(body[0])?;
    Ok((mt, body[1..].to_vec()))
}

/// Basic host metrics the agent samples on each heartbeat: cpu/mem/disk are
/// percentages (0-100); net_rx/tx are bytes per second since the previous
/// heartbeat. Every field is optional so a value the agent couldn't read, and
/// the first heartbeat of a session (before a network rate exists), send none
/// rather than a misleading zero. Mirrors internal/agentproto.HeartbeatRequestPayload.
#[derive(serde::Serialize, Debug, Default)]
pub struct HeartbeatRequestPayload {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cpu_pct: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mem_pct: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub disk_pct: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub net_rx_bps: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub net_tx_bps: Option<f64>,
}

#[derive(serde::Deserialize, Debug)]
pub struct HeartbeatResponsePayload {
    pub next_poll_ms: u64,
    /// Interactive-shell session ids a client has requested on this object that
    /// have no agent connection yet. For each, the agent opens a second mTLS
    /// connection and attaches as the "agent" half. Defaulted so an older
    /// gateway (which never sends the field) deserializes to an empty list.
    #[serde(default)]
    pub pending_sessions: Vec<String>,
}

/// The first frame on a shell-relay connection the agent opens: which session
/// and that this is the agent (PTY-holding) half. Mirrors
/// internal/agentproto.ShellAttachPayload (the agent only ever sends the agent
/// role, so object_id/cols/rows -- client-only fields -- are omitted).
#[derive(serde::Serialize, Debug)]
pub struct ShellAttachPayload {
    pub session_id: String,
    pub role: String,
}

/// A terminal-size change the agent receives from the client half.
#[derive(serde::Deserialize, Debug)]
pub struct ShellResizePayload {
    pub cols: u16,
    pub rows: u16,
}

/// Ends a session, sent or received by the agent. Mirrors
/// internal/agentproto.ShellClosePayload.
#[derive(serde::Serialize, serde::Deserialize, Debug, Default)]
pub struct ShellClosePayload {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

/// The agent role string for ShellAttachPayload.role.
pub const SHELL_ROLE_AGENT: &str = "agent";

#[derive(serde::Deserialize, Debug, Clone)]
pub struct Task {
    pub id: String,
    pub command: String,
    pub payload: serde_json::Value,
}

#[derive(serde::Deserialize, Debug)]
pub struct GetTaskResponsePayload {
    pub task: Option<Task>,
}

/// One `validate:` check's real, structured outcome -- kind/args/passed/
/// message, matching `validator_result`'s own typed columns
/// (migrations/00004) exactly, so the gateway can persist it as one row
/// per check rather than the single flattened text blob `output`/`error`
/// otherwise carries (see commands::Outcome's own doc comment: "always
/// one of these" -- this is the one real exception, for validate only).
#[derive(serde::Serialize, Debug, Clone)]
pub struct ValidatorResultPayload {
    pub kind: String,
    #[serde(skip_serializing_if = "serde_json::Value::is_null")]
    pub args: serde_json::Value,
    pub passed: bool,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub message: String,
}

#[derive(serde::Serialize, Debug, Default)]
pub struct ReportStatusRequestPayload {
    pub task_id: String,
    pub status: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub output: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub error: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub validator_results: Vec<ValidatorResultPayload>,
}

#[derive(serde::Deserialize, Debug)]
#[allow(dead_code)]
pub struct ReportStatusResponsePayload {
    #[allow(dead_code)]
    pub ok: bool,
}

/// One captured console line, as the agent sends it: when (epoch milliseconds,
/// the agent's wall clock), which stream, and the text. The gateway enriches it
/// with the object's build/team/name/kind. `dropped` > 0 marks a synthetic
/// record standing in for that many lines the bounded buffer had to drop before
/// this one. Mirrors internal/agentproto.LogLine field-for-field.
#[derive(serde::Serialize, Debug, Clone)]
pub struct LogLine {
    pub ts_ms: i64,
    pub stream: &'static str, // "stdout" | "stderr"
    pub line: String,
    #[serde(skip_serializing_if = "is_zero")]
    pub dropped: usize,
}

fn is_zero(n: &usize) -> bool {
    *n == 0
}

#[derive(serde::Serialize, Debug)]
pub struct LogBatchRequestPayload {
    pub records: Vec<LogLine>,
}

#[derive(serde::Deserialize, Debug)]
#[allow(dead_code)]
pub struct LogBatchResponsePayload {
    #[allow(dead_code)]
    pub ok: bool,
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    #[test]
    fn write_read_round_trip() {
        let mut buf = Vec::new();
        write_frame(&mut buf, MessageType::HeartbeatRequest, &[]).unwrap();
        write_frame(&mut buf, MessageType::GetTaskResponse, b"{\"task\":null}").unwrap();

        let mut cursor = Cursor::new(buf);
        let (mt, body) = read_frame(&mut cursor).unwrap();
        assert_eq!(mt, MessageType::HeartbeatRequest);
        assert!(body.is_empty());

        let (mt, body) = read_frame(&mut cursor).unwrap();
        assert_eq!(mt, MessageType::GetTaskResponse);
        assert_eq!(body, b"{\"task\":null}");
    }

    #[test]
    fn rejects_zero_length() {
        let mut cursor = Cursor::new(0u32.to_be_bytes().to_vec());
        assert!(read_frame(&mut cursor).is_err());
    }

    #[test]
    fn rejects_oversized_claim() {
        let mut cursor = Cursor::new((MAX_FRAME_SIZE + 1).to_be_bytes().to_vec());
        assert!(read_frame(&mut cursor).is_err());
    }

    #[test]
    fn rejects_truncated_body() {
        let mut data = 100u32.to_be_bytes().to_vec();
        data.extend_from_slice(b"short");
        let mut cursor = Cursor::new(data);
        assert!(read_frame(&mut cursor).is_err());
    }

    #[test]
    fn rejects_unknown_message_type() {
        let mut data = 1u32.to_be_bytes().to_vec();
        data.push(0xff);
        let mut cursor = Cursor::new(data);
        assert!(read_frame(&mut cursor).is_err());
    }

    /// The same property internal/agentproto's FuzzReadFrame checks --
    /// never panic, on anything -- run over a fixed adversarial corpus
    /// rather than under `cargo fuzz` (a heavier, nightly-only toolchain
    /// not installed this session). Every
    /// one of these was chosen to hit a different edge of the length
    /// handling.
    #[test]
    fn never_panics_on_adversarial_input() {
        let cases: Vec<Vec<u8>> = vec![
            vec![],
            vec![0, 0, 0, 0],
            vec![0, 0, 0, 1],
            vec![0xff, 0xff, 0xff, 0xff],
            vec![0, 0, 0, 5, 1, 2, 3],
            vec![0, 0, 0, 1, 0xff],
            vec![0, 0],
            vec![255],
        ];
        for case in cases {
            let mut cursor = Cursor::new(case.clone());
            let _ = read_frame(&mut cursor); // must not panic, error is fine
        }
    }
}
