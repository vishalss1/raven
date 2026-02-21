package parser

import (
	"net/url"
	"regexp"
)

func ExtractLinks(body string) []string {
	re := regexp.MustCompile(`href="(http[s]?://[^"]+)"`)
	matches := re.FindAllStringSubmatch(body, -1)

	var links []string
	for _, match := range matches {
		links = append(links, match[1])
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
