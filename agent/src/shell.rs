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
use std::sync::{mpsc, Arc, Mutex};

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
    // Trace each step. The shell relay is where the agent is most likely to die,
    // and with panic="abort" a panic leaves no close line -- so the LAST of these
    // lines before the log goes quiet tells us exactly how far the session got.
    // None of it is sensitive (plumbing, not payloads), and it only ever reaches
    // the file on an agent-debug build.
    dlog!("laforge-agent: shell {session_id}: dialing gateway relay (OUT) at {addr}");
    // Two strictly one-directional connections: OUT for shell output, IN for
    // client input. Each is owned by exactly one thread (see the module doc).
    let tls_out = dial_attach(session_id, addr, host_only, &tls_config, SHELL_DIR_OUT)?;
    dlog!("laforge-agent: shell {session_id}: OUT attached; dialing relay (IN)");
    let tls_in = dial_attach(session_id, addr, host_only, &tls_config, SHELL_DIR_IN)?;
    dlog!("laforge-agent: shell {session_id}: IN attached; starting shell");

    // Pick the backend. A real PTY (openpty on Unix, ConPTY on modern Windows)
    // gives a fully interactive terminal. But a Windows older than 10 1809 /
    // Server 2019 (e.g. Server 2016) has NO ConPTY: portable-pty PANICS looking
    // up CreatePseudoConsole, which under panic="abort" would take the whole
    // agent down. On such a host we must not even reach that call -- fall back to
    // a pipe-based shell (plain stdin/stdout/stderr pipes): no resize or rich
    // terminal handling, but a working prompt for debugging scripts.
    if conpty_available() {
        run_session_pty(session_id, tls_out, tls_in)
    } else {
        dlog!("laforge-agent: shell {session_id}: ConPTY unavailable (Windows < 1809 / Server 2019); using non-PTY fallback shell");
        run_session_piped(session_id, tls_out, tls_in)
    }
}

// run_session_pty is the interactive-terminal backend: a real PTY whose master
// read/write/resize drive the shared relay. Used on Unix always, and on Windows
// that has ConPTY.
fn run_session_pty(session_id: &str, tls_out: TlsConn, tls_in: TlsConn) -> io::Result<()> {
    let pty = native_pty_system();
    let pair = pty
        .openpty(PtySize { rows: 24, cols: 80, pixel_width: 0, pixel_height: 0 })
        .map_err(|e| io::Error::other(format!("openpty: {e}")))?;
    let shell = shell_command();
    let shell_prog = shell.get_argv().first().map(|s| s.to_string_lossy().into_owned()).unwrap_or_default();
    dlog!("laforge-agent: shell {session_id}: PTY open; spawning {shell_prog}");
    let child = pair.slave.spawn_command(shell).map_err(|e| io::Error::other(format!("spawning shell: {e}")))?;
    drop(pair.slave); // we only need the master from here on

    let writer = pair.master.take_writer().map_err(|e| io::Error::other(format!("pty writer: {e}")))?;
    let reader = pair.master.try_clone_reader().map_err(|e| io::Error::other(format!("pty reader: {e}")))?;
    // The master stays alive inside the resize closure, which the relay calls on
    // a ShellResize frame -- the one thing a PTY can do that a pipe cannot.
    let master = pair.master;
    let resize: Box<dyn FnMut(u16, u16) + Send> = Box::new(move |rows, cols| {
        let _ = master.resize(PtySize { rows, cols, pixel_width: 0, pixel_height: 0 });
    });
    let child: Arc<Mutex<dyn ShellChild>> = Arc::new(Mutex::new(child));
    relay(session_id, tls_out, tls_in, reader, writer, Some(resize), child)
}

