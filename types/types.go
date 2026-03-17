package types

import "net/http"

// Task is the unit of work Araneae hands to RAVEN.
// Metadata is opaque — RAVEN passes it through untouched to Result.
type Task struct {
	URL      string
	Method   string // defaults to GET if empty
	Headers  map[string]string
	Depth    int
	Metadata map[string]any
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

// Result is what RAVEN emits back to Araneae after processing a Task.
// Metadata is the same map that arrived in the Task.
type Result struct {
	URL         string
	StatusCode  int
	Discoveries []Discovery
	Metadata    map[string]any
	Err         error
}
