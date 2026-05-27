package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const cacheTTL = 30 * 24 * time.Hour // 30 days

func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "pantheon-uli")
}

func cacheFile() string {
	return filepath.Join(cacheDir(), "sites.txt")
}

// LoadSites reads cached sites from disk. Returns nil if cache is missing.
func LoadSites() ([]string, error) {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		return nil, err
	}
	var sites []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			sites = append(sites, line)
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// CacheIsStale returns true if the cache file is missing or older than cacheTTL.
func CacheIsStale() bool {
	info, err := os.Stat(cacheFile())
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > cacheTTL
}

// RefreshCache calls terminus site:list and writes the result to the cache file.
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

	if err := os.MkdirAll(cacheDir(), 0o755); err != nil {
		return sites, err
	}
	return sites, os.WriteFile(cacheFile(), []byte(strings.Join(sites, "\n")+"\n"), 0o644)
}
