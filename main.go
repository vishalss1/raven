package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
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

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func sameHost(baseHost, link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return u.Host == baseHost
}

func extractLinks(body string) []string {
	re := regexp.MustCompile(`href="(http[s]?://[^"]+)"`)
	matches := re.FindAllStringSubmatch(body, -1)
	var links []string
	for _, match := range matches {
		links = append(links, match[1])
	}
	return links
}

func fetch(rawURL string) []string {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "RAVEN-Bot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return extractLinks(string(bodyBytes))
}

func worker(id int, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {
		fmt.Printf("Worker %d [depth %d] fetching: %s\n", id, job.Depth, job.URL)
		links := fetch(job.URL)
		results <- Result{Job: job, Links: links}
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <depth> <maxPages> <url1> <url2> ...")
		os.Exit(1)
	}

	maxDepth, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Depth must be an integer")
		os.Exit(1)
	}

	maxPages, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("maxPages must be an integer")
		os.Exit(1)
	}

	workerCount := 3
	jobs := make(chan Job, workerCount*10)
	results := make(chan Result, workerCount)
	visited := NewVisited()

	for i := 1; i <= workerCount; i++ {
		go worker(i, jobs, results)
	}

	activeJobs := 0

	for i := 3; i < len(os.Args); i++ {
		root := os.Args[i]
		parsed, err := url.Parse(root)
		if err != nil {
			continue
		}
		normalized := normalizeURL(root)
		if visited.CheckAndMark(normalized) {
			activeJobs++
			jobs <- Job{URL: root, BaseHost: parsed.Host, Depth: 0}
		}
	}

	for activeJobs > 0 {
		result := <-results
		activeJobs--

		job := result.Job

		if job.Depth >= maxDepth {
			continue
		}

		for _, link := range result.Links {
			if !sameHost(job.BaseHost, link) {
				continue
			}

			if visited.Size() >= maxPages {
				continue
			}

			normalized := normalizeURL(link)
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
	fmt.Printf("Done. Crawled %d pages.\n", visited.Size())
}
