package collection

import (
	"encoding/json"
	"strings"
)

// See https://gist.github.com/sevein/dd36c2af23fd0d9e2e2438d8eb091314.
type Status uint

const (
	StatusNew        Status = iota // Unused!
	StatusInProgress               // Undergoing work.
	StatusDone                     // Work has completed.
	StatusError                    // Processing failed.
	StatusUnknown                  // Unused!
	StatusQueued                   // Awaiting resource allocation.
	StatusAbandoned                // User abandoned processing.
	StatusPending                  // Awaiting user decision.
)

func NewStatus(status string) Status {
	var s Status

	switch strings.ToLower(status) {
	case "new":
		s = StatusNew
	case "in progress":
		s = StatusInProgress
	case "done":
		s = StatusDone
	case "error":
		s = StatusError
	case "queued":
		s = StatusQueued
	case "abandoned":
		s = StatusAbandoned
	case "pending":
		s = StatusPending
	default:
		s = StatusUnknown
	}

	return s
}

func (p Status) String() string {
	switch p {
	case StatusNew:
		return "new"
	case StatusInProgress:
		return "in progress"
	case StatusDone:
		return "done"
	case StatusError:
		return "error"
	case StatusQueued:
		return "queued"
	case StatusAbandoned:
		return "abandoned"
	case StatusPending:
		return "pending"
	}
	return "unknown"
}

func (p Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *Status) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*p = NewStatus(s)

	return nil
}
