package engine

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vishalss1/raven/types"
	"github.com/vishalss1/raven/visited"
	"golang.org/x/net/html"
)

// Config controls the crawl engine behaviour.
type Config struct {
	Fetcher    Fetcher
	Renderer   Renderer
	Extractors []Extractor
	Queue      Queue
	Workers    int
	Visited    VisitedStore // nil → no dedup

	MaxDepth   int  // 0 = unlimited
	MaxPages   int  // 0 = unlimited
	MaxEdges   int  // 0 = unlimited
	DomainOnly bool // restrict crawl to the seed URL's host (or subdomains)
	AllowSubdomains bool // if true, allows subdomains of the seed domain

	// OnDiscover is called each time a new edge is found. Optional.
	OnDiscover func(parent, child string, depth int)

	// OnError is called when a fetch or process fails. Optional.
	OnError func(task types.Task, err error)
}

// Engine is a breadth-first crawl engine that builds an in-memory graph.
type Engine struct {
	cfg        Config
	seedDomain string

	mu        sync.Mutex
	nodes     map[string]*types.Node
	edges     []types.Edge
	seenEdges map[string]bool

	pageCount atomic.Int64
	pendingTasks atomic.Int64
	stopped   atomic.Bool
}

// New creates a new Engine with the given Config.
func New(cfg Config) *Engine {
	if cfg.Workers <= 0 {
		cfg.Workers = 5
	}
	return &Engine{
		cfg:       cfg,
		nodes:     make(map[string]*types.Node),
		seenEdges: make(map[string]bool),
	}
}

// Run starts a breadth-first crawl from seedURL and returns the resulting
// navigation graph when all stop conditions are met.
func (e *Engine) Run(ctx context.Context, seedURL string) types.Graph {
	// normalise seed
	seedURL = visited.NormalizeURL(seedURL)

	// extract domain for DomainOnly filtering
	if e.cfg.DomainOnly {
		e.seedDomain = domainOf(seedURL)
	}

	// create seed node
	e.addNode(seedURL, 0)

	// push seed task
	e.pendingTasks.Add(1)
	e.cfg.Queue.Push(types.Task{
		URL:   seedURL,
		Depth: 0,
	})

	// launch workers
	var wg sync.WaitGroup
	wg.Add(e.cfg.Workers)
	for i := 0; i < e.cfg.Workers; i++ {
		go func() {
			defer wg.Done()
			e.work(ctx)
		}()
	}

	wg.Wait()

	e.mu.Lock()
	defer e.mu.Unlock()
	return types.Graph{
		Nodes: e.nodes,
		Edges: e.edges,
	}
}

func (e *Engine) work(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		task, ok := e.cfg.Queue.Pop(ctx)
		if !ok {
			return
		}

		if ctx.Err() != nil {
			return
		}

		// visited check — skip if already seen (seed is pre-marked)
		if e.cfg.Visited != nil && !e.cfg.Visited.CheckAndMark(task.URL) {
			continue
		}

		// maxPages check
		if e.cfg.MaxPages > 0 {
			count := e.pageCount.Add(1)
			if count > int64(e.cfg.MaxPages) {
				// already past the limit — mark done and stop
				e.markDone()
				return
			}
		} else {
			e.pageCount.Add(1)
		}

		// process the page (fetch → render → extract)
		result := e.process(ctx, task)

		// update node with status code
		e.mu.Lock()
		if n, exists := e.nodes[task.URL]; exists {
			n.StatusCode = result.StatusCode
		}
		e.mu.Unlock()

		if result.Err != nil {
			if e.cfg.OnError != nil {
				e.cfg.OnError(task, result.Err)
			}
			e.taskFinished()
			continue
		}

		// enqueue discovered links
		for _, d := range result.Discoveries {
			if d.Type != types.DiscoveryLink {
				continue
			}

			childURL := visited.NormalizeURL(d.Value)
			childDepth := task.Depth + 1

			// depth gate
			if e.cfg.MaxDepth > 0 && childDepth > e.cfg.MaxDepth {
				continue
			}

			// domain gate
			if !e.isAllowedDomain(childURL) {
				continue
			}

			// record edge and deduplicate
			if !e.addEdge(task.URL, childURL) {
				continue // edge limit reached or duplicate edge
			}

			// check if child is already known
			e.mu.Lock()
			_, known := e.nodes[childURL]
			e.mu.Unlock()

			if known {
				continue
			}

			// new node
			e.addNode(childURL, childDepth)

			if e.cfg.OnDiscover != nil {
				e.cfg.OnDiscover(task.URL, childURL, childDepth)
			}

			e.pendingTasks.Add(1)
			e.cfg.Queue.Push(types.Task{
				URL:    childURL,
				Depth:  childDepth,
				Parent: task.URL,
			})
		}

		e.taskFinished()
	}
}

