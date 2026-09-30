// Package envfile loads a plain KEY=VALUE .env file into the process
// environment -- "Configuration by environment variables and files",
// so a developer can drop
// real values (a GitHub App's client secret, webhook secret, and so on)
// into one gitignored file instead of re-exporting them in every shell
// session. Deliberately minimal: no interpolation, no `export` keyword,
// no multiline values -- every real .env this project needs is secrets
// and connection strings, one per line, which is all this reads.
//
// This only ever matters for local, non-container development
// (`go run ./cmd/laforge-api`). A real deployment (or docker-compose,
// which has its own native env_file handling) sets real environment
// variables directly and never has a .env file on disk at all -- Load
// treats a missing file as a no-op, not an error.
package envfile

import (
	"bufio"
	"os"
	"strings"
)

// Load reads path and calls os.Setenv for each KEY=VALUE line it finds,
// skipping blank lines and lines starting with "#". A key already set in
// the real environment is left alone -- the actual environment always
// wins over the file, matching every other dotenv tool's own convention,
// so a value exported in your shell (or injected by a real deployment)
// can't be silently overridden by a stale .env.
func Load(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = unquote(value)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, value)
	}
	return scanner.Err()
}

// unquote strips one layer of matching "..." or '...' quotes, if
// present -- enough to let a value contain a leading/trailing space or a
// "#" without it being mistaken for a comment, without taking on real
// shell-quoting semantics this project's own .env values never need.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
