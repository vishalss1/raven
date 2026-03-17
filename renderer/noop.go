// Package renderer contains Renderer implementations for RAVEN.
//
// # When RAVEN invokes the Renderer
//
// The Engine calls Renderer.Render only when one is configured. If nil,
// it parses the raw HTTP body directly. Configure a browser renderer when:
//
//   - The target page builds its DOM in JS (React/Vue/Angular SPAs).
//   - You need Page.NetworkLogs — XHR/fetch calls the page makes at runtime,
//     surfaced by the API extractor as undocumented endpoint discoveries.
//   - Content is gated behind client-side auth or route transitions.
//
// # Implementing a browser renderer
//
// Satisfy engine.Renderer:
//
//	type Renderer interface {
//	    Render(ctx context.Context, resp types.Response) (types.Page, error)
//	}
//
// Typical Playwright (go-playwright) implementation:
//  1. Launch or reuse a persistent browser context.
//  2. Open a new Page and register a network route handler to populate NetworkLogs.
//  3. Navigate to resp.URL (or set content from resp.Body to skip a second fetch).
//  4. Wait for selector or networkidle.
//  5. Capture page.Content() → RawHTML, page.URL() → FinalURL.
//  6. Parse RawHTML into DOM, assemble and return types.Page.
package renderer

import (
	"context"
	"strings"

	"github.com/vishalss1/raven/types"
	"golang.org/x/net/html"
)

// Noop satisfies engine.Renderer without launching a browser.
// It parses the raw Response body exactly as the Engine would when Renderer
// is nil. Use in tests and as a placeholder until a real browser renderer
// is needed.
type Noop struct{}

func (Noop) Render(_ context.Context, resp types.Response) (types.Page, error) {
	doc, err := html.Parse(strings.NewReader(resp.Body))
	dom := resp.Body
	if err == nil {
		var b strings.Builder
		html.Render(&b, doc) //nolint:errcheck
		dom = b.String()
	}
	return types.Page{
		FinalURL:    resp.URL,
		DOM:         dom,
		RawHTML:     resp.Body,
		NetworkLogs: nil,
	}, nil
}
