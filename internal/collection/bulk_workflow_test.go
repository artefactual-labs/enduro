package collection

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
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
		"cancel queued collections": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationCancel,
				Status:    StatusQueued,
			},
			wantAction: bulkWorkflowActionCancel,
		},
		"reject abandon for errors": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationAbandon,
				Status:    StatusError,
			},
			wantErr: "bulk abandon is not supported for error collections",
		},
		"reject cancel for pending collections": {
			params: BulkWorkflowInput{
				Operation: BulkWorkflowOperationCancel,
				Status:    StatusPending,
			},
			wantErr: "bulk cancel is not supported for pending collections",
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

func TestValidateBulkCancelCollection(t *testing.T) {
	t.Parallel()

	transferID := "transfer-id"
	emptyTransferID := ""
	tests := map[string]struct {
		status     string
		transferID *string
		wantSkip   bool
		wantErr    string
	}{
		"accepts queued collection without transfer": {
			status: StatusQueued.String(),
		},
		"accepts queued collection with empty transfer": {
			status:     StatusQueued.String(),
			transferID: &emptyTransferID,
		},
		"skips queued collection with transfer": {
			status:     StatusQueued.String(),
			transferID: &transferID,
			wantSkip:   true,
		},
		"rejects collection that is no longer queued": {
			status:  StatusInProgress.String(),
			wantErr: "collection 42 is no longer queued",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			skip, err := validateBulkCancelCollection(42, tc.status, tc.transferID)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, skip, tc.wantSkip)
		})
	}
}

func TestBulkActivityExecuteSkipsQueuedCollectionsWithTransfer(t *testing.T) {
	t.Parallel()

	transferID := "transfer-id"
	ctrl := gomock.NewController(t)
	colsvc := NewMockBulkCollectionService(ctrl)

	colsvc.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, payload *goacollection.ListPayload) (*goacollection.ListResult, error) {
			if payload.Status == nil || *payload.Status != StatusQueued.String() {
				return nil, fmt.Errorf("unexpected list status: %v", payload.Status)
			}

			return &goacollection.ListResult{
				Items: []*goacollection.EnduroStoredCollection{
					{ID: 10, Status: StatusQueued.String()},
					{ID: 11, Status: StatusQueued.String()},
					{ID: 12, Status: StatusQueued.String()},
				},
			}, nil
		})
	colsvc.EXPECT().
		Show(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, payload *goacollection.ShowPayload) (*goacollection.EnduroDetailedStoredCollection, error) {
			switch payload.ID {
			case 10:
				return &goacollection.EnduroDetailedStoredCollection{
					ID:         payload.ID,
					Status:     StatusQueued.String(),
					TransferID: &transferID,
				}, nil
			case 11:
				return &goacollection.EnduroDetailedStoredCollection{
					ID:     payload.ID,
					Status: StatusQueued.String(),
				}, nil
			default:
				return nil, fmt.Errorf("unexpected show ID: %d", payload.ID)
			}
		}).
		Times(2)
	colsvc.EXPECT().
		Cancel(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, payload *goacollection.CancelPayload) error {
			assert.Equal(t, payload.ID, uint(11))
			return nil
		})

	activity := newBulkActivity(colsvc)
	err := activity.Execute(context.Background(), BulkWorkflowInput{
		Status:    StatusQueued,
		Operation: BulkWorkflowOperationCancel,
		Size:      1,
	})

	assert.NilError(t, err)
}