func (e *Engine) taskFinished() {
	if e.pendingTasks.Add(-1) == 0 {
		e.markDone()
	} else {
		e.checkDone() // Still check maxPages/maxEdges
	}
}

// process fetches, renders, and extracts discoveries from a single task.
func (e *Engine) process(ctx context.Context, task types.Task) types.Result {
	base := types.Result{URL: task.URL, Metadata: task.Metadata}

	// 1. Fetch
	resp, err := e.cfg.Fetcher.Fetch(ctx, task)
	if err != nil {
		base.Err = fmt.Errorf("fetch: %w", err)
		return base
	}
	base.StatusCode = resp.StatusCode

	// 2. Render or parse
	var page types.Page
	if e.cfg.Renderer != nil {
		page, err = e.cfg.Renderer.Render(ctx, resp)
		if err != nil {
			base.Err = fmt.Errorf("render: %w", err)
			return base
		}
	} else {
		page = parseHTML(resp)
	}

	// 3. Extract — run all extractors, merge
	var all []types.Discovery
	for _, ex := range e.cfg.Extractors {
		discoveries, exErr := ex.Extract(ctx, page)
		if exErr != nil {
			continue
		}
		all = append(all, discoveries...)
	}

	base.Discoveries = all
	return base
}

// ── graph helpers ───────────────────────────────────────────────────────────

func (e *Engine) addNode(urlStr string, depth int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.nodes[urlStr]; !exists {
		e.nodes[urlStr] = &types.Node{
			URL:   urlStr,
			Depth: depth,
		}
	}
}

func (e *Engine) addEdge(from, to string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := from + "|" + to
	if e.seenEdges[key] {
		return false
	}
	
	if e.cfg.MaxEdges > 0 && len(e.edges) >= e.cfg.MaxEdges {
		return false
	}

	e.seenEdges[key] = true
	e.edges = append(e.edges, types.Edge{From: from, To: to})
	
	if e.cfg.MaxEdges > 0 && len(e.edges) >= e.cfg.MaxEdges {
		go e.markDone() // Non-blocking mark done
	}
	
	return true
}

// ── stop-condition helpers ──────────────────────────────────────────────────

func (e *Engine) markDone() {
	if e.stopped.CompareAndSwap(false, true) {
		e.cfg.Queue.Done()
	}
}

func (e *Engine) checkDone() {
	if e.cfg.MaxPages > 0 && e.pageCount.Load() >= int64(e.cfg.MaxPages) {
		e.markDone()
	}
}

// ── URL helpers ─────────────────────────────────────────────────────────────

func domainOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func (e *Engine) isAllowedDomain(rawURL string) bool {
	if !e.cfg.DomainOnly {
		return true
	}
	host := domainOf(rawURL)
	if e.cfg.AllowSubdomains {
		return host == e.seedDomain || strings.HasSuffix(host, "."+e.seedDomain)
	}
	return host == e.seedDomain
}

func parseHTML(resp types.Response) types.Page {
	doc, err := html.Parse(strings.NewReader(resp.Body))
	dom := resp.Body
	if err == nil {
		var b strings.Builder
		html.Render(&b, doc) //nolint:errcheck
		dom = b.String()
	}
	return types.Page{
		FinalURL: resp.URL,
		DOM:      dom,
		RawHTML:  resp.Body,
	}
}
