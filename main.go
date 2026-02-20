package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

func worker(id int, jobs <-chan string, wg *sync.WaitGroup) {
	for url := range jobs {
		fmt.Println("Worker", id, "feching:", url)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Worker", id, "error:", err)
			wg.Done()
			continue
		}

		fmt.Println("Worker", id, "status:", resp.Status)
		resp.Body.Close()

		wg.Done()
	}
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
