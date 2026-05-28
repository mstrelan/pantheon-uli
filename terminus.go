package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var urlRe = regexp.MustCompile(`https?://\S+`)

// GetULI runs terminus remote:drush <site.env> -- uli and returns the raw URL.
func GetULI(ctx context.Context, siteEnv string) (string, error) {
	cmd := exec.CommandContext(ctx, "terminus", "remote:drush", siteEnv, "--", "uli")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("drush uli failed: %s", strings.TrimSpace(string(out)))
	}
	url := urlRe.FindString(string(out))
	if url == "" {
		return "", fmt.Errorf("no URL found in drush uli output: %s", strings.TrimSpace(string(out)))
	}
	return url, nil
}

// GetVanityDomain returns the first non-pantheon domain for a site.env, or "" if none.
func GetVanityDomain(ctx context.Context, siteEnv string) string {
	cmd := exec.CommandContext(ctx, "terminus", "domain:list", siteEnv, "--format=list", "--fields=id")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		d := strings.TrimSpace(line)
		if d == "" {
			continue
		}
		if strings.Contains(d, "pantheonsite.io") || strings.Contains(d, "pantheon.io") || strings.Contains(d, "drush.in") {
			continue
		}
		return d
	}
	return ""
}

// GetLockCreds returns (username, password) for HTTP basic auth on a locked site.
func GetLockCreds(ctx context.Context, siteEnv string) (string, string) {
	userCmd := exec.CommandContext(ctx, "terminus", "lock:info", siteEnv, "--field=username")
	passCmd := exec.CommandContext(ctx, "terminus", "lock:info", siteEnv, "--field=password")

	userOut, err1 := userCmd.Output()
	passOut, err2 := passCmd.Output()

	if err1 != nil || err2 != nil {
		return "", ""
	}
	u := strings.TrimSpace(string(userOut))
	p := strings.TrimSpace(string(passOut))
	return u, p
}
