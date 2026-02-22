package crawler

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RobotsCache struct {
	mu       sync.Mutex
	disallow map[string][]string // domain → disallowed paths
}

func NewRobotsCache() *RobotsCache {
	return &RobotsCache{
		disallow: make(map[string][]string),
	}
}

func (rc *RobotsCache) fetch(ctx context.Context, domain string) {
	url := "https://" + domain + "/robots.txt"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "RAVEN-Bot/1.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return // no robots.txt → allow everything
	}
	defer resp.Body.Close()

	var paths []string
	scanner := bufio.NewScanner(resp.Body)
	applies := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "User-agent:") {
			agent := strings.TrimSpace(strings.TrimPrefix(line, "User-agent:"))
			applies = agent == "*" || strings.EqualFold(agent, "RAVEN-Bot")
		}

		if applies && strings.HasPrefix(line, "Disallow:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "Disallow:"))
			if path != "" {
				paths = append(paths, path)
			}
		}
	}

	rc.mu.Lock()
	rc.disallow[domain] = paths
	rc.mu.Unlock()

	fmt.Printf("  robots.txt fetched for %s — %d disallowed paths\n", domain, len(paths))
}

// IsAllowed returns true if RAVEN is allowed to crawl the given URL
func (rc *RobotsCache) IsAllowed(ctx context.Context, domain, path string) bool {
	rc.mu.Lock()
	_, fetched := rc.disallow[domain]
	rc.mu.Unlock()

	if !fetched {
		rc.fetch(ctx, domain)
	}

	rc.mu.Lock()
	paths := rc.disallow[domain]
	rc.mu.Unlock()

	for _, disallowed := range paths {
		if strings.HasPrefix(path, disallowed) {
			return false
		}
	}
	return true
}
