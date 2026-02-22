package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Record struct {
	URL        string    `json:"url"`
	Depth      int       `json:"depth"`
	LinksFound int       `json:"links_found"`
	CrawledAt  time.Time `json:"crawled_at"`
}

type Writer struct {
	mu      sync.Mutex
	records []Record
	path    string
}

func NewWriter(path string) *Writer {
	return &Writer{
		path:    path,
		records: []Record{},
	}
}

func (w *Writer) Add(url string, depth, linksFound int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.records = append(w.records, Record{
		URL:        url,
		Depth:      depth,
		LinksFound: linksFound,
		CrawledAt:  time.Now(),
	})
}

func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	fmt.Println("Flushing", len(w.records), "records to", w.path)

	f, err := os.Create(w.path)
	if err != nil {
		fmt.Println("ERROR creating file:", err)
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(w.records)
	if err != nil {
		fmt.Println("ERROR encoding JSON:", err)
	}
	return err
}
