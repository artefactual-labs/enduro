package collection

import (
	"encoding/json"
	"strings"
)

type Status uint

const (
	StatusNew Status = iota
	StatusInProgress
	StatusDone
	StatusError
	StatusUnknown
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
