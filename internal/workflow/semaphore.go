package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	cadencesdk_workflow "go.uber.org/cadence/workflow"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type releaser func(ctx cadencesdk_workflow.Context, m *manager.Manager, pipelineName string) error

// acquirePipeline acquires the pipeline semaphore. It returns a releaser that
// users should execute. It's safe to execute more than once or when the acquire
// operation failed (no-op).
func acquirePipeline(ctx cadencesdk_workflow.Context, m *manager.Manager, pipelineName string, colID uint) (bool, releaser, error) {
	var acquired bool

	// The releaser defaults to a no-op operation, a nil value would panic.
	var relfn releaser = func(ctx cadencesdk_workflow.Context, m *manager.Manager, pipelineName string) error { return nil }

	// Acquire the pipeline semaphore.
	{
		ctx := cadencesdk_workflow.WithActivityOptions(ctx, cadencesdk_workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    forever,
			HeartbeatTimeout:       time.Minute,
			WaitForCancellation:    false,
		})
		if err := cadencesdk_workflow.ExecuteActivity(ctx, activities.AcquirePipelineActivityName, pipelineName).Get(ctx, nil); err != nil {
			return acquired, relfn, fmt.Errorf("error acquiring pipeline: %w", err)
		}

		acquired = true
	}

	// Create the function that releases the pipeline that we've just acquired.
	var once sync.Once
	relfn = func(ctx cadencesdk_workflow.Context, m *manager.Manager, pipelineName string) error {
		if !acquired {
			return nil
		}
		var err error
		once.Do(func() {
			ctx = withLocalActivityWithoutRetriesOpts(ctx)
			ctx, _ = cadencesdk_workflow.NewDisconnectedContext(ctx)
			err = cadencesdk_workflow.ExecuteLocalActivity(ctx, releasePipelineLocalActivity, m.Logger, m.Pipelines, pipelineName).Get(ctx, nil)
			if err != nil {
				err = fmt.Errorf("error releasing pipeline semaphore: %w", err)
			}
		})
		return err
	}

	// Set in-progress status.
	{
		ctx := withLocalActivityOpts(ctx)
		err := cadencesdk_workflow.ExecuteLocalActivity(ctx, setStatusInProgressLocalActivity, m.Collection, colID, time.Now().UTC()).Get(ctx, nil)
		if err != nil {
			return acquired, relfn, fmt.Errorf("error updating collection status: %w", err)
		}
	}

	return acquired, relfn, nil
}

func releasePipelineLocalActivity(ctx context.Context, logger logr.Logger, registry *pipeline.Registry, pipelineName string) error {
	p, err := registry.ByName(pipelineName)
	if err != nil {
		return wferrors.NonRetryableError(err)
	}

	p.Release()

	return nil
}
