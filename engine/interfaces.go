package engine

import (
	"context"

	"github.com/vishalss1/raven/types"
)

// Fetcher performs the HTTP request for a Task and returns a raw Response.
// Swap implementations freely — plain HTTP, mTLS, proxy-aware, etc.
type Fetcher interface {
	Fetch(ctx context.Context, task types.Task) (types.Response, error)
}

// Renderer takes a raw Response and returns a fully-rendered Page.
// Implement this with a headless browser (Playwright, rod, chromedp) to
// handle JS-rendered content and populate Page.NetworkLogs.
// When nil, the Engine parses Response.Body directly into a Page.
type Renderer interface {
	Render(ctx context.Context, resp types.Response) (types.Page, error)
}

// Extractor inspects a Page and returns zero or more Discoveries.
// Each Extractor is independent — add new ones without touching the Engine.
type Extractor interface {
	Extract(ctx context.Context, page types.Page) ([]types.Discovery, error)
}

// Queue is the task transport for the Engine workers.
// Push adds work dynamically during the crawl.
// Pop blocks until a task is available, the queue is done, or ctx is cancelled.
// Done signals that no more tasks will be pushed — workers drain then stop.
type Queue interface {
	Push(task types.Task)
	Pop(ctx context.Context) (types.Task, bool) // false → queue done+empty or ctx cancelled
	Done()
}

// VisitedStore tracks which URLs have already been processed.
// CheckAndMark returns true if the URL is new and marks it as seen.
// Returns false if it was already seen — engine skips it.
type VisitedStore interface {
	CheckAndMark(url string) bool
}
