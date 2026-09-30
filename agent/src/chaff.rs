// Thin wrapper around the per-team structural-distinctness blob build.rs
// generates from LAFORGE_TEAM_SALT (see that file's own doc comment).
// CHAFF_BLOB's bytes
// and the chaff_N functions' constants carry no meaning of their own --
// they exist only to vary this binary's compiled size and layout per
// team, the same way two different source files compile to two
// different binaries. perturb() is the one real use of them (main.rs
// folds it into the reconnect-backoff jitter), which is also what keeps
// LTO from discarding them as dead code.
include!(concat!(env!("OUT_DIR"), "/chaff.rs"));

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn perturb_is_deterministic_for_a_fixed_seed() {
        assert_eq!(perturb(42), perturb(42));
        assert_eq!(perturb(0), perturb(0));
    }

    #[test]
    fn perturb_varies_with_seed() {
        // Not a mathematical guarantee for every possible pair (it's a
        // mixing function, not a bijection proof), but true for these
        // two concrete seeds, and a real regression check that perturb
        // isn't secretly a constant function.
        assert_ne!(perturb(1), perturb(2));
    }

    #[test]
    fn chaff_blob_is_within_build_rs_own_documented_bounds() {
        // build.rs generates a length in [2048, 2048+4096); this doesn't
        // pin an exact value (that would defeat the point -- it's
        // supposed to vary per LAFORGE_TEAM_SALT), just confirms the
        // generated file actually landed and is a real, non-trivial size.
        assert!(CHAFF_BLOB.len() >= 2048 && CHAFF_BLOB.len() < 2048 + 4096);
    }
}
