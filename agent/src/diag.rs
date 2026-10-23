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
// turn logging on by editing the systemd unit or the Windows service.

use std::fs::OpenOptions;
use std::io::Write;
use std::panic;
use std::path::PathBuf;
use std::sync::RwLock;
use std::time::{SystemTime, UNIX_EPOCH};

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

/// install_panic_hook routes a panic's message and location through the sink
/// before the process dies. This matters because the release profile is
/// panic="abort": a panic in ANY thread (e.g. the interactive-shell relay's PTY
/// read on Windows) kills the whole agent instantly, with no unwinding and no
/// trace -- the default hook prints to a stderr that a Windows service has no
/// console for, so the crash is invisible and looks like the agent just
/// "restarting". With the hook, a panic leaves one line in the debug log naming
/// the file:line, which is the difference between a diagnosable crash and a
/// silent one. It writes through the same sink as dlog!, so it stays silent on a
/// production (non-debug) build just like everything else. The previous hook is
/// chained so dev builds keep their normal stderr backtrace.
pub fn install_panic_hook() {
    let prev = panic::take_hook();
    panic::set_hook(Box::new(move |info| {
        let loc = info
            .location()
            .map(|l| format!("{}:{}", l.file(), l.line()))
            .unwrap_or_else(|| "unknown".to_string());
        let msg = info
            .payload()
            .downcast_ref::<&str>()
            .copied()
            .or_else(|| info.payload().downcast_ref::<String>().map(String::as_str))
            .unwrap_or("<non-string panic payload>");
        let thread = std::thread::current().name().unwrap_or("unnamed").to_string();
        write_line(&format!("laforge-agent: PANIC on thread \"{thread}\" at {loc}: {msg}"));
        prev(info);
    }));
}

// Line ending for the log file, chosen for the target OS: CRLF on Windows so
// the file opens correctly in Notepad (which treats a lone LF as no break and
// shows everything on one line), plain LF everywhere else. This is the target's
// convention, not the host's -- a Windows build (incl. the cross-compiled
// windows-gnu agent) gets CRLF because cfg(windows) follows the target.
#[cfg(windows)]
const LINE_END: &str = "\r\n";
#[cfg(not(windows))]
const LINE_END: &str = "\n";

/// write_line is the one place a diagnostic line is emitted; the dlog! macro
/// calls it. Everything is serialized through the sink lock (diagnostics are
/// low-frequency, so contention is a non-issue). Each line is prefixed with a
/// UTC timestamp so events can be placed in time and correlated with the gateway
/// and api logs (which are UTC too). The line terminator is the target OS's
/// (LINE_END) rather than writeln!'s hardcoded "\n", so the Windows log file is
/// CRLF-terminated and readable in Notepad.
pub fn write_line(line: &str) {
    let mut guard = match SINK.write() {
        Ok(g) => g,
        Err(_) => return,
    };
    // A production build is Silent -- bail before even formatting a timestamp,
    // keeping the hot path (every dlog! on a deployed agent) free.
    if matches!(&*guard, Sink::Silent) {
        return;
    }
    let ts = timestamp_utc();
    match &mut *guard {
        Sink::Silent => {}
        Sink::Stderr => {
            let _ = write!(std::io::stderr(), "{ts} {line}{LINE_END}");
        }
        Sink::File(f) => {
            let _ = write!(f, "{ts} {line}{LINE_END}");
            let _ = f.flush();
        }
    }
}

/// timestamp_utc renders the current time as an ISO-8601 UTC instant
/// (YYYY-MM-DDТHH:MM:SS.mmmZ), computed from the Unix epoch with no time crate
/// (the agent is deliberately dependency-light). Epoch seconds are
/// timezone-independent, so this is true UTC regardless of the host's clock
/// settings, which is exactly what makes it line up with the server-side logs.
fn timestamp_utc() -> String {
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default();
    let secs = now.as_secs();
    let millis = now.subsec_millis();
    let (y, mo, d) = civil_from_days((secs / 86_400) as i64);
    let rem = secs % 86_400;
    let (hh, mm, ss) = (rem / 3_600, (rem % 3_600) / 60, rem % 60);
    format!("{y:04}-{mo:02}-{d:02}T{hh:02}:{mm:02}:{ss:02}.{millis:03}Z")
}

/// civil_from_days converts a count of days since 1970-01-01 into a proleptic
/// Gregorian (year, month, day). Howard Hinnant's well-known public-domain
/// algorithm -- exact, branch-light, and library-free.
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719_468; // shift epoch to 0000-03-01
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = (z - era * 146_097) as i64; // day of era [0, 146096]
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365; // [0, 399]
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    let mp = (5 * doy + 2) / 153; // [0, 11]
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32; // [1, 31]
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32; // [1, 12]
    (if m <= 2 { y + 1 } else { y }, m, d)
}

/// dlog! replaces eprintln!/println! everywhere in the agent: same formatting,
/// but routed through the sink above (Silent by default) instead of the
/// console.
#[macro_export]
macro_rules! dlog {
    ($($arg:tt)*) => { $crate::diag::write_line(&format!($($arg)*)) };
}

#[cfg(test)]
mod tests {
    use super::civil_from_days;

    // Spot-check the epoch->date math across month, year, and leap-day
    // boundaries (day counts since 1970-01-01).
    #[test]
    fn civil_from_days_known_dates() {
        let cases: &[(i64, (i64, u32, u32))] = &[
            (0, (1970, 1, 1)),
            (31, (1970, 2, 1)),
            (59, (1970, 3, 1)),   // 1970 is not a leap year
            (365, (1971, 1, 1)),
            (789, (1972, 2, 29)), // 1972 IS a leap year
            (790, (1972, 3, 1)),
        ];
        for &(days, want) in cases {
            let got = civil_from_days(days);
            assert_eq!(got, want, "civil_from_days({days})");
        }
    }
}
