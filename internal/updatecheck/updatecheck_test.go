package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v3.0.0", "v3.0.1", -1},
		{"v3.0.1", "v3.0.0", 1},
		{"v3.0.0", "v3.0.0", 0},
		{"3.0.0", "v3.0.0", 0},       // leading v optional on either side
		{"v3", "v3.0.0", 0},          // missing components are zero
		{"v3.1", "v3.0.9", 1},        // minor dominates patch
		{"v2.9.9", "v3.0.0", -1},     // major dominates
		{"v3.0.0-rc1", "v3.0.0", -1}, // a pre-release precedes the release
		{"v3.0.0", "v3.0.0-rc1", 1},
		{"v3.0.0-rc1", "v3.0.0-rc2", -1}, // pre-releases compared textually
		{"v10.0.0", "v9.0.0", 1},         // numeric, not lexical, compare
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsRelease(t *testing.T) {
	for _, v := range []string{"", "dev", "  "} {
		if IsRelease(v) {
			t.Errorf("IsRelease(%q) = true, want false", v)
		}
	}
	for _, v := range []string{"v3.0.0", "3.1.2", "v3.0.0-rc1"} {
		if !IsRelease(v) {
			t.Errorf("IsRelease(%q) = false, want true", v)
		}
	}
}

func TestFetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request missing User-Agent (GitHub rejects those)")
		}
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v3.4.5"})
	}))
	defer srv.Close()

	got, err := fetchLatest(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetchLatest: %v", err)
	}
	if got != "v3.4.5" {
		t.Fatalf("fetchLatest = %q, want v3.4.5", got)
	}
}

func TestFetchLatestNoReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := fetchLatest(context.Background(), srv.URL); err == nil {
		t.Fatal("fetchLatest on 404 should error (nothing to compare against)")
	}
}

func TestCheckDevShortCircuits(t *testing.T) {
	// A dev build must not make any network call.
	res, err := Check(context.Background(), "dev")
	if err != nil {
		t.Fatalf("Check(dev): %v", err)
	}
	if res.Outdated || res.Latest != "" {
		t.Fatalf("Check(dev) = %+v, want not outdated and no latest", res)
	}
}

func TestCheckCached(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v3.9.0"})
	}))
	defer srv.Close()

	// Seed a fresh cache directly so CheckCached reads it without any network.
	cachePath := filepath.Join(t.TempDir(), "update-check.json")
	seed := cacheEntry{CheckedAt: time.Now(), Latest: "v3.9.0"}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := CheckCached(context.Background(), "v3.0.0", cachePath, 24*time.Hour)
	if err != nil {
		t.Fatalf("CheckCached: %v", err)
	}
	if !res.Outdated || res.Latest != "v3.9.0" {
		t.Fatalf("CheckCached fresh cache = %+v, want outdated against v3.9.0", res)
	}
	if res.Message() == "" {
		t.Fatal("an outdated result must produce a nudge message")
	}
	if hits != 0 {
		t.Fatalf("a fresh cache must not hit the network (hits=%d)", hits)
	}

	// Current == latest: not outdated.
	res, err = CheckCached(context.Background(), "v3.9.0", cachePath, 24*time.Hour)
	if err != nil {
		t.Fatalf("CheckCached (current): %v", err)
	}
	if res.Outdated {
		t.Fatalf("CheckCached with current==latest should not be outdated: %+v", res)
	}
}

func TestMessageEmptyWhenCurrent(t *testing.T) {
	if (Result{Current: "v3.0.0", Latest: "v3.0.0", Outdated: false}).Message() != "" {
		t.Fatal("a non-outdated result must have an empty message")
	}
}
