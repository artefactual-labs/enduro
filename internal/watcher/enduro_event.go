package watcher

const EnduroEventTypeObjectCreated = "object.created"

// EnduroEvent represents a normalized object storage event produced by an
// Enduro-owned adapter.
type EnduroEvent struct {
	Version string `json:"version"`
	Type    string `json:"type"`
	Bucket  string `json:"bucket"`
	Key     string `json:"key"`
	Source  string `json:"source,omitempty"`
}
