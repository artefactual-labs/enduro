package workflow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	cadencesdk "go.uber.org/cadence"
	cadencesdk_activity "go.uber.org/cadence/activity"
	cadencesdk_testsuite "go.uber.org/cadence/testsuite"
	cadencesdk_workflow "go.uber.org/cadence/workflow"

	"github.com/artefactual-labs/enduro/internal/workflow"
)

func TestTimer(t *testing.T) {
	wts := cadencesdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context) error {
			time.Sleep(time.Minute)
			return nil
		},
		cadencesdk_activity.RegisterOptions{
			Name: "activity",
		},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx cadencesdk_workflow.Context, duration time.Duration) error {
			// Our timer implements a workflow goroutine that cancels the
			// context when the timeout is exceeded. As a result, the activity
			// should return a CanceledError.
			timer := workflow.NewTimer()
			ctx, cancel := timer.WithTimeout(ctx, duration)
			defer cancel()

			future := cadencesdk_workflow.ExecuteActivity(
				cadencesdk_workflow.WithActivityOptions(ctx, cadencesdk_workflow.ActivityOptions{
					ScheduleToStartTimeout: time.Hour,
					StartToCloseTimeout:    time.Hour,
				}),
				"activity",
			)

			err := future.Get(ctx, nil)
			if cadencesdk.IsCanceledError(err) && timer.Exceeded() {
				return fmt.Errorf("deadline exceeded: %s", duration)
			}

			return err
		},
		cadencesdk_workflow.RegisterOptions{
			Name: "workflow",
		},
	)

	const deadline = time.Duration(time.Millisecond)
	env.ExecuteWorkflow("workflow", deadline)

	// Workflow should end with an error: deadline exceeded.
	assert.True(t, env.IsWorkflowCompleted())
	assert.Equal(t, fmt.Sprintf("deadline exceeded: %s", deadline), env.GetWorkflowError().Error())
}