// run_session_piped is the fallback for a Windows without ConPTY: spawn the
// shell with plain pipes and relay bytes. No PTY means no terminal semantics
// (no resize, no color/line editing from a real console), but cmd/PowerShell
// still runs commands and returns output -- enough to debug a host. stdout and
// stderr are merged into one ordered stream for the relay.
fn run_session_piped(session_id: &str, tls_out: TlsConn, tls_in: TlsConn) -> io::Result<()> {
    let mut cmd = shell_std_command();
    cmd.stdin(std::process::Stdio::piped())
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());
    let prog = cmd.get_program().to_string_lossy().into_owned();
    dlog!("laforge-agent: shell {session_id}: spawning {prog} (no PTY)");
    let mut child = cmd.spawn().map_err(|e| io::Error::other(format!("spawning fallback shell: {e}")))?;
    let stdin = child.stdin.take().ok_or_else(|| io::Error::other("fallback shell has no stdin"))?;
    let stdout = child.stdout.take().ok_or_else(|| io::Error::other("fallback shell has no stdout"))?;
    let stderr = child.stderr.take().ok_or_else(|| io::Error::other("fallback shell has no stderr"))?;

    // Merge stdout+stderr: a pump thread per pipe feeds one channel, and
    // ChannelReader serves the bytes. Both pumps dropping their sender (both
    // pipes at EOF) makes the reader return EOF, ending the relay's output side.
    let (tx, rx) = mpsc::channel::<Vec<u8>>();
    spawn_pipe_pump(stdout, tx.clone());
    spawn_pipe_pump(stderr, tx);
    let reader: Box<dyn Read + Send> = Box::new(ChannelReader { rx, buf: Vec::new(), pos: 0 });
    let writer: Box<dyn Write + Send> = Box::new(stdin);
    let child: Arc<Mutex<dyn ShellChild>> = Arc::new(Mutex::new(child));
    relay(session_id, tls_out, tls_in, reader, writer, None, child)
}

// relay is the backend-independent pump: it bridges the two one-directional
// gateway connections to a shell's reader/writer. OUTPUT thread copies shell ->
// client; the INPUT loop (this thread) copies client -> shell and applies
// resizes (when the backend supports them). When either side ends it kills the
// shell and shuts the other socket down so both unblock. Shared by the PTY and
// pipe backends so their lifecycle, logging, and teardown stay identical.
fn relay(
    session_id: &str,
    mut tls_out: TlsConn,
    mut tls_in: TlsConn,
    mut reader: Box<dyn Read + Send>,
    mut writer: Box<dyn Write + Send>,
    mut resize: Option<Box<dyn FnMut(u16, u16) + Send>>,
    child: Arc<Mutex<dyn ShellChild>>,
) -> io::Result<()> {
    // Clones of the raw sockets so each side can unblock the other on exit.
    let out_sd = tls_out.sock.try_clone()?;
    let in_sd = tls_in.sock.try_clone()?;
    dlog!("laforge-agent: shell {session_id}: relaying");

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
        out_child.lock().unwrap().kill();
        let _ = in_sd.shutdown(Shutdown::Both);
    });

    let in_reason = loop {
        match protocol::read_frame(&mut tls_in) {
            Ok((MessageType::ShellData, payload)) => {
                if writer.write_all(&payload).is_err() {
                    break "shell closed";
                }
                writer.flush().ok();
            }
            Ok((MessageType::ShellResize, payload)) => {
                if let Some(r) = resize.as_mut() {
                    if let Ok(rz) = serde_json::from_slice::<ShellResizePayload>(&payload) {
                        r(rz.rows, rz.cols);
                    }
                }
            }
            Ok((MessageType::ShellClose, _)) => break "closed by client",
            Ok(_) => {} // ignore anything unexpected on a shell connection
            Err(_) => break "client disconnected",
        }
    };
    dlog!("laforge-agent: shell {session_id}: input side ended ({in_reason})");
    child.lock().unwrap().kill();
    let _ = out_sd.shutdown(Shutdown::Both);
    out_thread.join().ok();
    child.lock().unwrap().wait();
    Ok(())
}

// spawn_pipe_pump drains one child pipe into the merge channel until EOF or
// error, then drops its sender (so the reader learns when both pipes are done).
fn spawn_pipe_pump(mut src: impl Read + Send + 'static, tx: mpsc::Sender<Vec<u8>>) {
    std::thread::spawn(move || {
        let mut buf = [0u8; PTY_READ_BUF];
        loop {
            match src.read(&mut buf) {
                Ok(0) | Err(_) => break,
                Ok(n) => {
                    if tx.send(buf[..n].to_vec()).is_err() {
                        break; // the reader is gone
                    }
                }
            }
        }
    });
}

// ChannelReader turns the merged stdout/stderr channel into a blocking Read for
// the pipe fallback. recv() erroring means every sender dropped (both pipes at
// EOF), which it reports as end-of-stream.
struct ChannelReader {
    rx: mpsc::Receiver<Vec<u8>>,
    buf: Vec<u8>,
    pos: usize,
}

