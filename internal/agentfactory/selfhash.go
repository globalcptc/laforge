// Self-hash seal: the factory-side counterpart to agent/src/selfhash.rs.
// After the agent is compiled, SelfHashSeal writes the layout of the two
// masked regions into SELFHASH_LAYOUT and patches SELFHASH_EXPECTED with a
// SHA-256 of the binary taken with those two regions zeroed:
//   - the per-host identity region (IDENTITY_BLOB), so a later per-host
//     PatchBinary never invalidates the hash, and
//   - the hash slot itself, which is self-referential.
// The agent recomputes exactly the same masked hash at runtime and compares
// (see selfhash.rs). This is the "seal pass" of the compile-then-seal build:
// run once per team build, before any per-host identity patch.
//
// The marker bytes and REGION size here MUST match agent/src/selfhash.rs
// exactly; SelfHashSealRoundTrips in selfhash_test.go round-trips against a
// real compiled agent binary rather than trusting the two agree in the
// abstract.
package agentfactory

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const selfHashRegion = 256 // matches selfhash.rs REGION

var (
	hashMarker   = []byte{0x5E, 0x1F, 0xC0, 0xDE} // SELFHASH_EXPECTED fill
	layoutMarker = []byte{0x1A, 0x1A, 0x00, 0x17} // SELFHASH_LAYOUT fill
)

// SelfHashSeal seals a freshly compiled agent binary: it records the masked
// regions' layout and patches in the expected hash. It is a no-op (returns
// the input unchanged) when the binary carries no unsealed layout region --
// either it has no self-hash statics at all (an older agent) or it was
// already sealed -- so callers can apply it defensively without tracking
// whether a given base was sealed already. Errors only when the binary is
// self-hash-capable but malformed (a layout region present without exactly
// one hash region and one identity region), which means the two sides have
// drifted and must not be shipped.
func SelfHashSeal(base []byte) ([]byte, error) {
	layouts := findMarkerRegions(base, layoutMarker, selfHashRegion)
	switch len(layouts) {
	case 0:
		return base, nil // no unsealed layout: no self-hash statics, or already sealed
	case 1:
		// proceed
	default:
		return nil, fmt.Errorf("found %d self-hash layout regions, expected exactly 1", len(layouts))
	}
	layoutOff := layouts[0]

	hashes := findMarkerRegions(base, hashMarker, selfHashRegion)
	if len(hashes) != 1 {
		return nil, fmt.Errorf("found %d self-hash regions, expected exactly 1 (binary has a layout region but no single hash region -- selfhash.rs/selfhash.go drift?)", len(hashes))
	}
	hashOff := hashes[0]

	idOff, err := FindMarkerRegion(base)
	if err != nil {
		return nil, fmt.Errorf("locating identity region for self-hash layout: %w", err)
	}

	out := make([]byte, len(base))
	copy(out, base)

	// Record the two masked ranges for the agent to read back: identity
	// (offset, BlobSize) then hash slot (offset, selfHashRegion).
	var layout [16]byte
	binary.BigEndian.PutUint32(layout[0:4], uint32(idOff))
	binary.BigEndian.PutUint32(layout[4:8], uint32(BlobSize))
	binary.BigEndian.PutUint32(layout[8:12], uint32(hashOff))
	binary.BigEndian.PutUint32(layout[12:16], uint32(selfHashRegion))
	copy(out[layoutOff:layoutOff+16], layout[:])

	// Hash a canonical copy with both masked ranges zeroed -- identical to
	// what the agent reconstructs at runtime. The layout region just written
	// is NOT masked: it's stable from here on and is legitimate hash input.
	canon := make([]byte, len(out))
	copy(canon, out)
	zeroRange(canon, idOff, BlobSize)
	zeroRange(canon, hashOff, selfHashRegion)
	sum := sha256.Sum256(canon)

	copy(out[hashOff:hashOff+len(sum)], sum[:])
	return out, nil
}

func zeroRange(data []byte, off, length int) {
	end := off + length
	if end > len(data) {
		end = len(data)
	}
	for i := off; i < end; i++ {
		data[i] = 0
	}
}
