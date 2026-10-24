package agentfactory

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// runWithTimeout starts cmd, lets it run briefly (the agent's own
// connect-loop never exits on its own, by design -- "agents tolerate it
// being down... they retry"), then kills it and returns whatever it
// printed. Real process execution, not a mock -- just bounded, since this
// test only needs to observe the first few log lines.
func runWithTimeout(t *testing.T, cmd *exec.Cmd) ([]byte, error) {
	t.Helper()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting %s: %v", cmd.Path, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return buf.Bytes(), err
	case <-time.After(2 * time.Second):
		cmd.Process.Kill()
		<-done
		return buf.Bytes(), nil
	}
}

// realAgentBinary returns the path to a real, freshly-built (unpatched)
// laforge-agent binary -- `cargo build --release` in agent/, not a
// fixture. Skips if cargo isn't found anywhere rather than failing the
// whole Go suite over a missing Rust toolchain.
//
// findCargo locates cargo and builds an environment for running it that
// works regardless of whether the calling shell happened to have the
// rustup toolchain on PATH. `go test ./...` from a plain shell (no
// `export PATH=...rustup...` first) genuinely hit this: exec.LookPath
// found nothing, the Homebrew rustup shim fallback below found cargo
// itself, but cargo then failed with "could not execute process `rustc
// -vV`" because rustc's own directory still wasn't on PATH for the
// subprocess. Fixed by explicitly adding both the Homebrew rustup shim
// directory and every installed toolchain's bin/ directory (globbed,
// since the exact toolchain name like "stable-aarch64-apple-darwin" is
// machine-specific) to the child process's PATH, rather than assuming
// the caller's shell already set it up.
func findCargo(t *testing.T) (cargoPath string, env []string) {
	t.Helper()
	extra := []string{"/opt/homebrew/opt/rustup/bin"}
	if home, err := os.UserHomeDir(); err == nil {
		if matches, err := filepath.Glob(filepath.Join(home, ".rustup", "toolchains", "*", "bin")); err == nil {
			extra = append(extra, matches...)
		}
	}
	pathVal := strings.Join(extra, string(os.PathListSeparator)) + string(os.PathListSeparator) + os.Getenv("PATH")
	env = append(os.Environ(), "PATH="+pathVal)

	for _, dir := range extra {
		candidate := filepath.Join(dir, "cargo")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, env
		}
	}
	if found, err := exec.LookPath("cargo"); err == nil {
		return found, env
	}
	t.Skip("cargo not found (checked PATH and the usual Homebrew rustup locations) -- needs the Rust toolchain (see agent/)")
	return "", nil
}

func realAgentBinary(t *testing.T) string {
	t.Helper()
	cargo, env := findCargo(t)
	agentDir, err := filepath.Abs("../../agent")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(cargo, "build", "--release")
	cmd.Dir = agentDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargo build --release: %v\n%s", err, out)
	}
	bin := filepath.Join(agentDir, "target", "release", "laforge-agent")
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("expected binary at %s: %v", bin, err)
	}
	return bin
}

// TestFindMarkerRegionOnRealBinary proves the marker-scanning logic
// against the actual compiled Rust binary, not a synthetic byte slice --
// the whole risk this package exists to cover (the compiler relocating,
// deduplicating, or otherwise not laying out IDENTITY_BLOB the way source
// reading alone would suggest) can only be caught by looking at what
// rustc/the linker actually produced.
func TestFindMarkerRegionOnRealBinary(t *testing.T) {
	bin := realAgentBinary(t)
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}

	offset, err := FindMarkerRegion(data)
	if err != nil {
		t.Fatalf("FindMarkerRegion: %v", err)
	}
	if offset <= 0 || offset+BlobSize > len(data) {
		t.Fatalf("offset %d is out of range for a %d-byte binary", offset, len(data))
	}
	if !isFullMarkerRun(data[offset:offset+BlobSize], marker) {
		t.Fatal("the region FindMarkerRegion returned is not actually a full marker run")
	}
}

// TestPatchRealBinaryAndVerifyNoPlaintextLeak is the hardening
// check: patch a real binary with a real (throwaway, test-only) identity,
// then run `strings` over the RESULT and confirm none of the sensitive
// plaintext -- the gateway address, the PEM headers, the raw key bytes --
// is recoverable. This is the literal test a competitor would run first
// against a captured agent binary.
func TestPatchRealBinaryAndVerifyNoPlaintextLeak(t *testing.T) {
	if _, err := exec.LookPath("strings"); err != nil {
		t.Skip("strings(1) not found on PATH")
	}
	bin := realAgentBinary(t)
	base, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}

	const gatewayAddr = "gw-secret-host-test.laforge.internal:8443"
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nFAKECADATAFORTESTONLY\n-----END CERTIFICATE-----\n")
	certPEM := []byte("-----BEGIN CERTIFICATE-----\nFAKECLIENTCERTDATAFORTESTONLY\n-----END CERTIFICATE-----\n")
	keyPEM := []byte("-----BEGIN EC PRIVATE KEY-----\nFAKEPRIVATEKEYDATAFORTESTONLY\n-----END EC PRIVATE KEY-----\n")

	patched, err := PatchBinary(base, gatewayAddr, caPEM, certPEM, keyPEM, false)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	if len(patched) != len(base) {
		t.Fatalf("patched binary is %d bytes, base was %d -- patching must not change the file size", len(patched), len(base))
	}
	if bytes.Equal(patched, base) {
		t.Fatal("patched binary is byte-identical to the base -- nothing was actually written")
	}

	patchedPath := filepath.Join(t.TempDir(), "laforge-agent-patched")
	if err := os.WriteFile(patchedPath, patched, 0o755); err != nil {
		t.Fatalf("writing patched binary: %v", err)
	}

	out, err := exec.Command("strings", patchedPath).CombinedOutput()
	if err != nil {
		t.Fatalf("strings: %v", err)
	}
	dump := string(out)

	for _, secret := range []string{
		gatewayAddr,
		"gw-secret-host-test",
		"FAKECADATAFORTESTONLY",
		"FAKECLIENTCERTDATAFORTESTONLY",
		"FAKEPRIVATEKEYDATAFORTESTONLY",
		"BEGIN CERTIFICATE",
		"BEGIN EC PRIVATE KEY",
	} {
		if containsPlaintext(dump, secret) {
			t.Errorf("strings output contains %q in plaintext -- the identity blob leaked", secret)
		}
	}
}

