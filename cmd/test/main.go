package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/vishalss1/raven/engine"
	"github.com/vishalss1/raven/extractor"
	"github.com/vishalss1/raven/fetcher"
	"github.com/vishalss1/raven/queue"
	"github.com/vishalss1/raven/types"
	"github.com/vishalss1/raven/visited"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	q := queue.NewMemory(64)

	eng := engine.New(engine.Config{
		Fetcher: fetcher.NewHTTP(fetcher.Options{}),
		Extractors: []engine.Extractor{
			extractor.Link{},
			extractor.Form{},
			extractor.Asset{},
			extractor.API{},
		},
		Queue:   q,
		Workers: 3,
		Visited: visited.NewMemory(),
		OnResult: func(r types.Result) {
			fmt.Printf("[%d] %s — %d discoveries\n",
				r.StatusCode, r.URL, len(r.Discoveries))
			for _, d := range r.Discoveries {
				fmt.Printf("    %s  %s\n", d.Type, d.Value)
			}
		},
		OnError: func(task types.Task, err error) {
			fmt.Printf("  ERROR %s: %v\n", task.URL, err)
		},
	})

	// seed one URL
	q.Push(types.Task{URL: "https://github.com"})
	q.Close() // single page — close immediately after seeding

	eng.Run(ctx)
	fmt.Println("done")
}
