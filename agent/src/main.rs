// laforge-agent: "a single fully-embedded binary with no external
// configuration... opens a TLS connection to the gateway... and presents
// its client certificate; the gateway presents a certificate the agent
// pinned at build time." This file is only ever the connection loop --
// framing lives in protocol.rs, identity in identity.rs, command
// execution in commands.rs and validators.rs.

mod antitamper;
mod chaff;
mod commands;
mod identity;
mod obfuscate;
mod protocol;
mod selfhash;
mod validators;

use protocol::{GetTaskResponsePayload, HeartbeatResponsePayload, MessageType, ReportStatusRequestPayload};
use rustls::pki_types::{CertificateDer, PrivateKeyDer, ServerName};
use rustls::{ClientConfig, ClientConnection, RootCertStore, StreamOwned};
use std::io;
use std::net::TcpStream;
use std::sync::Arc;
use std::time::{Duration, Instant};

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
        Some((identity, tls_config, host_only)) => agent_loop(&identity, &host_only, tls_config),
        None => std::process::exit(1),
    }
}

// run_container starts the gateway loop in the background (if the agent can
// connect) and then supervises the image's real command. The command runs
// regardless of the gateway -- the container's service must be up even if agent
// management can't connect -- so a broken/absent identity never takes the
// service down, it only means no check-in. load_agent logs why, if it can't.
fn run_container(cmd: String) -> ! {
    if let Some((identity, tls_config, host_only)) = load_agent() {
        std::thread::spawn(move || agent_loop(&identity, &host_only, tls_config));
    }
    supervise_app(cmd)
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
fn agent_loop(identity: &identity::Identity, host_only: &str, tls_config: Arc<ClientConfig>) -> ! {
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
    loop {
        let started = Instant::now();
        match run_session(&identity.gateway_addr, host_only, tls_config.clone()) {
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
fn supervise_app(cmd: String) -> ! {
    let mut child = match std::process::Command::new("sh").arg("-c").arg(&cmd).spawn() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("laforge-agent: supervisor: failed to start app command {cmd:?}: {e}");
            std::process::exit(1);
        }
    };
    let status = child.wait().unwrap_or_else(|e| {
        eprintln!("laforge-agent: supervisor: waiting on app: {e}");
        std::process::exit(1);
    });
    let code = status.code().unwrap_or(1);
    eprintln!("laforge-agent: supervised app exited (code {code}); stopping container");
    std::process::exit(code);
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

fn run_session(addr: &str, host_only: &str, tls_config: Arc<ClientConfig>) -> io::Result<()> {
    let tcp = TcpStream::connect(addr)?;
    tcp.set_nodelay(true).ok();

    let server_name = ServerName::try_from(host_only.to_string())
        .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, format!("invalid server name {host_only:?}: {e}")))?;
    let conn = ClientConnection::new(tls_config, server_name).map_err(|e| io::Error::other(format!("starting TLS: {e}")))?;
    let mut tls = StreamOwned::new(conn, tcp);

    loop {
        protocol::write_frame(&mut tls, MessageType::HeartbeatRequest, &[])?;
        let (mt, body) = protocol::read_frame(&mut tls)?;
        expect(mt, MessageType::HeartbeatResponse)?;
        let hb: HeartbeatResponsePayload = serde_json::from_slice(&body)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("decoding heartbeat response: {e}")))?;

        protocol::write_frame(&mut tls, MessageType::GetTaskRequest, &[])?;
        let (mt, body) = protocol::read_frame(&mut tls)?;
        expect(mt, MessageType::GetTaskResponse)?;
        let gt: GetTaskResponsePayload = serde_json::from_slice(&body)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, format!("decoding get-task response: {e}")))?;

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

fn expect(got: MessageType, want: MessageType) -> io::Result<()> {
    if got != want {
        return Err(io::Error::new(io::ErrorKind::InvalidData, format!("got message type {got:?}, want {want:?}")));
    }
    Ok(())
}
