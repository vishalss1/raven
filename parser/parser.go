package parser

import (
	"net/url"
	"regexp"
)

func ExtractLinks(body, baseURL string) []string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`href="([^"]+)"`)
	matches := re.FindAllStringSubmatch(body, -1)

	seen := make(map[string]bool)
	var links []string

	for _, match := range matches {
		raw := match[1]

		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}

		// resolve relative to base
		resolved := base.ResolveReference(parsed)

		// only http/https
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			continue
		}

		normalized := NormalizeURL(resolved.String())

		if !seen[normalized] {
			seen[normalized] = true
			links = append(links, normalized)
		}
	}

	return links
}

func NormalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func SameHost(baseHost, link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return u.Host == baseHost
}
