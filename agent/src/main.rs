// laforge-agent: "a single fully-embedded binary with no external
// configuration... opens a TLS connection to the gateway... and presents
// its client certificate; the gateway presents a certificate the agent
// pinned at build time." This file is only ever the connection loop --
// framing lives in protocol.rs, identity in identity.rs, command
// execution in commands.rs and validators.rs.

mod antitamper;
mod applog;
mod chaff;
mod commands;
mod identity;
mod metrics;
mod obfuscate;
mod protocol;
mod selfhash;
mod validators;

use protocol::{GetTaskResponsePayload, HeartbeatResponsePayload, LogBatchRequestPayload, MessageType, ReportStatusRequestPayload};
use rustls::pki_types::{CertificateDer, PrivateKeyDer, ServerName};
use rustls::{ClientConfig, ClientConnection, RootCertStore, StreamOwned};
use std::io;
use std::io::{BufRead, BufReader, Read, Write};
use std::net::TcpStream;
use std::sync::Arc;
use std::time::{Duration, Instant};

// Bounds for container console-log forwarding: how many lines the agent buffers
// before dropping the oldest (memory cap), and the most it ships in one frame.
const LOG_BUFFER_CAP: usize = 10_000;
const LOG_BATCH_MAX: usize = 500;

fn main() {
    // Basic anti-debug/anti-tamper self-checks -- see antitamper.rs's own
    // doc comment for exactly what's real here and what's scoped out.
    // Deliberately non-load-bearing: log and
    // continue, never refuse to start or change behavior on a finding.
    for finding in antitamper::self_check() {
        eprintln!("laforge-agent: self-check: {finding}");
    }

    // Native OCI container supervisor mode: when LAFORGE_SUPERVISE is set, the
    // agent is the container's entrypoint (the Incus builder sets it to the
    // image's original command). It supervises that command and runs the normal
    // gateway loop alongside, so a container checks in and runs steps/validators
    // exactly like a host. On a host, LAFORGE_SUPERVISE is unset.
    match std::env::var("LAFORGE_SUPERVISE").ok().filter(|s| !s.trim().is_empty()) {
        Some(cmd) => run_container(cmd),
        None => run_host(),
    }
}

// run_host is the ordinary agent: an identity is required to do anything.
// load_agent already logs the specific reason it couldn't connect (missing
// identity vs. a bad TLS config -- distinct causes that must not be conflated),
// so this just exits on failure.
fn run_host() -> ! {
    match load_agent() {
        // A host has no supervised app, so no console log queue -- host log
        // collection (journald/services) is a separate follow-up.
        Some((identity, tls_config, host_only)) => agent_loop(&identity, &host_only, tls_config, None),
        None => std::process::exit(1),
    }
}

// run_container starts the gateway loop in the background (if the agent can
// connect) and then supervises the image's real command. The command runs
// regardless of the gateway -- the container's service must be up even if agent
// management can't connect -- so a broken/absent identity never takes the
// service down, it only means no check-in. load_agent logs why, if it can't.
fn run_container(cmd: String) -> ! {
    // One buffer shared between the app's output readers (push) and the gateway
    // loop (drain + ship). Created even if the agent can't connect, so capture
    // is independent of gateway health.
    let logq = Arc::new(applog::LogQueue::new(LOG_BUFFER_CAP));
    if let Some((identity, tls_config, host_only)) = load_agent() {
        let lq = logq.clone();
        std::thread::spawn(move || agent_loop(&identity, &host_only, tls_config, Some(lq)));
    }
    supervise_app(cmd, logq)
}

// load_agent loads the embedded/dev identity and builds its TLS config, or None
// -- logging the SPECIFIC reason (no identity, or a TLS-config failure), which
// are different causes a caller must not conflate: a patched binary whose cert
// fails to parse has an identity, it just can't build a client, and reporting
// that as "no identity" would be wrong.
fn load_agent() -> Option<(identity::Identity, Arc<ClientConfig>, String)> {
    let identity = match identity::load() {
        Some(id) => id,
        None => {
            eprintln!("laforge-agent: no identity available (binary is unpatched and no LAFORGE_DEV_* env vars set)");
            return None;
        }
    };
    let tls_config = match build_tls_config(&identity) {
        Ok(cfg) => Arc::new(cfg),
        Err(e) => {
            eprintln!("laforge-agent: building TLS config: {e}");
            return None;
        }
    };
    let host_only = identity.gateway_addr.split(':').next().unwrap_or(&identity.gateway_addr).to_string();
    Some((identity, tls_config, host_only))
}

