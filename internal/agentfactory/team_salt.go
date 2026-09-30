package agentfactory

import (
	"crypto/sha256"
	"encoding/hex"
)

// TeamSalt derives a per-team compile-time salt from a team identifier.
// Deterministic -- the same team ID always yields the same salt, so
// re-running "build --team X" (say, after extending a competition and
// needing to rebuild) doesn't gratuitously churn that team's binary for
// no reason -- and distinct across teams: a SHA-256 digest of a
// fixed-prefix + team ID, not a short or guessable scheme, so two
// different team IDs collide only with cryptographic-hash-collision
// probability, not by construction.
//
// Consumed entirely on the Rust side: cmd/laforge-agent-factory's `build
// --team <id>` flag is the real caller (see that file), setting the
// result as the LAFORGE_TEAM_SALT environment variable before invoking
// `cargo build`. agent/build.rs reads it and bakes a salt-derived,
// variable-length "chaff" blob and a variable number of junk functions
// into the compiled binary (see that file and agent/src/chaff.rs) --
// real structural distinctness per team, not just a different value in
// the same patched region PatchBinary overwrites afterward per host.
// This function itself does nothing with the salt beyond deriving it;
// it exists in Go only because the factory's own CLI (Go) is what needs
// to hand it to `cargo build` as an environment variable.
func TeamSalt(teamID string) string {
	sum := sha256.Sum256([]byte("laforge-agent-team-salt:" + teamID))
	return hex.EncodeToString(sum[:])
}
