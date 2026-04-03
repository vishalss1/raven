package raven

import (
	"github.com/vishalss1/raven/engine"
	"github.com/vishalss1/raven/extractor"
	"github.com/vishalss1/raven/fetcher"
	"github.com/vishalss1/raven/queue"
	"github.com/vishalss1/raven/types"
	"github.com/vishalss1/raven/visited"
)

// re-export everything the caller needs
type Config = engine.Config
type Task = types.Task
type Node = types.Node
type Edge = types.Edge
type Graph = types.Graph

// keep for backward compatibility — still used internally
type Result = types.Result
type Discovery = types.Discovery
type DiscoveryType = types.DiscoveryType

var (
	NewEngine    = engine.New
	NewQueue     = queue.NewMemory
	NewVisited   = visited.NewMemory
	NewHTTP      = fetcher.NewHTTP
	NormalizeURL = visited.NormalizeURL

	// DefaultExtractors for graph crawling — only links matter for graph edges.
	// Pass additional extractors (Form, Asset, API) explicitly if needed.
	DefaultExtractors = []engine.Extractor{
		extractor.Link{},
	}
)
