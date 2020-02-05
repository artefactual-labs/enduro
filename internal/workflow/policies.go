package workflow

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
)

// Cadence doesn't seem to have a concept of unlimited duration. We use this
// constant to represent a long period of time (10 years).
const forever = time.Hour * 24 * 365 * 10

// withActivityOptsForLongLivedRequest returns a workflow context with activity
// options suited for long-running activities without heartbeats
func withActivityOptsForLongLivedRequest(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: forever,
		StartToCloseTimeout:    time.Minute * 10,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Minute * 10,
			ExpirationInterval:       time.Minute * 10,
			MaximumAttempts:          5,
			NonRetriableErrorReasons: []string{wferrors.NRE},
		},
	})
}

// withActivityOptsForHeartbeatedRequest returns a workflow context with
// activity options suited for long-lived activities implementing heartbeats.
//
// Remember that Cadence passes the cancellation signal to these activities.
// The activity should not ignore cancellation signals!
//
// The activity is responsible for returning a NRE error. Otherwise it will be
// retried "forever".
func withActivityOptsForHeartbeatedRequest(ctx workflow.Context, heartbeatTimeout time.Duration) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: forever,
		StartToCloseTimeout:    forever, // Real cap is workflow.ExecutionStartToCloseTimeout.
		HeartbeatTimeout:       heartbeatTimeout,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Second * 10,
			ExpirationInterval:       forever,
			NonRetriableErrorReasons: []string{wferrors.NRE},
		},
	})
}

// withActivityOptsForRequest returns a workflow context with activity options
// suited for short-lived requests that may require multiple attempts.
func withActivityOptsForRequest(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: forever,
		StartToCloseTimeout:    time.Second * 10,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Minute * 5,
			ExpirationInterval:       time.Minute * 5,
			MaximumAttempts:          20,
			NonRetriableErrorReasons: []string{wferrors.NRE},
		},
	})
}

// withActivityOptionsForNoOp returns a workflow context with activity options
// suited for no-op activities.
//nolint:deadcode,unused
func withActivityOptsForNoOp(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: forever,
		StartToCloseTimeout:    time.Second * 10,
	})
}

// withLocalActivityOpts returns a workflow context with activity options suited
// for local and short-lived activities with a few retries.
func withLocalActivityOpts(ctx workflow.Context) workflow.Context {
	return workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:          time.Second,
			BackoffCoefficient:       2,
			MaximumInterval:          time.Minute,
			MaximumAttempts:          3,
			NonRetriableErrorReasons: []string{wferrors.NRE},
		},
	})
}

// withActivityOptsForAsyncCompletion returns a workflow context with activity
// options for local and short-lived activities that don't deserve retries.
func withLocalActivityWithoutRetriesOpts(ctx workflow.Context) workflow.Context {
	return workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
	})
}

// withActivityOptsForAsyncCompletion returns a workflow context with activity
// options suited for asynchronous completion, embracing the fact that users
// can be away from keyboard for long periods (weekends, holidays...).
func withActivityOptsForAsyncCompletion(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: forever,
		StartToCloseTimeout:    time.Hour * 24 * 7,
	})
}
