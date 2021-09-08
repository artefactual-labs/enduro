package workflow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/artefactual-labs/enduro/internal/workflow"

	"github.com/stretchr/testify/assert"
	cadence "go.uber.org/cadence"
	cadenceactivity "go.uber.org/cadence/activity"
	cadencetestsuite "go.uber.org/cadence/testsuite"
	cadenceworkflow "go.uber.org/cadence/workflow"
)

func TestTimer(t *testing.T) {
	wts := cadencetestsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context) error {
			time.Sleep(time.Minute)
			return nil
		},
		cadenceactivity.RegisterOptions{
			Name: "activity",
		},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx cadenceworkflow.Context, duration time.Duration) error {
			// Our timer implements a workflow goroutine that cancels the
			// context when the timeout is exceeded. As a result, the activity
			// should return a CanceledError.
			timer := workflow.NewTimer()
			ctx, cancel := timer.WithTimeout(ctx, duration)
			defer cancel()

			future := cadenceworkflow.ExecuteActivity(
				cadenceworkflow.WithActivityOptions(ctx, cadenceworkflow.ActivityOptions{
					ScheduleToStartTimeout: time.Hour,
					StartToCloseTimeout:    time.Hour,
				}),
				"activity",
			)

			err := future.Get(ctx, nil)
			if cadence.IsCanceledError(err) && timer.Exceeded() {
				return fmt.Errorf("deadline exceeded: %s", duration)
			}

			return err
		},
		cadenceworkflow.RegisterOptions{
			Name: "workflow",
		},
	)

	const deadline = time.Duration(time.Millisecond)
	env.ExecuteWorkflow("workflow", deadline)

	// Workflow should end with an error: deadline exceeded.
	assert.True(t, env.IsWorkflowCompleted())
	assert.Equal(t, fmt.Sprintf("deadline exceeded: %s", deadline), env.GetWorkflowError().Error())
}
