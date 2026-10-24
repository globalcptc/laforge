// Self-hash anti-tamper: the agent hashes its own on-disk binary at
// runtime and compares it against a value the factory sealed in after
// compilation (internal/agentfactory.SelfHashSeal). Like every check in
// antitamper.rs this only ever produces a finding to log -- it never
// refuses to run, exits, or changes behavior (see antitamper.rs's top doc
// comment: obfuscation buys time, it is not the boundary).
//
// The delivered binary legitimately varies in two byte ranges, which are
// therefore EXCLUDED from the hash by zeroing them before hashing --
// identically here and on the factory side:
//   - the per-host identity region (identity.rs's IDENTITY_BLOB, patched
//     per host at delivery time), and
//   - the hash slot itself (self-referential: it holds the answer).
// The offsets/lengths of both ranges are read from SELFHASH_LAYOUT, which
// the factory fills in during the seal pass, so the agent needs no ELF/PE
// parsing to locate them. SELFHASH_LAYOUT is itself part of the file and
// NOT masked -- it's written once at seal time and never changes again, so
// it's stable input to the hash.
//
// A binary that was never sealed (a plain `cargo run` dev build, or a base
// built without the factory) leaves SELFHASH_LAYOUT as its raw marker; the
// check detects that and skips silently -- there is no expected hash to
// compare against.
//
// MARKER bytes and region sizes here MUST match
// internal/agentfactory/selfhash.go exactly. As with identity.rs there's
// deliberately no shared source of truth (Go can't import Rust), so the
// factory's tests round-trip against a real sealed binary rather than
// trusting the two definitions agree in the abstract.

use ring::digest;
use std::hint::black_box;

// Distinct from identity.rs's own 0xDEADBEEF marker so the three regions
// (identity / hash / layout) are never confused for one another by the
// factory's per-marker scans.
const HASH_MARKER: [u8; 4] = [0x5E, 0x1F, 0xC0, 0xDE];
const LAYOUT_MARKER: [u8; 4] = [0x1A, 0x1A, 0x00, 0x17];

// Both regions are a fixed 256 bytes of repeated marker in the unsealed
// binary -- long enough (64 repeats) that a coincidental identical run in
// compiled code is vanishingly unlikely, so the factory's "expect exactly
// one" scan is reliable. The hash region carries a 32-byte SHA-256 in its
// first bytes; the whole region is masked from the hash. The layout region
// carries four big-endian u32s (identity offset, identity length, hash
// offset, hash length) in its first 16 bytes and is NOT masked.
const REGION: usize = 256;
const SHA256_LEN: usize = 32;

const fn marker_fill(m: [u8; 4]) -> [u8; REGION] {
    let mut arr = [0u8; REGION];
    let mut i = 0;
    while i < REGION {
        arr[i] = m[i % 4];
        i += 1;
    }
    arr
}

#[no_mangle]
pub static SELFHASH_EXPECTED: [u8; REGION] = marker_fill(HASH_MARKER);

#[no_mangle]
pub static SELFHASH_LAYOUT: [u8; REGION] = marker_fill(LAYOUT_MARKER);

fn be_u32(b: &[u8]) -> usize {
    u32::from_be_bytes([b[0], b[1], b[2], b[3]]) as usize
}

fn is_unsealed(region: &[u8], marker: [u8; 4]) -> bool {
    region.chunks(4).all(|c| c == marker)
}

fn zero_range(data: &mut [u8], off: usize, len: usize) {
    let end = off.saturating_add(len).min(data.len());
    if off < end {
        for b in &mut data[off..end] {
            *b = 0;
        }
    }
}

/// self_hash_finding hashes the running binary with the identity and hash
/// regions masked and compares it to the sealed-in expected value, returning
/// a finding string on mismatch. Returns None when everything matches, when
/// the binary was never sealed, or when the binary can't be read at all (a
/// sandbox that hides current_exe is not itself evidence of tampering).
pub fn self_hash_finding() -> Option<String> {
    // black_box the static reads so the optimizer can't fold in the known
    // marker initializer and skip the real memory load of the sealed value.
    let layout = black_box(&SELFHASH_LAYOUT);
    let expected = black_box(&SELFHASH_EXPECTED);

    if is_unsealed(layout, LAYOUT_MARKER) {
        return None; // never sealed -- nothing to check (dev build, unsealed base)
    }

    let exe = std::env::current_exe().ok()?;
    let mut data = std::fs::read(&exe).ok()?;

    let id_off = be_u32(&layout[0..4]);
    let id_len = be_u32(&layout[4..8]);
    let hash_off = be_u32(&layout[8..12]);
    let hash_len = be_u32(&layout[12..16]);
    zero_range(&mut data, id_off, id_len);
    zero_range(&mut data, hash_off, hash_len);

    let got = digest::digest(&digest::SHA256, &data);
    if got.as_ref() == &expected[0..SHA256_LEN] {
        None
    } else {
        Some("the agent's on-disk binary does not match the hash sealed at build time -- it may have been modified".to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unsealed_binary_is_skipped_not_flagged() {
        // A plain `cargo test` binary is never sealed, so the layout is
        // still all-marker and the check must return None (skip), not a
        // false tamper finding.
        assert!(self_hash_finding().is_none());
    }

    #[test]
    fn marker_regions_are_full_runs() {
        assert!(is_unsealed(&SELFHASH_LAYOUT, LAYOUT_MARKER));
        assert!(is_unsealed(&SELFHASH_EXPECTED, HASH_MARKER));
    }
}
