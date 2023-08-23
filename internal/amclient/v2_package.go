package amclient

import (
	"context"
	"encoding/base64"
	"fmt"
)

const packageBasePath = "api/v2beta/package"

type PackageService interface {
	Create(context.Context, *PackageCreateRequest) (*PackageCreateResponse, *Response, error)
}

type PackageServiceOp struct {
	client *Client
}

var _ PackageService = &PackageServiceOp{}

type PackageCreateRequest struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Path             string `json:"path"`
	Accession        string `json:"accession,omitempty"`
	AccessSystemID   string `json:"access_system_id,omitempty"`
	MetadataSetID    string `json:"metadata_set_id,omitempty"`
	ProcessingConfig string `json:"processing_config,omitempty"`
	AutoApprove      bool   `json:"auto_approve,omitempty"`
}

type PackageCreateResponse struct {
	ID string `json:"id,omitempty"`
}

func (s *PackageServiceOp) Create(ctx context.Context, r *PackageCreateRequest) (*PackageCreateResponse, *Response, error) {
	path := fmt.Sprintf("%s/", packageBasePath)

	r.Path = base64.StdEncoding.EncodeToString([]byte(r.Path))

	req, err := s.client.NewRequestJSON(ctx, "POST", path, r)
	if err != nil {
		return nil, nil, err
	}

	payload := &PackageCreateResponse{}
	resp, err := s.client.Do(ctx, req, payload)

	return payload, resp, err
}
