// Package updatecheck tells a built laforge binary whether a newer release
// exists on GitHub, so the CLI and the language server can nudge people to
// update instead of silently running a stale version against a moved-on
// server. Every call is best-effort: a dev build, no network, GitHub being
// slow or rate-limiting, or an unparseable version all resolve to "not
// outdated" with no error surfaced to the user -- an update nudge must never
// get in the way of the actual work.
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub repository releases are published to. The latest release
// tag is what a stamped build compares itself against.
const Repo = "globalcptc/laforge"

// LatestURL is the GitHub REST endpoint for the newest published release. It
// resolves the highest non-prerelease, non-draft release -- exactly the one a
// person should be on.
const LatestURL = "https://api.github.com/repos/" + Repo + "/releases/latest"

// defaultTimeout bounds every network call. A short cap keeps the CLI snappy
// even on the one invocation a day that actually refreshes.
const defaultTimeout = 3 * time.Second

// Result is the outcome of a check.
type Result struct {
	Current  string // the version this binary was built as
	Latest   string // the latest release tag seen on GitHub ("" if unknown)
	Outdated bool   // Current is a release and older than Latest
}

// Message is the one-line nudge to show when Outdated, or "" otherwise.
func (r Result) Message() string {
	if !r.Outdated {
		return ""
	}
	return fmt.Sprintf("A newer LaForge release is available: %s (you have %s). Update: https://github.com/%s/releases/latest",
		r.Latest, r.Current, Repo)
}

// IsRelease reports whether v is a real stamped release version rather than a
// local build. A plain `go build`/`go install` leaves version "dev" (or
// empty); those can't be meaningfully compared to a release tag, so checks are
// skipped for them.
func IsRelease(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && v != "dev"
}

// Disabled reports whether the environment has opted out of update checks
// (LAFORGE_NO_UPDATE_CHECK set to anything non-empty). Honoured by every
// caller so an air-gapped or CI run never even attempts the network call.
func Disabled() bool {
	return strings.TrimSpace(os.Getenv("LAFORGE_NO_UPDATE_CHECK")) != ""
}

// FetchLatest returns the latest release tag (e.g. "v3.0.1") from GitHub.
func FetchLatest(ctx context.Context) (string, error) {
	return fetchLatest(ctx, LatestURL)
}

func fetchLatest(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// GitHub requires a User-Agent and serves the stable v3 REST media type.
	req.Header.Set("User-Agent", "laforge-updatecheck")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// 404 = no releases published yet; 403 = rate-limited. Either way
		// there's nothing to compare against, not an error worth surfacing.
		return "", fmt.Errorf("github releases: unexpected status %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimSpace(body.TagName), nil
}

// Check fetches the latest release and compares it to current. A non-release
// current (dev/empty) short-circuits to not-outdated without any network call.
func Check(ctx context.Context, current string) (Result, error) {
	res := Result{Current: current}
	if !IsRelease(current) {
		return res, nil
	}
	latest, err := FetchLatest(ctx)
	if err != nil {
		return res, err
	}
	res.Latest = latest
	res.Outdated = latest != "" && Compare(current, latest) < 0
	return res, nil
}

// cacheEntry is the throttle state CheckCached persists between runs.
type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// CheckCached is Check throttled to at most one network call per ttl, using a
// small JSON stamp file at cachePath. Between refreshes it still compares the
// current build against the last-seen latest, so a binary that is now behind a
// release recorded earlier is flagged instantly without any network call. All
// cache I/O is best-effort: an unreadable or unwritable cache just means the
// next run refreshes again.
func CheckCached(ctx context.Context, current, cachePath string, ttl time.Duration) (Result, error) {
	res := Result{Current: current}
	if !IsRelease(current) {
		return res, nil
	}
	if entry, ok := readCache(cachePath); ok && time.Since(entry.CheckedAt) < ttl {
		res.Latest = entry.Latest
		res.Outdated = entry.Latest != "" && Compare(current, entry.Latest) < 0
		return res, nil
	}
	latest, err := FetchLatest(ctx)
	if err != nil {
		// Fall back to a stale cache rather than losing a known nudge on a
		// flaky network.
		if entry, ok := readCache(cachePath); ok {
			res.Latest = entry.Latest
			res.Outdated = entry.Latest != "" && Compare(current, entry.Latest) < 0
		}
		return res, err
	}
	writeCache(cachePath, cacheEntry{CheckedAt: time.Now(), Latest: latest})
	res.Latest = latest
	res.Outdated = latest != "" && Compare(current, latest) < 0
	return res, nil
}

func readCache(path string) (cacheEntry, bool) {
	if path == "" {
		return cacheEntry{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, false
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, false
	}
	return entry, true
}

func writeCache(path string, entry cacheEntry) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

// Compare orders two version strings like semantic versions: it returns -1 if
// a < b, 0 if equal, 1 if a > b. A leading "v" is ignored, missing numeric
// components count as 0 ("v3" == "v3.0.0"), and a pre-release suffix
// ("-rc1") sorts BEFORE the same version without one. Anything it can't parse
// as numbers falls back to a plain string comparison, so a weird tag never
// panics -- it just might order oddly, which only ever means a spurious (or
// missing) nudge, never a crash.
func Compare(a, b string) int {
	anum, apre := splitVersion(a)
	bnum, bpre := splitVersion(b)
	for i := 0; i < 3; i++ {
		if anum[i] != bnum[i] {
			if anum[i] < bnum[i] {
				return -1
			}
			return 1
		}
	}
	// Equal release numbers: no pre-release outranks a pre-release.
	switch {
	case apre == "" && bpre == "":
		return 0
	case apre == "" && bpre != "":
		return 1
	case apre != "" && bpre == "":
		return -1
	}
	return strings.Compare(apre, bpre)
}

// splitVersion parses "v3.1.2-rc1" into [3,1,2] and "rc1". Non-numeric major
// (an unparseable tag) yields an all-zero triple and the whole string as the
// pre-release part, which Compare then falls back to comparing as text.
func splitVersion(v string) ([3]int, string) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	var nums [3]int
	parts := strings.Split(v, ".")
	parsedAny := false
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}, strings.TrimSpace(v + pre)
		}
		nums[i] = n
		parsedAny = true
	}
	if !parsedAny {
		return [3]int{}, pre
	}
	return nums, pre
}
