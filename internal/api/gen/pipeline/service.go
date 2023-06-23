// Code generated by goa v3.11.3, DO NOT EDIT.
//
// pipeline service
//
// Command:
// $ goa-v3.11.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
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
	// List all processing configurations of a pipeline given its ID
	Processing(context.Context, *ProcessingPayload) (res []string, err error)
}

// ServiceName is the name of the service as defined in the design. This is the
// same value that is set in the endpoint request contexts under the ServiceKey
// key.
const ServiceName = "pipeline"

// MethodNames lists the service method names as defined in the design. These
// are the same values that are set in the endpoint request contexts under the
// MethodKey key.
var MethodNames = [3]string{"list", "show", "processing"}

// EnduroStoredPipeline is the result type of the pipeline service show method.
type EnduroStoredPipeline struct {
	// Identifier of the pipeline
	ID *string
	// Name of the pipeline
	Name string
	// Maximum concurrent transfers
	Capacity *int64
	// Current transfers
	Current *int64
	Status  *string
}

// ListPayload is the payload type of the pipeline service list method.
type ListPayload struct {
	Name   *string
	Status bool
}

// Pipeline not found.
type PipelineNotFound struct {
	// Message of error
	Message string
	// Identifier of missing pipeline
	ID string
}

// ProcessingPayload is the payload type of the pipeline service processing
// method.
type ProcessingPayload struct {
	// Identifier of pipeline
	ID string
}

// ShowPayload is the payload type of the pipeline service show method.
type ShowPayload struct {
	// Identifier of pipeline to show
	ID string
}

// Error returns an error description.
func (e *PipelineNotFound) Error() string {
	return "Pipeline not found."
}

// ErrorName returns "PipelineNotFound".
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e *PipelineNotFound) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns "PipelineNotFound".
func (e *PipelineNotFound) GoaErrorName() string {
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
		Status:   vres.Status,
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
		Status:   res.Status,
	}
	return vres
}
