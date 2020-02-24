package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"github.com/go-logr/logr"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

func acquirePipeline(ctx workflow.Context, manager *manager.Manager, pipelineName string, colID uint) error {
	// Acquire the pipeline semaphore.
	{
		ctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    time.Second * 5,
			WaitForCancellation:    true,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:    time.Second * 2,
				BackoffCoefficient: 1,
				MaximumInterval:    time.Second * 2,
				ExpirationInterval: forever,
			},
		})

		if err := workflow.ExecuteActivity(ctx, activities.AcquirePipelineActivityName, pipelineName).Get(ctx, nil); err != nil {
			return fmt.Errorf("error acquiring pipeline: %w", err)
		}
	}

	// Set in-progress status.
	{
		ctx := withLocalActivityOpts(ctx)
		err := workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, manager.Collection, colID, time.Now().UTC()).Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error updating collection status: %w", err)
		}
	}

	return nil
}

func releasePipeline(ctx workflow.Context, manager *manager.Manager, pipelineName string) error {
	// Not using a disconnected workflow because the semaphore is local and we
	// don't want to release when the process is back up. We just need a real
	// semaphore.
	ctx = withLocalActivityWithoutRetriesOpts(ctx)

	err := workflow.ExecuteLocalActivity(ctx, releasePipelineLocalActivity, manager.Logger, manager.Pipelines, pipelineName).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("error releasing pipeline semaphore: %w", err)
	}

	return nil
}

func releasePipelineLocalActivity(ctx context.Context, logger logr.Logger, registry *pipeline.Registry, pipelineName string) error {
	p, err := registry.ByName(pipelineName)
	if err != nil {
		return wferrors.NonRetryableError(err)
	}

	// It's possible that we're trying to release more than is held by
	// the semaphore, since the semaphore is volatile and local!
	defer func() {
		if err := recover(); err != nil {
			logger.WithName("releasePipelineLocalActivity").Info("Pipeline lock release failed", "err", err)
		}
	}()

	p.Release()

	return nil
}
