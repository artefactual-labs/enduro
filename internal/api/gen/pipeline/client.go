// Code generated by goa v3.5.5, DO NOT EDIT.
//
// pipeline client
//
// Command:
// $ goa-v3.5.5 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package pipeline

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// Client is the "pipeline" service client.
type Client struct {
	ListEndpoint       goa.Endpoint
	ShowEndpoint       goa.Endpoint
	ProcessingEndpoint goa.Endpoint
}

// NewClient initializes a "pipeline" service client given the endpoints.
func NewClient(list, show, processing goa.Endpoint) *Client {
	return &Client{
		ListEndpoint:       list,
		ShowEndpoint:       show,
		ProcessingEndpoint: processing,
	}
}

// List calls the "list" endpoint of the "pipeline" service.
func (c *Client) List(ctx context.Context, p *ListPayload) (res []*EnduroStoredPipeline, err error) {
	var ires interface{}
	ires, err = c.ListEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.([]*EnduroStoredPipeline), nil
}

// Show calls the "show" endpoint of the "pipeline" service.
// Show may return the following errors:
//	- "not_found" (type *PipelineNotFound): Pipeline not found
//	- error: internal error
func (c *Client) Show(ctx context.Context, p *ShowPayload) (res *EnduroStoredPipeline, err error) {
	var ires interface{}
	ires, err = c.ShowEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*EnduroStoredPipeline), nil
}

// Processing calls the "processing" endpoint of the "pipeline" service.
// Processing may return the following errors:
//	- "not_found" (type *PipelineNotFound): Pipeline not found
//	- error: internal error
func (c *Client) Processing(ctx context.Context, p *ProcessingPayload) (res []string, err error) {
	var ires interface{}
	ires, err = c.ProcessingEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.([]string), nil
}
