package crawler

import (
	"context"
	"fmt"
	"net/url"

	"raven/output"
	"raven/parser"
)

type Config struct {
	MaxDepth    int
	MaxPages    int
	WorkerCount int
	OutputPath  string
}

func Run(ctx context.Context, seeds []string, cfg Config) {
	jobs := make(chan Job, cfg.WorkerCount*10)
	results := make(chan Result, cfg.WorkerCount)
	visited := NewVisited()
	rl := NewRateLimiter(1, 3)
	writer := output.NewWriter(cfg.OutputPath)

	for i := 1; i <= cfg.WorkerCount; i++ {
		go Worker(ctx, i, jobs, results, rl)
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
		select {
		case <-ctx.Done():
			fmt.Println("\nCrawl cancelled — shutting down cleanly")
			close(jobs)
			writer.Flush()
			return
		case result := <-results:
			activeJobs--

			job := result.Job
			writer.Add(job.URL, job.Depth, len(result.Links))

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
						select {
						case jobs <- j:
						case <-ctx.Done():
						}
					}(Job{
						URL:      normalized,
						BaseHost: job.BaseHost,
						Depth:    job.Depth + 1,
					})
				}
			}
		}
	}

	close(jobs)
	writer.Flush()
	fmt.Printf("Done. Crawled %d pages → %s\n", visited.Size(), cfg.OutputPath)
}
