package amclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

const jobsBasePath = "api/v2beta/jobs"

type JobsService interface {
	List(context.Context, string, *JobsListRequest) ([]Job, *Response, error)
}

type JobsServiceOp struct {
	client *Client
}

var _ JobsService = &JobsServiceOp{}

type JobsListRequest struct {
	Microservice string `json:"microservice,omitempty"`
	LinkID       string `json:"link_uuid,omitempty"`
	Name         string `json:"name,omitempty"`
}

type Job struct {
	ID           string    `json:"uuid"`
	Name         string    `json:"name"`
	Status       JobStatus `json:"status"`
	Microservice string    `json:"microservice"`
	LinkID       string    `json:"link_uuid"`
	Tasks        []Task    `json:"tasks"`
}

type JobStatus int

const (
	JobStatusUnknown JobStatus = iota
	JobStatusUserInput
	JobStatusProcessing
	JobStatusComplete
	JobStatusFailed
)

var jobStatusToString = map[JobStatus]string{
	JobStatusUnknown:    "UNKNOWN",
	JobStatusUserInput:  "USER_INPUT",
	JobStatusProcessing: "PROCESSING",
	JobStatusComplete:   "COMPLETE",
	JobStatusFailed:     "FAILED",
}

var jobStatusToID = map[string]JobStatus{
	"UNKNOWN":    JobStatusUnknown,
	"USER_INPUT": JobStatusUserInput,
	"PROCESSING": JobStatusProcessing,
	"COMPLETE":   JobStatusComplete,
	"FAILED":     JobStatusFailed,
}

// MarshalJSON marshals the enum as a quoted json string
func (s JobStatus) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(jobStatusToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *JobStatus) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	*s = jobStatusToID[j]
	return nil
}

type Task struct {
	ID       string `json:"uuid"`
	ExitCode uint8  `json:"exit_code"`
}

func (s *JobsServiceOp) List(ctx context.Context, ID string, r *JobsListRequest) ([]Job, *Response, error) {
	path := fmt.Sprintf("%s/%s", jobsBasePath, ID)

	req, err := s.client.NewRequestJSON(ctx, "GET", path, r)
	if err != nil {
		return nil, nil, err
	}

	payload := []Job{}
	resp, err := s.client.Do(ctx, req, &payload)

	return payload, resp, err
}
