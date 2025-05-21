package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
)

type asyncDecision string

const (
	retry     asyncDecision = "RETRY"
	retryOnce asyncDecision = "RETRY_ONCE"
	abandon   asyncDecision = "ABANDON"
)

var ErrAsyncCompletionAbandoned = errors.New("user abandoned")

// executeActivityWithAsyncErrorHandling executes a workflow activity with a
// retry mechanism that can interface with users via asynchronous completion.
//
// It returns a future which is resolved only when the underlying activity
// completes successfully or the retry mechanism keeps working.
//
// TODO: state changes in collection could be performed via hook functions,
// generalize and convert into a struct.
func executeActivityWithAsyncErrorHandling(ctx temporalsdk_workflow.Context, colsvc collection.Service, colID uint, opts temporalsdk_workflow.ActivityOptions, act any, args ...any) temporalsdk_workflow.Future {
	future, settable := temporalsdk_workflow.NewFuture(ctx)

	temporalsdk_workflow.Go(ctx, func(ctx temporalsdk_workflow.Context) {
		retryWithPolicy := true
		retryPolicy := opts.RetryPolicy
		var attempts uint

		for {
			attempts++

			if retryWithPolicy {
				opts.RetryPolicy = retryPolicy
			} else {
				opts.RetryPolicy = nil
			}

			// Set in-progress status on new attempts - presumably coming from "pending".
			if attempts > 0 {
				_ = temporalsdk_workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, colsvc, colID, time.Time{}).Get(ctx, nil)
			}

			// Execute the activity that we're wrapping.
			activityOpts := temporalsdk_workflow.WithActivityOptions(ctx, opts)
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, act, args...).Get(activityOpts, nil)

			// We're done here if the activity did not fail.
			if err == nil {
				settable.Set(nil, nil)
				return
			}

			// Execute the activity that performs asynchronous completion.
			var decision asyncDecision
			activityOpts = withActivityOptsForAsyncCompletion(ctx)
			err = temporalsdk_workflow.ExecuteActivity(activityOpts, AsyncCompletionActivityName, colID).Get(activityOpts, &decision)
			// Asynchronous completion failed.
			if err != nil {
				if temporalsdk_temporal.IsTimeoutError(err) {
					decision = abandon
				} else {
					settable.Set(nil, err)
					return
				}
			}

			switch decision {
			case retry:
				retryWithPolicy = true
				continue
			case retryOnce:
				retryWithPolicy = false
				continue
			case abandon:
				settable.Set(nil, ErrAsyncCompletionAbandoned)
				return
			default:
				settable.Set(nil, temporalsdk_temporal.NewApplicationError("received decision is unknown", ""))
				return
			}
		}
	})

	return future
}

var AsyncCompletionActivityName = "async-completion-activity"

type AsyncCompletionActivity struct {
	colsvc collection.Service
}

func NewAsyncCompletionActivity(colsvc collection.Service) *AsyncCompletionActivity {
	return &AsyncCompletionActivity{colsvc: colsvc}
}

func (a *AsyncCompletionActivity) Execute(ctx context.Context, colID uint) (string, error) {
	info := temporalsdk_activity.GetInfo(ctx)

	if err := a.colsvc.SetStatusPending(ctx, colID, info.TaskToken); err != nil {
		return "", fmt.Errorf("error saving task token: %v", err)
	}

	return "", temporalsdk_activity.ErrResultPending
}
