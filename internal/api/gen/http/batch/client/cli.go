// Code generated by goa v3.5.4, DO NOT EDIT.
//
// batch HTTP client CLI support package
//
// Command:
// $ goa-v3.5.4 gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"encoding/json"
	"fmt"

	batch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
)

// BuildSubmitPayload builds the payload for the batch submit endpoint from CLI
// flags.
func BuildSubmitPayload(batchSubmitBody string) (*batch.SubmitPayload, error) {
	var err error
	var body SubmitRequestBody
	{
		err = json.Unmarshal([]byte(batchSubmitBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"completed_dir\": \"Eum quis nihil soluta ut molestiae et.\",\n      \"path\": \"Labore impedit rerum laborum.\",\n      \"pipeline\": \"Provident voluptates iure et.\",\n      \"processing_config\": \"Ut dolor est.\",\n      \"retention_period\": \"Sit sed laboriosam.\"\n   }'")
		}
	}
	v := &batch.SubmitPayload{
		Path:             body.Path,
		Pipeline:         body.Pipeline,
		ProcessingConfig: body.ProcessingConfig,
		CompletedDir:     body.CompletedDir,
		RetentionPeriod:  body.RetentionPeriod,
	}

	return v, nil
}
