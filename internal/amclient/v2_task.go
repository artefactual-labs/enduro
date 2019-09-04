package amclient

import (
	"context"
	"fmt"
	"time"
)

const taskBasePath = "api/v2beta/task"

type TaskService interface {
	Read(context.Context, string) (*TaskDetailed, *Response, error)
}

type TaskServiceOp struct {
	client *Client
}

var _ TaskService = &TaskServiceOp{}

type TaskDateTime struct {
	time.Time
}

func (t *TaskDateTime) UnmarshalJSON(data []byte) error {
	s := string(data)
	s = s[1 : len(s)-1]
	td, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	t.Time = td
	return err
}

type TaskDetailed struct {
	ID          string       `json:"uuid"`
	ExitCode    uint8        `json:"exit_code"`
	FileID      string       `json:"file_uuid"`
	Filename    string       `json:"file_name"`
	TimeCreated TaskDateTime `json:"time_created"`
	TimeStarted TaskDateTime `json:"time_started"`
	TimeEnded   TaskDateTime `json:"time_ended"`
	Duration    uint32       `json:"duration"`
}

func (s *TaskServiceOp) Read(ctx context.Context, ID string) (*TaskDetailed, *Response, error) {
	path := fmt.Sprintf("%s/%s", taskBasePath, ID)

	req, err := s.client.NewRequestJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	payload := &TaskDetailed{}
	resp, err := s.client.Do(ctx, req, payload)

	return payload, resp, err
}
