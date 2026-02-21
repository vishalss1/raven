package crawler

import (
	"fmt"
	"io"
	"net/http"
	"raven/parser"
	"time"
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

func fetch(rawURL string) []string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil
	}
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
	return parser.ExtractLinks(string(bodyBytes))
}

func Worker(id int, jobs <-chan Job, results chan<- Result, rl *RateLimiter) {
	for job := range jobs {
		//fmt.Printf("Worker %d [depth %d] fetching: %s\n", id, job.Depth, job.URL)
		fmt.Printf("Worker %d [depth %d] %s fetching: %s\n", id, job.Depth, time.Now().Format("15:04:05.000"), job.URL)
		rl.Wait(job.BaseHost) // blocks here if too fast
		links := fetch(job.URL)
		results <- Result{Job: job, Links: links}
	}
}
