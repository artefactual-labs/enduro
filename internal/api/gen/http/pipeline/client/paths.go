// Code generated by goa v3.4.3, DO NOT EDIT.
//
// HTTP request path constructors for the pipeline service.
//
// Command:
// $ goa-v3.4.3 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

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
