package crawler

import "sync"

type Visited struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewVisited() *Visited {
	return &Visited{seen: make(map[string]bool)}
}

func (v *Visited) CheckAndMark(u string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen[u] {
		return false
	}
	v.seen[u] = true
	return true
}

func (v *Visited) Size() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.seen)
}
