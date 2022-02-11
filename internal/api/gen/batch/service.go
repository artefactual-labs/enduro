// Code generated by goa v3.5.5, DO NOT EDIT.
//
// batch service
//
// Command:
// $ goa-v3.5.5 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package batch

import (
	"context"

	goa "goa.design/goa/v3/pkg"
)

// The batch service manages batches of collections.
type Service interface {
	// Submit a new batch
	Submit(context.Context, *SubmitPayload) (res *BatchResult, err error)
	// Retrieve status of current batch operation.
	Status(context.Context) (res *BatchStatusResult, err error)
	// Retrieve form hints
	Hints(context.Context) (res *BatchHintsResult, err error)
}

// ServiceName is the name of the service as defined in the design. This is the
// same value that is set in the endpoint request contexts under the ServiceKey
// key.
const ServiceName = "batch"

// MethodNames lists the service method names as defined in the design. These
// are the same values that are set in the endpoint request contexts under the
// MethodKey key.
var MethodNames = [3]string{"submit", "status", "hints"}

// SubmitPayload is the payload type of the batch service submit method.
type SubmitPayload struct {
	Path             string
	Pipeline         string
	ProcessingConfig *string
	CompletedDir     *string
	RetentionPeriod  *string
}

// BatchResult is the result type of the batch service submit method.
type BatchResult struct {
	WorkflowID string
	RunID      string
}

// BatchStatusResult is the result type of the batch service status method.
type BatchStatusResult struct {
	Running    bool
	Status     *string
	WorkflowID *string
	RunID      *string
}

// BatchHintsResult is the result type of the batch service hints method.
type BatchHintsResult struct {
	// A list of known values of completedDir used by existing watchers.
	CompletedDirs []string
}

// MakeNotAvailable builds a goa.ServiceError from an error.
func MakeNotAvailable(err error) *goa.ServiceError {
	return &goa.ServiceError{
		Name:    "not_available",
		ID:      goa.NewErrorID(),
		Message: err.Error(),
	}
}

// MakeNotValid builds a goa.ServiceError from an error.
func MakeNotValid(err error) *goa.ServiceError {
	return &goa.ServiceError{
		Name:    "not_valid",
		ID:      goa.NewErrorID(),
		Message: err.Error(),
	}
}
