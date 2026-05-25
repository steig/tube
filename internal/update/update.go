// Package update implements the lightweight update-availability check used by
// the root command's post-run hook. The design priorities, in order:
//
//  1. Never block command execution. We read the cache synchronously and
//     refresh it in a background goroutine, so the first run shows nothing
//     and every subsequent run has fresh-enough data.
//  2. No new dependencies. We resolve the latest tag the same way the
//     install script does — by following the redirect on /releases/latest.
//  3. Quiet by default. Any failure (network down, GitHub rate-limited,
//     cache write fails) is swallowed — an update notice is a nice-to-have,
//     not a guarantee.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// CacheTTL is how long we trust a cached "latest version" lookup before
	// kicking off another refresh.
	CacheTTL = 24 * time.Hour

	// fetchTimeout caps the network call in the background refresh.
	fetchTimeout = 3 * time.Second

	defaultRepo = "steig/tube"
)

// cacheEntry is what we serialize to ~/.tube/update-cache.json.
type cacheEntry struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

// Result is what callers get back. Newer is true only if Latest is a
// version greater than the running build.
type Result struct {
	Latest  string
	Current string
	Newer   bool
}

// Check reads the cached "latest version" and compares it against current.
// If the cache is stale or missing, a background goroutine refreshes it for
// the next invocation. Returns a zero Result with no error when there's
// nothing useful to report yet.
//
// `current` is the version string the binary was built with (e.g. "0.2.0").
// Pass "dev" or "" to suppress all checks.
func Check(current, cachePath string) (Result, error) {
	if shouldSkip(current) {
		return Result{}, nil
	}

	entry, _ := readCache(cachePath)
	if time.Since(entry.CheckedAt) > CacheTTL {
		// Capture the fetcher locally — tests swap the package var while
		// commands are running and the goroutine must read a stable value.
		fetch := fetchLatest
		go func() {
			latest, err := fetch(defaultRepo)
			if err != nil || latest == "" {
				return
			}
			_ = writeCache(cachePath, cacheEntry{
				CheckedAt:     time.Now().UTC(),
				LatestVersion: latest,
			})
		}()
	}

	if entry.LatestVersion == "" {
		return Result{Current: current}, nil
	}

	return Result{
		Latest:  entry.LatestVersion,
		Current: current,
		Newer:   isNewer(entry.LatestVersion, current),
	}, nil
}

// shouldSkip returns true for builds that shouldn't bother with the check:
// dev builds, empty version strings, or when the user has opted out.
func shouldSkip(current string) bool {
	if os.Getenv("TUBE_NO_UPDATE_CHECK") != "" {
		return true
	}
	if current == "" || current == "dev" || current == "unknown" {
		return true
	}
	return false
}

// DefaultCachePath returns ~/.tube/update-cache.json without requiring the
// caller to load the full Config.
func DefaultCachePath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".tube", "update-cache.json")
	}
	return ""
}

func readCache(path string) (cacheEntry, error) {
	if path == "" {
		return cacheEntry{}, errors.New("no cache path")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, err
	}
	var e cacheEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return cacheEntry{}, err
	}
	return e, nil
}

func writeCache(path string, e cacheEntry) error {
	if path == "" {
		return errors.New("no cache path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// fetchLatest follows the redirect on /releases/latest to get the actual
// tag. We avoid the GitHub API to dodge rate limits and the need for
// authentication on public repos.
//
// Override via a hook for tests.
var fetchLatest = fetchLatestFromGitHub

func fetchLatestFromGitHub(repo string) (string, error) {
	if repo == "" {
		repo = defaultRepo
	}
	url := fmt.Sprintf("https://github.com/%s/releases/latest", repo)

	client := &http.Client{
		Timeout: fetchTimeout,
		// Don't follow redirects — we just want the Location header.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no Location header (status %d)", resp.StatusCode)
	}
	// loc looks like https://github.com/owner/repo/releases/tag/vX.Y.Z
	idx := strings.LastIndex(loc, "/tag/")
	if idx < 0 {
		return "", fmt.Errorf("unexpected redirect: %s", loc)
	}
	return loc[idx+len("/tag/"):], nil
}

// isNewer reports whether latest is a strictly greater semver than current.
// Both may have a leading "v"; both should be of the form MAJOR.MINOR.PATCH
// (extra suffixes are tolerated by ignoring them).
//
// Conservative: returns false on any parse failure rather than spamming a
// false-positive "upgrade available" message.
func isNewer(latest, current string) bool {
	la, lb, lc, ok1 := parseSemver(latest)
	ca, cb, cc, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	switch {
	case la != ca:
		return la > ca
	case lb != cb:
		return lb > cb
	default:
		return lc > cc
	}
}

func parseSemver(v string) (int, int, int, bool) {
	v = strings.TrimPrefix(v, "v")
	// Trim any pre-release / build metadata.
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	c, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return a, b, c, true
}

// Once gates an update message to fire at most once per process. The PostRun
// hook may run for every subcommand, but we want exactly one notice.
var Once sync.Once

// Fetch synchronously hits the network, updates the cache, and returns a
// fresh Result. Use this when the user explicitly asked (e.g. `tube
// upgrade --check`) — for ambient-notice paths, use Check instead.
func Fetch(current, cachePath string) (Result, error) {
	if shouldSkip(current) {
		return Result{Current: current}, nil
	}
	latest, err := fetchLatest(defaultRepo)
	if err != nil {
		return Result{Current: current}, err
	}
	_ = writeCache(cachePath, cacheEntry{
		CheckedAt:     time.Now().UTC(),
		LatestVersion: latest,
	})
	return Result{
		Latest:  latest,
		Current: current,
		Newer:   isNewer(latest, current),
	}, nil
}