func containsPlaintext(haystack, needle string) bool {
	return len(needle) > 0 && bytes.Contains([]byte(haystack), []byte(needle))
}

// TestPatchedBinaryActuallyRunsWithItsPatchedIdentity closes the loop:
// not just "the plaintext isn't visible to strings," but "a patched
// binary, run for real, actually recovers the identity that was patched
// in." Since this agent has no LAFORGE_DEV_* env vars set and a real
// (patched, not marker) IDENTITY_BLOB, identity::load() must take the
// patched path, not the dev fallback -- observable from its attempt to
// dial the (fake, unreachable) gateway address that was patched in,
// which fails fast with a connection error rather than the "no identity
// available" message an unpatched binary prints.
func TestPatchedBinaryActuallyRunsWithItsPatchedIdentity(t *testing.T) {
	bin := realAgentBinary(t)
	base, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nnotreallyacert\n-----END CERTIFICATE-----\n")
	certPEM := []byte("-----BEGIN CERTIFICATE-----\nnotreallyacert\n-----END CERTIFICATE-----\n")
	keyPEM := []byte("-----BEGIN EC PRIVATE KEY-----\nnotreallyakey\n-----END EC PRIVATE KEY-----\n")
	// Port 1 is never a real listening gateway -- the point is only to
	// observe the agent try, and fail at TLS setup rather than report "no
	// identity available." Patch with agent-debug ON: a real (debug-off)
	// patched agent is deliberately SILENT (nothing to stdout/stderr), so the
	// debug log file beside the binary is how we observe what it did. Baking
	// debug here also proves the Go-encoded 5th blob section is read back as
	// debug=true by the Rust agent.
	patched, err := PatchBinary(base, "127.0.0.1:1", caPEM, certPEM, keyPEM, true)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	patchedPath := filepath.Join(t.TempDir(), "laforge-agent-patched-run")
	if err := os.WriteFile(patchedPath, patched, 0o755); err != nil {
		t.Fatalf("writing patched binary: %v", err)
	}
	resignForMacOSTesting(t, patchedPath)

	cmd := exec.Command(patchedPath)
	cmd.Env = []string{} // deliberately no LAFORGE_DEV_* vars -- must use the patched blob
	_, _ = runWithTimeout(t, cmd)

	// The agent writes its debug log next to the binary.
	logPath := filepath.Join(filepath.Dir(patchedPath), "laforge-agent.log")
	logData, _ := os.ReadFile(logPath)

	// A non-empty debug log is the proof the process actually ran: it loaded
	// the patched identity and reached TLS setup. An empty/absent log would
	// mean it never got that far -- e.g. SIGKILLed before main() ran, which
	// bit a real run of this test on macOS (PatchBinary's byte overwrite
	// invalidates the Mach-O ad-hoc signature and AMFI kills the process;
	// resignForMacOSTesting is the fix). Requiring non-empty output is what
	// catches that, rather than passing for the wrong reason.
	if len(logData) == 0 {
		t.Fatalf("patched binary wrote no debug log at %s -- it may have been killed before running (see resignForMacOSTesting's doc comment)", logPath)
	}
	if bytes.Contains(logData, []byte("no identity available")) {
		t.Fatalf("patched binary reported no identity available -- it should have loaded the patched blob. Log:\n%s", logData)
	}
	// Reaching TLS setup (which fails on the deliberately fake, non-PEM
	// cert/key -- acceptable here, we don't mint a real CA just to prove the
	// identity loaded) is exactly the "took the patched path, not the dev
	// fallback" signal we want.
	if !bytes.Contains(logData, []byte("building TLS config")) {
		t.Fatalf("expected the agent to reach TLS setup with the patched identity; log was:\n%s", logData)
	}
}

// resignForMacOSTesting re-signs a binary with an ad-hoc signature so it
// can actually execute on this machine. Needed ONLY because macOS's code
// signing enforcement (AMFI) SIGKILLs a Mach-O binary whose bytes were
// modified after the linker embedded its own ad-hoc signature -- exactly
// what PatchBinary does. Found by running a freshly patched binary
// directly and observing total silence and exit code 137 (SIGKILL), then
// confirming with `codesign -dv` that the embedded CodeDirectory hash was
// stale against the now-patched bytes.
//
// This has no equivalent on the real Linux/Windows targets (a plain ELF
// or PE binary carries no OS-enforced signature by default), so
// PatchBinary itself stays portable and never shells out to codesign --
// this exists purely to make local testing on a Mac possible, and is a
// no-op everywhere else.
func resignForMacOSTesting(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	out, err := exec.Command("codesign", "--sign", "-", "--force", path).CombinedOutput()
	if err != nil {
		t.Fatalf("codesign --sign - --force %s: %v\n%s", path, err, out)
	}
}
