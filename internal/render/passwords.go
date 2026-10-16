package render

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Password generators, exposed to templates (`{{ passphrase 3 }}`) and to the
// set_password/create_user steps' `generate:` option.
//
// They must be STABLE, not random per call: LaForge renders the same content
// many times -- at step materialization, every reconcile pass's fingerprint,
// each scheduled fire, the Render tab -- and a value that changed between
// renders would mean a host whose recorded password isn't what's on the box,
// or a fingerprint that never settles (an endless rebuild). So each value is
// derived, never drawn: HMAC-SHA256 of the build's secret (Context.Secret,
// random per build and stored with it) over the value's identity seeds a
// ChaCha8 stream. Same build + same identity -> same password, every render;
// a different build -> a different password.
//
// The identity is (scope, generator, size, label):
//   - scope defaults to this copy (team + hostname), so `{{ passphrase 3 }}` in
//     a script and `generate: {type: passphrase, size: 3}` on the same host are
//     the SAME password -- a script can write out the password a step set.
//   - label tells several passwords on one host apart: `{{ passphrase 3 "svc" }}`.
//   - a label prefixed "team:" scopes the value to the whole team (every host in
//     the team gets the same one -- a shared domain account), "build:" to the
//     whole build (every team).
//
// Without a build secret (laforge check / render / the LSP preview, which have
// no build) a fixed preview secret is used: values are well-formed and stable,
// but not secret, and not what a real build will get.

// GeneratorTypes are the generator names, as the set_password/create_user
// `generate.type` field and the template functions spell them.
const (
	GenPassphrase = "passphrase"
	GenWordchain  = "wordchain"
	GenRandom     = "random"
)

var previewSecret = []byte("laforge-preview-not-a-secret")

// generatorBounds are each generator's allowed size (words, or characters for
// random) -- generous, but enough to reject a typo'd 300.
var generatorBounds = map[string][2]int{
	GenPassphrase: {1, 12},
	GenWordchain:  {1, 12},
	GenRandom:     {8, 128},
}

// GeneratePassword returns the stable password for (gen, size, label) in ctx's
// scope. gen is one of the Gen* constants; size is the word count (passphrase,
// wordchain) or length (random).
func GeneratePassword(ctx *Context, gen string, size int, label string) (string, error) {
	b, ok := generatorBounds[gen]
	if !ok {
		return "", fmt.Errorf("unknown password generator %q (want %s, %s or %s)", gen, GenPassphrase, GenWordchain, GenRandom)
	}
	if size < b[0] || size > b[1] {
		return "", fmt.Errorf("%s size %d out of range %d-%d", gen, size, b[0], b[1])
	}
	r := rand.New(rand.NewChaCha8(generatorSeed(ctx, gen, size, label)))
	switch gen {
	case GenPassphrase:
		return passphrase(r, size), nil
	case GenWordchain:
		return wordchain(r, size), nil
	default:
		return randomPassword(r, size), nil
	}
}

func generatorSeed(ctx *Context, gen string, size int, label string) [32]byte {
	secret := ctx.Secret
	if len(secret) == 0 {
		secret = previewSecret
	}
	var scope string
	switch {
	case strings.HasPrefix(label, "build:"):
		scope = "build"
	case strings.HasPrefix(label, "team:"):
		scope = "team\x00" + strconv.Itoa(ctx.Team)
	default:
		scope = "host\x00" + strconv.Itoa(ctx.Team) + "\x00" + ctx.As
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("laforge-password-v1\x00" + scope + "\x00" + gen + "\x00" + strconv.Itoa(size) + "\x00" + label))
	var seed [32]byte
	copy(seed[:], mac.Sum(nil))
	return seed
}

// passphrase: size dictionary words (4-7 letters, capitalized), hyphenated,
// then a 4-digit number -- "Patch-Session-Query-4484".
func passphrase(r *rand.Rand, size int) string {
	parts := make([]string, 0, size+1)
	for range size {
		w := longWords[r.IntN(len(longWords))]
		parts = append(parts, strings.ToUpper(w[:1])+w[1:])
	}
	parts = append(parts, strconv.Itoa(1000+r.IntN(9000)))
	return strings.Join(parts, "-")
}

// wordchain: size short words (3-5 letters), hyphenated, each randomly
// capitalized or not -- "fork-Brick-Spoon-Snail-test".
func wordchain(r *rand.Rand, size int) string {
	parts := make([]string, 0, size)
	for range size {
		w := shortWords[r.IntN(len(shortWords))]
		if r.IntN(2) == 0 {
			w = strings.ToUpper(w[:1]) + w[1:]
		}
		parts = append(parts, w)
	}
	return strings.Join(parts, "-")
}

// randomSymbols avoids characters that break quoting in the places these
// passwords land: no quotes, backslash, $, `, % (batch), # (YAML/shell
// comments), &, |, ;, < or >, or spaces.
const (
	randomUpper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	randomLower   = "abcdefghijkmnopqrstuvwxyz"
	randomDigits  = "23456789"
	randomSymbols = "-_.!@+=*"
)

// randomPassword: size characters with at least one upper, lower, digit and
// symbol (Windows complexity rules), look-alikes (0/O, 1/l/I) left out.
func randomPassword(r *rand.Rand, size int) string {
	all := randomUpper + randomLower + randomDigits + randomSymbols
	out := []byte{
		randomUpper[r.IntN(len(randomUpper))],
		randomLower[r.IntN(len(randomLower))],
		randomDigits[r.IntN(len(randomDigits))],
		randomSymbols[r.IntN(len(randomSymbols))],
	}
	for len(out) < size {
		out = append(out, all[r.IntN(len(all))])
	}
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return string(out)
}

// generatorFuncs are the template forms: `{{ passphrase 3 }}`,
// `{{ wordchain 5 "svc" }}`, `{{ randompassword 16 }}`. An optional single
// label argument picks a distinct value (and its scope -- see above).
func generatorFuncs(ctx *Context) map[string]any {
	mk := func(gen string) func(int, ...string) (string, error) {
		return func(size int, label ...string) (string, error) {
			if len(label) > 1 {
				return "", fmt.Errorf("%s takes a size and at most one label", gen)
			}
			l := ""
			if len(label) == 1 {
				l = label[0]
			}
			return GeneratePassword(ctx, gen, size, l)
		}
	}
	return map[string]any{
		"passphrase":     mk(GenPassphrase),
		"wordchain":      mk(GenWordchain),
		"randompassword": mk(GenRandom),
	}
}
