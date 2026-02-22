package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"raven/parser"
)

type Job struct {
	URL      string
	BaseHost string
	Depth    int
}

type Result struct {
	Job   Job
	Links []string
}

const (
	maxRetries = 3
	baseDelay  = 2 * time.Second
)

func fetch(ctx context.Context, rawURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RAVEN-Bot/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// treat 5xx as retryable errors
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parser.ExtractLinks(string(bodyBytes), rawURL), nil
}

func fetchWithRetry(ctx context.Context, rawURL string) []string {
	for attempt := 0; attempt < maxRetries; attempt++ {
		links, err := fetch(ctx, rawURL)
		if err == nil {
			return links
		}

		// don't retry if context is cancelled
		if ctx.Err() != nil {
			return nil
		}

		// exponential backoff: 2s, 4s, 8s
		delay := baseDelay * time.Duration(1<<attempt)
		fmt.Printf("  retry %d/%d for %s (wait %s): %v\n",
			attempt+1, maxRetries, rawURL, delay, err)

		select {
		case <-time.After(delay):
			// wait then retry
		case <-ctx.Done():
			return nil // cancelled during wait
		}
	}

	fmt.Printf("  gave up on %s after %d attempts\n", rawURL, maxRetries)
	return nil
}

func Worker(ctx context.Context, id int, jobs <-chan Job, results chan<- Result, rl *RateLimiter) {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d shutting down\n", id)
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			fmt.Printf("Worker %d [depth %d] %s fetching: %s\n",
				id, job.Depth, time.Now().Format("15:04:05.000"), job.URL)
			rl.Wait(job.BaseHost)
			links := fetchWithRetry(ctx, job.URL)
			results <- Result{Job: job, Links: links}
		}
	}
}
