package agentfactory

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// recomputeMaskedHashAt reproduces agent/src/selfhash.rs's runtime
// computation in Go: given the sealed binary and the layout region's offset
// (captured before sealing, while its marker still existed), read the four
// offsets, zero the identity and hash regions, and SHA-256 the result. The
// whole point of the seal is that this equals the 32 bytes stored in the
// hash slot -- for the sealed binary AND after a per-host identity is patched
// over it.
func recomputeMaskedHashAt(bin []byte, layoutOff int) (got [32]byte, stored []byte) {
	idOff := int(binary.BigEndian.Uint32(bin[layoutOff : layoutOff+4]))
	idLen := int(binary.BigEndian.Uint32(bin[layoutOff+4 : layoutOff+8]))
	hashOff := int(binary.BigEndian.Uint32(bin[layoutOff+8 : layoutOff+12]))
	hashLen := int(binary.BigEndian.Uint32(bin[layoutOff+12 : layoutOff+16]))
	canon := make([]byte, len(bin))
	copy(canon, bin)
	zeroRange(canon, idOff, idLen)
	zeroRange(canon, hashOff, hashLen)
	got = sha256.Sum256(canon)
	stored = bin[hashOff : hashOff+32]
	return
}

// TestSelfHashSealRoundTrips is the cross-platform correctness proof, against
// a real compiled agent: seal it, then confirm the value the seal stored is
// exactly what the agent recomputes -- both for the sealed binary and after a
// per-host identity is patched over it (which must NOT invalidate the hash,
// since the identity region is masked). No process is run, so it needs no
// code signing and works on every OS.
func TestSelfHashSealRoundTrips(t *testing.T) {
	bin := realAgentBinary(t)
	base, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}

	// The base must carry exactly one of each region, all still full marker.
	layouts := findMarkerRegions(base, layoutMarker, selfHashRegion)
	if len(layouts) != 1 {
		t.Fatalf("found %d self-hash layout regions in the base, want 1 -- selfhash.rs and selfhash.go may have drifted", len(layouts))
	}
	if got := len(findMarkerRegions(base, hashMarker, selfHashRegion)); got != 1 {
		t.Fatalf("found %d self-hash regions in the base, want 1", got)
	}
	layoutOff := layouts[0]

	sealed, err := SelfHashSeal(base)
	if err != nil {
		t.Fatalf("SelfHashSeal: %v", err)
	}
	if len(sealed) != len(base) {
		t.Fatalf("sealing changed the size: %d -> %d", len(base), len(sealed))
	}
	if bytes.Equal(sealed, base) {
		t.Fatal("sealed binary is byte-identical to the base -- nothing was sealed")
	}
	// The layout region is no longer raw marker (it now holds offsets).
	if isFullMarkerRun(sealed[layoutOff:layoutOff+selfHashRegion], layoutMarker) {
		t.Fatal("layout region is still all-marker after sealing")
	}

	got, stored := recomputeMaskedHashAt(sealed, layoutOff)
	if !bytes.Equal(got[:], stored) {
		t.Fatalf("sealed hash mismatch: agent would recompute %x, seal stored %x", got, stored)
	}

	// Patch a per-host identity over the sealed base; the hash must still
	// verify, because the identity region is masked out of it.
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nnotreallyacert\n-----END CERTIFICATE-----\n")
	certPEM := caPEM
	keyPEM := []byte("-----BEGIN EC PRIVATE KEY-----\nnotreallyakey\n-----END EC PRIVATE KEY-----\n")
	patched, err := PatchBinary(sealed, "127.0.0.1:1", caPEM, certPEM, keyPEM, false)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	got2, stored2 := recomputeMaskedHashAt(patched, layoutOff)
	if !bytes.Equal(got2[:], stored2) {
		t.Fatal("after patching a per-host identity, the self-hash no longer verifies -- the identity region is not being masked correctly")
	}
}

