package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCachedReusesUntilContentChanges(t *testing.T) {
	root := t.TempDir()
	write := func(body string) {
		if err := os.WriteFile(filepath.Join(root, "env.yaml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("type: environment\nname: e\n")

	a, err := LoadCached(root)
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	b, err := LoadCached(root)
	if err != nil {
		t.Fatalf("LoadCached (2nd): %v", err)
	}
	if a != b {
		t.Fatal("unchanged content should be a cache hit (same *Content pointer)")
	}

	// A content change (different size busts the signature regardless of mtime
	// granularity) must re-Load, not serve the stale cached parse.
	write("type: environment\nname: e\nteams: 5\n# changed, longer than before\n")
	c, err := LoadCached(root)
	if err != nil {
		t.Fatalf("LoadCached (after change): %v", err)
	}
	if c == a {
		t.Fatal("changed content must bust the cache (new *Content pointer)")
	}
}
