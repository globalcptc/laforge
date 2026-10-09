// Interactive shell sessions: the agent's half of the PTY relay. When the
// gateway tells the agent (via the heartbeat response's pending_sessions) that a
// client wants a shell, the agent opens TWO mTLS connections dedicated to that
// session, attaches each as the "agent" half, spawns a real PTY shell as its own
// identity (root on Linux, SYSTEM on Windows), and relays raw bytes both ways.
//
// Why two connections. rustls is a single state machine -- you cannot read and
// write one ClientConnection from two threads at once. An earlier design used a
// single connection with a short socket read timeout to interleave reads and
// writes on one thread; it worked on Linux but on Windows inbound keystrokes
// never arrived (the timeout/read semantics differ). So each session now uses
// two strictly one-directional connections, each owned by exactly one thread:
//   - OUT: the agent only ever WRITES PTY output here (ShellDirOut).
//   - IN:  the agent only ever READS client input here  (ShellDirIn).
// No timeouts, no interleave, no shared TLS state -- just blocking reads on one
// connection and blocking writes on the other, which behaves identically on
// Linux and Windows. The gateway pairs both against the single (duplex) client
// connection. When one side ends, it unblocks the other: the input side kills
// the shell (so the output thread's PTY read returns EOF) and the output side
// shuts down the IN socket (so the input thread's blocking read returns).

use std::collections::HashSet;
use std::io::{self, Read, Write};
use std::net::{Shutdown, TcpStream};
use std::sync::{Arc, Mutex};

use portable_pty::{native_pty_system, CommandBuilder, PtySize};
use rustls::pki_types::ServerName;
use rustls::{ClientConfig, ClientConnection, StreamOwned};

use crate::dlog;
use crate::protocol::{
    self, MessageType, ShellAttachPayload, ShellClosePayload, ShellResizePayload, SHELL_DIR_IN, SHELL_DIR_OUT, SHELL_ROLE_AGENT,
};

const PTY_READ_BUF: usize = 8192;

type TlsConn = StreamOwned<ClientConnection, TcpStream>;

/// maybe_spawn starts a shell-session thread for each pending id not already
/// being handled, recording it in `handled` so a id advertised again on the
/// next heartbeat (before both connections attach) is not spawned twice.
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
        dlog!("laforge-agent: gateway requested shell session {id}; opening connections");
        std::thread::spawn(move || {
            if let Err(e) = run_session(&id, &addr, &host_only, tls_config) {
                dlog!("laforge-agent: shell session {id} ended: {e}");
            }
        });
    }
}

/// dial_attach opens one mTLS connection to the gateway relay and sends the
/// opening ShellAttach for the agent half in the given direction. The gateway
/// already knows (from the client's trusted attach) which object may serve this
/// session and checks it against our cert; we only name the session, our role,
/// and which direction this connection carries.
fn dial_attach(session_id: &str, addr: &str, host_only: &str, tls_config: &Arc<ClientConfig>, dir: &str) -> io::Result<TlsConn> {
    let tcp = TcpStream::connect(addr)?;
    tcp.set_nodelay(true).ok();
    let server_name = ServerName::try_from(host_only.to_string())
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, format!("invalid server name {host_only:?}: {e}")))?;
    let conn = ClientConnection::new(tls_config.clone(), server_name).map_err(|e| io::Error::other(format!("starting TLS: {e}")))?;
    let mut tls = StreamOwned::new(conn, tcp);
    let attach = serde_json::to_vec(&ShellAttachPayload {
        session_id: session_id.to_string(),
        role: SHELL_ROLE_AGENT.to_string(),
        dir: dir.to_string(),
    })
    .map_err(|e| io::Error::other(format!("encoding attach: {e}")))?;
    protocol::write_frame(&mut tls, MessageType::ShellAttach, &attach)?;
    Ok(tls)
}

