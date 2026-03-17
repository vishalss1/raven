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
type Result = types.Result
type Task = types.Task
type Discovery = types.Discovery
type DiscoveryType = types.DiscoveryType

var (
	NewEngine  = engine.New
	NewQueue   = queue.NewMemory
	NewVisited = visited.NewMemory
	NewHTTP    = fetcher.NewHTTP

	// default extractor set
	DefaultExtractors = []engine.Extractor{
		extractor.Link{},
		extractor.Form{},
		extractor.Asset{},
		extractor.API{},
	}
)
