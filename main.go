package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"raven/crawler"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <depth> <maxPages> <url1> <url2> ...")
		os.Exit(1)
	}

	maxDepth, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("depth must be an integer")
		os.Exit(1)
	}

	maxPages, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("maxPages must be an integer")
		os.Exit(1)
	}

	seeds := os.Args[3:]

	// ctx cancels automatically on Ctrl+C
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := crawler.Config{
		MaxDepth:    maxDepth,
		MaxPages:    maxPages,
		WorkerCount: 3,
	}

	crawler.Run(ctx, seeds, cfg)
}
