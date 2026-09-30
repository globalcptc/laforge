// The per-host identity: gateway address, the CA the agent pins (so it
// only ever trusts the real gateway, not just any TLS server), and this
// host's own client certificate + key. "A per-host identity/token baked
// into the binary at build time, by the agent factory." Nothing here is
// ever read from a file or an environment variable in a real build --
// IDENTITY_BLOB is patched directly into the compiled binary's data
// section by cmd/laforge-agent-factory (see that tool).
//
// XOR obfuscation only, matching a proven baseline
// (strip + byte-array obfuscation + static linking passed the `strings`
// test) -- "obfuscation buys time, it is not the boundary." The
// real boundary is that the gateway only accepts a certificate signed by
// its own pinned CA, which XOR does nothing to weaken or strengthen.

use std::env;
use std::fs;

const XOR_KEY: u8 = 0x5A;

/// BLOB_SIZE has to be big enough for four real PEM blocks (a CA cert,
/// this host's own cert, and its private key, all base64-armored text)
/// plus the gateway address -- ECDSA P-256 keeps each PEM small (a few
/// hundred bytes), so 16KiB is generous headroom, not a tight fit.
const BLOB_SIZE: usize = 16 * 1024;

/// MARKER is what cmd/laforge-agent-factory's patcher searches the
/// compiled binary for: BLOB_SIZE repeated copies of these 4 bytes,
/// contiguous. Chosen to be vanishingly unlikely to occur by coincidence
/// in compiled Rust/libc code -- the factory itself double-checks it
/// finds *exactly one* such run before ever patching, and refuses rather
/// than guessing if it finds zero or more than one.
const MARKER: [u8; 4] = [0xDE, 0xAD, 0xBE, 0xEF];

const fn make_marker_blob() -> [u8; BLOB_SIZE] {
    let mut arr = [0u8; BLOB_SIZE];
    let mut i = 0;
    while i < BLOB_SIZE {
        arr[i] = MARKER[i % 4];
        i += 1;
    }
    arr
}

#[no_mangle]
pub static IDENTITY_BLOB: [u8; BLOB_SIZE] = make_marker_blob();

pub struct Identity {
    pub gateway_addr: String,
    pub ca_pem: Vec<u8>,
    pub cert_pem: Vec<u8>,
    pub key_pem: Vec<u8>,
}

fn is_unpatched(data: &[u8]) -> bool {
    data.chunks(4).all(|c| c == MARKER)
}

fn deobfuscate(data: &[u8]) -> Vec<u8> {
    data.iter().map(|b| b ^ XOR_KEY).collect()
}

/// read_section reads one `[4-byte BE length][bytes]` section starting at
/// `pos`, returning the bytes and the position just past them. Returns
/// None on anything that doesn't fit -- a corrupt or truncated blob is a
/// clean "no identity," never a panic or an out-of-bounds read.
fn read_section(data: &[u8], pos: usize) -> Option<(Vec<u8>, usize)> {
    if pos + 4 > data.len() {
        return None;
    }
    let len = u32::from_be_bytes([data[pos], data[pos + 1], data[pos + 2], data[pos + 3]]) as usize;
    let start = pos + 4;
    let end = start.checked_add(len)?;
    if end > data.len() {
        return None;
    }
    Some((data[start..end].to_vec(), end))
}

fn parse_blob(deobfuscated: &[u8]) -> Option<Identity> {
    let (gw, pos) = read_section(deobfuscated, 0)?;
    let (ca, pos) = read_section(deobfuscated, pos)?;
    let (cert, pos) = read_section(deobfuscated, pos)?;
    let (key, _pos) = read_section(deobfuscated, pos)?;
    Some(Identity {
        gateway_addr: String::from_utf8(gw).ok()?,
        ca_pem: ca,
        cert_pem: cert,
        key_pem: key,
    })
}

