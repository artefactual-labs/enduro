// Code generated by goa v3.4.3, DO NOT EDIT.
//
// pipeline service
//
// Command:
// $ goa-v3.4.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package pipeline

import (
	"context"

	pipelineviews "github.com/artefactual-labs/enduro/internal/api/gen/pipeline/views"
)

// The pipeline service manages Archivematica pipelines.
type Service interface {
	// List all known pipelines
	List(context.Context, *ListPayload) (res []*EnduroStoredPipeline, err error)
	// Show pipeline by ID
	Show(context.Context, *ShowPayload) (res *EnduroStoredPipeline, err error)
}

// ServiceName is the name of the service as defined in the design. This is the
// same value that is set in the endpoint request contexts under the ServiceKey
// key.
const ServiceName = "pipeline"

// MethodNames lists the service method names as defined in the design. These
// are the same values that are set in the endpoint request contexts under the
// MethodKey key.
var MethodNames = [2]string{"list", "show"}

// ListPayload is the payload type of the pipeline service list method.
type ListPayload struct {
	Name *string
}

// ShowPayload is the payload type of the pipeline service show method.
type ShowPayload struct {
	// Identifier of pipeline to show
	ID string
}

// EnduroStoredPipeline is the result type of the pipeline service show method.
type EnduroStoredPipeline struct {
	// Name of the collection
	ID *string
	// Name of the collection
	Name string
	// Maximum concurrent transfers
	Capacity *int64
	// Current transfers
	Current *int64
}

// NotFound is the type returned when attempting to operate with a collection
// that does not exist.
type NotFound struct {
	// Message of error
	Message string
	// Identifier of missing collection
	ID uint
}

// Error returns an error description.
func (e *NotFound) Error() string {
	return "NotFound is the type returned when attempting to operate with a collection that does not exist."
}

// ErrorName returns "NotFound".
func (e *NotFound) ErrorName() string {
	return e.Message
}

// NewEnduroStoredPipeline initializes result type EnduroStoredPipeline from
// viewed result type EnduroStoredPipeline.
func NewEnduroStoredPipeline(vres *pipelineviews.EnduroStoredPipeline) *EnduroStoredPipeline {
	return newEnduroStoredPipeline(vres.Projected)
}

// NewViewedEnduroStoredPipeline initializes viewed result type
// EnduroStoredPipeline from result type EnduroStoredPipeline using the given
// view.
func NewViewedEnduroStoredPipeline(res *EnduroStoredPipeline, view string) *pipelineviews.EnduroStoredPipeline {
	p := newEnduroStoredPipelineView(res)
	return &pipelineviews.EnduroStoredPipeline{Projected: p, View: "default"}
}

// newEnduroStoredPipeline converts projected type EnduroStoredPipeline to
// service type EnduroStoredPipeline.
func newEnduroStoredPipeline(vres *pipelineviews.EnduroStoredPipelineView) *EnduroStoredPipeline {
	res := &EnduroStoredPipeline{
		ID:       vres.ID,
		Capacity: vres.Capacity,
		Current:  vres.Current,
	}
	if vres.Name != nil {
		res.Name = *vres.Name
	}
	return res
}

// newEnduroStoredPipelineView projects result type EnduroStoredPipeline to
// projected type EnduroStoredPipelineView using the "default" view.
func newEnduroStoredPipelineView(res *EnduroStoredPipeline) *pipelineviews.EnduroStoredPipelineView {
	vres := &pipelineviews.EnduroStoredPipelineView{
		ID:       res.ID,
		Name:     &res.Name,
		Capacity: res.Capacity,
		Current:  res.Current,
	}
	return vres
}
