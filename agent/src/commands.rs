// The agent's command set -- "Ansible-local is dropped... every command
// takes structured arguments... reports status, output, and an error
// separately." Scoped to Linux: user/group/service management shell out to
// real Linux tools (useradd, chpasswd, usermod, systemctl), which exist
// on the real target OS but not on this development machine -- covered by
// unit tests for payload parsing and idempotent-exit-code handling, not
// by literally creating a Linux user from this repo's test suite.
//
// `download` and `extract` are real: download does an HTTP(S) GET of an
// arbitrary URL (files.cp.tc, IP-limited to the builder environments, or
// anywhere on the internet) to a path on the host; extract unpacks a
// zip/tar/tgz already on the host. `upload` is still a clear "not implemented"
// failure -- sending a file BACK to LaForge needs a file-store service that
// doesn't exist yet (a deliberately larger, separately-planned project), so it
// fails loudly rather than pretending. Reboot is genuinely wired to a real
// reboot command (`shutdown -r`) -- deliberately never invoked by anything in
// this repository's automated tests, for the obvious reason that it would
// reboot whatever machine ran them.

use serde_json::Value;
use std::fs;
use std::io::{Read, Write as _};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{mpsc, Arc, Mutex, OnceLock};
use std::time::{Duration, Instant};

/// Outcome is (status, output, error) exactly matching
/// agentproto.ReportStatusRequestPayload's three fields -- run() always
/// returns one of these, never panics, for any payload shape including a
/// deliberately malformed one (see the malformed-payload tests below).
pub struct Outcome {
    pub status: &'static str, // "done" | "failed" | "timeout" (execute only)
    pub output: String,
    pub error: String,
    /// Only ever non-empty for a `validate` task (see validators::run) --
    /// every other command's own Outcome literal below sets this to an
    /// empty Vec, since only validate produces per-check structured
    /// results at all.
    pub validator_results: Vec<crate::protocol::ValidatorResultPayload>,
}

type Handler = fn(&Value) -> Result<String, String>;

/// The command dispatch table: built once per process and cached, as a
/// permuted function-pointer jump table rather than a plain `match` --
/// see obfuscate.rs's own doc comment for exactly what this buys and
/// what it deliberately doesn't claim to be (not OLLVM, not control-flow
/// flattening). `run`'s observable behavior is unchanged either way --
/// every test below only asserts on Outcome, never on how dispatch got
/// there.
static DISPATCH: OnceLock<(u64, crate::obfuscate::JumpTable<Handler>)> = OnceLock::new();

fn dispatch_table() -> &'static (u64, crate::obfuscate::JumpTable<Handler>) {
    DISPATCH.get_or_init(|| {
        let seed = crate::obfuscate::runtime_seed();
        let entries: Vec<(&str, Handler)> = vec![
            ("execute", cmd_execute),
            ("write_file", cmd_write_file),
            ("append_file", cmd_append_file),
            ("delete", cmd_delete),
            ("change_perms", cmd_change_perms),
            ("create_user", cmd_create_user),
            ("set_password", cmd_set_password),
            ("add_to_group", cmd_add_to_group),
            ("service", cmd_service),
            ("reboot", cmd_reboot),
            ("disconnect_sessions", cmd_disconnect_sessions),
            ("download", cmd_download),
            ("extract", cmd_extract),
        ];
        (seed, crate::obfuscate::JumpTable::build(entries, seed))
    })
}

pub fn run(command: &str, payload: &Value) -> Outcome {
    let (seed, table) = dispatch_table();
    // opaque_true(*seed) always holds (see obfuscate::opaque_true's own
    // doc comment) -- this branch never changes which arm below actually
    // runs. It exists to give a disassembler a genuine extra condition
    // to resolve around the real dispatch, rather than one it can prove
    // dead at a glance. The `else` arm is real, compiled code -- not
    // `unreachable!()` -- for the same reason: an obviously-unreachable
    // marker would itself be a giveaway of which side is the decoy.
    let result: Result<String, String> = if crate::obfuscate::opaque_true(*seed) {
        match command {
            "upload" => Err(
                "upload is not implemented yet -- sending a file back to LaForge needs a file-store service that doesn't exist yet (separately planned)".to_string()
            ),
            other => match table.get(other, *seed) {
                Some(handler) => handler(payload),
                None => Err(format!("unknown command {other:?}")),
            },
        }
    } else {
        Err(format!("unknown command {command:?}"))
    };
    match result {
        Ok(output) => Outcome { status: "done", output, error: String::new(), validator_results: Vec::new() },
        Err(error) => {
            // A timed-out execute reports "timeout" so the gateway can treat it as
            // terminal (no retry); older gateways see any non-"done" as failed.
            let status = if command == "execute" && error.starts_with(&format!("{} {TIMED_OUT}", opt_str(payload, "command"))) {
                "timeout"
            } else {
                "failed"
            };
            Outcome { status, output: String::new(), error, validator_results: Vec::new() }
        }
    }
}

/// Follows the command name in a timed-out execute's error; run() keys the
/// "timeout" status off it, and the gateway falls back to matching it.
const TIMED_OUT: &str = "timed out after ";

fn req_str<'a>(payload: &'a Value, field: &str) -> Result<&'a str, String> {
    payload.get(field).and_then(Value::as_str).ok_or_else(|| format!("missing or non-string field {field:?}"))
}

fn opt_str<'a>(payload: &'a Value, field: &str) -> &'a str {
    payload.get(field).and_then(Value::as_str).unwrap_or("")
}

fn cmd_execute(payload: &Value) -> Result<String, String> {
    let command = req_str(payload, "command")?;
    let args: Vec<String> = payload
        .get("args")
        .and_then(Value::as_array)
        .map(|a| a.iter().filter_map(|v| v.as_str().map(String::from)).collect())
        .unwrap_or_default();
    let mut cmd = Command::new(command);
    cmd.args(&args);
    if let Some(dir) = payload.get("working_dir").and_then(Value::as_str) {
        if !dir.is_empty() {
            cmd.current_dir(dir);
        }
    }
    // The gateway sends the step's timeout_sec; absent or 0 means no deadline.
    let timeout_sec = payload.get("timeout_sec").and_then(Value::as_u64).unwrap_or(0);
    let timeout = (timeout_sec > 0).then(|| Duration::from_secs(timeout_sec));
    let out = run_captured(cmd, timeout, PIPE_DRAIN_GRACE).map_err(|e| format!("spawning {command}: {e}"))?;
    match out.status {
        Some(status) if status.success() => Ok(String::from_utf8_lossy(&out.stdout).to_string()),
        Some(status) => Err(format!("{command} exited {status}: {}", failure_detail(&out.stdout, &out.stderr))),
        None => Err(format!(
            "{command} {TIMED_OUT}{timeout_sec}s and was killed along with its child processes; output before the kill: {}",
            failure_detail(&out.stdout, &out.stderr)
        )),
    }
}