impl Read for ChannelReader {
    fn read(&mut self, out: &mut [u8]) -> io::Result<usize> {
        while self.pos >= self.buf.len() {
            match self.rx.recv() {
                Ok(v) => {
                    self.buf = v;
                    self.pos = 0;
                }
                Err(_) => return Ok(0), // all senders dropped -> EOF
            }
        }
        let n = (self.buf.len() - self.pos).min(out.len());
        out[..n].copy_from_slice(&self.buf[self.pos..self.pos + n]);
        self.pos += n;
        Ok(n)
    }
}

// ShellChild abstracts the two spawned-process handles -- a portable-pty Child
// (PTY backend) and a std process Child (pipe fallback) -- so relay can kill and
// reap either one the same way.
trait ShellChild: Send {
    fn kill(&mut self);
    fn wait(&mut self);
}

impl ShellChild for Box<dyn portable_pty::Child + Send + Sync> {
    fn kill(&mut self) {
        let _ = (**self).kill();
    }
    fn wait(&mut self) {
        let _ = (**self).wait();
    }
}

impl ShellChild for std::process::Child {
    fn kill(&mut self) {
        let _ = std::process::Child::kill(self);
    }
    fn wait(&mut self) {
        let _ = std::process::Child::wait(self);
    }
}

// conpty_available reports whether this host has the ConPTY API that
// portable-pty's Windows backend requires. CreatePseudoConsole was added in
// Windows 10 1809 / Server 2019; on anything older portable-pty panics, so we
// check for the symbol ourselves first and take the pipe fallback instead.
#[cfg(windows)]
fn conpty_available() -> bool {
    use std::os::raw::c_void;
    extern "system" {
        fn GetModuleHandleA(name: *const u8) -> *mut c_void;
        fn GetProcAddress(module: *mut c_void, name: *const u8) -> *mut c_void;
    }
    // kernel32 is always loaded; CreatePseudoConsole is only exported on 1809+.
    unsafe {
        let k32 = GetModuleHandleA(b"kernel32.dll\0".as_ptr());
        if k32.is_null() {
            return false;
        }
        !GetProcAddress(k32, b"CreatePseudoConsole\0".as_ptr()).is_null()
    }
}

#[cfg(unix)]
fn conpty_available() -> bool {
    true // Unix always has openpty; the ConPTY question is Windows-only.
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

// shell_std_command is the pipe-fallback counterpart of shell_command: the same
// shell choice, but as a std::process::Command (no PTY). Cross-platform so the
// fallback and its relay path compile and test on Unix even though it's only
// reached at runtime on a ConPTY-less Windows.
#[cfg(unix)]
fn shell_std_command() -> std::process::Command {
    let shell = if std::path::Path::new("/bin/bash").exists() { "/bin/bash" } else { "/bin/sh" };
    std::process::Command::new(shell)
}

#[cfg(windows)]
fn shell_std_command() -> std::process::Command {
    let powershell = r"C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe";
    if std::path::Path::new(powershell).exists() {
        std::process::Command::new(powershell)
    } else {
        std::process::Command::new("cmd.exe")
    }
}

#[cfg(test)]
mod tests {
    use super::ChannelReader;
    use std::io::Read;
    use std::sync::mpsc;

    // The merged stdout/stderr reader must serve bytes in order across chunks and
    // report EOF once every sender (both pipe pumps) has dropped.
    #[test]
    fn channel_reader_orders_chunks_then_eof() {
        let (tx, rx) = mpsc::channel();
        tx.send(b"hello ".to_vec()).unwrap();
        tx.send(b"world".to_vec()).unwrap();
        drop(tx);
        let mut r = ChannelReader { rx, buf: Vec::new(), pos: 0 };
        let mut out = String::new();
        r.read_to_string(&mut out).unwrap();
        assert_eq!(out, "hello world");
    }

    // A chunk larger than the caller's buffer is handed back across several reads
    // without losing or reordering bytes, and the final read returns 0 (EOF).
    #[test]
    fn channel_reader_partial_reads() {
        let (tx, rx) = mpsc::channel();
        tx.send(b"abcde".to_vec()).unwrap();
        drop(tx);
        let mut r = ChannelReader { rx, buf: Vec::new(), pos: 0 };
        let mut b = [0u8; 2];
        assert_eq!(r.read(&mut b).unwrap(), 2);
        assert_eq!(&b, b"ab");
        assert_eq!(r.read(&mut b).unwrap(), 2);
        assert_eq!(&b, b"cd");
        assert_eq!(r.read(&mut b).unwrap(), 1);
        assert_eq!(&b[..1], b"e");
        assert_eq!(r.read(&mut b).unwrap(), 0);
    }
}
