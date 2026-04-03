package visited

import (
	"net/url"
	"strings"
	"sync"
)

// NormalizeURL canonicalises a URL for deduplication:
//   - force scheme to https (treat http and https as equivalent)
//   - lowercase host
//   - strip fragment (#...)
//   - strip query parameters (?ref=...)
//   - remove trailing slash from path
//   - collapse duplicate slashes in path
func NormalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	// treat http and https as same resource
	if u.Scheme == "http" {
		u.Scheme = "https"
	}

	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawQuery = ""
	u.ForceQuery = false

	// collapse duplicate slashes and remove trailing slash
	u.Path = collapseSlashes(u.Path)
	u.Path = strings.TrimRight(u.Path, "/")

	return u.String()
}

// collapseSlashes replaces runs of consecutive slashes with a single slash.
func collapseSlashes(path string) string {
	var b strings.Builder
	b.Grow(len(path))
	prev := byte(0)
	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '/' && prev == '/' {
			continue
		}
		b.WriteByte(c)
		prev = c
	}
	return b.String()
}

type Memory struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewMemory() *Memory {
	return &Memory{seen: make(map[string]bool)}
}

// CheckAndMark normalises the URL then returns true if new (and marks it).
// Returns false if already seen.
func (v *Memory) CheckAndMark(rawURL string) bool {
	norm := NormalizeURL(rawURL)
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen[norm] {
		return false
	}
	v.seen[norm] = true
	return true
}

func (v *Memory) Size() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.seen)
}
