package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
)

type releaser func(ctx temporalsdk_workflow.Context) error

var noopReleaser = releaser(func(ctx temporalsdk_workflow.Context) error {
	return nil
})

// acquirePipeline acquires the pipeline semaphore. It returns a releaser that
// users should execute. It's safe to execute more than once or when the acquire
// operation failed (no-op).
func acquirePipeline(ctx temporalsdk_workflow.Context, colsvc collection.Service, pipelineRegistry *pipeline.Registry, pipelineName string, colID uint) (bool, releaser, error) {
	var acquired bool

	// The releaser defaults to a no-op operation, a nil value would panic.
	relfn := noopReleaser

	// Acquire the pipeline semaphore.
	{
		ctx := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
			HeartbeatTimeout:       time.Minute,
			WaitForCancellation:    false,
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    forever,
			RetryPolicy: &temporalsdk_temporal.RetryPolicy{
				MaximumAttempts: 1,
			},
		})
		if err := temporalsdk_workflow.ExecuteActivity(ctx, activities.AcquirePipelineActivityName, pipelineName).Get(ctx, nil); err != nil {
			return acquired, relfn, fmt.Errorf("error acquiring pipeline: %w", err)
		}

		acquired = true
	}

	// Create the function that releases the pipeline that we've just acquired.
	var once sync.Once
	relfn = func(ctx temporalsdk_workflow.Context) error {
		if !acquired {
			return nil
		}
		var err error
		once.Do(func() {
			ctx = withLocalActivityWithoutRetriesOpts(ctx)
			ctx, _ = temporalsdk_workflow.NewDisconnectedContext(ctx)
			err = temporalsdk_workflow.ExecuteLocalActivity(ctx, releasePipelineLocalActivity, pipelineRegistry, pipelineName).Get(ctx, nil)
			if err != nil {
				err = fmt.Errorf("error releasing pipeline semaphore: %w", err)
			}
		})
		return err
	}

	// Set in-progress status.
	{
		ctx := withLocalActivityOpts(ctx)
		err := temporalsdk_workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, colsvc, colID, time.Now().UTC()).Get(ctx, nil)
		if err != nil {
			return acquired, relfn, fmt.Errorf("error updating collection status: %w", err)
		}
	}

	return acquired, relfn, nil
}

func releasePipelineLocalActivity(ctx context.Context, registry *pipeline.Registry, pipelineName string) error {
	p, err := registry.ByName(pipelineName)
	if err != nil {
		return temporal.NewNonRetryableError(err)
	}

	p.Release()

	return nil
}
