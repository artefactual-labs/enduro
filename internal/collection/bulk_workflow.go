package collection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/oklog/run"
	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_activity "go.uber.org/cadence/activity"
	cadencesdk_workflow "go.uber.org/cadence/workflow"
	"go.uber.org/yarpc/yarpcerrors"

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

type BulkWorkflowInput struct {
	// Status of collections where bulk is performed.
	Status Status

	// Type of operation that is performed, e.g. "retry", "cancel"...
	Operation BulkWorkflowOperation

	// Max. number of collections affected. Zero means no cap established.
	Size uint
}

// BulkWorkflow is a Cadence workflow that performs bulk operations.
func BulkWorkflow(ctx cadencesdk_workflow.Context, params BulkWorkflowInput) error {
	opts := cadencesdk_workflow.WithActivityOptions(ctx, cadencesdk_workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Hour * 24 * 365,
		StartToCloseTimeout:    time.Hour * 24 * 365,
		WaitForCancellation:    true,
		HeartbeatTimeout:       time.Second * 5,
	})

	return cadencesdk_workflow.ExecuteActivity(opts, BulkActivityName, params).Get(opts, nil)
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

						cadencesdk_activity.RecordHeartbeat(ctx, cp)
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

							switch params.Operation {
							case BulkWorkflowOperationRetry:
								err = a.Retry(ctx, item.ID)
							default:
								return fmt.Errorf("bulk %s not supported yet", params.Operation)
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

func (a *BulkActivity) Retry(ctx context.Context, ID uint) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err := a.colsvc.Goa().Retry(ctx, &collection.RetryPayload{ID: ID})

	// User may have already started it manually.
	var werr *cadencesdk_gen_shared.WorkflowExecutionAlreadyStartedError
	if errors.As(err, &werr) {
		return nil
	}

	// Cadence seems to retry on cases where we'd rather give up right away,
	// e.g. when the history is unavailable.
	var yarpcerr *yarpcerrors.Status
	if errors.As(err, &yarpcerr) {
		switch yarpcerr.Code() {
		case yarpcerrors.CodeInternal:
			if strings.Contains(yarpcerr.Message(), "requested workflow history does not exist") {
				return nil
			}
		case yarpcerrors.CodeDeadlineExceeded:
			// At some point I'm seeing the timeout to be exceeded while the
			// underlying error (presumably history does not exist) is not
			// visible.
			return nil
		}
	}

	return err
}
