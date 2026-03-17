# RAVEN

**Rate-Limited Asynchronous Visited-Aware Engine for Network Crawling**

A programmable crawl engine for Go.

---

## One import

```go
import "github.com/vishalss1/raven"
```

That's it. Everything you need comes from that one package.

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

## What it does

You push a `Task` into the queue. RAVEN fetches the page, extracts everything on it, and calls `OnResult` with what it found. That's the whole loop.

What you do with the result — enqueue new URLs, write to a database, filter by domain, track depth — is entirely up to you. RAVEN has no opinions about crawl strategy.

---

## Discoveries

Every page produces a list of typed discoveries:

| Type | What it is |
|---|---|
| `link` | Every `href` on the page, resolved and normalised |
| `form` | Form action URL, method, and field names |
| `asset` | Images, stylesheets |
| `script` | External JS files |
| `api` | Endpoints found in network logs or JS source |

```go
type Discovery struct {
    Type      string         // link | form | asset | script | api
    Value     string         // the URL
    SourceURL string         // which page it came from
    Metadata  map[string]any // extractor-specific detail
}
```

---

## Configuration

```go
raven.Config{
    Fetcher    // how to fetch — default: raven.NewHTTP()
    Renderer   // optional browser renderer for JS-heavy pages
    Extractors // what to extract — default: raven.DefaultExtractors
    Queue      // task queue — default: raven.NewQueue()
    Visited    // deduplication — default: raven.NewVisited()
    Workers    // goroutine concurrency
    OnResult   // called for every completed page
    OnError    // called on fetch failure
}
```

All fields except `Fetcher` and `Queue` are optional.

---

## Plugging in a browser renderer

By default RAVEN parses raw HTML. For JS-rendered pages, implement `raven.Renderer` with a headless browser (Playwright, rod, chromedp) and pass it into `Config.Renderer`. The extractors receive a `Page` either way — they never know whether a browser was involved.

See `renderer/noop.go` for the implementation guide.

---

## Bringing your own deduplication

`raven.NewVisited()` is an in-memory set — fine for a single process. For distributed crawls, implement `raven.VisitedStore`:

```go
type VisitedStore interface {
    CheckAndMark(url string) bool
}
```

Back it with Redis or Postgres and pass it in. RAVEN doesn't care what's behind the interface.