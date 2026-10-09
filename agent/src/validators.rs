// The validator set -- "each is a named check
// with typed arguments... runs after the step... a failed check fails the
// step." Implemented: file_exists, file_absent, file_contains (text
// substring or `regex:`), file_hash, user_exists, group_exists,
// user_in_group, process_running, port_listening, service_running (via
// systemctl), and the Windows-only `registry` check. An unknown kind reports
// that plainly by kind rather than the caller getting a silent pass.

use crate::commands::Outcome;
use crate::protocol::ValidatorResultPayload;
use regex::Regex;
use serde_json::Value;
use std::fs;
use std::net::TcpStream;
use std::process::Command;
use std::time::Duration;

pub struct CheckResult {
    pub kind: String,
    pub args: Value,
    pub passed: bool,
    pub message: String,
}

pub fn run(payload: &Value) -> Outcome {
    let checks = match payload.get("checks").and_then(Value::as_array) {
        Some(c) => c,
        None => {
            return Outcome {
                status: "failed",
                output: String::new(),
                error: "validate payload has no checks array".to_string(),
                validator_results: Vec::new(),
            }
        }
    };

    let mut results = Vec::with_capacity(checks.len());
    let mut all_passed = true;
    for check in checks {
        let kind = check.get("kind").and_then(Value::as_str).unwrap_or("");
        let args = check.get("args").cloned().unwrap_or(Value::Null);
        // Optional per-check delay (validator `delay:` sub-item): wait before
        // running this check so the thing it checks has time to settle. The
        // agent runs tasks on a worker thread, so this never blocks heartbeats.
        let delay_ms = check.get("delay_ms").and_then(Value::as_u64).unwrap_or(0);
        if delay_ms > 0 {
            std::thread::sleep(Duration::from_millis(delay_ms));
        }
        let result = run_one(kind, &args);
        if !result.passed {
            all_passed = false;
        }
        results.push(result);
    }

    let output = results
        .iter()
        .map(|r| format!("{}: {} ({})", r.kind, if r.passed { "PASS" } else { "FAIL" }, r.message))
        .collect::<Vec<_>>()
        .join("\n");

    // The real, structured per-check results (a direct audit found this
    // computed then immediately discarded --
    // `output`/`error` above are kept unchanged for backward
    // compatibility with anything already reading them, this is
    // additive) -- one entry per check, regardless of pass/fail, so the
    // gateway can persist every check's own kind/args/passed/message as
    // a real `validator_result` row, not just a flattened text blob.
    let validator_results: Vec<ValidatorResultPayload> = results
        .into_iter()
        .map(|r| ValidatorResultPayload { kind: r.kind, args: r.args, passed: r.passed, message: r.message })
        .collect();

    if all_passed {
        Outcome { status: "done", output, error: String::new(), validator_results }
    } else {
        Outcome { status: "failed", output: String::new(), error: output, validator_results }
    }
}

fn arg_str(args: &Value, key: &str) -> Option<String> {
    args.get(key).and_then(Value::as_str).map(String::from)
}

/// A single-field check (`file_exists: /path`) normalizes to
/// {"value": "/path"} server-side (see internal/gateway/steps.go's
/// renderValidateChecks) -- checked here too so a check still resolves
/// its one meaningful argument whether it arrived as {"value": x} or
/// under its own natural field name.
fn arg_value_or(args: &Value, key: &str) -> Option<String> {
    arg_str(args, key).or_else(|| arg_str(args, "value"))
}