// TestSelfHashSealIsIdempotent: sealing an already-sealed binary is a no-op
// (its layout marker is gone), so callers -- the factory build step AND
// agentdelivery, which both seal -- can apply it without double-sealing.
func TestSelfHashSealIsIdempotent(t *testing.T) {
	bin := realAgentBinary(t)
	base, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}
	sealed, err := SelfHashSeal(base)
	if err != nil {
		t.Fatalf("SelfHashSeal: %v", err)
	}
	again, err := SelfHashSeal(sealed)
	if err != nil {
		t.Fatalf("second SelfHashSeal: %v", err)
	}
	if !bytes.Equal(again, sealed) {
		t.Fatal("sealing an already-sealed binary changed it -- SelfHashSeal is not idempotent")
	}
}

// TestSealedAgentReportsNoTamper closes the loop end to end: a sealed +
// identity-patched binary, run for real, does NOT print the self-hash tamper
// finding. Skipped on macOS, where a patched Mach-O must be re-signed to run
// at all and re-signing itself rewrites bytes the self-hash covers (a
// local-testing artifact with no equivalent on the real Linux/Windows
// targets); TestSelfHashSealRoundTrips covers the logic cross-platform.
func TestSealedAgentReportsNoTamper(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS re-signs a patched binary, which alters bytes the self-hash covers; the round-trip test covers this cross-platform")
	}
	patchedPath := sealedPatchedAgent(t)
	cmd := exec.Command(patchedPath)
	cmd.Env = []string{}
	out, _ := runWithTimeout(t, cmd)
	if len(out) == 0 {
		t.Fatal("sealed agent produced no output -- it may have been killed before running")
	}
	if bytes.Contains(out, []byte("does not match the hash sealed")) {
		t.Fatalf("a correctly sealed binary reported tampering. Output:\n%s", out)
	}
}

// TestTamperedAgentReportsTamper is the positive: flip one code byte of a
// sealed + patched binary (outside the masked regions) and confirm the agent
// flags it. Same macOS skip as above.
func TestTamperedAgentReportsTamper(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS re-signing interferes with post-seal byte edits; round-trip test covers the logic cross-platform")
	}
	patchedPath := sealedPatchedAgent(t)
	data, err := os.ReadFile(patchedPath)
	if err != nil {
		t.Fatalf("reading sealed agent: %v", err)
	}
	// Flip a byte early in the file (ELF header/code area) -- well clear of
	// the identity and hash regions, which live in a data section far in.
	data[0x400] ^= 0xFF
	if err := os.WriteFile(patchedPath, data, 0o755); err != nil {
		t.Fatalf("writing tampered agent: %v", err)
	}
	cmd := exec.Command(patchedPath)
	cmd.Env = []string{}
	out, _ := runWithTimeout(t, cmd)
	if !bytes.Contains(out, []byte("does not match the hash sealed")) {
		t.Fatalf("a tampered binary did not report tampering. Output:\n%s", out)
	}
}

// sealedPatchedAgent builds a real agent, seals it, patches a throwaway
// identity, writes it out executable, and returns the path.
func sealedPatchedAgent(t *testing.T) string {
	t.Helper()
	bin := realAgentBinary(t)
	base, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("reading %s: %v", bin, err)
	}
	sealed, err := SelfHashSeal(base)
	if err != nil {
		t.Fatalf("SelfHashSeal: %v", err)
	}
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nnotreallyacert\n-----END CERTIFICATE-----\n")
	keyPEM := []byte("-----BEGIN EC PRIVATE KEY-----\nnotreallyakey\n-----END EC PRIVATE KEY-----\n")
	patched, err := PatchBinary(sealed, "127.0.0.1:1", caPEM, caPEM, keyPEM, false)
	if err != nil {
		t.Fatalf("PatchBinary: %v", err)
	}
	path := filepath.Join(t.TempDir(), "laforge-agent-sealed")
	if err := os.WriteFile(path, patched, 0o755); err != nil {
		t.Fatalf("writing sealed agent: %v", err)
	}
	return path
}
