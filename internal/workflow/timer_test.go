package workflow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/workflow"
)

func TestTimer(t *testing.T) {
	wts := temporalsdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context) error {
			time.Sleep(time.Minute)
			return nil
		},
		temporalsdk_activity.RegisterOptions{
			Name: "activity",
		},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx temporalsdk_workflow.Context, duration time.Duration) error {
			// Our timer implements a workflow goroutine that cancels the
			// context when the timeout is exceeded. As a result, the activity
			// should return a CanceledError.
			timer := workflow.NewTimer()
			ctx, cancel := timer.WithTimeout(ctx, duration)
			defer cancel()

			future := temporalsdk_workflow.ExecuteActivity(
				temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
					ScheduleToStartTimeout: time.Hour,
					StartToCloseTimeout:    time.Hour,
					RetryPolicy: &temporalsdk_temporal.RetryPolicy{
						MaximumAttempts: 1,
					},
				}),
				"activity",
			)

			err := future.Get(ctx, nil)
			if temporalsdk_temporal.IsCanceledError(err) && timer.Exceeded() {
				return fmt.Errorf("deadline exceeded: %s", duration)
			}

			return err
		},
		temporalsdk_workflow.RegisterOptions{
			Name: "workflow",
		},
	)

	const deadline = time.Duration(time.Millisecond)
	env.ExecuteWorkflow("workflow", deadline)

	// Workflow should end with an error: deadline exceeded.
	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.ErrorContains(t, env.GetWorkflowError(), fmt.Sprintf("deadline exceeded: %s", deadline))
}