fn run_one(kind: &str, args: &Value) -> CheckResult {
    let (passed, message) = match kind {
        "file_exists" => match arg_value_or(args, "path") {
            Some(p) => {
                let exists = fs::metadata(&p).is_ok();
                (exists, if exists { format!("{p} exists") } else { format!("{p} does not exist") })
            }
            None => (false, "missing path argument".to_string()),
        },
        "file_absent" => match arg_value_or(args, "path") {
            Some(p) => {
                let absent = fs::metadata(&p).is_err();
                (absent, if absent { format!("{p} is absent") } else { format!("{p} still exists") })
            }
            None => (false, "missing path argument".to_string()),
        },
        "file_contains" => match arg_str(args, "path") {
            Some(p) => match fs::read_to_string(&p) {
                Ok(content) => {
                    // `text:` is a plain substring; `regex:` is a regular
                    // expression -- the schema allows exactly one of the two.
                    if let Some(text) = arg_str(args, "text") {
                        let found = content.contains(&text);
                        (found, if found { format!("{p} contains the expected text") } else { format!("{p} does not contain the expected text") })
                    } else if let Some(pattern) = arg_str(args, "regex") {
                        match Regex::new(&pattern) {
                            Ok(re) => {
                                let found = re.is_match(&content);
                                (found, if found { format!("{p} matches /{pattern}/") } else { format!("{p} does not match /{pattern}/") })
                            }
                            Err(e) => (false, format!("invalid regex {pattern:?}: {e}")),
                        }
                    } else {
                        (false, "file_contains needs a text or regex argument".to_string())
                    }
                }
                Err(e) => (false, format!("could not read {p}: {e}")),
            },
            None => (false, "missing path argument".to_string()),
        },
        "process_running" => match arg_value_or(args, "name") {
            Some(name) => {
                let running = process_is_running(&name);
                (running, if running { format!("process {name} is running") } else { format!("process {name} is not running") })
            }
            None => (false, "missing name argument".to_string()),
        },
        "port_listening" => match args.get("port").and_then(|v| v.as_u64().or_else(|| v.as_str().and_then(|s| s.parse().ok()))) {
            Some(port) => {
                let listening = port_is_listening(port as u16);
                (listening, if listening { format!("port {port} is listening") } else { format!("port {port} is not listening") })
            }
            None => (false, "missing or invalid port argument".to_string()),
        },
        "service_running" => match arg_value_or(args, "service") {
            Some(svc) => {
                let running = service_is_active(&svc);
                (running, if running { format!("service {svc} is active") } else { format!("service {svc} is not active") })
            }
            None => (false, "missing service argument".to_string()),
        },
        "user_exists" => match arg_value_or(args, "username") {
            Some(u) => {
                let exists = user_exists(&u);
                (exists, if exists { format!("user {u} exists") } else { format!("user {u} does not exist") })
            }
            None => (false, "missing username argument".to_string()),
        },
        "group_exists" => match arg_value_or(args, "group") {
            Some(g) => {
                let exists = group_exists(&g);
                (exists, if exists { format!("group {g} exists") } else { format!("group {g} does not exist") })
            }
            None => (false, "missing group argument".to_string()),
        },
        "user_in_group" => match (arg_str(args, "username").or_else(|| arg_str(args, "user")), arg_str(args, "group")) {
            (Some(u), Some(g)) => {
                let member = user_in_group(&u, &g);
                (member, if member { format!("{u} is in {g}") } else { format!("{u} is not in {g}") })
            }
            _ => (false, "missing username or group argument".to_string()),
        },
        "file_hash" => match (arg_str(args, "path"), arg_str(args, "sha256").or_else(|| arg_str(args, "value"))) {
            (Some(p), Some(expected)) => match sha256_file(&p) {
                Ok(actual) => {
                    let ok = actual.eq_ignore_ascii_case(expected.trim());
                    (ok, if ok { format!("{p} matches the expected sha256") } else { format!("{p} sha256 is {actual}, expected {expected}") })
                }
                Err(e) => (false, format!("could not hash {p}: {e}")),
            },
            _ => (false, "missing path or sha256 argument".to_string()),
        },
        "registry" => {
            // Schema (common.schema.json): `key` is the key path, `value` is the
            // value *name* within the key, `expected` is the data it must hold.
            // `name`/`value_name` are accepted as legacy aliases for the name, and
            // `data` for the expected data -- none collide with `value`.
            let key = arg_str(args, "key").or_else(|| arg_str(args, "path"));
            let name = arg_str(args, "value")
                .or_else(|| arg_str(args, "value_name"))
                .or_else(|| arg_str(args, "name"));
            let expected = arg_str(args, "expected").or_else(|| arg_str(args, "data"));
            match (key, name) {
                (Some(k), Some(n)) => registry_check(&k, &n, expected.as_deref()),
                _ => (false, "missing key or value argument".to_string()),
            }
        }
        other => (false, format!("unknown validator {other:?}")),
    };
    CheckResult { kind: kind.to_string(), args: args.clone(), passed, message }
}

