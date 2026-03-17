package extractor

import (
	"context"
	"net/url"
	"strings"

	"github.com/vishalss1/raven/types"
	"golang.org/x/net/html"
)

// Form extracts every HTML form: its resolved action URL, HTTP method,
// and the names of its input/textarea/select fields.
type Form struct{}

func (Form) Extract(_ context.Context, page types.Page) ([]types.Discovery, error) {
	base, err := url.Parse(page.FinalURL)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(page.DOM))
	if err != nil {
		return nil, err
	}

	var out []types.Discovery

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			action := nodeAttr(n, "action")
			method := strings.ToUpper(nodeAttr(n, "method"))
			if method == "" {
				method = "GET"
			}

			actionURL := page.FinalURL
			if action != "" {
				if parsed, err := url.Parse(action); err == nil {
					actionURL = base.ResolveReference(parsed).String()
				}
			}

			var fields []string
			collectFields(n, &fields)

			out = append(out, types.Discovery{
				Type:      types.DiscoveryForm,
				Value:     actionURL,
				SourceURL: page.FinalURL,
				Metadata: map[string]any{
					"method": method,
					"fields": fields,
				},
			})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}

func collectFields(form *html.Node, out *[]string) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "input", "textarea", "select":
				if name := nodeAttr(n, "name"); name != "" {
					*out = append(*out, name)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(form)
}
