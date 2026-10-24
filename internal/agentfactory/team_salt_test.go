package agentfactory

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTeamSaltDeterministicAndDistinctAcrossTeams(t *testing.T) {
	a1 := TeamSalt("team-alpha")
	a2 := TeamSalt("team-alpha")
	b := TeamSalt("team-bravo")

	if a1 != a2 {
		t.Fatalf("TeamSalt is not deterministic: %q != %q for the same team id", a1, a2)
	}
	if a1 == b {
		t.Fatal("TeamSalt produced the same salt for two different team ids")
	}
	if len(a1) != 64 { // hex-encoded SHA-256
		t.Fatalf("expected a 64-char hex salt, got %d chars: %q", len(a1), a1)
	}
}

func TestTeamSaltDiffersFromPlainTeamID(t *testing.T) {
	// Not load-bearing for anything downstream, but confirms TeamSalt
	// doesn't just echo its input back (which would make it no better
	// than passing the raw team id straight through as the salt).
	if TeamSalt("team-x") == "team-x" {
		t.Fatal("TeamSalt returned its input unchanged")
	}
}

// buildRealAgentBinaryWithTeamSalt is realAgentBinary's (patch_test.go)
// sibling: builds the real agent/ Rust project with LAFORGE_TEAM_SALT
// set to salt (or left unset if salt == ""), returning the built
// binary's own bytes -- copied out before a later call reuses and
// overwrites the same target/release/laforge-agent path. Skips if cargo
// isn't available, matching every other real-binary test in this
// package.
func buildRealAgentBinaryWithTeamSalt(t *testing.T, salt string) []byte {
	t.Helper()
	cargo, env := findCargo(t)
	agentDir, err := filepath.Abs("../../agent")
	if err != nil {
		t.Fatal(err)
	}
	if salt != "" {
		env = append(env, "LAFORGE_TEAM_SALT="+salt)
	}
	cmd := exec.Command(cargo, "build", "--release")
	cmd.Dir = agentDir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargo build --release (salt=%q): %v\n%s", salt, err, out)
	}
	bin := filepath.Join(agentDir, "target", "release", "laforge-agent")
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading built binary: %v", err)
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp
}

// TestTeamSaltProducesStructurallyDistinctBinaries is the real proof
// that two teams' compiled agent binaries
// (built through the exact mechanism cmd/laforge-agent-factory's
// `build --team` flag drives -- TeamSalt -> LAFORGE_TEAM_SALT ->
// agent/build.rs) differ in far more than the BlobSize identity region
// internal/agentfactory.PatchBinary overwrites per host. Real cargo
// builds, not a synthetic byte comparison -- slow (three real release
// builds), matching this package's existing real-binary tests'
// willingness to pay that cost for a real proof.
func TestTeamSaltProducesStructurallyDistinctBinaries(t *testing.T) {
	binDefault := buildRealAgentBinaryWithTeamSalt(t, "")
	binA := buildRealAgentBinaryWithTeamSalt(t, TeamSalt("team-alpha"))
	binB := buildRealAgentBinaryWithTeamSalt(t, TeamSalt("team-bravo"))

	if bytes.Equal(binDefault, binA) {
		t.Fatal("a team-salted build is byte-identical to the no-salt default build -- LAFORGE_TEAM_SALT had no effect")
	}
	if bytes.Equal(binA, binB) {
		t.Fatal("two different teams' builds are byte-identical -- per-team structural distinctness isn't real")
	}

	diffCount := 0
	n := len(binA)
	if len(binB) < n {
		n = len(binB)
	}
	for i := 0; i < n; i++ {
		if binA[i] != binB[i] {
			diffCount++
		}
	}
	lenDiff := len(binA) - len(binB)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	// The real property item 4 cares about: two teams' binaries must
	// differ in far more than what PatchBinary would later patch (the
	// BlobSize identity region) -- confirming this isn't just moving the
	// goalposts, but a genuine change to the binary's own structure.
	if diffCount+lenDiff < BlobSize*4 {
		t.Fatalf("only %d differing bytes (plus a %d byte length difference) between two teams' builds -- expected far more than BlobSize (%d), since this is supposed to be structural, not just a patched region",
			diffCount, lenDiff, BlobSize)
	}
}