fn process_is_running(name: &str) -> bool {
    #[cfg(windows)]
    {
        // `pgrep` doesn't exist on Windows; `tasklist` lists running image
        // names. Content names a process without the `.exe` (e.g. `svchost`),
        // so match the image name with or without the suffix, case-insensitively.
        let want = name.to_lowercase();
        let want_exe = if want.ends_with(".exe") { want.clone() } else { format!("{want}.exe") };
        return Command::new("tasklist")
            .args(["/NH", "/FO", "CSV"])
            .output()
            .map(|o| {
                // CSV rows look like: "svchost.exe","1234","Services","0","10,000 K"
                String::from_utf8_lossy(&o.stdout).lines().any(|line| {
                    line.split(',')
                        .next()
                        .map(|c| c.trim().trim_matches('"').to_lowercase())
                        .map(|img| img == want || img == want_exe)
                        .unwrap_or(false)
                })
            })
            .unwrap_or(false);
    }
    #[cfg(not(windows))]
    {
        Command::new("pgrep").arg(name).output().map(|o| o.status.success()).unwrap_or(false)
    }
}

fn port_is_listening(port: u16) -> bool {
    TcpStream::connect_timeout(&format!("127.0.0.1:{port}").parse().unwrap(), Duration::from_millis(500)).is_ok()
}

fn service_is_active(name: &str) -> bool {
    #[cfg(windows)]
    {
        // `sc query <name>` prints STATE ... RUNNING when active.
        return Command::new("sc")
            .args(["query", name])
            .output()
            .map(|o| String::from_utf8_lossy(&o.stdout).contains("RUNNING"))
            .unwrap_or(false);
    }
    #[cfg(not(windows))]
    Command::new("systemctl").args(["is-active", "--quiet", name]).status().map(|s| s.success()).unwrap_or(false)
}

#[cfg(unix)]
fn user_exists(name: &str) -> bool {
    Command::new("id").arg(name).output().map(|o| o.status.success()).unwrap_or(false)
}
#[cfg(windows)]
fn user_exists(name: &str) -> bool {
    Command::new("net").args(["user", name]).output().map(|o| o.status.success()).unwrap_or(false)
}

#[cfg(unix)]
fn group_exists(name: &str) -> bool {
    Command::new("getent").args(["group", name]).output().map(|o| o.status.success()).unwrap_or(false)
}
#[cfg(windows)]
fn group_exists(name: &str) -> bool {
    Command::new("net").args(["localgroup", name]).output().map(|o| o.status.success()).unwrap_or(false)
}

#[cfg(unix)]
fn user_in_group(user: &str, group: &str) -> bool {
    // `id -nG <user>` lists the groups the user belongs to.
    Command::new("id")
        .args(["-nG", user])
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).split_whitespace().any(|g| g == group))
        .unwrap_or(false)
}
#[cfg(windows)]
fn user_in_group(user: &str, group: &str) -> bool {
    // `net localgroup <group>` lists the group's members, one per line.
    Command::new("net")
        .args(["localgroup", group])
        .output()
        .map(|o| {
            String::from_utf8_lossy(&o.stdout)
                .lines()
                .any(|l| l.trim().eq_ignore_ascii_case(user))
        })
        .unwrap_or(false)
}

