package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"sync"
)

// LoadCached is Load with a cheap content-signature cache.
//
// A checkout directory for a given commit is immutable, yet the orchestrator
// re-Loads it (parse + schema-validate every YAML, read every people CSV, fold
// extends, cross-check) multiple times per reconcile pass and on every pass --
// which for a real content repo is the bulk of a pass's time. LoadCached first
// computes a stat signature (relative path + size + mtime of every content
// file) -- cheap, no reads -- and returns the previously parsed *Content when
// the signature is unchanged. A re-checkout (new commit) or a live dev-tree
// edit changes an mtime/size and busts the entry, so it never serves stale
// content; only an unchanged tree is reused.
//
// The returned *Content is SHARED across callers and passes and MUST be treated
// as read-only. Reconcile and render already treat loaded content as immutable;
// callers that need to mutate must Load (uncached) their own copy.
func LoadCached(root string) (*Content, error) {
	sig, sigErr := contentSignature(root)
	if sigErr == nil {
		loadCacheMu.Lock()
		e, ok := loadCache[root]
		loadCacheMu.Unlock()
		if ok && e.sig == sig {
			return e.content, nil
		}
	}
	c, err := Load(root)
	if err != nil {
		return c, err
	}
	if sigErr == nil {
		loadCacheMu.Lock()
		// Bounded: checkout dirs are per-commit, so this only grows as commits
		// churn. Reset wholesale past the cap rather than track per-entry LRU --
		// live builds re-cache on their next pass.
		if len(loadCache) >= loadCacheMax {
			loadCache = map[string]loadCacheEntry{}
		}
		loadCache[root] = loadCacheEntry{sig: sig, content: c}
		loadCacheMu.Unlock()
	}
	return c, nil
}

type loadCacheEntry struct {
	sig     string
	content *Content
}

const loadCacheMax = 64

var (
	loadCacheMu sync.Mutex
	loadCache   = map[string]loadCacheEntry{}
)

// contentSignature hashes the relative path, size and mtime of every file under
// root (skipping the same non-content directories Load skips). It reads no file
// contents, so it's far cheaper than Load; any change that would change Load's
// output -- a re-checkout, or an edit in a dev tree -- changes a size or mtime
// and therefore the signature.
func contentSignature(root string) (string, error) {
	type ent struct {
		rel  string
		size int64
		mod  int64
	}
	var ents []ent
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			if rel != "." && isNonContentDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		ents = append(ents, ent{rel: filepath.ToSlash(rel), size: info.Size(), mod: info.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(ents, func(i, j int) bool { return ents[i].rel < ents[j].rel })
	h := sha256.New()
	for _, e := range ents {
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", e.rel, e.size, e.mod)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