// --- run_captured: execute with a deadline and a whole-tree kill ----------
//
// Command::output() waits forever, so a hung script used to hold its host's
// task lane indefinitely regardless of the step's timeout. run_captured spawns
// the child into its own killable unit -- a process group on Unix, a Job Object
// on Windows -- drains stdout/stderr on threads, and polls for exit against the
// deadline. On expiry the WHOLE tree is killed, not just the direct child: a
// shell wrapper killed alone leaves its grandchildren running and holding the
// output pipes open.
//
// Escape hatches remain: a Unix grandchild that calls setsid() leaves the
// group, and on Windows a grandchild started in the instant between spawn and
// job assignment isn't in the job. Both are far outside what a step script does
// in practice.

/// How long to keep reading a command's output after it exits (or is killed).
/// A script that backgrounds a daemon without redirecting it leaves that daemon
/// holding the pipe -- Command::output() would wait on it forever. After the
/// grace the capture keeps draining (so the daemon never blocks on a full pipe)
/// but discards, and the step reports what it had.
const PIPE_DRAIN_GRACE: Duration = Duration::from_secs(5);

const WAIT_POLL: Duration = Duration::from_millis(50);

struct Captured {
    /// None when the deadline expired and the tree was killed.
    status: Option<ExitStatus>,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
}

fn run_captured(mut cmd: Command, timeout: Option<Duration>, drain_grace: Duration) -> std::io::Result<Captured> {
    cmd.stdin(Stdio::null()).stdout(Stdio::piped()).stderr(Stdio::piped());
    let mut tree = ProcessTree::spawn(cmd)?;
    let stdout = Capture::start(tree.child.stdout.take());
    let stderr = Capture::start(tree.child.stderr.take());
    let deadline = timeout.map(|t| Instant::now() + t);
    let status = loop {
        match tree.child.try_wait() {
            Ok(Some(status)) => break Some(status),
            Ok(None) => {}
            Err(e) => {
                tree.kill();
                return Err(e);
            }
        }
        let mut nap = WAIT_POLL;
        if let Some(d) = deadline {
            let left = d.saturating_duration_since(Instant::now());
            if left.is_zero() {
                tree.kill();
                break None;
            }
            nap = nap.min(left);
        }
        std::thread::sleep(nap);
    };
    let drain_until = Instant::now() + drain_grace;
    Ok(Captured { status, stdout: stdout.finish(drain_until), stderr: stderr.finish(drain_until) })
}

/// Capture drains one output pipe on its own thread into a shared buffer, so
/// the result can be taken at a deadline even if the pipe never reaches EOF.
struct Capture {
    buf: Arc<Mutex<Vec<u8>>>,
    detached: Arc<AtomicBool>,
    done: mpsc::Receiver<()>,
}

impl Capture {
    fn start<R: Read + Send + 'static>(src: Option<R>) -> Capture {
        let buf = Arc::new(Mutex::new(Vec::new()));
        let detached = Arc::new(AtomicBool::new(false));
        let (tx, done) = mpsc::channel();
        if let Some(mut src) = src {
            let (buf, detached) = (buf.clone(), detached.clone());
            std::thread::spawn(move || {
                let mut chunk = [0u8; 8192];
                loop {
                    match src.read(&mut chunk) {
                        Ok(0) | Err(_) => break,
                        Ok(n) if !detached.load(Ordering::Relaxed) => {
                            buf.lock().unwrap_or_else(|e| e.into_inner()).extend_from_slice(&chunk[..n])
                        }
                        Ok(_) => {}
                    }
                }
                let _ = tx.send(());
            });
        }
        Capture { buf, detached, done }
    }

    /// Waits for EOF until `until`, then takes whatever was read. The channel
    /// also disconnects (returning at once) when there was no pipe to read.
    fn finish(self, until: Instant) -> Vec<u8> {
        let _ = self.done.recv_timeout(until.saturating_duration_since(Instant::now()));
        self.detached.store(true, Ordering::Relaxed);
        std::mem::take(&mut *self.buf.lock().unwrap_or_else(|e| e.into_inner()))
    }
}

/// ProcessTree is a spawned child plus the handle that kills everything it
/// started: its process group on Unix, its Job Object on Windows.
struct ProcessTree {
    child: Child,
    #[cfg(windows)]
    job: Option<job::Job>,
}

#[cfg(unix)]
impl ProcessTree {
    fn spawn(mut cmd: Command) -> std::io::Result<ProcessTree> {
        use std::os::unix::process::CommandExt;
        cmd.process_group(0); // the child leads a new group whose id is its pid
        Ok(ProcessTree { child: cmd.spawn()? })
    }

