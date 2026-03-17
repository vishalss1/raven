package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/vishalss1/raven/types"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 3
	defaultBaseDelay  = 2 * time.Second
	defaultUserAgent  = "RAVEN-Bot/1.0"
)

// Options configures the HTTP fetcher.
type Options struct {
	Timeout    time.Duration
	MaxRetries int
	BaseDelay  time.Duration
	UserAgent  string
	// RateLimit controls requests-per-second per domain (0 = unlimited).
	RateLimitRPS   float64
	RateLimitBurst float64
}

func (o *Options) withDefaults() Options {
	if o == nil {
		o = &Options{}
	}
	out := *o
	if out.Timeout == 0 {
		out.Timeout = defaultTimeout
	}
	if out.MaxRetries == 0 {
		out.MaxRetries = defaultMaxRetries
	}
	if out.BaseDelay == 0 {
		out.BaseDelay = defaultBaseDelay
	}
	if out.UserAgent == "" {
		out.UserAgent = defaultUserAgent
	}
	if out.RateLimitRPS == 0 {
		out.RateLimitRPS = 1
	}
	if out.RateLimitBurst == 0 {
		out.RateLimitBurst = 3
	}
	return out
}

// HTTP is a plain HTTP/HTTPS Fetcher with per-domain rate limiting and
// exponential-backoff retries on 5xx responses.
type HTTP struct {
	opts    Options
	client  *http.Client
	limiter *rateLimiter
}

func NewHTTP(opts Options) *HTTP {
	o := opts.withDefaults()
	return &HTTP{
		opts:    o,
		client:  &http.Client{Timeout: o.Timeout},
		limiter: newRateLimiter(o.RateLimitRPS, o.RateLimitBurst),
	}
}

func (f *HTTP) Fetch(ctx context.Context, task types.Task) (types.Response, error) {
	method := task.Method
	if method == "" {
		method = http.MethodGet
	}

	var (
		resp types.Response
		err  error
	)

	for attempt := 0; attempt < f.opts.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return types.Response{}, ctx.Err()
		}

		// rate-limit per domain
		f.limiter.wait(domainOf(task.URL))

		resp, err = f.do(ctx, method, task)
		if err == nil {
			return resp, nil
		}

		// only retry on server-side errors
		if resp.StatusCode > 0 && resp.StatusCode < 500 {
			return resp, err
		}

		if ctx.Err() != nil {
			return types.Response{}, ctx.Err()
		}

		delay := f.opts.BaseDelay * time.Duration(1<<attempt)
		fmt.Printf("  retry %d/%d for %s (wait %s): %v\n",
			attempt+1, f.opts.MaxRetries, task.URL, delay, err)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return types.Response{}, ctx.Err()
		}
	}

	return types.Response{}, fmt.Errorf("gave up after %d attempts: %w", f.opts.MaxRetries, err)
}

func (f *HTTP) do(ctx context.Context, method string, task types.Task) (types.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, task.URL, nil)
	if err != nil {
		return types.Response{}, err
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)
	for k, v := range task.Headers {
		req.Header.Set(k, v)
	}

	httpResp, err := f.client.Do(req)
	if err != nil {
		return types.Response{}, err
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return types.Response{}, err
	}

	r := types.Response{
		URL:        httpResp.Request.URL.String(), // follows redirects
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       string(raw),
		RawBytes:   raw,
	}

	if httpResp.StatusCode >= 500 {
		return r, fmt.Errorf("server error: %d", httpResp.StatusCode)
	}

	return r, nil
}

// domainOf extracts the host from a URL cheaply.
func domainOf(rawURL string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(rawURL) > len(prefix) && rawURL[:len(prefix)] == prefix {
			rest := rawURL[len(prefix):]
			for i, c := range rest {
				if c == '/' || c == '?' || c == '#' {
					return rest[:i]
				}
			}
			return rest
		}
	}
	return rawURL
}

// ── token-bucket rate limiter (per domain) ─────────────────────────────────

type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	max        float64
	refillRate float64
	lastRefill time.Time
}

func (tb *tokenBucket) wait() {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastRefill).Seconds()
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.max {
			tb.tokens = tb.max
		}
		tb.lastRefill = now

		if tb.tokens >= 1.0 {
			tb.tokens--
			tb.mu.Unlock()
			return
		}
		wait := time.Duration((1.0-tb.tokens)/tb.refillRate*1000) * time.Millisecond
		tb.mu.Unlock()
		time.Sleep(wait)
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   float64
}

func newRateLimiter(rps, burst float64) *rateLimiter {
	return &rateLimiter{buckets: make(map[string]*tokenBucket), rps: rps, burst: burst}
}

func (rl *rateLimiter) wait(domain string) {
	rl.mu.Lock()
	b, ok := rl.buckets[domain]
	if !ok {
		b = &tokenBucket{tokens: rl.burst, max: rl.burst, refillRate: rl.rps, lastRefill: time.Now()}
		rl.buckets[domain] = b
	}
	rl.mu.Unlock()
	b.wait()
}