fn run_session(session_id: &str, addr: &str, host_only: &str, tls_config: Arc<ClientConfig>) -> io::Result<()> {
    // Two strictly one-directional connections: OUT for PTY output, IN for
    // client input. Each is owned by exactly one thread (see the module doc).
    let mut tls_out = dial_attach(session_id, addr, host_only, &tls_config, SHELL_DIR_OUT)?;
    let mut tls_in = dial_attach(session_id, addr, host_only, &tls_config, SHELL_DIR_IN)?;

    // Clones of the raw sockets so each side can unblock the other on exit.
    let out_sd = tls_out.sock.try_clone()?;
    let in_sd = tls_in.sock.try_clone()?;

    // Open the PTY and launch the shell as our own identity (root/SYSTEM).
    let pty = native_pty_system();
    let pair = pty
        .openpty(PtySize { rows: 24, cols: 80, pixel_width: 0, pixel_height: 0 })
        .map_err(|e| io::Error::other(format!("openpty: {e}")))?;
    let child = Arc::new(Mutex::new(
        pair.slave.spawn_command(shell_command()).map_err(|e| io::Error::other(format!("spawning shell: {e}")))?,
    ));
    drop(pair.slave); // we only need the master from here on

    let mut writer = pair.master.take_writer().map_err(|e| io::Error::other(format!("pty writer: {e}")))?;
    let mut reader = pair.master.try_clone_reader().map_err(|e| io::Error::other(format!("pty reader: {e}")))?;

    // OUTPUT thread: PTY -> client. Pure blocking reads of the PTY master, pure
    // writes on the OUT connection. On exit (shell exited, or the client went
    // away) it tells the client, kills the shell, and shuts down the IN socket
    // so the input thread's blocking read returns.
    let out_child = Arc::clone(&child);
    let out_sid = session_id.to_string();
    let out_thread = std::thread::spawn(move || {
        let mut buf = [0u8; PTY_READ_BUF];
        let reason = loop {
            match reader.read(&mut buf) {
                Ok(0) => break "shell exited",
                Ok(n) => {
                    if protocol::write_frame(&mut tls_out, MessageType::ShellData, &buf[..n]).is_err() {
                        break "client disconnected";
                    }
                }
                Err(_) => break "shell exited",
            }
        };
        let body = serde_json::to_vec(&ShellClosePayload { reason: reason.to_string() }).unwrap_or_default();
        let _ = protocol::write_frame(&mut tls_out, MessageType::ShellClose, &body);
        dlog!("laforge-agent: shell {out_sid}: output side ended ({reason})");
        let _ = out_child.lock().unwrap().kill();
        let _ = in_sd.shutdown(Shutdown::Both);
    });

    // INPUT loop (this thread): client -> PTY. Pure blocking reads on the IN
    // connection; frames drive the PTY. On exit it kills the shell (which makes
    // the output thread's PTY read return EOF) and shuts down the OUT socket.
    let in_reason = loop {
        match protocol::read_frame(&mut tls_in) {
            Ok((MessageType::ShellData, payload)) => {
                if writer.write_all(&payload).is_err() {
                    break "pty closed";
                }
                writer.flush().ok();
            }
            Ok((MessageType::ShellResize, payload)) => {
                if let Ok(r) = serde_json::from_slice::<ShellResizePayload>(&payload) {
                    pair.master.resize(PtySize { rows: r.rows, cols: r.cols, pixel_width: 0, pixel_height: 0 }).ok();
                }
            }
            Ok((MessageType::ShellClose, _)) => break "closed by client",
            Ok(_) => {} // ignore anything unexpected on a shell connection
            Err(_) => break "client disconnected",
        }
    };
    dlog!("laforge-agent: shell {session_id}: input side ended ({in_reason})");
    let _ = child.lock().unwrap().kill();
    let _ = out_sd.shutdown(Shutdown::Both);
    out_thread.join().ok();
    let _ = child.lock().unwrap().wait();
    Ok(())
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
