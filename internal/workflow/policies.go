package workflow

import (
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

// We use this constant to represent a long period of time (10 years).
const forever = time.Hour * 24 * 365 * 10

// withActivityOptsForLongLivedRequest returns a workflow context with activity
// options suited for long-running activities without heartbeats
func withActivityOptsForLongLivedRequest(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 2,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute * 10,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"TemporalTimeout:StartToClose",
			},
		},
	})
}

// withActivityOptsForHeartbeatedRequest returns a workflow context with
// activity options suited for long-lived activities implementing heartbeats.
//
// Remember that Temporal passes the cancellation signal to these activities.
// The activity should not ignore cancellation signals!
//
// The activity is responsible for returning a NRE error.
func withActivityOptsForHeartbeatedRequest(ctx temporalsdk_workflow.Context, heartbeatTimeout time.Duration) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: forever,
		HeartbeatTimeout:       heartbeatTimeout,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Second * 10,
		},
	})
}

// withActivityOptsForRequest returns a workflow context with activity options
// suited for short-lived requests that may require multiple attempts.
func withActivityOptsForRequest(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout:    time.Second * 10,
		ScheduleToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    20,
		},
	})
}

// withActivityOptsForLocalAction returns a workflow context with activity
// options suited for local activities like disk operations that should not
// require a retry policy attached.
func withActivityOptsForLocalAction(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

// withLocalActivityOpts returns a workflow context with activity options suited
// for local and short-lived activities with a few retries.
func withLocalActivityOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithLocalActivityOptions(ctx, temporalsdk_workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	})
}

// withActivityOptsForAsyncCompletion returns a workflow context with activity
// options for local and short-lived activities that don't deserve retries.
func withLocalActivityWithoutRetriesOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithLocalActivityOptions(ctx, temporalsdk_workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}

// withActivityOptsForAsyncCompletion returns a workflow context with activity
// options suited for asynchronous completion, embracing the fact that users
// can be away from keyboard for long periods (weekends, holidays...).
func withActivityOptsForAsyncCompletion(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 24 * 7,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
