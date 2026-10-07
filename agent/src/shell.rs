// Interactive shell sessions: the agent's half of the PTY relay. When the
// gateway tells the agent (via the heartbeat response's pending_sessions) that a
// client wants a shell, the agent opens a SECOND mTLS connection dedicated to
// that session, attaches as the "agent" half, spawns a real PTY shell as its own
// identity (root on Linux, SYSTEM on Windows), and relays raw bytes both ways.
//
// Duplex over one blocking TLS connection is the tricky part. rustls is a single
// state machine -- you cannot read and write it from two threads at once -- and
// read_frame is not resumable across a socket read timeout. So ONE thread owns
// the TLS connection and does everything: a short socket read timeout lets it
// read whatever bytes are available (rustls tolerates partial records and
// resumes), accumulate them, and parse any complete frames, then drain outbound
// PTY bytes and write them. A separate thread does nothing but blocking-read the
// PTY master and hand bytes to a channel -- it never touches TLS. This is fully
// portable (no fd polling), so it works identically on Linux and Windows.

use std::collections::HashSet;
use std::io::{self, Read, Write};
use std::net::TcpStream;
use std::sync::mpsc;
use std::sync::{Arc, Mutex};
use std::time::Duration;

use portable_pty::{native_pty_system, CommandBuilder, PtySize};
use rustls::pki_types::ServerName;
use rustls::{ClientConfig, ClientConnection, StreamOwned};

use crate::protocol::{self, MessageType, ShellAttachPayload, ShellClosePayload, ShellResizePayload, SHELL_ROLE_AGENT};

// How long a read blocks before returning so the loop can write pending PTY
// output. Short enough that keystroke->screen latency is imperceptible, long
// enough not to spin the CPU.
const READ_TIMEOUT: Duration = Duration::from_millis(10);
const PTY_READ_BUF: usize = 8192;

/// maybe_spawn starts a shell-session thread for each pending id not already
/// being handled, recording it in `handled` so a id advertised again on the
/// next heartbeat (before the attach completes) is not spawned twice.
pub fn maybe_spawn(pending: &[String], handled: &Arc<Mutex<HashSet<String>>>, addr: &str, host_only: &str, tls_config: &Arc<ClientConfig>) {
    for id in pending {
        {
            let mut set = handled.lock().unwrap();
            if set.contains(id) {
                continue;
            }
            set.insert(id.clone());
        }
        let (id, addr, host_only, tls_config) = (id.clone(), addr.to_string(), host_only.to_string(), tls_config.clone());
        // Diagnostic: if the gateway logs "signaling agent" but you never see
        // this line on the host, this agent is a stale build without shell
        // support -- redeploy the host.
        eprintln!("laforge-agent: gateway requested shell session {id}; opening a connection");
        std::thread::spawn(move || {
            if let Err(e) = run_session(&id, &addr, &host_only, tls_config) {
                eprintln!("laforge-agent: shell session {id} ended: {e}");
            }
        });
    }
}

