// Basic anti-debug / anti-tamper self-checks -- "Basic
// anti-debug / anti-tamper self-checks (self-hash, ptrace/
// IsDebuggerPresent detection) that degrade gracefully (log + continue)
// rather than being load-bearing for security -- the protocol obscurity
// and mTLS/token model are the real control; obfuscation buys time, it
// is not the boundary." Every check in this file follows that rule
// exactly: self_check() only ever returns findings to log. Nothing here
// ever refuses to start, exits, or changes agent behavior based on the
// result -- a false positive (a legitimate `strace`, a loaded system,
// CI) must never turn into a denial-of-service against the agent's own
// real job.
//
// Honestly scoped, not silently under-delivered:
//   - ptrace detection is real, but Linux-only, via /proc/self/status's
//     TracerPid field, not the more commonly cited "call
//     PTRACE_TRACEME on yourself and see if it fails" trick. That trick
//     was deliberately NOT used: on success it actually attaches this
//     process as its own tracee, and a parent that isn't prepared to
//     reap ptrace stops (a container's PID 1, a plain shell, systemd)
//     can leave the agent stuck stopped the next time it receives a
//     signal that would otherwise just be delivered -- a real
//     production risk for a fixed, no-real-benefit-over-/proc technique.
//     Reading TracerPid has the identical detection power with zero side
//     effects.
//   - Windows uses kernel32's IsDebuggerPresent, the Win32 equivalent of
//     the Linux TracerPid read. debugger_attached() is only a no-op on
//     targets that are neither Linux nor Windows.
//   - Self-hashing (hash the binary at runtime, compare against a value
//     sealed in after compilation) lives in selfhash.rs and is folded into
//     self_check() below. It uses the seal-pass build machinery in
//     internal/agentfactory.SelfHashSeal (compile, then hash with the
//     per-host identity region and the hash slot masked, then patch the
//     hash), so a per-host identity patch never invalidates it. Like every
//     other check here it only ever logs a finding.
//   - The timing check below is the one real, working, cross-platform
//     technique in this file: a fixed amount of real integer work,
//     flagged only if it took implausibly long, which is the observable
//     symptom of single-stepping through it in a debugger or a
//     breakpoint pausing mid-loop.

use std::time::{Duration, Instant};

#[cfg(target_os = "linux")]
fn debugger_attached() -> bool {
    match std::fs::read_to_string("/proc/self/status") {
        Ok(status) => status
            .lines()
            .find_map(|l| l.strip_prefix("TracerPid:"))
            .map(|v| v.trim() != "0")
            .unwrap_or(false),
        // /proc unreadable (a sandboxed or unusual environment) is not
        // itself suspicious -- treat it the same as "not traced," never
        // as a finding of its own.
        Err(_) => false,
    }
}

#[cfg(windows)]
fn debugger_attached() -> bool {
    // The Win32 equivalent of the /proc TracerPid check: kernel32's
    // IsDebuggerPresent reports whether a user-mode debugger is attached
    // to this process. Declared directly rather than pulling in a Windows
    // API crate, to keep the agent's dependency set as small as the rest
    // of it.
    #[link(name = "kernel32")]
    extern "system" {
        fn IsDebuggerPresent() -> i32;
    }
    unsafe { IsDebuggerPresent() != 0 }
}

#[cfg(not(any(target_os = "linux", windows)))]
fn debugger_attached() -> bool {
    false // no real equivalent implemented for this target yet -- see this file's own top doc comment
}

/// A generous threshold, deliberately: real scheduling jitter, a loaded
/// CI runner, or a slow VM can all push a tight loop's wall time up by a
/// meaningful multiple without anything actually being wrong. 2,000,000
/// iterations of trivial integer math finishes in low single-digit
/// milliseconds on any real target this agent runs on; 500ms is roughly
/// two orders of magnitude of headroom above that, chosen to keep this
/// check from ever crying wolf on ordinary hardware while still catching
/// the multi-second-plus pause a debugger breakpoint or single-stepping
/// actually causes.
const TIMING_THRESHOLD: Duration = Duration::from_millis(500);

fn timing_anomaly() -> bool {
    let start = Instant::now();
    let mut acc: u64 = 0;
    for i in 0..2_000_000u64 {
        acc = acc.wrapping_add(i ^ (i << 3)).rotate_left(1);
    }
    // Never optimized away: the loop's result feeds back into the
    // return-relevant elapsed-time measurement only via real, unavoidable
    // wall-clock work, but std::hint::black_box also pins `acc` itself so
    // the optimizer can't prove the whole loop is dead and skip it.
    std::hint::black_box(acc);
    start.elapsed() > TIMING_THRESHOLD
}

/// self_check runs every real check in this module and returns one
/// human-readable finding per check that looked suspicious. Called once
/// from main() at startup; findings are logged and the agent proceeds
/// regardless, exactly matching the "degrade
/// gracefully... not load-bearing" rule -- see this file's top doc
/// comment for what's real here versus scoped out.
pub fn self_check() -> Vec<String> {
    let mut findings = Vec::new();
    if debugger_attached() {
        findings.push("a debugger or tracer appears to be attached to this process".to_string());
    }
    if timing_anomaly() {
        findings.push(format!(
            "a fixed-work timing probe took longer than {TIMING_THRESHOLD:?} -- possible single-stepping or a breakpoint (or just a very loaded machine)"
        ));
    }
    if let Some(f) = crate::selfhash::self_hash_finding() {
        findings.push(f);
    }
    findings
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn self_check_never_panics() {
        // The actual claim this test can make: it runs to completion
        // without panicking. Asserting the findings are always empty
        // would be a real flake risk on a slow/loaded CI runner (see
        // TIMING_THRESHOLD's own doc comment) -- this file's whole design
        // point is that a false positive here is a logged line, not a
        // test or agent failure.
        let _ = self_check();
    }

    #[test]
    fn timing_anomaly_is_false_under_normal_conditions() {
        // Unlike self_check_never_panics, this one real assertion is
        // still worth making directly: a 500ms threshold for 2 million
        // trivial iterations should never trip on real test hardware,
        // and if it starts doing so that's worth knowing about (a
        // genuine regression in the check's own tuning), not silently
        // tolerating flakiness by never asserting anything at all.
        assert!(!timing_anomaly(), "timing_anomaly tripped under a plain cargo test run -- TIMING_THRESHOLD may need retuning");
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn debugger_attached_is_false_for_an_untraced_test_process() {
        assert!(!debugger_attached(), "cargo test itself should not read as traced");
    }
}
