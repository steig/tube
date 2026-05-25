package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.3.0", "0.2.0", true},
		{"v0.3.0", "v0.2.0", true},
		{"0.3.0", "0.3.0", false},
		{"0.2.9", "0.2.10", false}, // numeric, not lexicographic
		{"v1.0.0", "0.9.9", true},
		{"v0.2.0", "v0.3.0", false},
		{"v0.2.0-rc1", "0.1.0", true}, // pre-release suffix ignored
		{"not-a-version", "0.1.0", false},
		{"0.1.0", "dev", false}, // unparseable current
	}
	for _, c := range cases {
		got := isNewer(c.latest, c.current)
		if got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		in                    string
		a, b, c               int
		ok                    bool
	}{
		{"0.2.0", 0, 2, 0, true},
		{"v1.2.3", 1, 2, 3, true},
		{"v1.2.3-rc1", 1, 2, 3, true},
		{"v1.2.3+build", 1, 2, 3, true},
		{"1.2", 0, 0, 0, false},
		{"v1.x.0", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}
	for _, c := range cases {
		a, b, ce, ok := parseSemver(c.in)
		if ok != c.ok {
			t.Errorf("parseSemver(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && (a != c.a || b != c.b || ce != c.c) {
			t.Errorf("parseSemver(%q) = (%d,%d,%d), want (%d,%d,%d)", c.in, a, b, ce, c.a, c.b, c.c)
		}
	}
}

func TestShouldSkip(t *testing.T) {
	t.Setenv("TUBE_NO_UPDATE_CHECK", "")
	cases := []struct {
		v    string
		want bool
	}{
		{"", true},
		{"dev", true},
		{"unknown", true},
		{"0.2.0", false},
	}
	for _, c := range cases {
		if got := shouldSkip(c.v); got != c.want {
			t.Errorf("shouldSkip(%q) = %v, want %v", c.v, got, c.want)
		}
	}

	t.Setenv("TUBE_NO_UPDATE_CHECK", "1")
	if !shouldSkip("0.2.0") {
		t.Error("shouldSkip should respect TUBE_NO_UPDATE_CHECK")
	}
}

func TestCheck_UsesCachedValue(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "cache.json")

	// Seed a fresh cache with a known latest version.
	seed := cacheEntry{
		CheckedAt:     time.Now().UTC(),
		LatestVersion: "v0.5.0",
	}
	b, _ := json.Marshal(seed)
	if err := os.WriteFile(cachePath, b, 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	// Network-impossible URL to prove we don't hit the network when cache is fresh.
	orig := fetchLatest
	fetchLatest = func(string) (string, error) { t.Fatal("fetchLatest called unexpectedly"); return "", nil }
	t.Cleanup(func() { fetchLatest = orig })

	r, err := Check("0.2.0", cachePath)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if r.Latest != "v0.5.0" || !r.Newer {
		t.Errorf("Check returned %+v, want Latest=v0.5.0 Newer=true", r)
	}
}

func TestCheck_RefreshesStaleCache(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "cache.json")

	// Stale entry from 48h ago.
	stale := cacheEntry{
		CheckedAt:     time.Now().Add(-48 * time.Hour).UTC(),
		LatestVersion: "v0.2.0",
	}
	b, _ := json.Marshal(stale)
	if err := os.WriteFile(cachePath, b, 0o644); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	called := make(chan struct{}, 1)
	orig := fetchLatest
	fetchLatest = func(string) (string, error) {
		called <- struct{}{}
		return "v0.9.0", nil
	}
	t.Cleanup(func() { fetchLatest = orig })

	r, err := Check("0.2.0", cachePath)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	// Call should return stale-cache contents without blocking.
	if r.Latest != "v0.2.0" {
		t.Errorf("Check returned %+v, want Latest=v0.2.0 (stale)", r)
	}

	// Background goroutine should have refreshed the cache.
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh did not call fetchLatest within 2s")
	}

	// Allow the goroutine's writeCache call to land.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entry, _ := readCache(cachePath)
		if entry.LatestVersion == "v0.9.0" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("cache was not refreshed by background goroutine")
}

func TestCheck_NoCacheFirstRun(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, "cache.json")

	done := make(chan struct{})
	orig := fetchLatest
	fetchLatest = func(string) (string, error) {
		// Defer signaling so the cache write inside Check's goroutine has
		// completed before the test cleanup tears down TempDir. Without
		// this, on Go 1.22 the goroutine writes after t.TempDir's cleanup
		// runs, leaving the dir non-empty and failing the test.
		defer close(done)
		return "v0.9.0", nil
	}
	t.Cleanup(func() {
		<-done
		fetchLatest = orig
	})

	r, err := Check("0.2.0", cachePath)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	// No cache means no version to report on this call. Background refresh
	// will populate for next time.
	if r.Latest != "" || r.Newer {
		t.Errorf("first run returned %+v, want empty Latest", r)
	}

	// Wait for the goroutine before the test returns so cleanup can drain.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh did not run within 2s")
	}
}