// agent_loop is the steady-state connection loop: connect to the gateway, run a
// session, reconnect with a short delay. Never returns.
fn agent_loop(identity: &identity::Identity, host_only: &str, tls_config: Arc<ClientConfig>, logq: Option<Arc<applog::LogQueue>>) -> ! {
    // "Agents tolerate it being down. They retry with backoff and keep their
    // current work." Exponential backoff between reconnects: start at BASE and
    // double up to MAX, so a brief blip retries almost immediately but a long
    // gateway outage doesn't hammer it every few seconds. A session that
    // actually connected and ran for a while (>= RESET_AFTER) resets the
    // backoff, so only genuinely-failing reconnects escalate -- a normal
    // heartbeat session that ends is not a "failure" to back off from.
    const BASE: Duration = Duration::from_secs(2);
    const MAX: Duration = Duration::from_secs(60);
    const RESET_AFTER: Duration = Duration::from_secs(10);
    let mut backoff = BASE;
    // One metrics collector for the agent's whole life (not per session): CPU and
    // network are deltas since the last sample, so it must persist across
    // reconnects to keep producing real averages. Both hosts and containers
    // report metrics -- the collector is here, above the host/container split.
    let mut metrics = metrics::Collector::new();
    loop {
        let started = Instant::now();
        match run_session(&identity.gateway_addr, host_only, tls_config.clone(), logq.as_ref(), &mut metrics) {
            Ok(()) => {}
            Err(e) => eprintln!("laforge-agent: session ended: {e}"),
        }
        if started.elapsed() >= RESET_AFTER {
            backoff = BASE; // it was really connected; next reconnect starts fast again
        } else {
            backoff = std::cmp::min(backoff * 2, MAX);
        }
        // Jitter (0-511ms, from chaff::perturb) is added on top so many hosts
        // don't reconnect in lockstep -- and so the per-team chaff module (see
        // chaff.rs/build.rs, item 4) keeps one genuine runtime use and can't be
        // dead-code-eliminated, the same way every chaff_N function it calls
        // needs a real caller to survive LTO.
        let jitter = Duration::from_millis(chaff::perturb(obfuscate::runtime_seed()) & 0x1ff);
        std::thread::sleep(backoff + jitter);
    }
}

// supervise_app starts the image's original command (cmd, the captured OCI
// entrypoint) as a child and waits on it, exiting the container with the app's
// status when it exits. cmd is run through `sh -c` so a shell-quoted OCI
// entrypoint works as-is; an image with no shell is a documented follow-up
// (parse the argv instead). Incus provides the container's real PID 1 and reaps
// orphans itself, so the agent only has to wait on its own child -- no libc/init
// duties needed. Never returns.
//
// The child's stdout/stderr are piped so the agent can capture the app's console
// for forwarding (logq). Each line is TEED: re-printed to the agent's own
// stdout/stderr (which is the container console, so `docker logs`/`incus
// console` still work live) AND pushed onto the bounded queue the gateway loop
// ships. Capture is best-effort: a read error on a stream just ends that
// stream's tee, never the app.
fn supervise_app(cmd: String, logq: Arc<applog::LogQueue>) -> ! {
    use std::process::Stdio;
    let mut child = match std::process::Command::new("sh")
        .arg("-c")
        .arg(&cmd)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
    {
        Ok(c) => c,
        Err(e) => {
            eprintln!("laforge-agent: supervisor: failed to start app command {cmd:?}: {e}");
            std::process::exit(1);
        }
    };
    if let Some(out) = child.stdout.take() {
        let q = logq.clone();
        std::thread::spawn(move || tee_stream(out, "stdout", q));
    }
    if let Some(err) = child.stderr.take() {
        let q = logq.clone();
        std::thread::spawn(move || tee_stream(err, "stderr", q));
    }
    let status = child.wait().unwrap_or_else(|e| {
        eprintln!("laforge-agent: supervisor: waiting on app: {e}");
        std::process::exit(1);
    });
    let code = status.code().unwrap_or(1);
    eprintln!("laforge-agent: supervised app exited (code {code}); stopping container");
    std::process::exit(code);
}

// tee_stream reads one of the app's console streams line by line, echoes each
// line to the agent's matching console stream (preserving the container's live
// console), and pushes it onto the log queue for the gateway loop to ship.
// Reads as UTF-8 lines; non-UTF-8 console output is a documented follow-up
// (read raw bytes instead). Returns when the stream closes (app exit) or on a
// read error.
fn tee_stream<R: Read>(r: R, stream: &'static str, logq: Arc<applog::LogQueue>) {
    let reader = BufReader::new(r);
    for line in reader.lines() {
        let line = match line {
            Ok(l) => l,
            Err(_) => break,
        };
        if stream == "stdout" {
            println!("{line}");
        } else {
            eprintln!("{line}");
        }
        logq.push(stream, line);
    }
}

