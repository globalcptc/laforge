// The agent's own diagnostic output. On a hostile student network the agent
// must leave NO local trace by default -- nothing to stdout, nothing to stderr,
// nothing a captured box can read. The ONLY thing that ever leaves the agent is
// what it sends the servers over mTLS. So every diagnostic line the agent would
// otherwise print goes through this sink instead of eprintln!/println!, and the
// sink is Silent unless explicitly turned on. Two things turn it on:
//   - a dev build (unpatched identity, i.e. `cargo run`) logs to stderr, so a
//     developer iterating on the agent still sees output;
//   - a real build whose baked identity carries the agent-debug flag writes to
//     a file next to the binary (and still nothing to the console).
// The flag is baked into the binary alongside the identity (see identity.rs),
// NOT read from an environment variable or the launcher -- so a student can't
// turn logging on by editing the systemd unit or the scheduled task.

use std::fs::OpenOptions;
use std::io::Write;
use std::path::PathBuf;
use std::sync::RwLock;

enum Sink {
    Silent,
    Stderr,
    File(std::fs::File),
}

static SINK: RwLock<Sink> = RwLock::new(Sink::Silent);

/// init_silent is the default and the production posture: diagnostics go
/// nowhere. Explicit so main() reads clearly.
pub fn init_silent() {
    if let Ok(mut s) = SINK.write() {
        *s = Sink::Silent;
    }
}

/// init_stderr is for dev builds only (`cargo run`), so output is visible while
/// iterating. A real deployed agent never selects this.
pub fn init_stderr() {
    if let Ok(mut s) = SINK.write() {
        *s = Sink::Stderr;
    }
}

/// init_file_next_to_binary points the sink at a log file beside the agent
/// binary (e.g. /usr/local/bin/laforge-agent.log). Best effort: if the path
/// can't be resolved or the file can't be opened, the sink is left as-is --
/// it never falls back to the console, because silence is the safe default.
pub fn init_file_next_to_binary() {
    let path = match log_path() {
        Some(p) => p,
        None => return,
    };
    if let Ok(f) = OpenOptions::new().create(true).append(true).open(&path) {
        if let Ok(mut s) = SINK.write() {
            *s = Sink::File(f);
        }
    }
}

fn log_path() -> Option<PathBuf> {
    let exe = std::env::current_exe().ok()?;
    Some(exe.parent()?.join("laforge-agent.log"))
}

/// write_line is the one place a diagnostic line is emitted; the dlog! macro
/// calls it. Everything is serialized through the sink lock (diagnostics are
/// low-frequency, so contention is a non-issue).
pub fn write_line(line: &str) {
    let mut guard = match SINK.write() {
        Ok(g) => g,
        Err(_) => return,
    };
    match &mut *guard {
        Sink::Silent => {}
        Sink::Stderr => {
            let _ = writeln!(std::io::stderr(), "{line}");
        }
        Sink::File(f) => {
            let _ = writeln!(f, "{line}");
            let _ = f.flush();
        }
    }
}

/// dlog! replaces eprintln!/println! everywhere in the agent: same formatting,
/// but routed through the sink above (Silent by default) instead of the
/// console.
#[macro_export]
macro_rules! dlog {
    ($($arg:tt)*) => { $crate::diag::write_line(&format!($($arg)*)) };
}
