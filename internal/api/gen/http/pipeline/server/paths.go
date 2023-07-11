// Code generated by goa v3.11.3, DO NOT EDIT.
//
// HTTP request path constructors for the pipeline service.
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package server

import (
	"fmt"
)

// ListPipelinePath returns the URL path to the pipeline service list HTTP endpoint.
func ListPipelinePath() string {
	return "/pipeline"
}

// ShowPipelinePath returns the URL path to the pipeline service show HTTP endpoint.
func ShowPipelinePath(id string) string {
	return fmt.Sprintf("/pipeline/%v", id)
}

// ProcessingPipelinePath returns the URL path to the pipeline service processing HTTP endpoint.
func ProcessingPipelinePath(id string) string {
	return fmt.Sprintf("/pipeline/%v/processing", id)
}
