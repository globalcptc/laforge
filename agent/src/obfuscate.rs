// Lightweight, source-level control-flow obfuscation.
//
// The agent-hardening design calls for "optional
// control-flow obfuscation via LLVM-based passes (OLLVM-style)."
// Real OLLVM tooling -- a patched/plugin-capable LLVM with the actual
// bogus-control-flow, control-flow-flattening, and instruction-
// substitution passes -- is NOT available in this environment: it needs
// a plugin interface clang's LLVM exposes but rustc's own (differently
// versioned, differently packaged) LLVM backend doesn't expose the same
// way, and no such toolchain is installed here. Claiming OLLVM-level
// obfuscation would be dishonest, so this doesn't.
//
// What's here instead, applied to the one real hot spot worth obscuring
// (command dispatch, in commands.rs) rather than the whole program:
//   - Opaque predicates: branches that are always true (or always false)
//     by a real number-theory identity, seeded from a runtime value so
//     the compiler can't constant-fold them away, giving a disassembler
//     a genuine extra branch to resolve instead of one it can trivially
//     prove dead at a glance.
//   - A permuted function-pointer jump table in place of a plain `match`,
//     so the compiled mapping from a command name to the function that
//     handles it isn't laid out in declaration order, and reshuffles
//     itself across process runs (the permutation seed is a runtime
//     value, not a build-time constant).
//
// This is a real, if narrow, technique -- it raises the cost of static
// analysis a little. It is not control-flow flattening and it is not
// OLLVM; this comment says so rather than letting the name imply more.

use std::time::{SystemTime, UNIX_EPOCH};

/// An opaque predicate: always true, for any seed, but not obviously so
/// from a disassembly. n*n mod 4 is always 0 (n even) or 1 (n odd),
/// never 2 or 3 -- a basic, easily-verified number-theory identity, not
/// a trick that happens to hold for the test inputs below.
#[inline(never)]
pub fn opaque_true(seed: u64) -> bool {
    let n = seed.wrapping_mul(0x9E3779B97F4A7C15).wrapping_add(1);
    let r = n.wrapping_mul(n) % 4;
    r == 0 || r == 1
}

/// The inverse: always false, same reasoning (n*n mod 4 is never 2 or 3).
/// Kept even though nothing in this file calls it yet -- it's the
/// natural counterpart to opaque_true and a real, tested building block
/// for the next real branch this technique gets applied to.
#[inline(never)]
#[allow(dead_code)]
pub fn opaque_false(seed: u64) -> bool {
    let n = seed.wrapping_mul(0x9E3779B97F4A7C15).wrapping_add(1);
    let r = n.wrapping_mul(n) % 4;
    r == 2 || r == 3
}

/// runtime_seed returns a value that varies at process start and is not
/// knowable at compile time -- required for opaque_true/opaque_false's
/// "not constant-foldable" property to actually hold. A fixed literal
/// seed would let the optimizer prove the predicate at compile time and
/// delete the dead branch outright, defeating the point of having one.
pub fn runtime_seed() -> u64 {
    let nanos = SystemTime::now().duration_since(UNIX_EPOCH).map(|d| d.subsec_nanos()).unwrap_or(0) as u64;
    nanos ^ (std::process::id() as u64).wrapping_mul(0x100000001b3)
}

/// A command-name -> handler jump table, permuted by a seed rather than
/// laid out in declaration or alphabetical order. The one concrete thing
/// this buys over a plain `match "execute" => ..., "write_file" => ...`:
/// a disassembler reading the compiled table can't assume the order it
/// sees corresponds to anything meaningful, and that order changes
/// across process runs (seed comes from runtime_seed(), not a constant).
pub struct JumpTable<T> {
    entries: Vec<(u32, T)>,
}

impl<T> JumpTable<T> {
    pub fn build(mut entries: Vec<(&str, T)>, seed: u64) -> Self {
        entries.sort_by_key(|(k, _)| hash_key(k, seed));
        JumpTable { entries: entries.into_iter().map(|(k, v)| (hash_key(k, seed), v)).collect() }
    }

    pub fn get(&self, key: &str, seed: u64) -> Option<&T> {
        let h = hash_key(key, seed);
        self.entries.iter().find(|(k, _)| *k == h).map(|(_, v)| v)
    }
}

fn hash_key(key: &str, seed: u64) -> u32 {
    let mut state = seed ^ 0xcbf29ce484222325;
    for b in key.as_bytes() {
        state ^= *b as u64;
        state = state.wrapping_mul(0x100000001b3);
    }
    ((state >> 32) as u32) ^ (state as u32)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn opaque_true_holds_for_many_seeds() {
        for seed in [0u64, 1, 2, 42, u64::MAX, 0xDEADBEEF, 123456789, u64::MAX / 3] {
            assert!(opaque_true(seed), "opaque_true({seed}) was false");
            assert!(!opaque_false(seed), "opaque_false({seed}) was true");
        }
    }

    #[test]
    fn jump_table_resolves_every_entry_for_a_fixed_seed() {
        let seed = 777u64;
        let table = JumpTable::build(vec![("a", 1), ("b", 2), ("c", 3)], seed);
        assert_eq!(table.get("a", seed), Some(&1));
        assert_eq!(table.get("b", seed), Some(&2));
        assert_eq!(table.get("c", seed), Some(&3));
        assert_eq!(table.get("nope", seed), None);
    }

    #[test]
    fn jump_table_lookup_requires_the_matching_seed() {
        // Not a security property (the seed isn't a secret -- it's
        // derived from process id/time, observable to anyone who can
        // already run the binary) -- just proves the table's layout is
        // genuinely seed-dependent, not a fixed hash in disguise.
        let table = JumpTable::build(vec![("a", 1)], 1);
        assert_eq!(table.get("a", 2), None, "lookup with a different seed should not resolve");
    }

    #[test]
    fn runtime_seed_is_nonzero_in_practice() {
        // Not a mathematical guarantee (a seed of exactly 0 is possible
        // in principle), just a sanity check that it's actually reading
        // real time/pid data rather than always returning a fixed value.
        assert_ne!(runtime_seed(), 0);
    }
}
