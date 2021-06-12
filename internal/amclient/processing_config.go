package amclient

import (
	"bytes"
	"context"
	"fmt"
)

const processingConfigBasePath = "api/processing-configuration"

// ProcessingConfigService is an interface for interfacing with the processing
// configuration endpoints of the Dashboard API.
type ProcessingConfigService interface {
	Get(context.Context, string) (*ProcessingConfig, *Response, error)
}

// ProcessingConfigOp handles communication with the Tranfer related methods of
// the Archivematica API.
type ProcessingConfigOp struct {
	client *Client
}

var _ ProcessingConfigService = &ProcessingConfigOp{}

// ProcessingConfig represents the processing configuration document returned
// by the Dashboard API.
type ProcessingConfig struct {
	bytes.Buffer
}

// Get obtains a processing configuration given its name.
func (s *ProcessingConfigOp) Get(ctx context.Context, name string) (*ProcessingConfig, *Response, error) {
	path := fmt.Sprintf("%s/%s/", processingConfigBasePath, name)

	req, err := s.client.NewRequest(ctx, "GET", path, nil, WithRequestAcceptXML())
	if err != nil {
		return nil, nil, err
	}

	payload := &ProcessingConfig{}
	resp, err := s.client.Do(ctx, req, payload)

	return payload, resp, err
}
