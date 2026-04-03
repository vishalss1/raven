# RAVEN

**Rate-Limited Asynchronous Visited-Aware Engine for Network Crawling**

A programmable breadth-first crawl engine for Go that outputs a structured navigation graph.

---

## One import

```go
import "github.com/vishalss1/raven"
```

That's it. Everything you need comes from that one package.

---

## Usage

```go
eng := raven.NewEngine(raven.Config{
    Fetcher:    raven.NewHTTP(fetcher.Options{}),
    Extractors: raven.DefaultExtractors,
    Queue:      raven.NewQueue(),
    Workers:    5,
    Visited:    raven.NewVisited(),
    MaxDepth:   3,
    MaxPages:   100,
    DomainOnly: true,
    OnDiscover: func(parent, child string, depth int) {
        fmt.Printf("[depth=%d] %s → %s\n", depth, parent, child)
    },
    OnError: func(task raven.Task, err error) {
        fmt.Printf("Error crawling %s: %v\n", task.URL, err)
    },
})

graph := eng.Run(ctx, "https://example.com")

fmt.Printf("Nodes: %d, Edges: %d\n", len(graph.Nodes), len(graph.Edges))
```

---

## What it does

You call `Run(ctx, seedURL)`. RAVEN performs a breadth-first crawl starting from the seed, following links across pages while respecting depth limits and domain constraints. It returns a `Graph` — a set of nodes (URLs with depth) and edges (parent → child relationships).

Each page is fetched, optionally rendered (for JS-heavy pages), and then extractors pull out every link. Discovered links become new tasks pushed into the queue automatically. The engine terminates when:

- the queue is empty (all reachable pages crawled)
- `MaxPages` is reached
- the context is cancelled

---

## Graph Output

The crawl produces a structured graph:

```go
type Graph struct {
    Nodes map[string]*Node  // keyed by normalised URL
    Edges []Edge            // parent → child relationships
}

type Node struct {
    URL        string
    Depth      int
    StatusCode int    // populated during crawl
    LatencyMs  int64  // placeholder for future use
}

type Edge struct {
    From string
    To   string
}
```

---

## Configuration

```go
raven.Config{
    Fetcher     // how to fetch — default: raven.NewHTTP()
    Renderer    // optional browser renderer for JS-heavy pages
    Extractors  // what to extract — default: raven.DefaultExtractors (Link only)
    Queue       // task queue — default: raven.NewQueue()
    Visited     // deduplication — default: raven.NewVisited()
    Workers     // goroutine concurrency (default: 5)

    MaxDepth    // max link depth from seed (0 = unlimited)
    MaxPages    // max pages to crawl (0 = unlimited)
    MaxEdges    // max edges in graph (0 = unlimited)
    DomainOnly  // restrict to seed URL's host or subdomains
    AllowSubdomains // allow subdomains of the seed host

    OnDiscover  // callback: func(parent, child string, depth int)
    OnError     // callback: func(task raven.Task, err error)
}
```

---

## URL Normalization

RAVEN normalises all URLs before deduplication:

- force scheme to `https` (treats http/https as same)
- lowercase host
- strip query parameters (`?ref=...`)
- strip fragments (`#...`)
- remove trailing slash
- collapse duplicate slashes (`//`)

Use `raven.NormalizeURL(url)` directly if needed.

---

## Domain Restriction

When `DomainOnly: true`, the engine only enqueues URLs that share the same hostname as the seed. Cross-domain links still appear as edges in the graph but are not crawled.

---

## Extractors

The default extractor set for graph crawling is `Link` only. Additional extractors are available:

| Extractor | What it finds |
|---|---|
| `extractor.Link{}` | Every `href` on the page, resolved and normalised |
| `extractor.Form{}` | Form action URL, method, and field names |
| `extractor.Asset{}` | Images, stylesheets |
| `extractor.API{}` | Endpoints found in network logs or JS source |

Pass them explicitly if you need richer discovery data:

```go
Extractors: []engine.Extractor{
    extractor.Link{},
    extractor.Form{},
    extractor.Asset{},
    extractor.API{},
},
```

Only `Link` discoveries are used for graph edge creation and task enqueuing.

---

## Plugging in a browser renderer

By default RAVEN parses raw HTML. For JS-rendered pages, implement `engine.Renderer` with a headless browser (Playwright, rod, chromedp) and pass it into `Config.Renderer`. The extractors receive a `Page` either way — they never know whether a browser was involved.

See `renderer/noop.go` for the implementation guide.

---

## Bringing your own deduplication

`raven.NewVisited()` is an in-memory set with URL normalisation — fine for a single process. For distributed crawls, implement `engine.VisitedStore`:

```go
type VisitedStore interface {
    CheckAndMark(url string) bool
}
```

Back it with Redis or Postgres and pass it in. RAVEN doesn't care what's behind the interface.