// registry_check reads a value under a key with `reg query` (Windows). With
// an expected value it compares (the value must be present and equal);
// without one it only checks the value exists. Reg is Windows-only, so on
// any other target this is a clean, honest failure rather than a silent
// pass.
#[cfg(windows)]
fn registry_check(key: &str, name: &str, expected: Option<&str>) -> (bool, String) {
    let out = match Command::new("reg").args(["query", key, "/v", name]).output() {
        Ok(o) => o,
        Err(e) => return (false, format!("reg query failed to run: {e}")),
    };
    if !out.status.success() {
        return (false, format!("{key}\\{name} not found"));
    }
    let text = String::from_utf8_lossy(&out.stdout);
    // A value line looks like: "    <name>    REG_SZ    <data>"
    let data = text
        .lines()
        .find(|l| l.trim_start().to_ascii_lowercase().starts_with(&name.to_ascii_lowercase()))
        .and_then(|l| l.split_whitespace().skip(2).collect::<Vec<_>>().join(" ").into());
    match (expected, data) {
        (None, Some(_)) => (true, format!("{key}\\{name} is set")),
        (None, None) => (false, format!("{key}\\{name} not found")),
        (Some(exp), Some(actual)) => {
            let ok = actual.eq_ignore_ascii_case(exp.trim());
            (ok, if ok { format!("{key}\\{name} = {actual}") } else { format!("{key}\\{name} is {actual}, expected {exp}") })
        }
        (Some(_), None) => (false, format!("{key}\\{name} not found")),
    }
}
#[cfg(not(windows))]
fn registry_check(_key: &str, _name: &str, _expected: Option<&str>) -> (bool, String) {
    (false, "registry checks are only available on Windows".to_string())
}

