// Code generated by goa v3.4.3, DO NOT EDIT.
//
// collection client
//
// Command:
// $ goa-v3.4.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package collection

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// Client is the "collection" service client.
type Client struct {
	ListEndpoint       goa.Endpoint
	ShowEndpoint       goa.Endpoint
	DeleteEndpoint     goa.Endpoint
	CancelEndpoint     goa.Endpoint
	RetryEndpoint      goa.Endpoint
	WorkflowEndpoint   goa.Endpoint
	DownloadEndpoint   goa.Endpoint
	DecideEndpoint     goa.Endpoint
	BulkEndpoint       goa.Endpoint
	BulkStatusEndpoint goa.Endpoint
}

// NewClient initializes a "collection" service client given the endpoints.
func NewClient(list, show, delete_, cancel, retry, workflow, download, decide, bulk, bulkStatus goa.Endpoint) *Client {
	return &Client{
		ListEndpoint:       list,
		ShowEndpoint:       show,
		DeleteEndpoint:     delete_,
		CancelEndpoint:     cancel,
		RetryEndpoint:      retry,
		WorkflowEndpoint:   workflow,
		DownloadEndpoint:   download,
		DecideEndpoint:     decide,
		BulkEndpoint:       bulk,
		BulkStatusEndpoint: bulkStatus,
	}
}

// List calls the "list" endpoint of the "collection" service.
func (c *Client) List(ctx context.Context, p *ListPayload) (res *ListResult, err error) {
	var ires interface{}
	ires, err = c.ListEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*ListResult), nil
}

// Show calls the "show" endpoint of the "collection" service.
// Show may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- error: internal error
func (c *Client) Show(ctx context.Context, p *ShowPayload) (res *EnduroStoredCollection, err error) {
	var ires interface{}
	ires, err = c.ShowEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*EnduroStoredCollection), nil
}

// Delete calls the "delete" endpoint of the "collection" service.
// Delete may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- error: internal error
func (c *Client) Delete(ctx context.Context, p *DeletePayload) (err error) {
	_, err = c.DeleteEndpoint(ctx, p)
	return
}

// Cancel calls the "cancel" endpoint of the "collection" service.
// Cancel may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- "not_running" (type *goa.ServiceError)
//	- error: internal error
func (c *Client) Cancel(ctx context.Context, p *CancelPayload) (err error) {
	_, err = c.CancelEndpoint(ctx, p)
	return
}

// Retry calls the "retry" endpoint of the "collection" service.
// Retry may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- "not_running" (type *goa.ServiceError)
//	- error: internal error
func (c *Client) Retry(ctx context.Context, p *RetryPayload) (err error) {
	_, err = c.RetryEndpoint(ctx, p)
	return
}

// Workflow calls the "workflow" endpoint of the "collection" service.
// Workflow may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- error: internal error
func (c *Client) Workflow(ctx context.Context, p *WorkflowPayload) (res *EnduroCollectionWorkflowStatus, err error) {
	var ires interface{}
	ires, err = c.WorkflowEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*EnduroCollectionWorkflowStatus), nil
}

// Download calls the "download" endpoint of the "collection" service.
// Download may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- error: internal error
func (c *Client) Download(ctx context.Context, p *DownloadPayload) (res []byte, err error) {
	var ires interface{}
	ires, err = c.DownloadEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.([]byte), nil
}

// Decide calls the "decide" endpoint of the "collection" service.
// Decide may return the following errors:
//	- "not_found" (type *NotFound): Collection not found
//	- "not_valid" (type *goa.ServiceError)
//	- error: internal error
func (c *Client) Decide(ctx context.Context, p *DecidePayload) (err error) {
	_, err = c.DecideEndpoint(ctx, p)
	return
}

// Bulk calls the "bulk" endpoint of the "collection" service.
// Bulk may return the following errors:
//	- "not_available" (type *goa.ServiceError)
//	- "not_valid" (type *goa.ServiceError)
//	- error: internal error
func (c *Client) Bulk(ctx context.Context, p *BulkPayload) (res *BulkResult, err error) {
	var ires interface{}
	ires, err = c.BulkEndpoint(ctx, p)
	if err != nil {
		return
	}
	return ires.(*BulkResult), nil
}

// BulkStatus calls the "bulk_status" endpoint of the "collection" service.
func (c *Client) BulkStatus(ctx context.Context) (res *BulkStatusResult, err error) {
	var ires interface{}
	ires, err = c.BulkStatusEndpoint(ctx, nil)
	if err != nil {
		return
	}
	return ires.(*BulkStatusResult), nil
}
