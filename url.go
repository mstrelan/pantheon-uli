package main

import (
	"net/url"
	"regexp"
	"strings"
)

var domainRe = regexp.MustCompile(`https?://([^/]+)`)

// EnforceHTTPS ensures the URL uses https://.
func EnforceHTTPS(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") {
		return "https://" + rawURL[len("http://"):]
	}
	return rawURL
}

// InjectCreds inserts URL-encoded username:password@ into the URL.
func InjectCreds(rawURL, username, password string) string {
	if username == "" || password == "" {
		return rawURL
	}
	encUser := url.PathEscape(username)
	encPass := url.PathEscape(password)
	creds := encUser + ":" + encPass + "@"

	if strings.HasPrefix(rawURL, "https://") {
		return "https://" + creds + rawURL[len("https://"):]
	}
	if strings.HasPrefix(rawURL, "http://") {
		return "http://" + creds + rawURL[len("http://"):]
	}
	return rawURL
}

// SwapDomain replaces the host in the URL with newDomain.
func SwapDomain(rawURL, newDomain string) string {
	if newDomain == "" {
		return rawURL
	}
	m := domainRe.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return rawURL
	}
	return strings.Replace(rawURL, m[1], newDomain, 1)
}
