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

func fetch(ctx context.Context, rawURL string) []string {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "RAVEN-Bot/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return parser.ExtractLinks(string(bodyBytes))
}

func Worker(ctx context.Context, id int, jobs <-chan Job, results chan<- Result, rl *RateLimiter) {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d shutting down\n", id)
			return
		case job, ok := <-jobs:
			if !ok {
				return // jobs channel closed
			}
			fmt.Printf("Worker %d [depth %d] %s fetching: %s\n",
				id, job.Depth, time.Now().Format("15:04:05.000"), job.URL)
			rl.Wait(job.BaseHost)
			links := fetch(ctx, job.URL)
			results <- Result{Job: job, Links: links}
		}
	}
}