fn run_session(session_id: &str, addr: &str, host_only: &str, tls_config: Arc<ClientConfig>) -> io::Result<()> {
    let tcp = TcpStream::connect(addr)?;
    tcp.set_nodelay(true).ok();
    let server_name = ServerName::try_from(host_only.to_string())
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, format!("invalid server name {host_only:?}: {e}")))?;
    let conn = ClientConnection::new(tls_config, server_name).map_err(|e| io::Error::other(format!("starting TLS: {e}")))?;
    let mut tls = StreamOwned::new(conn, tcp);

    // Attach as the agent half. The gateway already knows (from the client's
    // trusted attach) which object may serve this session and checks it against
    // our cert; we only name the session and our role.
    let attach = serde_json::to_vec(&ShellAttachPayload { session_id: session_id.to_string(), role: SHELL_ROLE_AGENT.to_string() })
        .map_err(|e| io::Error::other(format!("encoding attach: {e}")))?;
    protocol::write_frame(&mut tls, MessageType::ShellAttach, &attach)?;

    // Open the PTY and launch the shell.
    let pty = native_pty_system();
    let pair = pty
        .openpty(PtySize { rows: 24, cols: 80, pixel_width: 0, pixel_height: 0 })
        .map_err(|e| io::Error::other(format!("openpty: {e}")))?;
    let mut child = pair
        .slave
        .spawn_command(shell_command())
        .map_err(|e| io::Error::other(format!("spawning shell: {e}")))?;
    drop(pair.slave); // we only need the master from here on

    let mut writer = pair.master.take_writer().map_err(|e| io::Error::other(format!("pty writer: {e}")))?;
    let mut reader = pair.master.try_clone_reader().map_err(|e| io::Error::other(format!("pty reader: {e}")))?;

    // PTY output -> channel. On EOF (shell exit) or error the sender drops,
    // which the main loop observes as Disconnected.
    let (tx, rx) = mpsc::channel::<Vec<u8>>();
    std::thread::spawn(move || {
        let mut buf = [0u8; PTY_READ_BUF];
        loop {
            match reader.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    if tx.send(buf[..n].to_vec()).is_err() {
                        break;
                    }
                }
                Err(_) => break,
            }
        }
    });

    // Single TLS owner: read available inbound bytes (resumable), act on any
    // complete frames, then flush outbound PTY bytes.
    tls.sock.set_read_timeout(Some(READ_TIMEOUT)).ok();
    let mut inbuf: Vec<u8> = Vec::with_capacity(PTY_READ_BUF);
    let mut scratch = [0u8; PTY_READ_BUF];

    let close_reason: &str = 'session: loop {
        // 1) Pull whatever plaintext is ready right now. A read timeout
        //    (WouldBlock on Unix, TimedOut on Windows) just means "nothing this
        //    tick"; rustls keeps any partial record buffered and resumes.
        match tls.read(&mut scratch) {
            Ok(0) => break 'session "client disconnected",
            Ok(n) => inbuf.extend_from_slice(&scratch[..n]),
            Err(e) if e.kind() == io::ErrorKind::WouldBlock || e.kind() == io::ErrorKind::TimedOut => {}
            Err(e) => return Err(e),
        }

        // 2) Parse and act on every complete frame now buffered.
        while let Some((mt, payload)) = take_frame(&mut inbuf) {
            match mt {
                MessageType::ShellData => {
                    writer.write_all(&payload)?;
                    writer.flush().ok();
                }
                MessageType::ShellResize => {
                    if let Ok(r) = serde_json::from_slice::<ShellResizePayload>(&payload) {
                        pair.master
                            .resize(PtySize { rows: r.rows, cols: r.cols, pixel_width: 0, pixel_height: 0 })
                            .ok();
                    }
                }
                MessageType::ShellClose => break 'session "closed by client",
                _ => {} // ignore anything unexpected on a shell connection
            }
        }

        // 3) Flush PTY output. A Disconnected channel means the shell exited.
        loop {
            match rx.try_recv() {
                Ok(bytes) => protocol::write_frame(&mut tls, MessageType::ShellData, &bytes)?,
                Err(mpsc::TryRecvError::Empty) => break,
                Err(mpsc::TryRecvError::Disconnected) => break 'session "shell exited",
            }
        }
    };

    // Tell the other side we're done (best-effort), then reap the child.
    let body = serde_json::to_vec(&ShellClosePayload { reason: close_reason.to_string() }).unwrap_or_default();
    let _ = protocol::write_frame(&mut tls, MessageType::ShellClose, &body);
    let _ = child.kill();
    let _ = child.wait();
    Ok(())
}

/// take_frame removes and returns the first complete frame from buf, or None if
/// a full frame isn't buffered yet. Mirrors protocol::read_frame's length/type
/// layout but over an in-memory buffer so it's resumable across partial reads.
/// A malformed length (0 or over the cap) drains the buffer and returns None to
/// end the session cleanly rather than loop.
fn take_frame(buf: &mut Vec<u8>) -> Option<(MessageType, Vec<u8>)> {
    if buf.len() < 4 {
        return None;
    }
    let length = u32::from_be_bytes([buf[0], buf[1], buf[2], buf[3]]);
    if length == 0 || length > protocol::MAX_FRAME_SIZE {
        buf.clear();
        return None;
    }
    let total = 4 + length as usize;
    if buf.len() < total {
        return None;
    }
    let frame: Vec<u8> = buf.drain(..total).collect();
    let mt = MessageType::from_u8(frame[4]).ok()?;
    Some((mt, frame[5..].to_vec()))
}

/// shell_command picks the interactive shell for this OS. The agent already runs
/// as root (Linux) / SYSTEM (Windows), so the spawned shell inherits that.
#[cfg(unix)]
fn shell_command() -> CommandBuilder {
    let shell = if std::path::Path::new("/bin/bash").exists() { "/bin/bash" } else { "/bin/sh" };
    let mut cmd = CommandBuilder::new(shell);
    cmd.env("TERM", "xterm-256color");
    cmd
}

#[cfg(windows)]
fn shell_command() -> CommandBuilder {
    let powershell = r"C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe";
    let mut cmd = if std::path::Path::new(powershell).exists() {
        CommandBuilder::new(powershell)
    } else {
        CommandBuilder::new("cmd.exe")
    };
    cmd.env("TERM", "xterm-256color");
    cmd
}