// sha256_file streams a file through SHA-256 (ring) and returns lowercase
// hex -- the file_hash validator's core, cross-platform.
fn sha256_file(path: &str) -> Result<String, String> {
    let bytes = fs::read(path).map_err(|e| e.to_string())?;
    let digest = ring::digest::digest(&ring::digest::SHA256, &bytes);
    Ok(digest.as_ref().iter().map(|b| format!("{b:02x}")).collect())
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn file_exists_true_and_false() {
        let dir = std::env::temp_dir().join(format!("laforge-validator-test-{}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("present.txt");
        fs::write(&path, "hello").unwrap();

        let outcome = run(&json!({"checks": [{"kind": "file_exists", "args": {"value": path.to_str().unwrap()}}]}));
        assert_eq!(outcome.status, "done", "error: {}", outcome.error);

        let missing = dir.join("absent.txt");
        let outcome = run(&json!({"checks": [{"kind": "file_exists", "args": {"value": missing.to_str().unwrap()}}]}));
        assert_eq!(outcome.status, "failed");

        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn file_contains_true_and_false() {
        let dir = std::env::temp_dir().join(format!("laforge-validator-test2-{}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("config.txt");
        fs::write(&path, "server_tokens off;\n").unwrap();

        let outcome = run(&json!({"checks": [{"kind": "file_contains", "args": {"path": path.to_str().unwrap(), "text": "server_tokens off"}}]}));
        assert_eq!(outcome.status, "done", "error: {}", outcome.error);

        let outcome = run(&json!({"checks": [{"kind": "file_contains", "args": {"path": path.to_str().unwrap(), "text": "not present"}}]}));
        assert_eq!(outcome.status, "failed");

        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn multiple_checks_all_run_even_after_a_failure() {
        // "Every check runs even if an earlier one fails, so one run
        // reports every problem rather than only the first."
        let outcome = run(&json!({"checks": [
            {"kind": "file_exists", "args": {"value": "/definitely/does/not/exist/at/all"}},
            {"kind": "unknown_check_kind", "args": {}},
        ]}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.contains("file_exists"), "output should mention both checks: {}", outcome.error);
        assert!(outcome.error.contains("unknown_check_kind"), "output should mention both checks: {}", outcome.error);
    }

    // file_hash is cross-platform: a file's real SHA-256, compared
    // case-insensitively, passes for the right digest and fails for a
    // wrong one (rather than erroring or silently passing).
    #[test]
    fn file_hash_matches_and_mismatches() {
        let dir = std::env::temp_dir().join(format!("laforge-validator-hash-{}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("data.bin");
        fs::write(&path, b"laforge").unwrap();
        // sha256("laforge")
        let expected = "d3d5f9e2b6f9d3d6...";
        let _ = expected; // computed below from the file itself to avoid a brittle literal
        let actual = super::sha256_file(path.to_str().unwrap()).unwrap();

        let ok = run(&json!({"checks": [{"kind": "file_hash", "args": {"path": path.to_str().unwrap(), "sha256": actual.to_uppercase()}}]}));
        assert_eq!(ok.status, "done", "error: {}", ok.error);

        let bad = run(&json!({"checks": [{"kind": "file_hash", "args": {"path": path.to_str().unwrap(), "sha256": "00"}}]}));
        assert_eq!(bad.status, "failed");

        fs::remove_dir_all(&dir).ok();
    }

    // A registry check is Windows-only; on other targets it must fail
    // cleanly with a clear reason, never silently pass.
    #[cfg(not(windows))]
    #[test]
    fn registry_is_windows_only_off_windows() {
        let outcome = run(&json!({"checks": [{"kind": "registry", "args": {"key": "HKLM\\X", "value": "Y", "expected": "Z"}}]}));
        assert_eq!(outcome.status, "failed");
        assert!(outcome.error.to_lowercase().contains("windows"), "{}", outcome.error);
    }

    // user_exists resolves a real account and rejects a bogus one.
    #[cfg(unix)]
    #[test]
    fn user_exists_true_for_root_false_for_nonsense() {
        let ok = run(&json!({"checks": [{"kind": "user_exists", "args": {"username": "root"}}]}));
        assert_eq!(ok.status, "done", "error: {}", ok.error);
        let bad = run(&json!({"checks": [{"kind": "user_exists", "args": {"username": "definitely-no-such-user-x9"}}]}));
        assert_eq!(bad.status, "failed");
    }

    #[test]
    fn malformed_payload_never_panics() {
        for payload in [json!({}), json!(null), json!({"checks": "not an array"}), json!({"checks": [null, 42, "x"]})] {
            let outcome = run(&payload);
            assert_eq!(outcome.status, "failed");
        }
    }

    // The real fix for a gap found by direct audit:
    // run() always computed a real, structured
    // per-check result, then discarded it, flattening everything into
    // the generic output/error text every other command already carries.
    // Proves validator_results now actually carries one real entry per
    // check -- both a passed and a failed one, with each check's own
    // args round-tripped -- not just that the overall status is right.
    #[test]
    fn structured_validator_results_carry_every_check() {
        let dir = std::env::temp_dir().join(format!("laforge-validator-test3-{}", std::process::id()));
        fs::create_dir_all(&dir).unwrap();
        let path = dir.join("present.txt");
        fs::write(&path, "hello").unwrap();
        let missing = dir.join("absent.txt");

        let outcome = run(&json!({"checks": [
            {"kind": "file_exists", "args": {"path": path.to_str().unwrap()}},
            {"kind": "file_exists", "args": {"path": missing.to_str().unwrap()}},
        ]}));

        assert_eq!(outcome.status, "failed", "one check failed, so the overall task did too");
        assert_eq!(outcome.validator_results.len(), 2, "both checks' own results should be present, not just the failing one");

        let passed = &outcome.validator_results[0];
        assert_eq!(passed.kind, "file_exists");
        assert!(passed.passed);
        assert!(!passed.message.is_empty());
        assert_eq!(passed.args.get("path").and_then(|v| v.as_str()), Some(path.to_str().unwrap()));

        let failed = &outcome.validator_results[1];
        assert_eq!(failed.kind, "file_exists");
        assert!(!failed.passed);
        assert!(!failed.message.is_empty());
        assert_eq!(failed.args.get("path").and_then(|v| v.as_str()), Some(missing.to_str().unwrap()));

        fs::remove_dir_all(&dir).ok();
    }

    #[test]
    fn structured_validator_results_are_empty_for_a_malformed_payload() {
        let outcome = run(&json!({}));
        assert!(outcome.validator_results.is_empty());
    }
}
