// laforge-agent-factory is the agent factory: compile the Rust
// agent, then patch a per-host identity into a copy of it. See
// internal/agentfactory for the actual patching mechanism. What's
// currently simplified: compiling once per platform, not yet once per
// team per platform.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/globalcptc/laforge/internal/agentfactory"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "patch":
		err = runPatch(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `laforge-agent-factory

  laforge-agent-factory build --agent-dir <path> --target <rust-target-triple> --out <path> [--team <team-id>]
      Cross-compile the agent once for one platform (cargo build --release
      --target ...). With --team, also sets LAFORGE_TEAM_SALT (derived via
      internal/agentfactory.TeamSalt) so the resulting binary carries real,
      compile-time structural distinctness for that team -- see
      agent/build.rs. Without --team,
      behaves exactly as before (a fixed dev salt agent/build.rs falls
      back to), for every existing caller that never passes it.

  laforge-agent-factory patch --base <path> --out <path> --gateway-addr <host:port> --ca-cert <path> --client-cert <path> --client-key <path> [--debug true]
      Patch a per-host identity into a copy of a compiled agent binary.
      Milliseconds: no recompilation, just an in-memory byte region
      overwrite. --debug true bakes in the agent-debug flag (a local debug
      log beside the binary); default/omitted leaves the agent silent.`)
}

func runBuild(args []string) error {
	fs := newFlagSet(args, "build", []string{"agent-dir", "target", "out", "team"})
	agentDir := fs["agent-dir"]
	target := fs["target"]
	out := fs["out"]
	team := fs["team"]
	if agentDir == "" || target == "" || out == "" {
		return fmt.Errorf("usage: laforge-agent-factory build --agent-dir <path> --target <triple> --out <path> [--team <team-id>]")
	}

	cmd := exec.Command("cargo", "build", "--release", "--target", target)
	cmd.Dir = agentDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if team != "" {
		// Real per-team structural distinctness: a per-team compile-time
		// salt, baked into the binary
		// by agent/build.rs, not just the per-host identity patch that
		// runPatch below performs afterward. This is what makes "compile
		// once per team per platform, then patch per host" -- the
		// intended shape -- real: the caller (an
		// orchestration step outside this tool, invoking `build` once
		// per team and `patch` once per host of that team's build)
		// already matches that shape; this is what makes each of those
		// per-team compiles genuinely different from each other rather
		// than incidentally identical.
		salt := agentfactory.TeamSalt(team)
		cmd.Env = append(os.Environ(), "LAFORGE_TEAM_SALT="+salt)
		fmt.Printf("building for team %q (salt %s...)\n", team, salt[:12])
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo build: %w", err)
	}

	binName := "laforge-agent"
	if target == "x86_64-pc-windows-gnu" {
		binName += ".exe"
	}
	src := filepath.Join(agentDir, "target", target, "release", binName)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading built binary %s: %w", src, err)
	}
	// Seal pass: patch the self-hash layout + expected hash now, once per
	// team build, so every host patched from this base carries a valid
	// self-hash (the identity region is masked out of the hash). A no-op for
	// an agent without the self-hash statics.
	sealed, err := agentfactory.SelfHashSeal(data)
	if err != nil {
		return fmt.Errorf("sealing self-hash: %w", err)
	}
	data = sealed
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, data, 0o755); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("built %s (%d bytes) -> %s\n", target, len(data), out)
	return nil
}

func runPatch(args []string) error {
	fs := newFlagSet(args, "patch", []string{"base", "out", "gateway-addr", "ca-cert", "client-cert", "client-key", "debug"})
	base, out := fs["base"], fs["out"]
	gatewayAddr := fs["gateway-addr"]
	caCertPath, clientCertPath, clientKeyPath := fs["ca-cert"], fs["client-cert"], fs["client-key"]
	debug := fs["debug"] == "true" // optional: bake in the agent-debug flag (local debug log)
	if base == "" || out == "" || gatewayAddr == "" || caCertPath == "" || clientCertPath == "" || clientKeyPath == "" {
		return fmt.Errorf("usage: laforge-agent-factory patch --base <path> --out <path> --gateway-addr <host:port> --ca-cert <path> --client-cert <path> --client-key <path> [--debug true]")
	}

	baseData, err := os.ReadFile(base)
	if err != nil {
		return fmt.Errorf("reading base binary %s: %w", base, err)
	}
	caPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("reading ca-cert %s: %w", caCertPath, err)
	}
	certPEM, err := os.ReadFile(clientCertPath)
	if err != nil {
		return fmt.Errorf("reading client-cert %s: %w", clientCertPath, err)
	}
	keyPEM, err := os.ReadFile(clientKeyPath)
	if err != nil {
		return fmt.Errorf("reading client-key %s: %w", clientKeyPath, err)
	}

	patched, err := agentfactory.PatchBinary(baseData, gatewayAddr, caPEM, certPEM, keyPEM, debug)
	if err != nil {
		return fmt.Errorf("patching: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(out, patched, 0o755); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("patched %s (%d bytes) -> %s\n", base, len(patched), out)
	return nil
}

// newFlagSet is a tiny --key value parser -- this tool's flags are all
// required strings, so the stdlib flag package's extra ceremony (a
// FlagSet, a pointer per flag) buys nothing over a plain map here.
func newFlagSet(args []string, cmdName string, known []string) map[string]string {
	knownSet := make(map[string]bool, len(known))
	for _, k := range known {
		knownSet[k] = true
	}
	out := make(map[string]string)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) < 3 || a[:2] != "--" {
			continue
		}
		name := a[2:]
		if !knownSet[name] {
			fmt.Fprintf(os.Stderr, "laforge-agent-factory %s: unknown flag --%s\n", cmdName, name)
			continue
		}
		if i+1 < len(args) {
			out[name] = args[i+1]
			i++
		}
	}
	return out
}
