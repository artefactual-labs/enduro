// Code generated by goa v3.1.3, DO NOT EDIT.
//
// batch HTTP client types
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	batch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	goa "goa.design/goa/v3/pkg"
)

// SubmitRequestBody is the type of the "batch" service "submit" endpoint HTTP
// request body.
type SubmitRequestBody struct {
	Path     string `form:"path" json:"path" xml:"path"`
	Pipeline string `form:"pipeline" json:"pipeline" xml:"pipeline"`
}

// SubmitResponseBody is the type of the "batch" service "submit" endpoint HTTP
// response body.
type SubmitResponseBody struct {
	WorkflowID *string `form:"workflow_id,omitempty" json:"workflow_id,omitempty" xml:"workflow_id,omitempty"`
	RunID      *string `form:"run_id,omitempty" json:"run_id,omitempty" xml:"run_id,omitempty"`
}

// StatusResponseBody is the type of the "batch" service "status" endpoint HTTP
// response body.
type StatusResponseBody struct {
	Running    *bool   `form:"running,omitempty" json:"running,omitempty" xml:"running,omitempty"`
	Status     *string `form:"status,omitempty" json:"status,omitempty" xml:"status,omitempty"`
	WorkflowID *string `form:"workflow_id,omitempty" json:"workflow_id,omitempty" xml:"workflow_id,omitempty"`
	RunID      *string `form:"run_id,omitempty" json:"run_id,omitempty" xml:"run_id,omitempty"`
}

// SubmitNotAvailableResponseBody is the type of the "batch" service "submit"
// endpoint HTTP response body for the "not_available" error.
type SubmitNotAvailableResponseBody struct {
	// Name is the name of this class of errors.
	Name *string `form:"name,omitempty" json:"name,omitempty" xml:"name,omitempty"`
	// ID is a unique identifier for this particular occurrence of the problem.
	ID *string `form:"id,omitempty" json:"id,omitempty" xml:"id,omitempty"`
	// Message is a human-readable explanation specific to this occurrence of the
	// problem.
	Message *string `form:"message,omitempty" json:"message,omitempty" xml:"message,omitempty"`
	// Is the error temporary?
	Temporary *bool `form:"temporary,omitempty" json:"temporary,omitempty" xml:"temporary,omitempty"`
	// Is the error a timeout?
	Timeout *bool `form:"timeout,omitempty" json:"timeout,omitempty" xml:"timeout,omitempty"`
	// Is the error a server-side fault?
	Fault *bool `form:"fault,omitempty" json:"fault,omitempty" xml:"fault,omitempty"`
}

// SubmitNotValidResponseBody is the type of the "batch" service "submit"
// endpoint HTTP response body for the "not_valid" error.
type SubmitNotValidResponseBody struct {
	// Name is the name of this class of errors.
	Name *string `form:"name,omitempty" json:"name,omitempty" xml:"name,omitempty"`
	// ID is a unique identifier for this particular occurrence of the problem.
	ID *string `form:"id,omitempty" json:"id,omitempty" xml:"id,omitempty"`
	// Message is a human-readable explanation specific to this occurrence of the
	// problem.
	Message *string `form:"message,omitempty" json:"message,omitempty" xml:"message,omitempty"`
	// Is the error temporary?
	Temporary *bool `form:"temporary,omitempty" json:"temporary,omitempty" xml:"temporary,omitempty"`
	// Is the error a timeout?
	Timeout *bool `form:"timeout,omitempty" json:"timeout,omitempty" xml:"timeout,omitempty"`
	// Is the error a server-side fault?
	Fault *bool `form:"fault,omitempty" json:"fault,omitempty" xml:"fault,omitempty"`
}

// NewSubmitRequestBody builds the HTTP request body from the payload of the
// "submit" endpoint of the "batch" service.
func NewSubmitRequestBody(p *batch.SubmitPayload) *SubmitRequestBody {
	body := &SubmitRequestBody{
		Path:     p.Path,
		Pipeline: p.Pipeline,
	}
	return body
}

// NewSubmitBatchResultAccepted builds a "batch" service "submit" endpoint
// result from a HTTP "Accepted" response.
func NewSubmitBatchResultAccepted(body *SubmitResponseBody) *batch.BatchResult {
	v := &batch.BatchResult{
		WorkflowID: *body.WorkflowID,
		RunID:      *body.RunID,
	}

	return v
}

// NewSubmitNotAvailable builds a batch service submit endpoint not_available
// error.
func NewSubmitNotAvailable(body *SubmitNotAvailableResponseBody) *goa.ServiceError {
	v := &goa.ServiceError{
		Name:      *body.Name,
		ID:        *body.ID,
		Message:   *body.Message,
		Temporary: *body.Temporary,
		Timeout:   *body.Timeout,
		Fault:     *body.Fault,
	}

	return v
}

// NewSubmitNotValid builds a batch service submit endpoint not_valid error.
func NewSubmitNotValid(body *SubmitNotValidResponseBody) *goa.ServiceError {
	v := &goa.ServiceError{
		Name:      *body.Name,
		ID:        *body.ID,
		Message:   *body.Message,
		Temporary: *body.Temporary,
		Timeout:   *body.Timeout,
		Fault:     *body.Fault,
	}

	return v
}

// NewStatusBatchStatusResultOK builds a "batch" service "status" endpoint
// result from a HTTP "OK" response.
func NewStatusBatchStatusResultOK(body *StatusResponseBody) *batch.BatchStatusResult {
	v := &batch.BatchStatusResult{
		Running:    *body.Running,
		Status:     body.Status,
		WorkflowID: body.WorkflowID,
		RunID:      body.RunID,
	}

	return v
}

// ValidateSubmitResponseBody runs the validations defined on SubmitResponseBody
func ValidateSubmitResponseBody(body *SubmitResponseBody) (err error) {
	if body.WorkflowID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("workflow_id", "body"))
	}
	if body.RunID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("run_id", "body"))
	}
	return
}

// ValidateStatusResponseBody runs the validations defined on StatusResponseBody
func ValidateStatusResponseBody(body *StatusResponseBody) (err error) {
	if body.Running == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("running", "body"))
	}
	return
}

// ValidateSubmitNotAvailableResponseBody runs the validations defined on
// submit_not_available_response_body
func ValidateSubmitNotAvailableResponseBody(body *SubmitNotAvailableResponseBody) (err error) {
	if body.Name == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("name", "body"))
	}
	if body.ID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("id", "body"))
	}
	if body.Message == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("message", "body"))
	}
	if body.Temporary == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("temporary", "body"))
	}
	if body.Timeout == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("timeout", "body"))
	}
	if body.Fault == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("fault", "body"))
	}
	return
}

// ValidateSubmitNotValidResponseBody runs the validations defined on
// submit_not_valid_response_body
func ValidateSubmitNotValidResponseBody(body *SubmitNotValidResponseBody) (err error) {
	if body.Name == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("name", "body"))
	}
	if body.ID == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("id", "body"))
	}
	if body.Message == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("message", "body"))
	}
	if body.Temporary == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("temporary", "body"))
	}
	if body.Timeout == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("timeout", "body"))
	}
	if body.Fault == nil {
		err = goa.MergeErrors(err, goa.MissingFieldError("fault", "body"))
	}
	return
}