    fn kill(&mut self) {
        extern "C" {
            fn kill(pid: i32, sig: i32) -> i32;
        }
        const SIGKILL: i32 = 9;
        // A negative pid signals the whole process group.
        unsafe {
            kill(-(self.child.id() as i32), SIGKILL);
        }
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

#[cfg(windows)]
impl ProcessTree {
    fn spawn(mut cmd: Command) -> std::io::Result<ProcessTree> {
        use std::os::windows::io::AsRawHandle;
        let child = cmd.spawn()?;
        // Without a job, a timeout still kills the direct child; only the tree
        // kill degrades. Nested jobs need Windows 8 / Server 2012 or later.
        let job = job::Job::new().and_then(|j| j.assign(child.as_raw_handle()).map(|_| j));
        let job = match job {
            Ok(j) => Some(j),
            Err(e) => {
                crate::diag::write_line(&format!("laforge-agent: execute: no job object for pid {} ({e}); timeout will kill only the direct child", child.id()));
                None
            }
        };
        Ok(ProcessTree { child, job })
    }

    fn kill(&mut self) {
        if let Some(job) = &self.job {
            job.terminate();
        }
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

/// Minimal Job Object FFI, declared directly (like antitamper.rs and shell.rs)
/// rather than pulling in a Windows API crate. The job is created with
/// KILL_ON_JOB_CLOSE so the tree also dies if the agent itself does; on a
/// normal exit Drop clears that flag before closing, so a daemon a script
/// legitimately started outlives the step.
#[cfg(windows)]
mod job {
    use std::os::raw::c_void;

    type Handle = *mut c_void;

    #[repr(C)]
    #[derive(Default)]
    struct BasicLimitInformation {
        per_process_user_time_limit: i64,
        per_job_user_time_limit: i64,
        limit_flags: u32,
        minimum_working_set_size: usize,
        maximum_working_set_size: usize,
        active_process_limit: u32,
        affinity: usize,
        priority_class: u32,
        scheduling_class: u32,
    }

    #[repr(C)]
    #[derive(Default)]
    struct IoCounters {
        read_operation_count: u64,
        write_operation_count: u64,
        other_operation_count: u64,
        read_transfer_count: u64,
        write_transfer_count: u64,
        other_transfer_count: u64,
    }

    #[repr(C)]
    #[derive(Default)]
    struct ExtendedLimitInformation {
        basic: BasicLimitInformation,
        io: IoCounters,
        process_memory_limit: usize,
        job_memory_limit: usize,
        peak_process_memory_used: usize,
        peak_job_memory_used: usize,
    }

    const JOB_OBJECT_EXTENDED_LIMIT_INFORMATION: i32 = 9;
    const JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE: u32 = 0x2000;

    #[link(name = "kernel32")]
    extern "system" {
        fn CreateJobObjectW(attributes: *mut c_void, name: *const u16) -> Handle;
        fn SetInformationJobObject(job: Handle, class: i32, info: *mut c_void, len: u32) -> i32;
        fn AssignProcessToJobObject(job: Handle, process: Handle) -> i32;
        fn TerminateJobObject(job: Handle, exit_code: u32) -> i32;
        fn CloseHandle(handle: Handle) -> i32;
    }

    pub struct Job(Handle);

    impl Job {
        pub fn new() -> std::io::Result<Job> {
            let h = unsafe { CreateJobObjectW(std::ptr::null_mut(), std::ptr::null()) };
            if h.is_null() {
                return Err(std::io::Error::last_os_error());
            }
            let job = Job(h);
            job.set_limit_flags(JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)?;
            Ok(job)
        }

        pub fn assign(&self, process: std::os::windows::io::RawHandle) -> std::io::Result<()> {
            if unsafe { AssignProcessToJobObject(self.0, process as Handle) } == 0 {
                return Err(std::io::Error::last_os_error());
            }
            Ok(())
        }

        pub fn terminate(&self) {
            unsafe {
                TerminateJobObject(self.0, 1);
            }
        }

        fn set_limit_flags(&self, flags: u32) -> std::io::Result<()> {
            let mut info = ExtendedLimitInformation::default();
            info.basic.limit_flags = flags;
            let ok = unsafe {
                SetInformationJobObject(
                    self.0,
                    JOB_OBJECT_EXTENDED_LIMIT_INFORMATION,
                    &mut info as *mut _ as *mut c_void,
                    std::mem::size_of::<ExtendedLimitInformation>() as u32,
                )
            };
            if ok == 0 {
                return Err(std::io::Error::last_os_error());
            }
            Ok(())
        }
    }

    impl Drop for Job {
        fn drop(&mut self) {
            let _ = self.set_limit_flags(0);
            unsafe {
                CloseHandle(self.0);
            }
        }
    }
}

// failure_detail builds the diagnostic for a failed command from BOTH its stderr
// and stdout -- many tools (docker compose among them, and plain shell scripts)
// write their real error to stdout, so stderr alone comes back empty and a step
// failure reads as a bare "exited 1:" with nothing useful. Capped to a tail so a
// noisy command can't bloat the event payload (the error is almost always last).
fn failure_detail(stdout: &[u8], stderr: &[u8]) -> String {
    let mut parts: Vec<String> = Vec::new();
    let e = String::from_utf8_lossy(stderr);
    if !e.trim().is_empty() {
        parts.push(e.trim_end().to_string());
    }
    let o = String::from_utf8_lossy(stdout);
    if !o.trim().is_empty() {
        parts.push(o.trim_end().to_string());
    }
    let joined = if parts.is_empty() { "(no output)".to_string() } else { parts.join("\n") };
    tail(&joined, 16_000)
}

// tail keeps the last `max` bytes of s (on a char boundary), prefixing a marker
// when it had to cut -- a command's failure is almost always at the end.
fn tail(s: &str, max: usize) -> String {
    if s.len() <= max {
        return s.to_string();
    }
    let cut = s.len() - max;
    let start = (cut..s.len()).find(|&i| s.is_char_boundary(i)).unwrap_or(s.len());
    format!("…(truncated)…\n{}", &s[start..])
}

fn cmd_write_file(payload: &Value) -> Result<String, String> {
    let path = req_str(payload, "path")?;
    let content = opt_str(payload, "content");
    fs::write(path, content).map_err(|e| format!("writing {path}: {e}"))?;
    let mode = opt_str(payload, "mode");
    if !mode.is_empty() {
        set_mode(path, mode)?;
    }
    Ok(format!("wrote {} bytes to {path}", content.len()))
}

fn cmd_append_file(payload: &Value) -> Result<String, String> {
    let path = req_str(payload, "path")?;
    let content = opt_str(payload, "content");
    let mut f = fs::OpenOptions::new().create(true).append(true).open(path).map_err(|e| format!("opening {path}: {e}"))?;
    f.write_all(content.as_bytes()).map_err(|e| format!("appending to {path}: {e}"))?;
    Ok(format!("appended {} bytes to {path}", content.len()))
}

/// download fetches `from` (an http/https URL -- files.cp.tc or anywhere on the
/// internet) and streams it to `to` on the host. Any parent directories of `to`
/// are created. Streamed via io::copy so a large artifact never has to fit in
/// memory. A non-2xx response, an unreachable host, or an unwritable
/// destination is a real, reported failure.
fn cmd_download(payload: &Value) -> Result<String, String> {
    let from = req_str(payload, "from")?;
    let to = req_str(payload, "to")?;
    let agent = ureq::AgentBuilder::new()
        .timeout_connect(std::time::Duration::from_secs(30))
        .build();
    let resp = agent.get(from).call().map_err(|e| format!("downloading {from}: {e}"))?;
    if let Some(parent) = std::path::Path::new(to).parent() {
        if !parent.as_os_str().is_empty() {
            fs::create_dir_all(parent).map_err(|e| format!("creating {}: {e}", parent.display()))?;
        }
    }
    let mut reader = resp.into_reader();
    let mut file = fs::File::create(to).map_err(|e| format!("creating {to}: {e}"))?;
    let n = std::io::copy(&mut reader, &mut file).map_err(|e| format!("writing {to}: {e}"))?;
    Ok(format!("downloaded {n} bytes from {from} to {to}"))
}

/// extract unpacks an archive already on the host (`src`) into `dest`, creating
/// `dest` if needed. Type is chosen by `src`'s extension: .zip, .tar, or
/// .tar.gz/.tgz -- exactly what the schema documents. Both underlying crates
/// sanitize member paths, so a malicious "../" entry can't escape `dest`.
fn cmd_extract(payload: &Value) -> Result<String, String> {
    let src = req_str(payload, "src")?;
    let dest = req_str(payload, "dest")?;
    fs::create_dir_all(dest).map_err(|e| format!("creating {dest}: {e}"))?;
    let file = fs::File::open(src).map_err(|e| format!("opening {src}: {e}"))?;
    let lower = src.to_ascii_lowercase();
    if lower.ends_with(".zip") {
        let mut archive = zip::ZipArchive::new(file).map_err(|e| format!("reading zip {src}: {e}"))?;
        archive.extract(dest).map_err(|e| format!("extracting zip {src} to {dest}: {e}"))?;
    } else if lower.ends_with(".tar.gz") || lower.ends_with(".tgz") {
        let gz = flate2::read::GzDecoder::new(file);
        tar::Archive::new(gz).unpack(dest).map_err(|e| format!("extracting {src} to {dest}: {e}"))?;
    } else if lower.ends_with(".tar") {
        tar::Archive::new(file).unpack(dest).map_err(|e| format!("extracting {src} to {dest}: {e}"))?;
    } else {
        return Err(format!("unsupported archive type for {src} (expected .zip, .tar, .tar.gz, or .tgz)"));
    }
    Ok(format!("extracted {src} to {dest}"))
}

fn cmd_delete(payload: &Value) -> Result<String, String> {
    let path = req_str(payload, "path")?;
    match fs::metadata(path) {
        Ok(m) if m.is_dir() => fs::remove_dir_all(path).map_err(|e| e.to_string())?,
        Ok(_) => fs::remove_file(path).map_err(|e| e.to_string())?,
        // Idempotent: "safe to call more than once" -- deleting something
        // already gone is success, not an error.
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(format!("{path} already absent")),
        Err(e) => return Err(e.to_string()),
    }
    Ok(format!("deleted {path}"))
}

// --- set_mode: Unix chmod, Windows ACLs ---------------------------------
//
// The same authored `change_perms` step runs on both. On Unix it's an
// ordinary chmod. On Windows there is no mode bit, so the octal is mapped
// onto ACLs the closest honest way: the owner triad grants OWNER RIGHTS,
// the group triad grants the built-in Users group, and the other triad
// grants Everyone -- addressed by well-known SID so it works regardless of
// the system's language. Administrators and SYSTEM always keep Full so a
// step can't lock the machine (or this agent) out of a file it just
// touched. rwx maps to icacls simple rights: execute implies read (RX),
// otherwise read is R, and write adds W.
#[cfg(unix)]
fn set_mode(path: &str, mode: &str) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    let m = u32::from_str_radix(mode.trim_start_matches('0'), 8).map_err(|e| format!("parsing mode {mode:?}: {e}"))?;
    fs::set_permissions(path, fs::Permissions::from_mode(m)).map_err(|e| format!("chmod {path}: {e}"))
}
#[cfg(windows)]
fn set_mode(path: &str, mode: &str) -> Result<(), String> {
    let m = u32::from_str_radix(mode.trim_start_matches('0'), 8).map_err(|e| format!("parsing mode {mode:?}: {e}"))?;
    let triad_rights = |bits: u32| -> Option<String> {
        let mut rights: Vec<&str> = Vec::new();
        if bits & 0o1 != 0 {
            rights.push("RX"); // execute implies read
        } else if bits & 0o4 != 0 {
            rights.push("R");
        }
        if bits & 0o2 != 0 {
            rights.push("W");
        }
        if rights.is_empty() {
            None
        } else {
            Some(rights.join(","))
        }
    };
    // Reset inherited ACLs so the mapped ones are the whole story.
    run_ok("icacls", &[path, "/inheritance:r"])?;
    // Never lock the box (or ourselves) out.
    run_ok("icacls", &[path, "/grant:r", "*S-1-5-32-544:(F)", "/grant:r", "*S-1-5-18:(F)"])?;
    for (bits, sid) in [((m >> 6) & 7, "*S-1-3-4"), ((m >> 3) & 7, "*S-1-5-32-545"), (m & 7, "*S-1-1-0")] {
        if let Some(rights) = triad_rights(bits) {
            run_ok("icacls", &[path, "/grant:r", &format!("{sid}:({rights})")])?;
        }
    }
    Ok(())
}

// --- set_owner ----------------------------------------------------------
#[cfg(unix)]
fn set_owner(path: &str, owner: &str, group: &str) -> Result<(), String> {
    let spec = if group.is_empty() { owner.to_string() } else { format!("{owner}:{group}") };
    run_ok("chown", &[&spec, path])
}
#[cfg(windows)]
fn set_owner(path: &str, owner: &str, _group: &str) -> Result<(), String> {
    // Windows files have an owner but no separate group owner; only the
    // owner is set (icacls /setowner).
    run_ok("icacls", &[path, "/setowner", owner])
}

fn cmd_change_perms(payload: &Value) -> Result<String, String> {
    let path = req_str(payload, "path")?;
    let mode = opt_str(payload, "mode");
    if !mode.is_empty() {
        set_mode(path, mode)?;
    }
    let owner = opt_str(payload, "owner");
    if !owner.is_empty() {
        set_owner(path, owner, opt_str(payload, "group"))?;
    }
    Ok(format!("changed perms on {path}"))
}

// --- create_user: useradd / net user -----------------------------------
#[cfg(unix)]
fn create_user_impl(username: &str, groups: &[String]) -> Result<(), String> {
    let mut args = vec!["-m".to_string(), "-s".to_string(), "/bin/bash".to_string()];
    if !groups.is_empty() {
        args.push("-G".to_string());
        args.push(groups.join(","));
    }
    args.push(username.to_string());
    let out = Command::new("useradd").args(&args).output().map_err(|e| format!("spawning useradd: {e}"))?;
    // "Idempotent where it can be": useradd exits 9 for "user already
    // exists" -- a re-run over an existing user succeeds, per the plan's
    // own rule, rather than failing the whole step.
    if !out.status.success() && out.status.code() != Some(9) {
        return Err(format!("useradd exited {}: {}", out.status, String::from_utf8_lossy(&out.stderr)));
    }
    Ok(())
}
#[cfg(windows)]
fn create_user_impl(username: &str, groups: &[String]) -> Result<(), String> {
    // `net user <name> /add`. Exit 2 ("account already exists", NET
    // HELPMSG 2224) is the idempotent case, matching useradd's own exit-9
    // tolerance above.
    run_codes("net", &["user", username, "/add"], &[2])?;
    for g in groups {
        add_to_local_group(username, g)?;
    }
    Ok(())
}

fn cmd_create_user(payload: &Value) -> Result<String, String> {
    let username = req_str(payload, "username")?;
    let groups: Vec<String> = payload
        .get("groups")
        .and_then(Value::as_array)
        .map(|a| a.iter().filter_map(|v| v.as_str().map(String::from)).collect())
        .unwrap_or_default();
    create_user_impl(username, &groups)?;
    let password = opt_str(payload, "password");
    if !password.is_empty() {
        set_password_for(username, password)?;
    }
    Ok(format!("created user {username}"))
}

// --- set_password: chpasswd / net user <name> <password> ----------------
#[cfg(unix)]
fn set_password_for(username: &str, password: &str) -> Result<(), String> {
    let mut child = Command::new("chpasswd").stdin(Stdio::piped()).spawn().map_err(|e| format!("spawning chpasswd: {e}"))?;
    child
        .stdin
        .as_mut()
        .expect("stdin was requested as piped")
        .write_all(format!("{username}:{password}\n").as_bytes())
        .map_err(|e| format!("writing to chpasswd: {e}"))?;
    let status = child.wait().map_err(|e| e.to_string())?;
    if !status.success() {
        return Err(format!("chpasswd exited {status}"));
    }
    Ok(())
}
#[cfg(windows)]
fn set_password_for(username: &str, password: &str) -> Result<(), String> {
    run_ok("net", &["user", username, password])?;
    // Clear "user must change password at next logon". A freshly-created local
    // account (net user /add) keeps that flag set even after its password is
    // assigned above, which blocks the very login the password was set up for.
    // Setting the account's PasswordExpired to 0 via the WinNT ADSI provider is
    // the reliable, wmic-free way to drop the requirement. Single-quoted ADSI
    // path so there are no embedded double quotes to escape through argv.
    let ps = format!("$u=[ADSI]('WinNT://./{username},user'); $u.PasswordExpired=0; $u.SetInfo()");
    run_ok("powershell", &["-NoProfile", "-NonInteractive", "-Command", &ps])
}

fn cmd_set_password(payload: &Value) -> Result<String, String> {
    let username = req_str(payload, "username")?;
    let password = req_str(payload, "password")?;
    set_password_for(username, password)?;
    Ok(format!("password set for {username}"))
}

// --- add_to_group: usermod / net localgroup -----------------------------
#[cfg(unix)]
fn add_to_local_group(username: &str, group: &str) -> Result<(), String> {
    run_ok("usermod", &["-aG", group, username])
}
#[cfg(windows)]
fn add_to_local_group(username: &str, group: &str) -> Result<(), String> {
    // Exit 2 ("already a member", NET HELPMSG 1378) is idempotent.
    run_codes("net", &["localgroup", group, username, "/add"], &[2])
}

fn cmd_add_to_group(payload: &Value) -> Result<String, String> {
    let username = req_str(payload, "username")?;
    let group = req_str(payload, "group")?;
    add_to_local_group(username, group)?;
    Ok(format!("added {username} to {group}"))
}

// --- service: systemctl / sc -------------------------------------------
#[cfg(unix)]
fn service_action(name: &str, action: &str) -> Result<(), String> {
    run_ok("systemctl", &[action, name])
}
#[cfg(windows)]
fn service_action(name: &str, action: &str) -> Result<(), String> {
    match action {
        // sc start exits 1056 when already running, sc stop exits 1062
        // ("service not started") -- both are the idempotent no-op case.
        "start" => run_codes("sc", &["start", name], &[1056]),
        "stop" => run_codes("sc", &["stop", name], &[1062]),
        "restart" => {
            let _ = run_codes("sc", &["stop", name], &[1062]);
            run_codes("sc", &["start", name], &[1056])
        }
        "enable" => run_ok("sc", &["config", name, "start=", "auto"]),
        "disable" => run_ok("sc", &["config", name, "start=", "disabled"]),
        other => Err(format!("unsupported service action {other:?} on Windows")),
    }
}

fn cmd_service(payload: &Value) -> Result<String, String> {
    let name = req_str(payload, "name")?;
    let action = req_str(payload, "action")?;
    service_action(name, action)?;
    Ok(format!("{action} {name}"))
}

/// cmd_reboot is wired to a real reboot command -- "the full command
/// set" means this one too, not a stub -- but is never called by
/// anything in this repository's tests or by any verification code path,
/// for the obvious reason that doing so would reboot whatever machine ran
/// it.
#[cfg(unix)]
fn cmd_reboot(payload: &Value) -> Result<String, String> {
    let delay = payload.get("delay_sec").and_then(Value::as_i64).unwrap_or(0).max(0);
    // Cancel any already-pending shutdown so a re-issue is idempotent; ignore the
    // error when none is pending.
    let _ = run_ok("shutdown", &["-c"]);
    run_ok("shutdown", &["-r", &format!("+{}", (delay + 59) / 60)])?;
    Ok(format!("reboot scheduled in {delay}s"))
}
#[cfg(windows)]
fn cmd_reboot(payload: &Value) -> Result<String, String> {
    let delay = payload.get("delay_sec").and_then(Value::as_i64).unwrap_or(0).max(0);
    // Abort any shutdown this box already has scheduled, so re-issuing is
    // idempotent -- otherwise `shutdown /r` fails with 1190 ("a system shutdown
    // has already been scheduled"). Ignore the error: it fails when nothing is
    // scheduled, which is the normal case.
    let _ = run_ok("shutdown", &["/a"]);
    // /f forces the reboot through. A non-forced reboot issued by the agent
    // (running as a LocalSystem service, session 0) was observed being DEFERRED
    // until an interactive login released it -- leaving the host hung mid-build.
    run_ok("shutdown", &["/r", "/t", &delay.to_string(), "/f"])?;
    Ok(format!("reboot scheduled in {delay}s (forced)"))
}

/// parse_rdp_session_ids pulls the numeric session IDs of connected RDP
/// sessions out of `query session` output. A connected RDP session's
/// SESSIONNAME is "rdp-tcp#<n>"; the bare "rdp-tcp" listener, "console" and
/// "services" rows are skipped, so only live remote sessions are returned.
/// USERNAME can be blank, which shifts columns, so the ID is taken as the first
/// purely-numeric token on the row rather than by position ("rdp-tcp#2" is not
/// purely numeric, so the name is never mistaken for the ID). Kept pure and
/// platform-independent so it can be unit-tested anywhere, not only on Windows.
#[cfg_attr(not(windows), allow(dead_code))]
fn parse_rdp_session_ids(text: &str) -> Vec<u32> {
    let mut ids: Vec<u32> = Vec::new();
    for line in text.lines() {
        // A leading '>' marks the current session; strip it before matching.
        let line = line.trim_start_matches('>').trim_start();
        let mut toks = line.split_whitespace();
        let name = match toks.next() {
            Some(n) => n,
            None => continue,
        };
        if !name.starts_with("rdp-tcp#") {
            continue;
        }
        if let Some(id) = toks.find_map(|t| t.parse::<u32>().ok()) {
            ids.push(id);
        }
    }
    ids
}

/// cmd_disconnect_sessions forcibly ends live interactive remote-access
/// sessions WITHOUT disturbing the agent's own outbound connection to the
/// gateway. The access reconciler fires this at a team on close, because the
/// network firewall only blocks NEW connections: an already-established RDP/SSH
/// session rides conntrack (on a bridge the published port is a nat-proxy whose
/// DNAT conntrack survives the proxy's removal) and the firewall can't tear a
/// live flow down. The agent, reached over the one egress path a closed team
/// keeps, drops the sessions from inside the guest instead. It only ever acts
/// on the inbound/listening side (log off RDP sessions; reset established
/// sockets on service ports), never on its own outbound gateway socket.
#[cfg(windows)]
fn cmd_disconnect_sessions(_payload: &Value) -> Result<String, String> {
    // `query session` lists one row per session. A connected RDP session's
    // SESSIONNAME is "rdp-tcp#<n>"; the bare "rdp-tcp" row is the idle listener
    // and "console"/"services" are local, so keying on the "rdp-tcp#" prefix
    // hits exactly the live remote sessions. The agent runs as a LocalSystem
    // service in session 0, which has no "rdp-tcp#" name, so it is never a
    // candidate -- its gateway connection is untouched.
    let out = Command::new("query")
        .arg("session")
        .output()
        .map_err(|e| format!("spawning query session: {e}"))?;
    if !out.status.success() {
        return Err(format!("query session exited {}: {}", out.status, String::from_utf8_lossy(&out.stderr).trim()));
    }
    let text = String::from_utf8_lossy(&out.stdout);
    let ids = parse_rdp_session_ids(&text);
    if ids.is_empty() {
        return Ok("no active RDP sessions to disconnect".to_string());
    }
    let mut done: Vec<u32> = Vec::new();
    let mut errs: Vec<String> = Vec::new();
    for id in &ids {
        match Command::new("logoff").arg(id.to_string()).output() {
            Ok(o) if o.status.success() => done.push(*id),
            Ok(o) => errs.push(format!("session {id}: logoff exited {} {}", o.status, String::from_utf8_lossy(&o.stderr).trim())),
            Err(e) => errs.push(format!("session {id}: spawning logoff: {e}")),
        }
    }
    if done.is_empty() && !errs.is_empty() {
        return Err(format!("failed to log off RDP session(s): {}", errs.join("; ")));
    }
    let mut msg = format!("logged off {} RDP session(s): {:?}", done.len(), done);
    if !errs.is_empty() {
        msg.push_str(&format!(" ({} failed: {})", errs.len(), errs.join("; ")));
    }
    Ok(msg)
}

#[cfg(unix)]
fn cmd_disconnect_sessions(payload: &Value) -> Result<String, String> {
    // `ss -K` resets matching sockets; it needs CONFIG_INET_DIAG_DESTROY and
    // root (the agent has both). We match the LISTENING side (sport) of the
    // usual remote-access ports plus any the server passed in `ports`, so only
    // inbound sessions are reset -- the agent's own connection is OUTBOUND to
    // the gateway on a different port and is never matched. Best-effort per
    // port: a kernel without -K support is reported, not fatal.
    let mut ports: Vec<u64> = vec![22, 3389, 5900];
    if let Some(arr) = payload.get("ports").and_then(Value::as_array) {
        for v in arr {
            if let Some(p) = v.as_u64() {
                if p > 0 && p < 65536 {
                    ports.push(p);
                }
            }
        }
    }
    ports.sort_unstable();
    ports.dedup();
    let mut done: Vec<u64> = Vec::new();
    let mut errs: Vec<String> = Vec::new();
    for p in &ports {
        let out = Command::new("ss")
            .arg("-K")
            .arg("state")
            .arg("established")
            .arg(format!("sport = :{p}"))
            .output();
        match out {
            // ss exits 0 whether or not it killed anything on that port, so a
            // clean exit just means "nothing established there" or "reset them".
            Ok(o) if o.status.success() => done.push(*p),
            Ok(o) => errs.push(format!("port {p}: ss -K exited {} {}", o.status, String::from_utf8_lossy(&o.stderr).trim())),
            Err(e) => errs.push(format!("port {p}: spawning ss: {e}")),
        }
    }
    if done.is_empty() && !errs.is_empty() {
        return Err(format!("could not reset sessions: {}", errs.join("; ")));
    }
    let mut msg = format!("reset established sessions on port(s): {done:?}");
    if !errs.is_empty() {
        msg.push_str(&format!(" ({} port(s) failed: {})", errs.len(), errs.join("; ")));
    }
    Ok(msg)
}

fn run_ok(cmd: &str, args: &[&str]) -> Result<(), String> {
    run_codes(cmd, args, &[])
}

// run_codes runs a command, treating success -- or any exit code in
// ok_codes -- as fine. The ok_codes are how a re-run stays idempotent
// where the tool signals "already in that state" with a nonzero code
// (Windows `net`/`sc` do this) rather than succeeding silently.
fn run_codes(cmd: &str, args: &[&str], ok_codes: &[i32]) -> Result<(), String> {
    let out = Command::new(cmd).args(args).output().map_err(|e| format!("spawning {cmd}: {e}"))?;
    if out.status.success() {
        return Ok(());
    }
    if let Some(code) = out.status.code() {
        if ok_codes.contains(&code) {
            return Ok(());
        }
    }
    Err(format!("{cmd} exited {}: {}", out.status, String::from_utf8_lossy(&out.stderr)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn write_then_read_file_round_trip() {
        let dir = std::env::temp_dir().join(format!("laforge-agent-test-{}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("hello.txt");
        let payload = json!({"path": path.to_str().unwrap(), "content": "hello world", "mode": "0644"});
        let outcome = run("write_file", &payload);
        assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
        assert_eq!(fs::read_to_string(&path).unwrap(), "hello world");

        let outcome = run("append_file", &json!({"path": path.to_str().unwrap(), "content": "!"}));
        assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
        assert_eq!(fs::read_to_string(&path).unwrap(), "hello world!");

        let outcome = run("delete", &json!({"path": path.to_str().unwrap()}));
        assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
        assert!(!path.exists());

        // Deleting again (already gone) must still succeed -- idempotent.
        let outcome = run("delete", &json!({"path": path.to_str().unwrap()}));
        assert_eq!(outcome.status, "done", "delete should be idempotent, got error: {}", outcome.error);

        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn execute_real_command() {
        let outcome = run("execute", &json!({"command": "/bin/echo", "args": ["hi", "there"]}));
        assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
        assert_eq!(outcome.output.trim(), "hi there");
    }

    #[test]
    fn parses_only_connected_rdp_session_ids() {
        // Representative `query session` output: the services (0) and console
        // (1) rows and the idle "rdp-tcp" listener (65536) must be ignored; only
        // the two connected "rdp-tcp#N" rows -- one with a blank USERNAME,
        // shifting the columns, and the current one marked with '>' -- yield IDs.
        let text = "\
 SESSIONNAME       USERNAME                 ID  STATE   TYPE        DEVICE
 services                                    0  Disc
 console                                     1  Conn
>rdp-tcp#2         Administrator             2  Active  rdpwd
 rdp-tcp#5                                   5  Active  rdpwd
 rdp-tcp                                 65536  Listen
";
        assert_eq!(parse_rdp_session_ids(text), vec![2, 5]);
    }

    #[test]
    fn parses_no_sessions_when_only_listener_present() {
        let text = "\
 SESSIONNAME       USERNAME                 ID  STATE   TYPE        DEVICE
 services                                    0  Disc
 console                                     1  Conn
 rdp-tcp                                 65536  Listen
";
        assert!(parse_rdp_session_ids(text).is_empty());
    }

    #[test]
    fn execute_failing_command_reports_failed_not_panic() {
        let outcome = run("execute", &json!({"command": "/bin/sh", "args": ["-c", "exit 7"]}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.contains('7'), "error should mention the exit code: {}", outcome.error);
    }

    #[test]
    fn execute_nonexistent_binary_reports_failed_not_panic() {
        let outcome = run("execute", &json!({"command": "/no/such/binary/at/all"}));
        assert_eq!(outcome.status, "failed");
        assert!(!outcome.error.is_empty());
    }

    #[cfg(unix)]
    fn pid_alive(pid: i32) -> bool {
        extern "C" {
            fn kill(pid: i32, sig: i32) -> i32;
        }
        unsafe { kill(pid, 0) == 0 }
    }

    // Polls briefly: a SIGKILLed orphan stays a zombie until init reaps it.
    #[cfg(unix)]
    fn wait_dead(pid: i32) -> bool {
        let until = Instant::now() + Duration::from_secs(3);
        while Instant::now() < until {
            if !pid_alive(pid) {
                return true;
            }
            std::thread::sleep(Duration::from_millis(50));
        }
        false
    }

    #[cfg(unix)]
    fn grandchild_pid(output: &str) -> i32 {
        let rest = output.split("grandchild=").nth(1).unwrap_or_else(|| panic!("no grandchild pid in output: {output}"));
        rest.chars().take_while(char::is_ascii_digit).collect::<String>().parse().unwrap()
    }

    #[test]
    fn execute_timeout_kills_and_reports_tail() {
        let start = Instant::now();
        let outcome = run(
            "execute",
            &json!({"command": "/bin/sh", "args": ["-c", "echo before-the-hang; sleep 30"], "timeout_sec": 1}),
        );
        assert_eq!(outcome.status, "timeout");
        assert!(outcome.error.contains("timed out after 1s"), "error should say it timed out: {}", outcome.error);
        assert!(outcome.error.contains("before-the-hang"), "error should carry the output tail: {}", outcome.error);
        assert!(start.elapsed() < Duration::from_secs(4), "took {:?}", start.elapsed());
    }

    // The case that hung prod: the wrapper shell is killed, but a grandchild
    // it started still holds stdout/stderr open. Killing only the direct child
    // would leave the step waiting on the pipes (and the grandchild running).
    #[cfg(unix)]
    #[test]
    fn execute_timeout_kills_grandchild_holding_pipes() {
        let start = Instant::now();
        let outcome = run(
            "execute",
            &json!({"command": "/bin/sh", "args": ["-c", "sleep 30 & echo grandchild=$!; wait"], "timeout_sec": 1}),
        );
        assert_eq!(outcome.status, "timeout");
        assert!(outcome.error.contains("timed out"), "{}", outcome.error);
        // Well under PIPE_DRAIN_GRACE: the pipes closed because the grandchild died.
        assert!(start.elapsed() < Duration::from_secs(3), "took {:?}", start.elapsed());
        let pid = grandchild_pid(&outcome.error);
        assert!(wait_dead(pid), "grandchild {pid} survived the timeout");
    }

    // Only a real deadline expiry is "timeout"; a script that merely prints the
    // phrase and fails is an ordinary failure.
    #[test]
    fn execute_output_mentioning_timeout_is_still_failed() {
        let outcome = run("execute", &json!({"command": "/bin/sh", "args": ["-c", "echo 'curl timed out after 5s' >&2; exit 1"]}));
        assert_eq!(outcome.status, "failed", "{}", outcome.error);
    }

    #[test]
    fn execute_zero_or_missing_timeout_means_no_deadline() {
        for payload in [
            json!({"command": "/bin/sh", "args": ["-c", "sleep 1; echo ok"], "timeout_sec": 0}),
            json!({"command": "/bin/sh", "args": ["-c", "sleep 1; echo ok"]}),
        ] {
            let outcome = run("execute", &payload);
            assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
            assert_eq!(outcome.output.trim(), "ok");
        }
    }

    #[test]
    fn execute_finishing_inside_timeout_succeeds() {
        let outcome = run("execute", &json!({"command": "/bin/echo", "args": ["fast"], "timeout_sec": 30}));
        assert_eq!(outcome.status, "done", "error was: {}", outcome.error);
        assert_eq!(outcome.output.trim(), "fast");
    }

    // A script that exits normally but leaves a background process holding the
    // pipes must still return (after the drain grace) and must NOT kill that
    // process -- steps legitimately start daemons.
    #[cfg(unix)]
    #[test]
    fn exited_command_with_lingering_grandchild_returns_after_grace() {
        let mut cmd = Command::new("/bin/sh");
        cmd.args(["-c", "sleep 30 & echo grandchild=$!"]);
        let start = Instant::now();
        let out = run_captured(cmd, None, Duration::from_millis(300)).unwrap();
        assert!(start.elapsed() < Duration::from_secs(3), "took {:?}", start.elapsed());
        assert!(out.status.unwrap().success());
        let pid = grandchild_pid(&String::from_utf8_lossy(&out.stdout));
        assert!(pid_alive(pid), "a normal exit must not kill the script's background processes");
        unsafe {
            extern "C" {
                fn kill(pid: i32, sig: i32) -> i32;
            }
            kill(pid, 9);
        }
    }

    #[test]
    fn upload_still_reports_not_implemented() {
        let outcome = run("upload", &json!({"from": "/var/log/setup.log"}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.contains("not implemented"), "upload error was: {}", outcome.error);
    }

    #[test]
    fn download_and_extract_are_implemented_missing_fields_fail_cleanly() {
        // No longer the "not implemented" stub: a missing field is a real
        // validation failure, and run() never panics on a malformed payload.
        for cmd in ["download", "extract"] {
            let outcome = run(cmd, &json!({}));
            assert_eq!(outcome.status, "failed", "{cmd} with no fields should fail");
            assert!(!outcome.error.contains("not implemented"), "{cmd} should be implemented now: {}", outcome.error);
        }
    }

    #[test]
    fn download_fetches_from_a_real_http_server() {
        use std::io::Read as _;
        // A minimal one-shot HTTP/1.1 server on a real loopback socket.
        let listener = std::net::TcpListener::bind("127.0.0.1:0").unwrap();
        let addr = listener.local_addr().unwrap();
        let handle = std::thread::spawn(move || {
            let (mut sock, _) = listener.accept().unwrap();
            let mut buf = [0u8; 1024];
            let _ = sock.read(&mut buf); // consume the request line/headers
            let body = b"payload-bytes";
            let resp = format!(
                "HTTP/1.1 200 OK\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
                body.len()
            );
            sock.write_all(resp.as_bytes()).unwrap();
            sock.write_all(body).unwrap();
        });

        let dir = unique_tmp("download");
        fs::create_dir_all(&dir).unwrap();
        let to = dir.join("nested").join("out.bin"); // parent dir must be created
        let url = format!("http://{addr}/file");
        let outcome = run("download", &json!({"from": url, "to": to.to_str().unwrap()}));
        handle.join().unwrap();
        assert_eq!(outcome.status, "done", "download error: {}", outcome.error);
        assert_eq!(fs::read(&to).unwrap(), b"payload-bytes");
        assert!(outcome.output.contains("downloaded 13 bytes"), "output: {}", outcome.output);
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn download_unreachable_url_reports_failed_not_panic() {
        // Port 1 refuses instantly -- a real connection failure, surfaced (not a panic).
        let outcome = run("download", &json!({"from": "http://127.0.0.1:1/nope", "to": "/tmp/laforge-dl-test"}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.contains("downloading"), "error was: {}", outcome.error);
    }

    fn unique_tmp(tag: &str) -> std::path::PathBuf {
        let mut p = std::env::temp_dir();
        let nanos = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos();
        p.push(format!("laforge-{tag}-{nanos}"));
        p
    }

    #[test]
    fn extract_unpacks_tar_gz() {
        let dir = unique_tmp("extract-targz");
        fs::create_dir_all(&dir).unwrap();
        let archive = dir.join("payload.tar.gz");
        {
            let f = fs::File::create(&archive).unwrap();
            let enc = flate2::write::GzEncoder::new(f, flate2::Compression::default());
            let mut tb = tar::Builder::new(enc);
            let data = b"hi";
            let mut header = tar::Header::new_gnu();
            header.set_path("hello.txt").unwrap();
            header.set_size(data.len() as u64);
            header.set_mode(0o644);
            header.set_cksum();
            tb.append(&header, &data[..]).unwrap();
            tb.into_inner().unwrap().finish().unwrap();
        }
        let dest = dir.join("out");
        let outcome = run("extract", &json!({"src": archive.to_str().unwrap(), "dest": dest.to_str().unwrap()}));
        assert_eq!(outcome.status, "done", "extract error: {}", outcome.error);
        assert_eq!(fs::read_to_string(dest.join("hello.txt")).unwrap(), "hi");
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn extract_unpacks_zip() {
        let dir = unique_tmp("extract-zip");
        fs::create_dir_all(&dir).unwrap();
        let archive = dir.join("payload.zip");
        {
            let f = fs::File::create(&archive).unwrap();
            let mut zw = zip::ZipWriter::new(f);
            zw.start_file("hello.txt", zip::write::SimpleFileOptions::default()).unwrap();
            zw.write_all(b"hi").unwrap();
            zw.finish().unwrap();
        }
        let dest = dir.join("out");
        let outcome = run("extract", &json!({"src": archive.to_str().unwrap(), "dest": dest.to_str().unwrap()}));
        assert_eq!(outcome.status, "done", "extract error: {}", outcome.error);
        assert_eq!(fs::read_to_string(dest.join("hello.txt")).unwrap(), "hi");
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn extract_unknown_type_reports_failed() {
        let dir = unique_tmp("extract-bad");
        fs::create_dir_all(&dir).unwrap();
        let f = dir.join("payload.rar");
        fs::write(&f, b"not really a rar").unwrap();
        let outcome = run("extract", &json!({"src": f.to_str().unwrap(), "dest": dir.join("out").to_str().unwrap()}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.contains("unsupported archive type"), "error was: {}", outcome.error);
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn unknown_command_reports_failed_not_panic() {
        let outcome = run("teleport", &json!({}));
        assert_eq!(outcome.status, "failed");
    }

    #[test]
    fn malformed_payloads_never_panic() {
        // Every real command, fed a payload missing every field it needs,
        // or the wrong JSON shape entirely (a bare number instead of an
        // object) -- must report "failed" with a message, never panic.
        // This is the Rust agent's equivalent of agentproto's fuzz test:
        // untrusted structured input (here, whatever the gateway sends)
        // must never crash the process handling it.
        // "reboot" is deliberately EXCLUDED from this list: it is wired
        // to a real `shutdown -r` command (see cmd_reboot's own doc
        // comment), and this test suite must never risk actually
        // rebooting whatever machine runs `cargo test`. Its malformed-
        // payload handling (missing delay_sec) is covered by inspection
        // -- `.and_then(...).unwrap_or(0)` never panics on a missing or
        // wrong-typed field -- not by execution.
        let commands = [
            "execute", "write_file", "append_file", "delete", "change_perms",
            "create_user", "set_password", "add_to_group", "service",
        ];
        let bad_payloads = [json!({}), json!(null), json!(42), json!("a string"), json!([1, 2, 3])];
        for cmd in commands {
            for payload in &bad_payloads {
                let outcome = run(cmd, payload);
                assert_eq!(outcome.status, "failed", "{cmd} with payload {payload} should fail cleanly, got status={}", outcome.status);
            }
        }
    }
}
