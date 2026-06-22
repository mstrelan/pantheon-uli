package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

const cacheTTL    = 30 * 24 * time.Hour // 30 days — site list TTL
const envMetaTTL  = 30 * 24 * time.Hour // 30 days — per-environment metadata TTL

// EnvMeta holds cached per-environment metadata.
type EnvMeta struct {
	VanityDomain string    `json:"vanity_domain"`
	LockUser     string    `json:"lock_user"`
	LockPass     string    `json:"lock_pass"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// siteCache is the JSON structure written to sites.json.
type siteCache struct {
	Sites       []string           `json:"sites"`
	RefreshedAt time.Time          `json:"refreshed_at"`
	EnvMeta     map[string]EnvMeta `json:"env_meta,omitempty"`
}

func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "pantheon-uli")
}

func cacheFile() string {
	return filepath.Join(cacheDir(), "sites.json")
}

// readCache reads and parses the JSON cache file. Returns a zero-value
// siteCache (not an error) if the file is missing or malformed.
func readCache() (siteCache, error) {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		if os.IsNotExist(err) {
			return siteCache{}, nil
		}
		return siteCache{}, err
	}
	var c siteCache
	if err := json.Unmarshal(data, &c); err != nil {
		// Corrupt file — treat as empty cache.
		return siteCache{}, nil
	}
	return c, nil
}

// writeCache persists c to disk as JSON, creating the cache directory if needed.
func writeCache(c siteCache) error {
	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile(), data, 0o644)
}

// LoadSites reads cached sites from disk. Returns nil, err if the cache is
// missing or the site list is empty.
func LoadSites() ([]string, error) {
	c, err := readCache()
	if err != nil {
		return nil, err
	}
	if len(c.Sites) == 0 {
		return nil, fmt.Errorf("cache is empty")
	}
	sites := make([]string, len(c.Sites))
	copy(sites, c.Sites)
	sort.Strings(sites)
	return sites, nil
}

// CacheIsStale returns true if the JSON cache is missing, has no refresh
// timestamp, or the site list was last refreshed more than cacheTTL ago.
func CacheIsStale() bool {
	c, err := readCache()
	if err != nil {
		return true
	}
	if c.RefreshedAt.IsZero() {
		return true
	}
	return time.Since(c.RefreshedAt) > cacheTTL
}

// RefreshCache calls terminus site:list and writes the result to the JSON cache.
// Existing per-environment metadata is preserved.
func RefreshCache() ([]string, error) {
	cmd := exec.Command("terminus", "site:list", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terminus site:list failed: %w", err)
	}

	var result map[string]struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing terminus output: %w", err)
	}

	var sites []string
	for _, v := range result {
		if v.Name != "" {
			sites = append(sites, v.Name)
		}
	}
	sort.Strings(sites)

	// Preserve existing env metadata when refreshing the site list.
	existing, _ := readCache()
	c := siteCache{
		Sites:       sites,
		RefreshedAt: time.Now(),
		EnvMeta:     existing.EnvMeta,
	}
	return sites, writeCache(c)
}

// LoadEnvMeta returns cached per-environment metadata for the given siteEnv
// (e.g. "mysite.live"). ok is false if the entry is absent or stale.
func LoadEnvMeta(siteEnv string) (vanityDomain, lockUser, lockPass string, ok bool) {
	c, err := readCache()
	if err != nil {
		return "", "", "", false
	}
	if c.EnvMeta == nil {
		return "", "", "", false
	}
	meta, exists := c.EnvMeta[siteEnv]
	if !exists {
		return "", "", "", false
	}
	if time.Since(meta.UpdatedAt) > envMetaTTL {
		return "", "", "", false
	}
	return meta.VanityDomain, meta.LockUser, meta.LockPass, true
}

// SaveEnvMeta writes or updates per-environment metadata in the JSON cache.
func SaveEnvMeta(siteEnv, vanityDomain, lockUser, lockPass string) error {
	c, err := readCache()
	if err != nil {
		return err
	}
	if c.EnvMeta == nil {
		c.EnvMeta = make(map[string]EnvMeta)
	}
	c.EnvMeta[siteEnv] = EnvMeta{
		VanityDomain: vanityDomain,
		LockUser:     lockUser,
		LockPass:     lockPass,
		UpdatedAt:    time.Now(),
	}
	return writeCache(c)
}
