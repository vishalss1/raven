package types

import "net/http"

// Task is the unit of work the Engine processes.
// Parent tracks which URL spawned this task (empty for the seed).
type Task struct {
	URL      string
	Method   string // defaults to GET if empty
	Headers  map[string]string
	Depth    int
	Parent   string         // URL that discovered this task
	Metadata map[string]any // opaque — passed through untouched
}

// Response is the raw HTTP result produced by a Fetcher.
type Response struct {
	URL        string
	StatusCode int
	Headers    http.Header
	Body       string
	RawBytes   []byte
}

// Page is what Extractors consume. Produced by a Renderer (or by
// parsing RawBytes directly when no Renderer is configured).
type Page struct {
	FinalURL    string
	DOM         string // parsed HTML as a string; use with golang.org/x/net/html
	RawHTML     string
	NetworkLogs []NetworkEntry // populated only when a Renderer is active
}

// NetworkEntry records a single outbound request the page made.
type NetworkEntry struct {
	URL    string
	Method string
}

// DiscoveryType labels what kind of thing was found.
type DiscoveryType string

const (
	DiscoveryLink   DiscoveryType = "link"
	DiscoveryForm   DiscoveryType = "form"
	DiscoveryAPI    DiscoveryType = "api"
	DiscoveryAsset  DiscoveryType = "asset"
	DiscoveryScript DiscoveryType = "script"
	DiscoveryOther  DiscoveryType = "other"
)

// Discovery is the atomic output of an Extractor.
type Discovery struct {
	Type      DiscoveryType
	Value     string
	SourceURL string
	Metadata  map[string]any
}

// Result is what the engine produces after processing a Task internally.
type Result struct {
	URL         string
	StatusCode  int
	Discoveries []Discovery
	Metadata    map[string]any
	Err         error
}

// ── Graph output types ─────────────────────────────────────────────────────

// Node represents a single URL in the crawl graph.
type Node struct {
	URL        string `json:"url"`
	Depth      int    `json:"depth"`
	StatusCode int    `json:"status_code,omitempty"` // placeholder — not populated yet
	LatencyMs  int64  `json:"latency_ms,omitempty"`  // placeholder — not populated yet
}

// Edge represents a parent → child link in the crawl graph.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Graph is the structured output of a crawl: all discovered nodes and
// the edges connecting them.
type Graph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges []Edge           `json:"edges"`
}
