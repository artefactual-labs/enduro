package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
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
func executeActivityWithAsyncErrorHandling(ctx workflow.Context, colsvc collection.Service, colID uint, opts workflow.ActivityOptions, act interface{}, args ...interface{}) workflow.Future {
	future, settable := workflow.NewFuture(ctx)

	workflow.Go(ctx, func(ctx workflow.Context) {
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
				_ = workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, colsvc, colID, time.Time{}).Get(ctx, nil)
			}

			// Execute the activity that we're wrapping.
			activityOpts := workflow.WithActivityOptions(ctx, opts)
			err := workflow.ExecuteActivity(activityOpts, act, args...).Get(activityOpts, nil)

			// We're done here if the activity did not fail.
			if err == nil {
				settable.Set(nil, nil)
				return
			}

			// Execute the activity that performs asynchronous completion.
			var decision asyncDecision
			activityOpts = withActivityOptsForAsyncCompletion(ctx)
			err = workflow.ExecuteActivity(activityOpts, AsyncCompletionActivityName, colID).Get(activityOpts, &decision)

			// Asynchronous completion failed.
			if err != nil {
				if cadence.IsTimeoutError(err) {
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
				settable.Set(nil, cadence.NewCustomError("received decision is unknown"))
				return
			}
		}
	})

	return future
}

var AsyncCompletionActivityName = "async-completion-activity"

type AsyncCompletionActivity struct {
	manager *manager.Manager
}

func NewAsyncCompletionActivity(m *manager.Manager) *AsyncCompletionActivity {
	return &AsyncCompletionActivity{manager: m}
}

func (a *AsyncCompletionActivity) Execute(ctx context.Context, colID uint) (string, error) {
	info := activity.GetInfo(ctx)

	if err := a.manager.Collection.SetStatusPending(ctx, colID, info.TaskToken); err != nil {
		return "", fmt.Errorf("error saving task token: %v", err)
	}

	return "", activity.ErrResultPending
}
