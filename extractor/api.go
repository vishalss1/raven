package extractor

import (
	"context"
	"regexp"
	"strings"

	"github.com/vishalss1/raven/types"
)

// apiPatterns matches common patterns for API endpoint strings found in JS source:
// fetch("/api/..."), axios.get("/v1/..."), $.ajax({url:"/api/..."}), etc.
var apiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`fetch\(\s*["'` + "`" + `](\/[^"'` + "`" + `\s)]+)["'` + "`" + `]`),
	regexp.MustCompile(`axios\.\w+\(\s*["'` + "`" + `](\/[^"'` + "`" + `\s)]+)["'` + "`" + `]`),
	regexp.MustCompile(`\$\.ajax\(\s*\{[^}]*url\s*:\s*["'](\/[^"']+)["']`),
	regexp.MustCompile(`XMLHttpRequest[^;]*open\(\s*["']\w+["']\s*,\s*["'](\/[^"']+)["']`),
}

// API discovers API endpoints from two sources:
//  1. Page.NetworkLogs — populated when a browser Renderer is active; these are
//     real XHR/fetch calls the page made during rendering.
//  2. Inline JS in the page HTML — static pattern matching for common client-side
//     fetch/axios/XHR patterns. Less reliable than network logs but works without
//     a browser.
type API struct{}

func (API) Extract(_ context.Context, page types.Page) ([]types.Discovery, error) {
	seen := make(map[string]bool)
	var out []types.Discovery

	emit := func(value, method string) {
		key := method + ":" + value
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, types.Discovery{
			Type:      types.DiscoveryAPI,
			Value:     value,
			SourceURL: page.FinalURL,
			Metadata:  map[string]any{"method": method},
		})
	}

	// Source 1: network logs from the Renderer (high confidence)
	for _, entry := range page.NetworkLogs {
		if looksLikeAPI(entry.URL) {
			emit(entry.URL, entry.Method)
		}
	}

	// Source 2: static JS pattern matching (best-effort)
	for _, pat := range apiPatterns {
		for _, match := range pat.FindAllStringSubmatch(page.RawHTML, -1) {
			if len(match) >= 2 {
				emit(match[1], "GET") // method unknown from static analysis
			}
		}
	}

	return out, nil
}

// looksLikeAPI is a heuristic: URLs containing /api/, /v1/, /v2/, /graphql,
// or ending in .json are likely API endpoints rather than page navigations.
func looksLikeAPI(u string) bool {
	lower := strings.ToLower(u)
	indicators := []string{"/api/", "/v1/", "/v2/", "/v3/", "/graphql", ".json", "/rpc/"}
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}
