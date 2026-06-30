package collection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/run"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/api/gen/collection"
)

const (
	BulkWorkflowName              = "collection-bulk-workflow"
	BulkWorkflowID                = "collection-bulk-workflow"
	BulkWorkflowStateQueryHandler = "collection-bulk-state"
	BulkActivityName              = "collection-bulk-activity"
)

// BulkProgress reports bulk operation status - delivered as heartbeats.
type BulkProgress struct {
	CurrentID uint
	Count     uint
	Max       uint
}

type BulkWorkflowOperation string

const (
	BulkWorkflowOperationRetry   BulkWorkflowOperation = "retry"
	BulkWorkflowOperationCancel  BulkWorkflowOperation = "cancel"
	BulkWorkflowOperationAbandon BulkWorkflowOperation = "abandon"
)

type bulkWorkflowAction uint

const (
	bulkWorkflowActionRetry bulkWorkflowAction = iota
	bulkWorkflowActionDecide
)

const (
	collectionDecisionAbandon   = "ABANDON"
	collectionDecisionRetryOnce = "RETRY_ONCE"
)

var errBulkCancelSkipped = errors.New("bulk cancel skipped")

type BulkWorkflowInput struct {
	// Status of collections where bulk is performed.
	Status Status

	// Type of operation that is performed, e.g. "retry", "cancel"...
	Operation BulkWorkflowOperation

	// Max. number of collections affected. Zero means no cap established.
	Size uint
}

func bulkWorkflowInputAction(params BulkWorkflowInput) (bulkWorkflowAction, string, error) {
	switch params.Operation {
	case BulkWorkflowOperationRetry:
		switch params.Status {
		case StatusError, StatusAbandoned:
			return bulkWorkflowActionRetry, "", nil
		case StatusPending:
			return bulkWorkflowActionDecide, collectionDecisionRetryOnce, nil
		}
	case BulkWorkflowOperationAbandon:
		if params.Status == StatusPending {
			return bulkWorkflowActionDecide, collectionDecisionAbandon, nil
		}
	}

	return 0, "", fmt.Errorf("bulk %s is not supported for %s collections", params.Operation, params.Status)
}

// BulkWorkflow is a Temporal workflow that performs bulk operations.
func BulkWorkflow(ctx temporalsdk_workflow.Context, params BulkWorkflowInput) error {
	opts := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 24 * 365,
		WaitForCancellation: true,
		HeartbeatTimeout:    time.Second * 5,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})

	return temporalsdk_workflow.ExecuteActivity(opts, BulkActivityName, params).Get(opts, nil)
}

type BulkActivity struct {
	colsvc Service
}

func NewBulkActivity(colsvc Service) *BulkActivity {
	return &BulkActivity{
		colsvc: colsvc,
	}
}

func (a *BulkActivity) Execute(ctx context.Context, params BulkWorkflowInput) error {
	if _, _, err := bulkWorkflowInputAction(params); err != nil {
		return err
	}

	var group run.Group

	// One actor does the work while updating progress.
	// The other one sends the heartbeats.
	progress := &BulkProgress{}
	var mu sync.RWMutex

	{
		cancel := make(chan struct{})

		group.Add(
			func() error {
				ticker := time.NewTicker(time.Second * 1)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-cancel:
						return nil
					case <-ticker.C:
						mu.RLock()
						cp := progress
						mu.RUnlock()

						temporalsdk_activity.RecordHeartbeat(ctx, cp)
					}
				}
			},
			func(error) {
				close(cancel)
			},
		)
	}

	{
		cancel := make(chan struct{})

		group.Add(
			func() error {
				var nextCursor *string
				status := params.Status.String()
				var count uint
				for {
					select {
					case <-cancel:
						return nil
					default:
						res, err := a.colsvc.Goa().List(ctx, &collection.ListPayload{
							Status: &status,
							Cursor: nextCursor,
						})
						if err != nil {
							return err
						}

						for _, item := range res.Items {
							mu.Lock()
							progress = &BulkProgress{
								CurrentID: item.ID,
								Count:     count + 1,
								Max:       params.Size,
							}
							mu.Unlock()

							err = a.executeOperation(ctx, params, item.ID)
							if errors.Is(err, errBulkCancelSkipped) {
								continue
							}
							if err != nil {
								return fmt.Errorf("error executing bulk %s (failed on collection %d): %v", params.Operation, item.ID, err)
							}

							// Stop when cap is reached.
							count++
							if params.Size > 0 && count == params.Size {
								return nil
							}

							// Making sure that we don't overwork the system.
							time.Sleep(time.Millisecond * 50)
						}

						nextCursor = res.NextCursor
						if nextCursor == nil {
							return nil
						}
					}
				}
			},
			func(error) {
				close(cancel)
			},
		)
	}

	return group.Run()
}

func (a *BulkActivity) executeOperation(ctx context.Context, params BulkWorkflowInput, ID uint) error {
	action, decision, err := bulkWorkflowInputAction(params)
	if err != nil {
		return err
	}

	switch action {
	case bulkWorkflowActionRetry:
		return a.Retry(ctx, ID)
	case bulkWorkflowActionDecide:
		return a.Decide(ctx, ID, decision)
	default:
		return fmt.Errorf("bulk %s is not supported for %s collections", params.Operation, params.Status)
	}
}

func (a *BulkActivity) Retry(ctx context.Context, ID uint) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err := a.colsvc.Goa().Retry(ctx, &collection.RetryPayload{ID: ID})

	// User may have already started it manually.
	if temporalsdk_temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return nil
	}

	// TODO: ignore the error returned when the workflow history does not exist,
	// which is something that we used to do in Temporal.

	return err
}

func (a *BulkActivity) Decide(ctx context.Context, ID uint, option string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	return a.colsvc.Goa().Decide(ctx, &collection.DecidePayload{
		ID:     ID,
		Option: option,
	})
}
