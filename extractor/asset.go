package extractor

import (
	"context"
	"net/url"
	"strings"

	"github.com/vishalss1/raven/types"
	"golang.org/x/net/html"
)

// Asset extracts static asset references: images (src), stylesheets
// (link[rel=stylesheet]), and external script tags (script[src]).
type Asset struct{}

func (Asset) Extract(_ context.Context, page types.Page) ([]types.Discovery, error) {
	base, err := url.Parse(page.FinalURL)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(page.DOM))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var out []types.Discovery

	emit := func(t types.DiscoveryType, raw string, meta map[string]any) {
		if raw == "" {
			return
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			return
		}
		resolved := base.ResolveReference(parsed)
		if resolved.Scheme != "http" && resolved.Scheme != "https" {
			return
		}
		norm := resolved.String()
		if seen[norm] {
			return
		}
		seen[norm] = true
		out = append(out, types.Discovery{
			Type:      t,
			Value:     norm,
			SourceURL: page.FinalURL,
			Metadata:  meta,
		})
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "img":
				emit(types.DiscoveryAsset, nodeAttr(n, "src"), map[string]any{"tag": "img"})
			case "link":
				if strings.EqualFold(nodeAttr(n, "rel"), "stylesheet") {
					emit(types.DiscoveryAsset, nodeAttr(n, "href"), map[string]any{"tag": "link", "rel": "stylesheet"})
				}
			case "script":
				if src := nodeAttr(n, "src"); src != "" {
					emit(types.DiscoveryScript, src, map[string]any{"tag": "script"})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}
