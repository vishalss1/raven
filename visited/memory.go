package visited

import "sync"

type Memory struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewMemory() *Memory {
	return &Memory{seen: make(map[string]bool)}
}

func (v *Memory) CheckAndMark(url string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.seen[url] {
		return false
	}
	v.seen[url] = true
	return true
}

func (v *Memory) Size() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.seen)
}
