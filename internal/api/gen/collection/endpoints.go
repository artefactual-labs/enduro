// Code generated by goa v3.5.4, DO NOT EDIT.
//
// collection endpoints
//
// Command:
// $ goa-v3.5.4 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package collection

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// Endpoints wraps the "collection" service endpoints.
type Endpoints struct {
	Monitor    goa.Endpoint
	List       goa.Endpoint
	Show       goa.Endpoint
	Delete     goa.Endpoint
	Cancel     goa.Endpoint
	Retry      goa.Endpoint
	Workflow   goa.Endpoint
	Download   goa.Endpoint
	Decide     goa.Endpoint
	Bulk       goa.Endpoint
	BulkStatus goa.Endpoint
}

// MonitorEndpointInput holds both the payload and the server stream of the
// "monitor" method.
type MonitorEndpointInput struct {
	// Stream is the server stream used by the "monitor" method to send data.
	Stream MonitorServerStream
}

// NewEndpoints wraps the methods of the "collection" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	return &Endpoints{
		Monitor:    NewMonitorEndpoint(s),
		List:       NewListEndpoint(s),
		Show:       NewShowEndpoint(s),
		Delete:     NewDeleteEndpoint(s),
		Cancel:     NewCancelEndpoint(s),
		Retry:      NewRetryEndpoint(s),
		Workflow:   NewWorkflowEndpoint(s),
		Download:   NewDownloadEndpoint(s),
		Decide:     NewDecideEndpoint(s),
		Bulk:       NewBulkEndpoint(s),
		BulkStatus: NewBulkStatusEndpoint(s),
	}
}

// Use applies the given middleware to all the "collection" service endpoints.
func (e *Endpoints) Use(m func(goa.Endpoint) goa.Endpoint) {
	e.Monitor = m(e.Monitor)
	e.List = m(e.List)
	e.Show = m(e.Show)
	e.Delete = m(e.Delete)
	e.Cancel = m(e.Cancel)
	e.Retry = m(e.Retry)
	e.Workflow = m(e.Workflow)
	e.Download = m(e.Download)
	e.Decide = m(e.Decide)
	e.Bulk = m(e.Bulk)
	e.BulkStatus = m(e.BulkStatus)
}

// NewMonitorEndpoint returns an endpoint function that calls the method
// "monitor" of service "collection".
func NewMonitorEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		ep := req.(*MonitorEndpointInput)
		return nil, s.Monitor(ctx, ep.Stream)
	}
}

// NewListEndpoint returns an endpoint function that calls the method "list" of
// service "collection".
func NewListEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*ListPayload)
		return s.List(ctx, p)
	}
}

// NewShowEndpoint returns an endpoint function that calls the method "show" of
// service "collection".
func NewShowEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*ShowPayload)
		res, err := s.Show(ctx, p)
		if err != nil {
			return nil, err
		}
		vres := NewViewedEnduroStoredCollection(res, "default")
		return vres, nil
	}
}

// NewDeleteEndpoint returns an endpoint function that calls the method
// "delete" of service "collection".
func NewDeleteEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*DeletePayload)
		return nil, s.Delete(ctx, p)
	}
}

// NewCancelEndpoint returns an endpoint function that calls the method
// "cancel" of service "collection".
func NewCancelEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*CancelPayload)
		return nil, s.Cancel(ctx, p)
	}
}

// NewRetryEndpoint returns an endpoint function that calls the method "retry"
// of service "collection".
func NewRetryEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*RetryPayload)
		return nil, s.Retry(ctx, p)
	}
}

// NewWorkflowEndpoint returns an endpoint function that calls the method
// "workflow" of service "collection".
func NewWorkflowEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*WorkflowPayload)
		res, err := s.Workflow(ctx, p)
		if err != nil {
			return nil, err
		}
		vres := NewViewedEnduroCollectionWorkflowStatus(res, "default")
		return vres, nil
	}
}

// NewDownloadEndpoint returns an endpoint function that calls the method
// "download" of service "collection".
func NewDownloadEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*DownloadPayload)
		return s.Download(ctx, p)
	}
}

// NewDecideEndpoint returns an endpoint function that calls the method
// "decide" of service "collection".
func NewDecideEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*DecidePayload)
		return nil, s.Decide(ctx, p)
	}
}

// NewBulkEndpoint returns an endpoint function that calls the method "bulk" of
// service "collection".
func NewBulkEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		p := req.(*BulkPayload)
		return s.Bulk(ctx, p)
	}
}

// NewBulkStatusEndpoint returns an endpoint function that calls the method
// "bulk_status" of service "collection".
func NewBulkStatusEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return s.BulkStatus(ctx)
	}
}
