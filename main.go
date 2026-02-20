package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func worker(id int, jobs <-chan string, wg *sync.WaitGroup) {
	for url := range jobs {
		func() {
			defer wg.Done()

			fmt.Println("Worker", id, "fetching:", url)

			client := &http.Client{}
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", "RAVEN-Bot/1.0")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Worker", id, "error:", err)
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Worker", id, "read error:", err)
				return
			}

			links := extractLinks(string(bodyBytes))

			fmt.Println("Worker", id, "found links:")
			for _, link := range links {
				fmt.Println(" ", link)
			}

		}()
	}
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url> ...")
		os.Exit(1)
	}

	jobs := make(chan string)
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		go worker(i, jobs, &wg)
	}

	for i := 1; i < len(os.Args); i++ {
		wg.Add(1)
		jobs <- os.Args[i]
	}

	wg.Wait()
	close(jobs)

	fmt.Println("Done.")
}
