package crawler

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func newTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := time.Since(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now
}

func (tb *TokenBucket) Wait() {
	for {
		tb.mu.Lock()
		tb.refill()

		if tb.tokens >= 1.0 {
			tb.tokens -= 1.0
			tb.mu.Unlock()
			return
		}

		waitTime := time.Duration((1.0 - tb.tokens) / tb.refillRate * float64(time.Second))
		tb.mu.Unlock()

		time.Sleep(waitTime)
	}
}

type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*TokenBucket
	refillRate float64
	burst      float64
}

func NewRateLimiter(refillRate, burst float64) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		refillRate: refillRate,
		burst:      burst,
	}
}

// Wait blocks until the token bucket for the given domain allows a request
func (rl *RateLimiter) Wait(domain string) {
	rl.mu.Lock()
	bucket, exists := rl.buckets[domain]
	if !exists {
		bucket = newTokenBucket(rl.burst, rl.refillRate)
		rl.buckets[domain] = bucket
	}
	rl.mu.Unlock()

	bucket.Wait()
}
