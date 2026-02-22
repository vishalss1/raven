#  RAVEN

**Rate-Limited Asynchronous Visited-Aware Engine for Network Crawling**

A concurrent, domain-scoped web crawler written in Go. Built as a systems learning project — and evolved into a programmable crawl engine you can embed directly into your Go backend.

---

## What RAVEN Is

Most crawlers are tools you run. RAVEN is a library you call.

```go
cfg := crawler.Config{
    MaxDepth:    2,
    MaxPages:    50,
    WorkerCount: 3,
    OutputPath:  "results.json",
    OnResult: func(url string, depth, linksFound int) {
        db.Insert(url, depth, linksFound)   // your logic here
        queue.Push(url)                      // pipe anywhere
    },
}

crawler.Run(ctx, []string{"https://example.com"}, cfg)
```

Results stream into your code in real time via a callback. The JSON output and the callback run in parallel — zero coupling.

---

## Features

- **Concurrent worker pool** — N goroutines crawling in parallel
- **BFS crawling** — breadth-first graph traversal with depth control
- **Per-domain rate limiting** — manual token bucket, configurable burst and refill
- **Visited-aware deduplication** — thread-safe, URL-normalized
- **Relative link resolution** — resolves `href="/about"` against base URL correctly
- **robots.txt respect** — fetched once per domain, cached, applied to every URL
- **Retry with exponential backoff** — 2s → 4s → 8s, context-aware
- **Graceful shutdown** — Ctrl+C cancels in-flight requests cleanly via context
- **Structured JSON output** — url, depth, links_found, crawled_at per page
- **Streaming results callback** — embed RAVEN, consume results in real time

---

## Installation

```bash
git clone https://github.com/visha/raven
cd raven
go mod tidy
```

---

## CLI Usage

```bash
go run main.go <depth> <maxPages> <url>
```

```bash
# crawl github.com up to depth 2, max 50 pages
go run main.go 2 50 "https://github.com"

# multiple seeds
go run main.go 3 100 "https://github.com" "https://wikipedia.org"
```

Output is written to `results.json` in the project root.

---

## Library Usage

```go
import "github.com/visha/raven/crawler"

cfg := crawler.Config{
    MaxDepth:    3,
    MaxPages:    100,
    WorkerCount: 5,
    OutputPath:  "out.json",
    OnResult: func(url string, depth, linksFound int) {
        // fires for every crawled page in real time
        fmt.Println(url, depth, linksFound)
    },
}

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

crawler.Run(ctx, []string{"https://example.com"}, cfg)
```

---

## Architecture

```
main.go
  └── crawler.Run(ctx, seeds, config)
        ├── Worker Pool (N goroutines)
        │     ├── RateLimiter.Wait(domain)   per-domain token bucket
        │     ├── fetchWithRetry(ctx, url)   3 attempts, exponential backoff
        │     └── results <- Result
        │
        ├── Coordinator Loop
        │     ├── <-results                  drain results
        │     ├── RobotsCache.IsAllowed()    check before dispatch
        │     ├── Visited.CheckAndMark()     deduplication
        │     ├── cfg.OnResult()             stream to caller
        │     └── jobs <- Job               dispatch new work
        │
        └── Output Writer
              └── Flush() → results.json    on finish or Ctrl+C
```

**Web as a graph:**
- URLs = vertices
- Links = edges
- Workers = concurrent explorers
- jobs channel = frontier queue (BFS)

---

## Project Structure

```
raven/
├── main.go
├── crawler/
│   ├── crawler.go       coordinator loop, config, lifecycle
│   ├── worker.go        worker pool, fetch, retry logic
│   ├── visited.go       thread-safe deduplication
│   ├── ratelimiter.go   token bucket per domain
│   └── robots.go        robots.txt fetch, parse, cache
├── parser/
│   └── parser.go        ExtractLinks, NormalizeURL, SameHost
├── output/
│   └── output.go        JSON writer
└── go.mod
```

---

## Design Decisions

**Why a counter instead of WaitGroup for active jobs?**
`sync.WaitGroup` requires `Add()` before `Done()`. In a concurrent crawler, new jobs are discovered inside goroutines — making the Add/Done ordering impossible to guarantee safely. An `activeJobs` integer owned entirely by the coordinator loop avoids this race entirely.

**Why goroutines for job dispatch?**
The coordinator is both a consumer of results and a producer of jobs. If it blocks sending to a full `jobs` channel while workers are blocked trying to send to a full `results` channel — deadlock. Dispatching jobs in goroutines keeps the coordinator free to drain results.

**Why a manual token bucket instead of `golang.org/x/time/rate`?**
To understand it. The stdlib package is a well-tested token bucket — building it manually first means the abstraction is never a black box. Swapping to the stdlib version is a 10 minute refactor.

**Why `context` flows through `fetch()`?**
So Ctrl+C cancels in-flight HTTP requests immediately, not after the current fetch completes. `http.NewRequestWithContext` propagates the cancellation down to the transport layer.

**Why `OnResult` is a func and not a channel?**
A callback is simpler to embed — the caller doesn't manage a goroutine or drain a channel. If the caller wants a channel they can make one inside the callback. Keeps the API surface minimal.

---

## Versioning

| Version | What changed |
|---|---|
| v3 | BFS crawling, depth + page cap, deadlock fix |
| v4 | Modular structure, per-domain rate limiting |
| v5 | Context cancellation, graceful shutdown |
| v6 | Relative link resolution, JSON output layer |
| v7 | Retry with exponential backoff |
| v8 | robots.txt respect |
| v9 | Streaming results callback |
