package extractor

import (
	"context"
	"net/url"
	"strings"

	"github.com/vishalss1/raven/types"
	"golang.org/x/net/html"
)

// Link extracts every href from anchor tags, resolves relative URLs against
// the page's FinalURL, and normalises (strips query + fragment).
type Link struct{}

func (Link) Extract(_ context.Context, page types.Page) ([]types.Discovery, error) {
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

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				parsed, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				resolved := base.ResolveReference(parsed)
				if resolved.Scheme != "http" && resolved.Scheme != "https" {
					continue
				}
				resolved.RawQuery = ""
				resolved.Fragment = ""
				norm := resolved.String()
				if !seen[norm] {
					seen[norm] = true
					out = append(out, types.Discovery{
						Type:      types.DiscoveryLink,
						Value:     norm,
						SourceURL: page.FinalURL,
					})
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
