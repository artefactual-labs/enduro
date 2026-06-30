package collection

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
)

func TestBulkWorkflowInputAction(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params       BulkWorkflowInput
		wantAction   bulkWorkflowAction
		wantDecision string
		wantErr      string
	}{
		"retry errors": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationRetry,
				Status:    StatusError,
			},
			wantAction: bulkWorkflowActionRetry,
		},
		"retry abandoned collections": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationRetry,
				Status:    StatusAbandoned,
			},
			wantAction: bulkWorkflowActionRetry,
		},
		"retry pending collections via decision": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationRetry,
				Status:    StatusPending,
			},
			wantAction:   bulkWorkflowActionDecide,
			wantDecision: collectionDecisionRetryOnce,
		},
		"abandon pending collections via decision": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationAbandon,
				Status:    StatusPending,
			},
			wantAction:   bulkWorkflowActionDecide,
			wantDecision: collectionDecisionAbandon,
		},
		"reject abandon for errors": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationAbandon,
				Status:    StatusError,
			},
			wantErr: "bulk abandon is not supported for error collections",
		},
		"reject cancel before support is implemented": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationCancel,
				Status:    StatusQueued,
			},
			wantErr: "bulk cancel is not supported for queued collections",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotAction, gotDecision, err := bulkWorkflowInputAction(tc.params)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, gotAction, tc.wantAction)
			assert.Equal(t, gotDecision, tc.wantDecision)
		})
	}
}
