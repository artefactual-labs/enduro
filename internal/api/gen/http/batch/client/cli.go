// Code generated by goa v3.13.2, DO NOT EDIT.
//
// batch HTTP client CLI support package
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"encoding/json"
	"fmt"

	batch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	goa "goa.design/goa/v3/pkg"
)

// BuildSubmitPayload builds the payload for the batch submit endpoint from CLI
// flags.
func BuildSubmitPayload(batchSubmitBody string) (*batch.SubmitPayload, error) {
	var err error
	var body SubmitRequestBody
	{
		err = json.Unmarshal([]byte(batchSubmitBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"completed_dir\": \"abc123\",\n      \"depth\": 1,\n      \"exclude_hidden_files\": false,\n      \"path\": \"abc123\",\n      \"pipeline\": \"abc123\",\n      \"process_name_metadata\": false,\n      \"processing_config\": \"abc123\",\n      \"reject_duplicates\": false,\n      \"retention_period\": \"abc123\",\n      \"transfer_type\": \"abc123\"\n   }'")
		}
		if body.Depth < 0 {
			err = goa.MergeErrors(err, goa.InvalidRangeError("body.depth", body.Depth, 0, true))
		}
		if err != nil {
			return nil, err
		}
	}
	v := &batch.SubmitPayload{
		Path:                body.Path,
		Pipeline:            body.Pipeline,
		ProcessingConfig:    body.ProcessingConfig,
		CompletedDir:        body.CompletedDir,
		RetentionPeriod:     body.RetentionPeriod,
		RejectDuplicates:    body.RejectDuplicates,
		ExcludeHiddenFiles:  body.ExcludeHiddenFiles,
		TransferType:        body.TransferType,
		ProcessNameMetadata: body.ProcessNameMetadata,
		Depth:               body.Depth,
	}
	{
		var zero bool
		if v.RejectDuplicates == zero {
			v.RejectDuplicates = false
		}
	}
	{
		var zero bool
		if v.ExcludeHiddenFiles == zero {
			v.ExcludeHiddenFiles = false
		}
	}
	{
		var zero bool
		if v.ProcessNameMetadata == zero {
			v.ProcessNameMetadata = false
		}
	}
	{
		var zero int
		if v.Depth == zero {
			v.Depth = 0
		}
	}

	return v, nil
}
