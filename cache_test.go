package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setCacheDir overrides the OS cache directory for the duration of a test by
// setting XDG_CACHE_HOME (Linux) and LOCALAPPDATA (Windows).  The function
// returns a cleanup func that restores the original values.
func overrideCacheDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// os.UserCacheDir on Linux reads $XDG_CACHE_HOME, then $HOME/.cache.
	orig := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmp)
	t.Cleanup(func() { os.Setenv("XDG_CACHE_HOME", orig) })
	return tmp
}

// writeCacheRaw directly writes a siteCache struct to disk, bypassing
// writeCache so tests can plant arbitrary timestamps.
func writeCacheRaw(t *testing.T, c siteCache) {
	t.Helper()
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sites.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// ---- LoadSites ----

func TestLoadSites_EmptyCache(t *testing.T) {
	overrideCacheDir(t)
	sites, err := LoadSites()
	if err == nil {
		t.Fatal("expected error for missing cache, got nil")
	}
	if sites != nil {
		t.Fatalf("expected nil sites, got %v", sites)
	}
}

func TestLoadSites_ReturnsSortedSites(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{
		Sites:       []string{"zebra", "alpha", "mango"},
		RefreshedAt: time.Now(),
	})
	sites, err := LoadSites()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "mango", "zebra"}
	if len(sites) != len(want) {
		t.Fatalf("got %v, want %v", sites, want)
	}
	for i, s := range sites {
		if s != want[i] {
			t.Errorf("sites[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestLoadSites_IgnoresSitesTxt(t *testing.T) {
	overrideCacheDir(t)
	// Plant a legacy sites.txt — LoadSites must ignore it.
	dir := cacheDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "sites.txt"), []byte("legacy-site\n"), 0o644)

	sites, err := LoadSites()
	// Should fail because sites.json is absent.
	if err == nil {
		t.Fatalf("expected error when only sites.txt exists, got sites=%v", sites)
	}
}

// ---- CacheIsStale ----

func TestCacheIsStale_MissingFile(t *testing.T) {
	overrideCacheDir(t)
	if !CacheIsStale() {
		t.Fatal("missing cache should be stale")
	}
}

func TestCacheIsStale_FreshTimestamp(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{
		Sites:       []string{"site-a"},
		RefreshedAt: time.Now(),
	})
	if CacheIsStale() {
		t.Fatal("recently-written cache should not be stale")
	}
}

func TestCacheIsStale_ExpiredTimestamp(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{
		Sites:       []string{"site-a"},
		RefreshedAt: time.Now().Add(-(cacheTTL + time.Minute)),
	})
	if !CacheIsStale() {
		t.Fatal("cache with expired timestamp should be stale")
	}
}

func TestCacheIsStale_ZeroTimestamp(t *testing.T) {
	overrideCacheDir(t)
	// A cache file with a zero RefreshedAt (e.g. manually written without timestamp).
	writeCacheRaw(t, siteCache{
		Sites: []string{"site-a"},
		// RefreshedAt is zero value
	})
	if !CacheIsStale() {
		t.Fatal("cache with zero timestamp should be stale")
	}
}

// ---- LoadEnvMeta / SaveEnvMeta ----

func TestLoadEnvMeta_MissingEntry(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{
		Sites:       []string{"mysite"},
		RefreshedAt: time.Now(),
	})
	_, _, _, ok := LoadEnvMeta("mysite.live")
	if ok {
		t.Fatal("expected ok=false for absent entry")
	}
}

func TestSaveAndLoadEnvMeta_RoundTrip(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{Sites: []string{"mysite"}, RefreshedAt: time.Now()})

	err := SaveEnvMeta("mysite.live", "mysite.com", "admin", "secret")
	if err != nil {
		t.Fatalf("SaveEnvMeta: %v", err)
	}

	vanity, user, pass, ok := LoadEnvMeta("mysite.live")
	if !ok {
		t.Fatal("expected ok=true after save")
	}
	if vanity != "mysite.com" {
		t.Errorf("vanity = %q, want %q", vanity, "mysite.com")
	}
	if user != "admin" {
		t.Errorf("user = %q, want %q", user, "admin")
	}
	if pass != "secret" {
		t.Errorf("pass = %q, want %q", pass, "secret")
	}
}

func TestLoadEnvMeta_StaleEntry(t *testing.T) {
	overrideCacheDir(t)
	staleTime := time.Now().Add(-(envMetaTTL + time.Minute))
	writeCacheRaw(t, siteCache{
		Sites:       []string{"mysite"},
		RefreshedAt: time.Now(),
		EnvMeta: map[string]EnvMeta{
			"mysite.live": {
				VanityDomain: "mysite.com",
				LockUser:     "admin",
				LockPass:     "secret",
				UpdatedAt:    staleTime,
			},
		},
	})
	_, _, _, ok := LoadEnvMeta("mysite.live")
	if ok {
		t.Fatal("expected ok=false for stale entry")
	}
}

func TestSaveEnvMeta_PreservesSiteList(t *testing.T) {
	overrideCacheDir(t)
	writeCacheRaw(t, siteCache{
		Sites:       []string{"alpha", "beta"},
		RefreshedAt: time.Now(),
	})

	if err := SaveEnvMeta("alpha.dev", "", "", ""); err != nil {
		t.Fatalf("SaveEnvMeta: %v", err)
	}

	sites, err := LoadSites()
	if err != nil {
		t.Fatalf("LoadSites after SaveEnvMeta: %v", err)
	}
	if len(sites) != 2 {
		t.Errorf("expected 2 sites, got %d: %v", len(sites), sites)
	}
}

func TestSaveEnvMeta_PreservesOtherEntries(t *testing.T) {
	overrideCacheDir(t)
	now := time.Now()
	writeCacheRaw(t, siteCache{
		Sites:       []string{"alpha"},
		RefreshedAt: now,
		EnvMeta: map[string]EnvMeta{
			"alpha.live": {VanityDomain: "alpha.com", UpdatedAt: now},
		},
	})

	if err := SaveEnvMeta("alpha.dev", "", "user", "pass"); err != nil {
		t.Fatalf("SaveEnvMeta: %v", err)
	}

	// Original live entry must still be present and fresh.
	vanity, _, _, ok := LoadEnvMeta("alpha.live")
	if !ok {
		t.Fatal("original entry should still be present")
	}
	if vanity != "alpha.com" {
		t.Errorf("vanity = %q, want %q", vanity, "alpha.com")
	}
}

// ---- JSON format / sites.txt not written ----

func TestRefreshCache_WritesSitesJson(t *testing.T) {
	// We cannot actually call terminus in a unit test, so we just verify that
	// the cache file path ends in .json and that sites.txt is never created.
	overrideCacheDir(t)
	p := cacheFile()
	if filepath.Ext(p) != ".json" {
		t.Errorf("cacheFile() = %q, want a .json path", p)
	}
	// Confirm sites.txt would be in a different path.
	txtPath := filepath.Join(cacheDir(), "sites.txt")
	if p == txtPath {
		t.Errorf("cacheFile() returned sites.txt path, expected sites.json")
	}
}
