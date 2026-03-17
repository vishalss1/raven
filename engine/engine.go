package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/vishalss1/raven/types"
	"golang.org/x/net/html"
)

type Config struct {
	Fetcher    Fetcher
	Renderer   Renderer
	Extractors []Extractor
	Queue      Queue
	Workers    int
	Visited    VisitedStore // nil → no dedup, caller handles it
	OnResult   func(types.Result)
	OnError    func(task types.Task, err error)
}

type Engine struct {
	cfg Config
}

func New(cfg Config) *Engine {
	if cfg.Workers <= 0 {
		cfg.Workers = 5
	}
	return &Engine{cfg: cfg}
}

func (e *Engine) Run(ctx context.Context) {
	done := make(chan struct{}, e.cfg.Workers)

	for i := 0; i < e.cfg.Workers; i++ {
		go func(id int) {
			e.work(ctx, id)
			done <- struct{}{}
		}(i + 1)
	}

	for i := 0; i < e.cfg.Workers; i++ {
		<-done
	}
}

func (e *Engine) work(ctx context.Context, id int) {
	for {
		task, ok := e.cfg.Queue.Pop(ctx)
		if !ok {
			return
		}

		// visited check — skip if already seen
		if e.cfg.Visited != nil && !e.cfg.Visited.CheckAndMark(task.URL) {
			continue
		}

		result := e.process(ctx, task)
		if result.Err != nil {
			if e.cfg.OnError != nil {
				e.cfg.OnError(task, result.Err)
			} else {
				fmt.Printf("[worker %d] error processing %s: %v\n", id, task.URL, result.Err)
			}
		}

		if e.cfg.OnResult != nil {
			e.cfg.OnResult(result)
		}
	}
}

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
		discoveries, err := ex.Extract(ctx, page)
		if err != nil {
			fmt.Printf("  extractor error on %s: %v\n", task.URL, err)
			continue
		}
		all = append(all, discoveries...)
	}

	base.Discoveries = all
	return base
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
