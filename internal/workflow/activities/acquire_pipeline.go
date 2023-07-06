package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	temporalsdk_activity "go.temporal.io/sdk/activity"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// AcquirePipelineActivity acquires a lock in the weighted semaphore associated
// to a particular pipeline.
type AcquirePipelineActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewAcquirePipelineActivity(pipelineRegistry *pipeline.Registry) *AcquirePipelineActivity {
	return &AcquirePipelineActivity{pipelineRegistry: pipelineRegistry}
}

func (a *AcquirePipelineActivity) Execute(ctx context.Context, pipelineName string) error {
	p, err := a.pipelineRegistry.ByName(pipelineName)
	if err != nil {
		return temporal.NewNonRetryableError(err)
	}

	errAcquirePipeline := fmt.Errorf("error acquring semaphore: busy")

	err = backoff.RetryNotify(
		func() (err error) {
			ok := p.TryAcquire()
			if !ok {
				err = errAcquirePipeline
			}

			return err
		},
		backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx),
		func(err error, duration time.Duration) {
			temporalsdk_activity.RecordHeartbeat(ctx)
		},
	)

	return err
}
