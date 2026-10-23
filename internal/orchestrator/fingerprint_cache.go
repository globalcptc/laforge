package orchestrator

import "sync"

// fingerprintCache memoizes Fingerprint results across reconcile passes.
//
// A build's content is pinned to an immutable content revision, so an object's
// fingerprint is invariant for a given (content revision, object): recomputing
// it every 2-second reconcile pass -- which re-reads and re-renders every one of
// that object's step scripts (see Fingerprint / render.RenderScript) -- is pure
// waste once a build has settled. Reconcile keys this cache by the content
// revision plus the object's identity, so a genuine content change (a rebuild to
// a new commit → a new revision id) gets fresh keys and is never masked by a
// stale entry.
//
// It is a process-local, advisory cache: a miss (a cold replica, or after a
// capacity reset) simply recomputes, which is always correct, so there is no
// consistency obligation across the orchestrator's stateless replicas. The one
// thing it must never do is grow without bound, so it resets wholesale once it
// passes a generous cap rather than tracking per-entry LRU -- the next passes
// just repopulate the live builds' entries.
type fingerprintCache struct {
	mu  sync.Mutex
	m   map[string]string
	max int
}

// fpCache is the shared instance. Reconcile (and only Reconcile) uses it.
var fpCache = &fingerprintCache{m: make(map[string]string), max: 20000}

func (c *fingerprintCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *fingerprintCache) put(key, fp string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= c.max {
		// Bounded: drop everything and repopulate rather than evict per-entry.
		// Live builds re-cache within a pass or two; torn-down builds' entries
		// (their revisions never recur) are exactly what we want gone.
		c.m = make(map[string]string, c.max/2)
	}
	c.m[key] = fp
}
