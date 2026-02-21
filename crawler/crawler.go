package crawler

import (
	"net/url"

	"raven/parser"
)

type Config struct {
	MaxDepth    int
	MaxPages    int
	WorkerCount int
}

func Run(seeds []string, cfg Config) {
	jobs := make(chan Job, cfg.WorkerCount*10)
	results := make(chan Result, cfg.WorkerCount)
	visited := NewVisited()
	rl := NewRateLimiter(1, 3) // 1 req/sec per domain, burst of 3

	for i := 1; i <= cfg.WorkerCount; i++ {
		go Worker(i, jobs, results, rl)
	}

	activeJobs := 0

	for _, seed := range seeds {
		parsed, err := url.Parse(seed)
		if err != nil {
			continue
		}
		normalized := parser.NormalizeURL(seed)
		if visited.CheckAndMark(normalized) {
			activeJobs++
			jobs <- Job{URL: normalized, BaseHost: parsed.Host, Depth: 0}
		}
	}

	for activeJobs > 0 {
		result := <-results
		activeJobs--

		job := result.Job

		if job.Depth >= cfg.MaxDepth {
			continue
		}

		for _, link := range result.Links {
			if !parser.SameHost(job.BaseHost, link) {
				continue
			}
			if visited.Size() >= cfg.MaxPages {
				continue
			}
			normalized := parser.NormalizeURL(link)
			if visited.CheckAndMark(normalized) {
				activeJobs++
				go func(j Job) {
					jobs <- j
				}(Job{
					URL:      normalized,
					BaseHost: job.BaseHost,
					Depth:    job.Depth + 1,
				})
			}
		}
	}

	close(jobs)
}