fn build_tls_config(identity: &identity::Identity) -> Result<ClientConfig, String> {
    let mut roots = RootCertStore::empty();
    for cert in rustls_pemfile::certs(&mut identity.ca_pem.as_slice()) {
        let cert = cert.map_err(|e| format!("parsing CA cert: {e}"))?;
        roots.add(cert).map_err(|e| format!("adding CA cert to root store: {e}"))?;
    }

    let certs: Vec<CertificateDer<'static>> = rustls_pemfile::certs(&mut identity.cert_pem.as_slice())
        .collect::<Result<_, _>>()
        .map_err(|e| format!("parsing client cert: {e}"))?;
    let key: PrivateKeyDer<'static> = rustls_pemfile::private_key(&mut identity.key_pem.as_slice())
        .map_err(|e| format!("parsing client key: {e}"))?
        .ok_or_else(|| "no private key found in client key PEM".to_string())?;

    ClientConfig::builder()
        .with_root_certificates(roots)
        .with_client_auth_cert(certs, key)
        .map_err(|e| format!("building client config: {e}"))
}

fn run_session(addr: &str, host_only: &str, tls_config: Arc<ClientConfig>, logq: Option<&Arc<applog::LogQueue>>, metrics: &mut metrics::Collector) -> io::Result<()> {
    let tcp = TcpStream::connect(addr)?;
    tcp.set_nodelay(true).ok();

    let server_name = ServerName::try_from(host_only.to_string())
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, format!("invalid server name {host_only:?}: {e}")))?;
    let conn = ClientConnection::new(tls_config, server_name).map_err(|e| io::Error::other(format!("starting TLS: {e}")))?;
    let mut tls = StreamOwned::new(conn, tcp);

    loop {
        // Sample basic host metrics and carry them on the heartbeat. Best-effort:
        // if serialization somehow fails, send an empty heartbeat rather than
        // dropping the check-in.
        let hb_body = serde_json::to_vec(&metrics.sample()).unwrap_or_default();
        protocol::write_frame(&mut tls, MessageType::HeartbeatRequest, &hb_body)?;
        let (mt, body) = protocol::read_frame(&mut tls)?;
        expect(mt, MessageType::HeartbeatResponse)?;
        let hb: HeartbeatResponsePayload = serde_json::from_slice(&body)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("decoding heartbeat response: {e}")))?;

        protocol::write_frame(&mut tls, MessageType::GetTaskRequest, &[])?;
        let (mt, body) = protocol::read_frame(&mut tls)?;
        expect(mt, MessageType::GetTaskResponse)?;
        let gt: GetTaskResponsePayload = serde_json::from_slice(&body)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("decoding get-task response: {e}")))?;

        // Ship any buffered console lines every iteration, task or not, so logs
        // keep flowing at the poll cadence rather than only when idle.
        flush_logs(&mut tls, logq)?;

        if let Some(task) = gt.task {
            eprintln!("laforge-agent: running task {} ({})", task.id, task.command);
            let outcome = if task.command == "validate" {
                validators::run(&task.payload)
            } else {
                commands::run(&task.command, &task.payload)
            };
            eprintln!("laforge-agent: task {} -> {}", task.id, outcome.status);

            let report = ReportStatusRequestPayload {
                task_id: task.id,
                status: outcome.status.to_string(),
                output: outcome.output,
                error: outcome.error,
                validator_results: outcome.validator_results,
            };
            let body = serde_json::to_vec(&report)
                .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("encoding report-status: {e}")))?;
            protocol::write_frame(&mut tls, MessageType::ReportStatusRequest, &body)?;
            let (mt, _body) = protocol::read_frame(&mut tls)?;
            expect(mt, MessageType::ReportStatusResponse)?;
            // Immediately check for the next step -- no reason to wait
            // out the poll interval while there's known work queued.
            continue;
        }

        std::thread::sleep(Duration::from_millis(hb.next_poll_ms));
    }
}

// flush_logs drains buffered console lines and ships them as one LogBatch,
// in-band on the same mTLS session (request/response, like every other verb).
// A no-op when there is no queue (a host) or nothing buffered. The gateway acks
// and the agent reads that ack to keep the stream framed; the records
// themselves are fire-and-ack -- the agent does not resend on a non-ok.
fn flush_logs<S: Read + Write>(tls: &mut S, logq: Option<&Arc<applog::LogQueue>>) -> io::Result<()> {
    let q = match logq {
        Some(q) => q,
        None => return Ok(()),
    };
    let records = q.drain_batch(LOG_BATCH_MAX);
    if records.is_empty() {
        return Ok(());
    }
    let body = serde_json::to_vec(&LogBatchRequestPayload { records })
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("encoding log batch: {e}")))?;
    protocol::write_frame(tls, MessageType::LogBatchRequest, &body)?;
    let (mt, _body) = protocol::read_frame(tls)?;
    expect(mt, MessageType::LogBatchResponse)?;
    Ok(())
}

fn expect(got: MessageType, want: MessageType) -> io::Result<()> {
    if got != want {
        return Err(io::Error::new(io::ErrorKind::InvalidData, format!("got message type {got:?}, want {want:?}")));
    }
    Ok(())
}
