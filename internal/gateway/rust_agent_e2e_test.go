package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/agentfactory"
	"github.com/globalcptc/laforge/internal/db"
)

// buildRealAgentBinary compiles the actual agent/ Rust project (native,
// not cross-compiled -- this has to actually run on whatever machine is
// executing the test) and returns the path to the resulting binary.
// Skips (not fails) if cargo isn't available, matching
// internal/agentfactory's own real-binary tests.
func buildRealAgentBinary(t *testing.T) string {
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
	return filepath.Join(agentDir, "target", "release", "laforge-agent")
}

// TestRealRustAgentEndToEnd is the actual capstone: the REAL
// compiled Rust agent binary, patched with a REAL per-host identity by
// the REAL factory patcher, talking to the REAL Go gateway over REAL
// mTLS (both sides using their independent protocol implementations --
// agent/src/protocol.rs on one end, internal/agentproto on the other),
// backed by REAL Postgres via the gateway's actual restricted role. It
// heartbeats, receives a real task, actually writes a real file to disk,
// and reports success -- verified by reading that file back, not by
// trusting the agent's own claim.
func TestRealRustAgentEndToEnd(t *testing.T) {
	agentBin := buildRealAgentBinary(t)

	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	workDir := t.TempDir()
	targetFile := filepath.Join(workDir, "proof.txt")
	const expectedContent = "the real rust agent wrote this, for real"

	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "write_file",
		Payload: mustJSON(t, map[string]string{"path": targetFile, "content": expectedContent}),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}

	addr, ca, _ := startTestGatewayWithLease(t, 30*time.Second)

	// Note what buildRealAgentBinary's binary carries at this point: the
	// unpatched marker only. Everything below patches a REAL per-host
	// identity into a copy of it -- the patched binary needs no files on
	// disk at runtime (no LAFORGE_DEV_* env vars set either, see cmd.Env
	// below), only the identity baked into its own IDENTITY_BLOB.
	clientCert, clientKey, err := ca.IssueLeaf(host.deployedObjectID.String(), false, nil, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf: %v", err)
	}

	baseData, err := os.ReadFile(agentBin)
	if err != nil {
		t.Fatalf("reading agent binary: %v", err)
	}
	patched, err := agentfactory.PatchBinary(baseData, addr, ca.CertPEM, clientCert, clientKey)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	patchedPath := filepath.Join(t.TempDir(), "laforge-agent-e2e")
	if err := os.WriteFile(patchedPath, patched, 0o755); err != nil {
		t.Fatalf("writing patched binary: %v", err)
	}
	resignForMacOSTesting(t, patchedPath)

	cmd := exec.Command(patchedPath)
	cmd.Env = []string{} // no LAFORGE_DEV_* fallback -- must run on its patched identity alone
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting patched agent: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
		t.Logf("agent output:\n%s", logBuf.String())
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(targetFile)
		if err == nil && string(content) == expectedContent {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("the real agent never wrote %s: %v\nagent output so far:\n%s", targetFile, err, logBuf.String())
	}
	if string(content) != expectedContent {
		t.Fatalf("file content = %q, want %q", content, expectedContent)
	}

	// Confirm the loop closed all the way back through the gateway into
	// Postgres too, not just that the file landed.
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		task, err := adminQ.GetAgentTask(ctx, step0.ID)
		if err == nil && task.Status == "done" {
			session, err := adminQ.GetAgentSessionByDeployedObject(ctx, host.deployedObjectID)
			if err != nil {
				t.Fatalf("GetAgentSessionByDeployedObject: %v", err)
			}
			if session.CertFingerprint == "" {
				t.Fatal("agent_session has no cert_fingerprint recorded")
			}
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("agent_task %s never reached status=done in Postgres", step0.ID)
}

