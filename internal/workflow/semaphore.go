package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	cadencesdk_workflow "go.uber.org/cadence/workflow"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

func acquirePipeline(ctx cadencesdk_workflow.Context, manager *manager.Manager, pipelineName string, colID uint) (bool, error) {
	var acquired bool

	// Acquire the pipeline semaphore.
	{
		ctx := cadencesdk_workflow.WithActivityOptions(ctx, cadencesdk_workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    forever,
			HeartbeatTimeout:       time.Minute,
			WaitForCancellation:    false,
		})
		if err := cadencesdk_workflow.ExecuteActivity(ctx, activities.AcquirePipelineActivityName, pipelineName).Get(ctx, nil); err != nil {
			return acquired, fmt.Errorf("error acquiring pipeline: %w", err)
		}
	}

	acquired = true

	// Set in-progress status.
	{
		ctx := withLocalActivityOpts(ctx)
		err := cadencesdk_workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, manager.Collection, colID, time.Now().UTC()).Get(ctx, nil)
		if err != nil {
			return acquired, fmt.Errorf("error updating collection status: %w", err)
		}
	}

	return acquired, nil
}

func releasePipeline(ctx cadencesdk_workflow.Context, manager *manager.Manager, pipelineName string) error {
	ctx = withLocalActivityWithoutRetriesOpts(ctx)
	ctx, _ = cadencesdk_workflow.NewDisconnectedContext(ctx)

	err := cadencesdk_workflow.ExecuteLocalActivity(ctx, releasePipelineLocalActivity, manager.Logger, manager.Pipelines, pipelineName).Get(ctx, nil)
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

	p.Release()

	return nil
}
