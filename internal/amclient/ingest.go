package amclient

import (
	"context"
	"fmt"
)

const ingestBasePath = "api/ingest"

// Ingest is an interface for interfacing with the Ingest endpoints of the
// Dashboard API.
type IngestService interface {
	Status(context.Context, string) (*IngestStatusResponse, *Response, error)
	Hide(context.Context, string) (*IngestHideResponse, *Response, error)
}

// IngestServiceOp handles communication with the Ingest related methods of
// the Archivematica API.
type IngestServiceOp struct {
	client *Client
}

var _ IngestService = &IngestServiceOp{}

type IngestStatusResponse struct {
	ID           string `json:"uuid"`
	Status       string `json:"status"`
	Name         string `json:"name"`
	SIPID        string `json:"sip_uuid"`
	Microservice string `json:"microservice"`
	Directory    string `json:"directory"`
	Path         string `json:"path"`
	Message      string `json:"message"`
	Type         string `json:"type"`
}

func (s *IngestServiceOp) Status(ctx context.Context, ID string) (*IngestStatusResponse, *Response, error) {
	path := fmt.Sprintf("%s/status/%s", ingestBasePath, ID)

	req, err := s.client.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	payload := &IngestStatusResponse{}
	resp, err := s.client.Do(ctx, req, payload)

	return payload, resp, err
}

type IngestHideResponse struct {
	Removed bool `json:"removed"`
}

func (s *IngestServiceOp) Hide(ctx context.Context, ID string) (*IngestHideResponse, *Response, error) {
	path := fmt.Sprintf("%s/%s/delete/", ingestBasePath, ID)

	req, err := s.client.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, nil, err
	}

	payload := &IngestHideResponse{}
	resp, err := s.client.Do(ctx, req, payload)

	return payload, resp, err
}