/// encode_blob is the factory-side counterpart to parse_blob: builds the
/// XOR'd, length-prefixed byte sequence that gets written over
/// IDENTITY_BLOB's marker region. Exposed from this module (not
/// duplicated in the factory) so the wire format is defined in exactly
/// one place; cmd/laforge-agent-factory is a Go program and can't import
/// this directly, so it re-implements the same layout independently --
/// see that tool's own tests, which round-trip against a real patched
/// binary rather than trusting the two implementations agree in the
/// abstract.
#[allow(dead_code)]
pub fn encode_blob(gateway_addr: &str, ca_pem: &[u8], cert_pem: &[u8], key_pem: &[u8]) -> Result<[u8; BLOB_SIZE], String> {
    let mut out = Vec::with_capacity(BLOB_SIZE);
    for section in [gateway_addr.as_bytes(), ca_pem, cert_pem, key_pem] {
        out.extend_from_slice(&(section.len() as u32).to_be_bytes());
        out.extend_from_slice(section);
    }
    if out.len() > BLOB_SIZE {
        return Err(format!("identity blob is {} bytes, exceeds BLOB_SIZE {}", out.len(), BLOB_SIZE));
    }
    out.resize(BLOB_SIZE, 0);
    let obfuscated: Vec<u8> = out.iter().map(|b| b ^ XOR_KEY).collect();
    let mut arr = [0u8; BLOB_SIZE];
    arr.copy_from_slice(&obfuscated);
    Ok(arr)
}

/// load resolves this binary's identity: the patched blob if one is
/// present, otherwise a dev-only fallback reading LAFORGE_DEV_* env vars
/// -- exists purely so the agent can be exercised with `cargo run`
/// while iterating on it, never how a real deployed agent starts. A real
/// build's IDENTITY_BLOB is never left as the raw marker.
pub fn load() -> Option<Identity> {
    if !is_unpatched(&IDENTITY_BLOB) {
        let deob = deobfuscate(&IDENTITY_BLOB);
        if let Some(id) = parse_blob(&deob) {
            return Some(id);
        }
        eprintln!("identity: IDENTITY_BLOB is patched but failed to parse -- refusing to fall back to dev env vars for a build that was clearly meant to carry a real identity");
        return None;
    }
    load_dev_fallback()
}

fn load_dev_fallback() -> Option<Identity> {
    let gateway_addr = env::var("LAFORGE_DEV_GATEWAY_ADDR").ok()?;
    let ca_path = env::var("LAFORGE_DEV_CA_CERT").ok()?;
    let cert_path = env::var("LAFORGE_DEV_CLIENT_CERT").ok()?;
    let key_path = env::var("LAFORGE_DEV_CLIENT_KEY").ok()?;
    eprintln!("identity: IDENTITY_BLOB is unpatched -- using LAFORGE_DEV_* env vars (dev-only fallback, never how a real deployed agent starts)");
    Some(Identity {
        gateway_addr,
        ca_pem: fs::read(ca_path).ok()?,
        cert_pem: fs::read(cert_path).ok()?,
        key_pem: fs::read(key_path).ok()?,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn encode_then_parse_round_trip() {
        let blob = encode_blob("gw.example:8443", b"CA_PEM_BYTES", b"CERT_PEM_BYTES", b"KEY_PEM_BYTES").unwrap();
        assert!(!is_unpatched(&blob), "a real encoded blob must not look like the unpatched marker");
        let deob = deobfuscate(&blob);
        let id = parse_blob(&deob).expect("parse_blob should succeed on a freshly encoded blob");
        assert_eq!(id.gateway_addr, "gw.example:8443");
        assert_eq!(id.ca_pem, b"CA_PEM_BYTES");
        assert_eq!(id.cert_pem, b"CERT_PEM_BYTES");
        assert_eq!(id.key_pem, b"KEY_PEM_BYTES");
    }

    #[test]
    fn unpatched_marker_is_detected() {
        assert!(is_unpatched(&IDENTITY_BLOB));
    }

    #[test]
    fn oversized_identity_is_rejected_not_truncated_silently() {
        let huge = vec![b'x'; BLOB_SIZE]; // way over budget once length-prefixed
        let err = encode_blob("gw:1", &huge, &huge, &huge);
        assert!(err.is_err());
    }
}
