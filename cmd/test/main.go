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

	eng := engine.New(engine.Config{
		Fetcher: fetcher.NewHTTP(fetcher.Options{}),
		Extractors: []engine.Extractor{
			extractor.Link{},
		},
		Queue:      queue.NewMemory(),
		Workers:    3,
		Visited:    visited.NewMemory(),
		MaxDepth:   2,
		MaxPages:   10,
		DomainOnly: true,
		AllowSubdomains: true,
		OnDiscover: func(parent, child string, depth int) {
			fmt.Printf("  [depth=%d] %s → %s\n", depth, parent, child)
		},
		OnError: func(task types.Task, err error) {
			fmt.Printf("  [ERROR] %s: %v\n", task.URL, err)
		},
	})

	seed := "https://github.com"
	if len(os.Args) > 1 {
		seed = os.Args[1]
	}

	fmt.Printf("crawling %s (maxDepth=2, maxPages=10, domainOnly=true)\n\n", seed)
	graph := eng.Run(ctx, seed)

	fmt.Printf("\n=== Graph: %d nodes, %d edges ===\n", len(graph.Nodes), len(graph.Edges))
	for u, node := range graph.Nodes {
		fmt.Printf("  [depth=%d] %s\n", node.Depth, u)
	}
	for _, edge := range graph.Edges {
		fmt.Printf("  %s → %s\n", edge.From, edge.To)
	}
}