// TestRealRustAgentSupervisorEndToEnd is the native-container capstone: the REAL
// agent in SUPERVISOR mode (LAFORGE_SUPERVISE set, as the incus builder sets it
// for a native OCI container) must do BOTH jobs at once -- start and supervise
// the image's command AND run the normal gateway loop, so a container checks in
// and runs its steps exactly like a host. It proves the supervised command ran
// (a marker file it creates) and that the agent still heartbeated, took a task,
// and reported it done through the gateway into Postgres.
func TestRealRustAgentSupervisorEndToEnd(t *testing.T) {
	agentBin := buildRealAgentBinary(t)

	adminPool := openAdminPool(t)
	host := seedDeployedHost(t, adminPool)
	adminQ := db.New(adminPool)
	ctx := context.Background()

	workDir := t.TempDir()
	targetFile := filepath.Join(workDir, "proof.txt")
	supervisedProof := filepath.Join(workDir, "supervised.txt")
	const expectedContent = "the supervised rust agent still checked in"

	step0, err := adminQ.CreateAgentTask(ctx, db.CreateAgentTaskParams{
		DeployedObjectID: host.deployedObjectID, StepIndex: 0, Command: "write_file",
		Payload: mustJSON(t, map[string]string{"path": targetFile, "content": expectedContent}),
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}

	addr, ca, _ := startTestGatewayWithLease(t, 30*time.Second)

	clientCert, clientKey, err := ca.IssueLeaf(host.deployedObjectID.String(), false, nil, nil, time.Hour)
	if err != nil {
		t.Fatalf("IssueLeaf: %v", err)
	}
	baseData, err := os.ReadFile(agentBin)
	if err != nil {
		t.Fatalf("reading agent binary: %v", err)
	}
	patched, err := agentfactory.PatchBinary(baseData, addr, ca.CertPEM, clientCert, clientKey)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	patchedPath := filepath.Join(t.TempDir(), "laforge-agent-sup-e2e")
	if err := os.WriteFile(patchedPath, patched, 0o755); err != nil {
		t.Fatalf("writing patched binary: %v", err)
	}
	resignForMacOSTesting(t, patchedPath)

	cmd := exec.Command(patchedPath)
	// Supervisor mode: LAFORGE_SUPERVISE is the "image's command" -- here a
	// marker touch then a sleep so the app stays alive (its exit would stop the
	// agent). PATH is provided so the agent's `sh -c` finds a shell, exactly as a
	// real container image has one.
	cmd.Env = []string{
		"LAFORGE_SUPERVISE=touch " + supervisedProof + "; sleep 300",
		"PATH=/bin:/usr/bin",
	}
	// Own process group so teardown kills the agent AND its supervised child --
	// otherwise the child (sleep) inherits the log pipe and blocks cmd.Wait().
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting patched agent: %v", err)
	}
	defer func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // negative pid = the whole group
		cmd.Wait()
		t.Logf("agent output:\n%s", logBuf.String())
	}()

	// The supervised command must have run.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(supervisedProof); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := os.Stat(supervisedProof); err != nil {
		t.Fatalf("supervised command never ran (no %s): %v\nagent output:\n%s", supervisedProof, err, logBuf.String())
	}

	// AND the agent must still have checked in and done the task.
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(targetFile)
		if err == nil && string(content) == expectedContent {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	content, err := os.ReadFile(targetFile)
	if err != nil || string(content) != expectedContent {
		t.Fatalf("the supervised agent never completed its task (%s): err=%v\nagent output:\n%s", targetFile, err, logBuf.String())
	}

	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if task, err := adminQ.GetAgentTask(ctx, step0.ID); err == nil && task.Status == "done" {
			return // success: supervised the app AND checked in
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("agent_task %s never reached status=done -- supervisor mode broke check-in", step0.ID)
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}
	return b
}

// resignForMacOSTesting re-signs a binary with an ad-hoc signature so it
// can actually execute on this machine -- see the identical helper (and
// its full explanation) in internal/agentfactory/patch_test.go, duplicated
// here rather than exported since it's test-only, macOS-development-only
// plumbing, not something the packages under test should know about.
//
// findCargo is the same duplication for the same reason -- see
// internal/agentfactory/patch_test.go's copy for the full explanation of
// why it needs to build its own PATH rather than trusting the caller's
// shell.
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
