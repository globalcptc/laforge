package render

import (
	"regexp"
	"strings"
	"testing"
)

func TestWordListsAreClean(t *testing.T) {
	lower := regexp.MustCompile(`^[a-z]+$`)
	check := func(name string, words []string, lo, hi int) {
		seen := map[string]bool{}
		for _, w := range words {
			if !lower.MatchString(w) || len(w) < lo || len(w) > hi {
				t.Errorf("%s: %q is not %d-%d lowercase letters", name, w, lo, hi)
			}
			if seen[w] {
				t.Errorf("%s: %q appears twice", name, w)
			}
			seen[w] = true
		}
		if len(words) < 200 {
			t.Errorf("%s has only %d words; too few for a usable password space", name, len(words))
		}
	}
	check("longWords", longWords, 4, 7)
	check("shortWords", shortWords, 3, 5)
}

func TestGeneratorShapes(t *testing.T) {
	ctx := &Context{Team: 1, As: "vdi-win-01", Secret: []byte("s")}
	pp, err := GeneratePassword(ctx, GenPassphrase, 3, "")
	if err != nil || !regexp.MustCompile(`^([A-Z][a-z]{3,6}-){3}[1-9][0-9]{3}$`).MatchString(pp) {
		t.Errorf("passphrase 3 = %q, %v; want Word-Word-Word-NNNN", pp, err)
	}
	wc, err := GeneratePassword(ctx, GenWordchain, 5, "")
	if err != nil || !regexp.MustCompile(`^[A-Za-z]{3,5}(-[A-Za-z]{3,5}){4}$`).MatchString(wc) {
		t.Errorf("wordchain 5 = %q, %v; want five hyphenated 3-5 letter words", wc, err)
	}
	rp, err := GeneratePassword(ctx, GenRandom, 16, "")
	if err != nil || len(rp) != 16 {
		t.Fatalf("random 16 = %q, %v", rp, err)
	}
	for _, class := range []string{randomUpper, randomLower, randomDigits, randomSymbols} {
		if !strings.ContainsAny(rp, class) {
			t.Errorf("random 16 = %q lacks a character from %q", rp, class)
		}
	}
	for _, bad := range []struct {
		gen  string
		size int
	}{{GenPassphrase, 0}, {GenWordchain, 13}, {GenRandom, 7}, {"nope", 3}} {
		if _, err := GeneratePassword(ctx, bad.gen, bad.size, ""); err == nil {
			t.Errorf("GeneratePassword(%q, %d) succeeded, want an error", bad.gen, bad.size)
		}
	}
}

// TestGeneratorsAreStableAndScoped is the property everything depends on: the
// same build, host and label always give the same password (renders repeat),
// while a different host, team, label or build secret gives a different one --
// and team:/build: labels share one value across hosts/teams.
func TestGeneratorsAreStableAndScoped(t *testing.T) {
	gen := func(secret string, team int, as, label string) string {
		t.Helper()
		v, err := GeneratePassword(&Context{Team: team, As: as, Secret: []byte(secret)}, GenPassphrase, 3, label)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	base := gen("s1", 1, "dc", "")
	if again := gen("s1", 1, "dc", ""); again != base {
		t.Fatalf("not stable: %q then %q", base, again)
	}
	for name, other := range map[string]string{
		"other host":   gen("s1", 1, "ws", ""),
		"other team":   gen("s1", 2, "dc", ""),
		"other label":  gen("s1", 1, "dc", "svc"),
		"other secret": gen("s2", 1, "dc", ""),
	} {
		if other == base {
			t.Errorf("%s produced the same password %q", name, base)
		}
	}
	if a, b := gen("s1", 1, "dc", "team:da"), gen("s1", 1, "ws", "team:da"); a != b {
		t.Errorf("team: label differs across hosts in a team: %q vs %q", a, b)
	}
	if a, b := gen("s1", 1, "dc", "team:da"), gen("s1", 2, "dc", "team:da"); a == b {
		t.Errorf("team: label shared across teams: %q", a)
	}
	if a, b := gen("s1", 1, "dc", "build:x"), gen("s1", 3, "ws", "build:x"); a != b {
		t.Errorf("build: label differs across teams: %q vs %q", a, b)
	}
}

// TestGeneratorTemplateFuncs checks the template forms and that a template
// call and GeneratePassword agree (a script can write out what a step set).
func TestGeneratorTemplateFuncs(t *testing.T) {
	ctx := &Context{Team: 2, As: "vdi-win-02", Secret: []byte("s")}
	want, _ := GeneratePassword(ctx, GenWordchain, 4, "svc")
	fns := generatorFuncs(ctx)
	got, err := fns["wordchain"].(func(int, ...string) (string, error))(4, "svc")
	if err != nil || got != want {
		t.Errorf("wordchain 4 \"svc\" = %q, %v; want %q", got, err, want)
	}
	if _, err := fns["passphrase"].(func(int, ...string) (string, error))(3, "a", "b"); err == nil {
		t.Error("two labels accepted; want an error")
	}
}
