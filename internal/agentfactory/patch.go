// Package agentfactory is "the agent factory": it
// compiles the Rust agent and patches a per-host identity into a copy of
// it. "The factory compiles per team per platform, then produces each
// host's binary by patching an encrypted identity blob into a copy of
// its team's build: milliseconds each."
//
// The factory compiles once per team per platform (TeamSalt, wired into
// cmd/laforge-agent-factory's `build --team` flag -- see team_salt.go)
// and patches per host, which is the part that has to be fast and is:
// BlobSize/16KiB overwritten in an in-memory copy, no recompilation.
// TeamSalt closes a real gap: a per-host identity
// patch alone can never make two teams' binaries structurally distinct
// (a byte-region overwrite can't change function layout or section
// sizes), only a real per-team compile can -- see agent/build.rs and
// agent/src/chaff.rs for what the salt actually does once it reaches the
// Rust side.
//
// BlobSize, the marker bytes, and the XOR key here MUST match
// agent/src/identity.rs exactly -- there is deliberately no shared
// source of truth between the two (Go can't import Rust code), so
// patch_test.go round-trips against a real compiled agent binary rather
// than trusting the two definitions agree in the abstract.
package agentfactory

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	BlobSize = 16 * 1024
	xorKey   = 0x5A
)

var marker = []byte{0xDE, 0xAD, 0xBE, 0xEF}

// FindMarkerRegion scans a compiled agent binary for the one contiguous
// BlobSize run of repeated marker bytes agent/src/identity.rs's
// IDENTITY_BLOB compiles down to, and returns its start offset. Errors --
// refuses to guess -- if it finds zero such runs (wrong binary, or the
// marker got optimized/relocated somehow) or more than one (which would
// make "patch it" ambiguous).
func FindMarkerRegion(data []byte) (int, error) {
	starts := findMarkerRegions(data, marker, BlobSize)
	switch len(starts) {
	case 0:
		return 0, fmt.Errorf("no identity marker region found in this binary -- was it really built from agent/src/identity.rs?")
	case 1:
		return starts[0], nil
	default:
		return 0, fmt.Errorf("found %d identity marker regions, expected exactly 1 -- refusing to guess which one to patch", len(starts))
	}
}

// findMarkerRegions returns the start offset of every contiguous `size`-byte
// run of the 4-byte marker repeated -- the compiled form of a marker-filled
// static (identity.rs's IDENTITY_BLOB, selfhash.rs's SELFHASH_* regions).
// Shared by FindMarkerRegion and the self-hash seal so both find their
// regions the exact same way.
func findMarkerRegions(data, marker []byte, size int) []int {
	var starts []int
	for i := 0; i+len(marker) <= len(data); i++ {
		if !bytes.Equal(data[i:i+len(marker)], marker) {
			continue
		}
		if i+size > len(data) || !isFullMarkerRun(data[i:i+size], marker) {
			continue
		}
		starts = append(starts, i)
		i += size - 1 // skip past this confirmed region rather than re-scanning it
	}
	return starts
}

func isFullMarkerRun(region, marker []byte) bool {
	for i := 0; i < len(region); i += len(marker) {
		if !bytes.Equal(region[i:i+len(marker)], marker) {
			return false
		}
	}
	return true
}

// EncodeBlob builds the same [len][bytes] x4, XOR'd layout
// agent/src/identity.rs's parse_blob expects: gateway address, CA cert
// PEM, this host's client cert PEM, this host's client key PEM, in that
// order.
func EncodeBlob(gatewayAddr string, caPEM, certPEM, keyPEM []byte) ([]byte, error) {
	var buf bytes.Buffer
	for _, section := range [][]byte{[]byte(gatewayAddr), caPEM, certPEM, keyPEM} {
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(section)))
		buf.Write(lenBuf[:])
		buf.Write(section)
	}
	if buf.Len() > BlobSize {
		return nil, fmt.Errorf("identity blob is %d bytes, exceeds BlobSize %d -- a cert/key grew unexpectedly large", buf.Len(), BlobSize)
	}
	out := make([]byte, BlobSize)
	copy(out, buf.Bytes())
	for i := range out {
		out[i] ^= xorKey
	}
	return out, nil
}

// PatchBinary returns a copy of baseBinary with its identity marker
// region overwritten with gatewayAddr/caPEM/certPEM/keyPEM, encoded and
// obfuscated exactly as the agent expects to read them back. baseBinary
// itself is never modified.
func PatchBinary(baseBinary []byte, gatewayAddr string, caPEM, certPEM, keyPEM []byte) ([]byte, error) {
	offset, err := FindMarkerRegion(baseBinary)
	if err != nil {
		return nil, err
	}
	blob, err := EncodeBlob(gatewayAddr, caPEM, certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(baseBinary))
	copy(out, baseBinary)
	copy(out[offset:offset+BlobSize], blob)
	return out, nil
}
